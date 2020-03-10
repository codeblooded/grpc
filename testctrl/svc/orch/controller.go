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

const executorCount = 1

type consoleEventRecorder struct{}

func (c consoleEventRecorder) Record(e types.Event) {
	log.Printf("controller: %s %s: %v", e.Kind, e.SubjectName, e.Message)
}

type Controller struct {
	clientset   *kubernetes.Clientset
	queue       workqueue.Interface
	delegator   Delegator
	quitWatcher chan struct{}
}

func NewController() *Controller {
	c := &Controller{
		queue: workqueue.New(),
	}
	return c
}

func (c *Controller) SetClientset(cls *kubernetes.Clientset) {
	c.clientset = cls
}

func (c *Controller) Enqueue(s *types.Session) {
	c.queue.Add(s)
	//c.recorder.Record(types.NewEvent(s, types.QueueEvent, "queued with %d sessions ahead", c.queue.Len()))
}

func (c *Controller) Start() error {
	ready, err := c.StartWatcher()
	if err != nil {
		return fmt.Errorf("controller start failed when starting watcher: %v", err)
	}

	<-ready
	c.StartExecutors()
	return nil
}

func (c *Controller) StartWatcher() (ready chan struct{}, err error) {
	if c.clientset == nil {
		return nil, fmt.Errorf("cannot start workers without Kubernetes clientset")
	}

	listOpts := metav1.ListOptions{
		Watch: true,
	}
	watcher, err := c.clientset.CoreV1().Pods(v1.NamespaceDefault).Watch(listOpts)
	if err != nil {
		return nil, fmt.Errorf("could not start a pod watcher with list options %v: %v", listOpts, err)
	}

	watcherChan := watcher.ResultChan()
	c.quitWatcher = make(chan struct{})
	ready = make(chan struct{})

	go func() {
		log.Println("Watcher listening for pod events to proxy")
		close(ready)

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

	return ready, nil
}

func (c *Controller) StartExecutors() {
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

func (c *Controller) Stop() {
	log.Println("Sending stop signal to watcher and executors")
	close(c.quitWatcher)
	c.queue.ShutDown()
	time.Sleep(3 * time.Second) // allow 3 seconds for log flush
}

func (c *Controller) assign(s *types.Session, eventChan chan watch.Event) {
	c.delegator.Set(s.Name(), &Denoiser{Channel: eventChan})
}

func (c *Controller) deassign(s *types.Session) {
	c.delegator.Unset(s.Name())
}

func (c *Controller) execute(s *types.Session, eventChan chan watch.Event) {
	c.deploy(s, s.Driver())

	for {
		select {
		case event := <-eventChan:
			log.Printf("denoiser worked with event kind: %v", event.Type)
		case <-time.After(10*time.Second):
			goto breakpoint
		}
	}

breakpoint:
	log.Printf("Sleeping for 30 seconds to make it look like a test ran :)")
	time.Sleep(30 * time.Second)

	c.teardown(s, s.Driver(), false)
}

func (c *Controller) deploy(s *types.Session, co *types.Component) error {
	log.Printf("Deploying %v/%v [%v]", s.ResourceName(), co.ResourceName(), co.Kind())
	depl := NewSpecBuilder(s, co).Deployment()
	if _, err := c.clientset.AppsV1().Deployments(v1.NamespaceDefault).Create(depl); err != nil {
		return fmt.Errorf("unable to deploy %v/%v [%v]: %v", s.ResourceName(), co.ResourceName(),
			co.Kind(), err)
	}
	return nil
}

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

