// Copyright 2020 gRPC authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orch

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/grpc/grpc/testctrl/svc/store"
	"github.com/grpc/grpc/testctrl/svc/types"
)

// executorCount specifies the maximum number of sessions that should be processed concurrently.
const executorCount = 1

// Controller serves as the coordinator for orchestrating sessions. It manages active and idle
// sessions, as well as, interactions with Kubernetes through a set of a internal types.
type Controller struct {
	store           store.Store
	pcd             podCreateDeleter
	pw              podWatcher
	nl              NodeLister
	watcher         *Watcher
	waitQueue       *queue
	activeCount     int
	running         bool
	wg              sync.WaitGroup
	mux             sync.Mutex
	newExecutorFunc func() Executor
}

// NewController creates a controller using a Kubernetes clientset and a store. The clientset allows
// the controller to interact with Kubernetes. The store is used to report significant orchestration
// events, so progress can be reported. A nil clientset will result in an error.
func NewController(clientset kubernetes.Interface, store store.Store) (*Controller, error) {
	if clientset == nil {
		return nil, errors.New("cannot create controller from nil kubernetes clientset")
	}

	coreV1Interface := clientset.CoreV1()
	podInterface := coreV1Interface.Pods(corev1.NamespaceDefault)

	c := &Controller{
		pcd:     podInterface,
		pw:      podInterface,
		nl:      coreV1Interface.Nodes(),
		watcher: NewWatcher(podInterface),
		store:   store,
	}

	c.newExecutorFunc = func() Executor {
		return newKubeExecutor(0, c.pcd, c.watcher, c.store)
	}

	return c, nil
}

// Schedule adds a session to the list of sessions waiting to run. An error is returned if the
// session is nil, or the controller was not started.
func (c *Controller) Schedule(s *types.Session) error {
	if s == nil {
		return fmt.Errorf("cannot schedule a <nil> session")
	}

	if c.Stopped() {
		return fmt.Errorf("controller was not started, cannot schedule sessions")
	}

	c.waitQueue.Enqueue(s)
	if c.store != nil {
		c.store.StoreEvent(s.Name, &types.Event{
			SubjectName: s.Name,
			Kind:        types.QueueEvent,
			Time:        time.Now(),
		})
	}
	return nil
}

// Start prepares a controller for session scheduling. It creates or configures a set of threads and
// types to queue sessions, assign sessions to threads and communicate with kubernetes.
//
// An error is returned if there are problems preparing a queue or setting up a watcher to monitor
// Kubernetes events.
func (c *Controller) Start() error {
	c.mux.Lock()
	c.running = true
	c.mux.Unlock()

	waitQueue, err := c.setupQueue()
	if err != nil {
		return fmt.Errorf("controller start failed when setting up queue: %v", err)
	}
	c.waitQueue = waitQueue

	if err = c.watcher.Start(); err != nil {
		return fmt.Errorf("controller start failed when starting watcher: %v", err)
	}

	go c.loop()
	return nil
}

// Stop attempts to terminate all orchestration threads spawned by a call to Start. It waits for a
// graceful shutdown until for a specified timeout.
//
// If the timeout is reached before shutdown, an improper shutdown will occur. This may result in
// unpredictable states for running sessions and their resources. To signal these potential issues,
// an error is returned when this occurs.
//
// If Start was not called prior to Stop, there will be no adverse effects and nil will be returned.
func (c *Controller) Stop(timeout time.Duration) error {
	defer c.watcher.Stop()

	c.mux.Lock()
	c.running = false
	c.mux.Unlock()

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		glog.Infof("controller: executors safely exited")
	case <-time.After(timeout):
		glog.Warning("controller: unable to wait for executors to safely exit, timed out")
		return fmt.Errorf("executors did not safely exit before timeout")
	}

	return nil
}

// Stopped returns true if the controller is not running. This indicates that either Start has not
// been invoked or Stop has been invoked.
func (c *Controller) Stopped() bool {
	c.mux.Lock()
	defer c.mux.Unlock()
	return !c.running
}

func (c *Controller) decExecutors() {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.activeCount--
	c.wg.Done()
}

func (c *Controller) incExecutors() {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.activeCount++
	c.wg.Add(1)
}

func (c *Controller) loop() {
	for {
		session, quit := c.next()
		if quit {
			return
		}

		if session == nil {
			time.Sleep(5 * time.Second)
			continue // retry
		}

		c.spawnExecutor(session)
	}
}

func (c *Controller) next() (session *types.Session, quit bool) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if !c.running {
		return nil, true
	}

	if c.activeCount > executorCount {
		return nil, false
	}

	return c.waitQueue.Dequeue(), false
}

func (c *Controller) setupQueue() (*queue, error) {
	pools, err := FindPools(c.nl)
	if err != nil {
		return nil, err
	}

	rm := NewReservationManager()
	var poolNames []string

	for name, pool := range pools {
		poolNames = append(poolNames, name)
		rm.AddPool(pool)
	}

	glog.Infof("discovered pools: %v", poolNames)
	return newQueue(rm), nil
}

func (c *Controller) spawnExecutor(session *types.Session) {
	executor := c.newExecutorFunc()
	glog.Infof("controller: creating and started an executor")
	c.incExecutors()

	go func() {
		defer c.decExecutors()

		if err := executor.Execute(session); err != nil {
			glog.Infof("%v", err)
		}
	}()
}
