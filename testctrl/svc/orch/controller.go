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
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/golang/glog"

	"github.com/grpc/grpc/testctrl/svc/types"
)

// executorCount specifies the maximum number of sessions that should be processed concurrently.
const executorCount = 1

// Controller manages active and idle sessions and their interactions with the Kubernetes API.
type Controller struct {
	pcd         podCreateDeleter
	pw          podWatcher
	nl          NodeLister
	watcher     *Watcher
	waitQueue   *queue
	activeCount int
	mux         sync.Mutex
	running     bool
	wg          sync.WaitGroup
}

// NewController constructs a Controller instance with a Kubernetes Clientset. This allows the
// controller to communicate with the Kubernetes API.
func NewController(clientset kubernetes.Interface) *Controller {
	coreV1Interface := clientset.CoreV1()
	podInterface := coreV1Interface.Pods(corev1.NamespaceDefault)

	c := &Controller{
		pcd:       podInterface,
		pw:        podInterface,
		nl:        coreV1Interface.Nodes(),
		watcher:   NewWatcher(podInterface),
	}
	return c
}

// Schedule adds a session to the controller's queue. It will remain in the queue until there are
// sufficient resources for processing and monitoring. An error is returned if the session is nil,
// or the controller was not started.
func (c *Controller) Schedule(s *types.Session) error {
	if s == nil {
		return fmt.Errorf("cannot schedule a <nil> session")
	}

	if c.Stopped() {
		return fmt.Errorf("controller was not started, cannot schedule sessions")
	}

	c.waitQueue.Enqueue(s)
	return nil
}

// Start spawns goroutines to monitor the Kubernetes cluster for updates and to process a limited
// number of sessions at a time. An error is returned if there are problems within the goroutines,
// such as the inability to connect to the Kubernetes API.
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

	go c.waitAndAssign() // wait for sessions and create executors for them when possible
	return nil
}

// Stop attempts to terminate all orchestration goroutines spawned by a call to Start. It waits for
// executors to exit. Then, it kills the kubernetes watcher.
//
// If the timeout is reached before executors exit, an error is returned. The kubernetes watcher is
// still terminated. Any sessions running on the unterminated executors will likely fail.
func (c *Controller) Stop(timeout time.Duration) error {
	var err error

	c.mux.Lock()
	c.running = false
	c.mux.Unlock()

	select {
	case <-c.awaitExecutorsDone():
		glog.Infof("controller: executors safely exited")
	case <-time.After(timeout):
		glog.Warning("controller: unable to wait for executors to safely exit, timed out")
		err = fmt.Errorf("executors did not safely exit before timeout")
	}

	c.watcher.Stop()
	return err
}

func (c *Controller) waitAndAssign() {
	for {
		activeCount, quit := c.activeExecutors()

		if quit {
			return
		}

		if activeCount >= executorCount {
			time.Sleep(5 * time.Second)
			continue // loop until some executors are released
		}

	retryDequeue:
		session := c.waitQueue.Dequeue()
		if session == nil {
			time.Sleep(5 * time.Second)
			goto retryDequeue // loop until machines are available
		}

		executor := newExecutor(0, c.pcd, c.watcher)
		glog.Infof("controller: creating and started executor[%v]", executor.name)
		c.incExecutors()

		go func() {
			defer c.decExecutors()

			if err := executor.Execute(session); err != nil {
				glog.Infof("%v", err)
			}
		}()
	}
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

func (c *Controller) incExecutors() {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.activeCount++
	c.wg.Add(1)
}

func (c *Controller) decExecutors() {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.activeCount--
	c.wg.Done()
}

func (c *Controller) awaitExecutorsDone() chan struct{} {
	done := make(chan struct{})

	go func() {
		c.wg.Wait()
		close(done)
	}()

	return done
}

func (c *Controller) Stopped() bool {
	c.mux.Lock()
	defer c.mux.Unlock()
	return !c.running
}

func (c *Controller) activeExecutors() (count int, quit bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.activeCount, !c.running
}
