package orch

import (
	"fmt"
	"log"
	"sync"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"

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
		log.Println("Watcher listening for pod events to proxy")

		for {
			select {
			case event := <-watcherChan:
				pod := event.Object.(*v1.Pod)
				sessionName := pod.Labels["session-name"]

				c.mux.Lock()
				monitor := c.monitors[sessionName]
				monitor.Update(pod)
				c.mux.Unlock()
			case <-c.quitWatcher:
				log.Println("Watcher is terminating gracefully")
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
		log.Printf("Creating and starting executor %v/%v", info.index, executorCount)

		go func() {
			log.Printf("Executor[%v] waiting for healthy broadcast from watcher", info.index)

			for {
				// start with clean state
				info.session = nil
				info.monitor = nil

				log.Printf("%v: waiting for a session", info)
				si, quit := c.queue.Get()
				if quit {
					log.Printf("%v: terminating gracefully", info)
					return
				}

				session := si.(*types.Session)
				monitor := NewMonitor()
				c.mux.Lock()
				c.monitors[session.Name()] = monitor
				c.mux.Unlock()

				log.Printf("%v: starting work on %v", info, session.ResourceName())
				info.session = session
				info.monitor = monitor
				if err := c.execute(info); err != nil {
					log.Printf("%v: failed: %v", info, err)
				}
				c.queue.Done(session)
				log.Printf("%v: finished work on %v", info, session.ResourceName())
			}
		}()
	}
}

type executorInfo struct {
	index   int
	session *types.Session
	monitor *Monitor
}

func (ei *executorInfo) String() string {
	message := fmt.Sprintf("executor[%v]", ei.index)

	if ei.session != nil {
		message = message + fmt.Sprintf(" (session: %v)", ei.session.Name())
	}

	return message
}

// execute performs the provision, monitoring and teardown of a session's resources.
func (c *Controller) execute(info *executorInfo) error {
	defer c.teardown(info)

	if err := c.provision(info); err != nil {
		return fmt.Errorf("failed to provision resources: %v", err)
	}

	if err := c.monitorRun(info); err != nil {
		return fmt.Errorf("error resulted in termination: %v", err)
	}

	return nil
}

// deploy creates all kubernetes resources for a component by submitting a spec.
func (c *Controller) deploy(info *executorInfo, co *types.Component) error {
	log.Printf("Deploying %v/%v [%v]", info.session.ResourceName(), co.ResourceName(), co.Kind())
	depl := NewSpecBuilder(info.session, co).Deployment()
	if _, err := c.clientset.AppsV1().Deployments(v1.NamespaceDefault).Create(depl); err != nil {
		return fmt.Errorf("unable to deploy %v/%v [%v]: %v", info.session.ResourceName(), co.ResourceName(),
			co.Kind(), err)
	}
	return nil
}

func (c *Controller) provision(info *executorInfo) error {
	drivers := NewResources(info.session.Driver())
	if count := len(drivers); count != 1 {
		return fmt.Errorf("sessions must have exactly 1 driver, but %v drivers were specified", count)
	}
	driver := drivers[0]

	servers := NewResources(info.session.ServerWorkers()...)
	if count := len(servers); count != 1 {
		return fmt.Errorf("sessions must have exactly 1 server component, but got %v", count)
	}
	server := servers[0]

	clients := NewResources(info.session.ClientWorkers()...)

	workers := []*Resource{server}
	workers = append(workers, clients...)
	var workerIPs []string

	for _, worker := range workers {
	//	if err := c.deploy(info, worker); err != nil {
	//		return err
	//	}

		for {
			if worker.Unhealthy() {
				return fmt.Errorf("%v terminated due to unhealthy status: %v", worker, worker.Error())
			}

			if info.monitor.Unhealthy() {
				return fmt.Errorf("terminating due to error in available test dependency: %v",
					worker, info.monitor.Error())
			}

			if ip := worker.PodStatus().PodIP; len(ip) > 0 {
				workerIPs = append(workerIPs, ip)
			}
		}
	}

	_ = driver
//	if err := c.deploy(info, driver); err != nil {
//		return fmt.Errorf("%v could not be deployed: %v", driver, err)
//	}

	return nil
}

func (c *Controller) monitorRun(info *executorInfo) error {
	for {
		if info.monitor.Unhealthy() {
			return fmt.Errorf("terminating due to unhealthy state: %v", info.monitor.Error())
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
		return fmt.Errorf("failed to delete kubernetes resources")
	}

	return nil
}

