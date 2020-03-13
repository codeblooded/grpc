package orch

import (
	"fmt"
	"log"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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
	quitWatcher chan struct{}
}

// NewController constructs a Controller instance with a Kubernetes Clientset. This allows the
// controller to communicate with the Kubernetes API.
func NewController(clientset *kubernetes.Clientset) *Controller {
	c := &Controller{
		clientset: clientset,
		queue: workqueue.New(),
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
				log.Printf("Watcher received an event: %v on pod: %v", event.Type, event.Object.(*v1.Pod).Name)
				c.proxy(event)
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
		index := i // define copy to prevent race with i++
		eventChan := make(chan watch.Event)
		log.Printf("Creating and starting executor %v/%v", index, executorCount)
		go func() {
			log.Printf("Executor[%v] waiting for healthy broadcast from watcher", index)

			for {
				log.Printf("Executor[%v] is waiting for a session", index)
				si, quit := c.queue.Get()
				if quit {
					log.Printf("Executor[%v] is terminating gracefully", index)
					return
				}

				s := si.(*types.Session)
				log.Printf("Executor[%v] starting work on %v", index, s.ResourceName())
				c.assign(s, eventChan)

				c.execute(s, eventChan)

				c.deassign(s)
				c.queue.Done(s)
				log.Printf("Executor[%v] completed work on %v", index, s.ResourceName())
			}
		}()
	}
}

// assign tells the controller to pass all kubernetes pod events related to a specific session
// through the specified channel.
func (c *Controller) assign(s *types.Session, eventChan chan watch.Event) {
	c.delegator.Set(s.Name(), &Denoiser{Channel: eventChan})
}

// deassign tells the controller to stop passing kubernetes pod events for a specific session
// through any channel.
func (c *Controller) deassign(s *types.Session) {
	c.delegator.Unset(s.Name())
}

// execute performs the provision, monitoring and teardown of a session's resources.
func (c *Controller) execute(s *types.Session, eventChan chan watch.Event) {
	c.deploy(s, s.Driver())
	c.teardown(s, s.Driver(), false)
}

// deploy creates all kubernetes resources for a component by submitting a spec.
func (c *Controller) deploy(s *types.Session, co *types.Component) error {
	log.Printf("Deploying %v/%v [%v]", s.ResourceName(), co.ResourceName(), co.Kind())
	depl := NewSpecBuilder(s, co).Deployment()
	if _, err := c.clientset.AppsV1().Deployments(v1.NamespaceDefault).Create(depl); err != nil {
		return fmt.Errorf("unable to deploy %v/%v [%v]: %v", s.ResourceName(), co.ResourceName(),
			co.Kind(), err)
	}
	return nil
}

// teardown deletes all kubernetes resources for a component. If propagate is true, it will delete
// all kubernetes resources for the entire session.
func (c *Controller) teardown(s *types.Session, co *types.Component, propagate bool) error {
	var labels string

	if propagate {
		log.Printf("Deleting all component deployments for %v", s.ResourceName())
		labels = fmt.Sprintf("session-name=%v", s.Name())
	} else {
		log.Printf("Deleting %v/%v [%v]", s.ResourceName(), co.ResourceName(), co.Kind())
		labels = fmt.Sprintf("component-name=%v", co.Name())
	}

	delForeground := metav1.DeletePropagationForeground

	delOpts := &metav1.DeleteOptions{
		PropagationPolicy: &delForeground,
	}

	listOpts := metav1.ListOptions{
		LabelSelector: labels,
	}

	err := c.clientset.AppsV1().Deployments(v1.NamespaceDefault).DeleteCollection(delOpts, listOpts)
	if err != nil {
		return fmt.Errorf("failed to delete with labels '%v' in %v: %v", labels, s.ResourceName(), err)
	}

	return nil
}

// proxy finds the session for an event and sends the event through the assigned channel.
func (c *Controller) proxy(e watch.Event) {
	pod := e.Object.(*v1.Pod)
	sessionName := pod.Labels["session-name"]
	if len(sessionName) != 0 {
		denoiser := c.delegator.Get(sessionName)
		if denoiser == nil {
			log.Printf("[WARNING] ORPHAN SESSION POD, REQUIRES MANUAL CLEANUP: %v", pod.Name)
			return
		}

		denoiser.Send(e)
	}
}

