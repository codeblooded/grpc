package orch

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"

	"github.com/golang/glog"

	"github.com/grpc/grpc/testctrl/svc/types"
)

// executorCount specifies the maximum number of sessions that should be processed concurrently.
const executorCount = 1

// Controller manages active and idle sessions, as well as, communications with the Kubernetes API.
type Controller struct {
	clientset   *kubernetes.Clientset
	queue       workqueue.Interface
	delegator   Delegator
	monitors    map[string]*Monitor
	mux         sync.Mutex
	quitWatcher chan struct{}
}

// NewController constructs a Controller instance with a Kubernetes Clientset. This allows the
// controller to communicate with the Kubernetes API.
func NewController(clientset *kubernetes.Clientset) *Controller {
	c := &Controller{
		clientset: clientset,
		queue:     workqueue.New(),
		monitors:  make(map[string]*Monitor),
	}
	return c
}

// Schedule adds a session to the controller's queue. It will remain in the queue until there are
// sufficient resources for processing and monitoring.
func (c *Controller) Schedule(s *types.Session) error {
	c.queue.Add(s)
	return nil
}

// Start spawns goroutines to monitor the Kubernetes cluster for updates and to process a limited
// number of sessions at a time.
func (c *Controller) Start() error {
	if err := c.startWatcher(); err != nil {
		return fmt.Errorf("controller start failed when starting watcher: %v", err)
	}

	c.startExecutors()
	return nil
}

// Stop safely terminates all goroutines spawned by the call to Start. It returns immediately, but
// it allows the active sessions to finish before terminating goroutines.
func (c *Controller) Stop() error {
	c.queue.ShutDown()
	close(c.quitWatcher)
	return nil
}

// startWatcher creates a goroutine which watches for all kubernetes pod events in the cluster.
func (c *Controller) startWatcher() error {
	if c.clientset == nil {
		return fmt.Errorf("cannot start workers without Kubernetes clientset")
	}

	listOpts := metav1.ListOptions{
		Watch: true,
	}
	watcher, err := c.clientset.CoreV1().Pods(v1.NamespaceDefault).Watch(listOpts)
	if err != nil {
		return fmt.Errorf("could not start a pod watcher with list options %v: %v", listOpts, err)
	}

	watcherChan := watcher.ResultChan()
	c.quitWatcher = make(chan struct{})

	go func() {
		glog.Info("watcher: listening for kubernetes pod events")

		for {
			select {
			case event := <-watcherChan:
				pod := event.Object.(*v1.Pod)
				sessionName := pod.Labels["session-name"]

				c.mux.Lock()
				if monitor := c.monitors[sessionName]; monitor != nil {
					monitor.Update(pod)
				} else {
					glog.Warningf("watcher: found pods for session %v, but no monitor", sessionName)
				}
				c.mux.Unlock()
			case <-c.quitWatcher:
				glog.Info("watcher: terminating gracefully")
				watcher.Stop()
				return
			}
		}
	}()

	return nil
}

// startExecutors create a set of goroutines. Each goroutine becomes responsible for a single
// session at a time.
func (c *Controller) startExecutors() {
	for i := 0; i < executorCount; i++ {
		info := &executorInfo{index: i}
		glog.Infof("controller: creating and starting executor[%v]", info.index)

		go func() {
			for {
				// start with clean state
				info.session = nil
				info.monitor = nil

				glog.Infof("executor[%v]: waiting for a session", info.index)
				si, quit := c.queue.Get()
				if quit {
					glog.Infof("executor[%v]: terminating gracefully", info.index)
					return
				}

				session := si.(*types.Session)
				monitor := NewMonitor()
				c.mux.Lock()
				c.monitors[session.Name()] = monitor
				c.mux.Unlock()

				glog.Infof("executor[%v]: starting work on session %v", info.index, session.Name())
				info.session = session
				info.monitor = monitor
				if err := c.execute(info); err != nil {
					glog.Infof("executor[%v]: session %v terminated: %v", info.index, info.session.Name(), err)
				}
				c.queue.Done(session)
				glog.Infof("executor[%v]: finished work on session %v", info.index, session.Name())
			}
		}()
	}
}

type executorInfo struct {
	index   int
	session *types.Session
	monitor *Monitor
}

// execute performs the provision, monitoring and teardown of a session's resources.
func (c *Controller) execute(info *executorInfo) error {
	defer c.teardown(info)

	if err := c.provision(info); err != nil {
		return fmt.Errorf("failed to provision resources with error: %v", err)
	}

	if err := c.monitorRun(info); err != nil {
		return fmt.Errorf("failed to finish run with error: %v", err)
	}

	return nil
}

// deploy creates all kubernetes resources for a component by submitting a spec.
func (c *Controller) deploy(info *executorInfo, co *types.Component) error {
	kind := strings.ToLower(co.Kind().String())

	glog.V(2).Infof("deploying %v component %v for session %v", kind, co.Name(), info.session.Name())

	depl := NewSpecBuilder(info.session, co).Deployment()
	if _, err := c.clientset.AppsV1().Deployments(v1.NamespaceDefault).Create(depl); err != nil {
		return fmt.Errorf("unable to deploy %v component %v with error: %v", kind, co.Name(), err)
	}
	return nil
}

func (c *Controller) provision(info *executorInfo) error {
	drivers := NewResources(info.session.Driver())
	if count := len(drivers); count != 1 {
		return fmt.Errorf("expected exactly 1 driver, but got %v drivers", count)
	}
	driver := drivers[0]

	servers := NewResources(info.session.ServerWorkers()...)
	if count := len(servers); count != 1 {
		return fmt.Errorf("expected exactly 1 server, but got %v servers", count)
	}
	server := servers[0]

	clients := NewResources(info.session.ClientWorkers()...)

	workers := []*Resource{server}
	workers = append(workers, clients...)
	var workerIPs []string

	for _, worker := range workers {
		info.monitor.Add(worker)

		if err := c.deploy(info, worker.Component()); err != nil {
			return err
		}

		var assignedIP bool

		for {

			if worker.Unhealthy() {
				return fmt.Errorf("component %v terminated due to unhealthy status: %v", worker.Name(), worker.Error())
			}

			if info.monitor.Unhealthy() {
				return fmt.Errorf("session %v terminating due to error in %v component: %v",
					info.session.Name(), info.monitor.ErrResource().Name(), info.monitor.Error())
			}

			if !assignedIP {
				if ip := worker.PodStatus().PodIP; len(ip) > 0 {
					assignedIP = true
					workerIPs = append(workerIPs, ip)
					glog.V(2).Infof("component %v was assigned IP address %v", worker.Name(), ip)
				}
			}
		}
	}

	if err := c.deploy(info, driver.Component()); err != nil {
		return fmt.Errorf("driver component %v could not be deployed with error: %v", driver.Name(), err)
	}

	return nil
}

func (c *Controller) monitorRun(info *executorInfo) error {
	for {
		if info.monitor.Unhealthy() {
			return fmt.Errorf("component %v is unhealthy, terminating with error: %v",
				info.monitor.ErrResource().Name(), info.monitor.Error())
		}

		if info.monitor.Done() {
			return nil
		}
	}
}

func (c *Controller) teardown(info *executorInfo) error {
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("session-name=%v", info.session.Name()),
	}

	err := c.clientset.AppsV1().Deployments(v1.NamespaceDefault).DeleteCollection(&metav1.DeleteOptions{}, listOpts)
	if err != nil {
		return fmt.Errorf("unable to teardown components with error: %v", err)
	}

	return nil
}

