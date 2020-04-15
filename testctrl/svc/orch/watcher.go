package orch

import (
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type Watcher struct {
	eventChans map[string]chan *PodWatchEvent
	quit       chan struct{}
	mux        sync.Mutex
	wi         watch.Interface
}

func NewWatcher() *Watcher {
	return &Watcher{
		eventChans: make(map[string]chan *PodWatchEvent),
		quit:       make(chan struct{}),
	}
}

func (w *Watcher) Start(pw podWatcher) error {
	wi, err := pw.Watch(metav1.ListOptions{Watch: true})
	if err != nil {
		return fmt.Errorf("could not start watcher: %v", err)
	}

	w.wi = wi
	go w.watch()
	return nil
}

func (w *Watcher) Stop() {
	close(w.quit)
	w.wi.Stop()
}

func (w *Watcher) Subscribe(sessionName string) (<-chan *PodWatchEvent, error) {
	w.mux.Lock()
	defer w.mux.Unlock()

	_, exists := w.eventChans[sessionName]
	if exists {
		return nil, fmt.Errorf("session %v already has a follower", sessionName)
	}

	eventChan := make(chan *PodWatchEvent)
	w.eventChans[sessionName] = eventChan
	return eventChan, nil
}

func (w *Watcher) Unsubscribe(sessionName string) error {
	w.mux.Lock()
	defer w.mux.Unlock()

	eventChan, exists := w.eventChans[sessionName]
	if !exists {
		return fmt.Errorf("cannot unfollow session %v, it does not have a follower", sessionName)
	}

	close(eventChan)
	delete(w.eventChans, sessionName)
	return nil
}

func (w *Watcher) watch() {
	glog.Infoln("watcher: listening for pod events")

	for {
		select {
		case wiEvent := <-w.wi.ResultChan():
			obj := wiEvent.Object
			if obj == nil {
				goto exit
			}

			pod := obj.(*corev1.Pod)
			sessionName, ok := pod.Labels["session-name"]
			if !ok {
				continue // must be a pod that is not for testing
			}

			health, err := w.diagnose(pod)
			event := &PodWatchEvent{
				SessionName:   sessionName,
				ComponentName: pod.Labels["component-name"],
				Health:        health,
				Error:         err,
				Pod:           pod,
				PodIP:         pod.Status.PodIP,
			}
			w.publish(event)
		case <-w.quit:
			goto exit
		}
	}

exit:
	glog.Infof("watcher: terminated gracefully")
}

func (w *Watcher) publish(event *PodWatchEvent) {
	w.mux.Lock()
	defer w.mux.Unlock()

	eventChan := w.eventChans[event.SessionName]
	eventChan <- event
}

func (w *Watcher) diagnose(pod *corev1.Pod) (Health, error) {
	status := pod.Status

	if count := len(status.ContainerStatuses); count != 1 {
		return NotReady, fmt.Errorf("pod has %v container statuses, expected 1", count)
	}
	containerStatus := status.ContainerStatuses[0]

	terminationState := containerStatus.LastTerminationState.Terminated
	if terminationState == nil {
		terminationState = containerStatus.State.Terminated
	}

	if terminationState != nil {
		if terminationState.ExitCode == 0 {
			return Succeeded, nil
		}

		return Failed, fmt.Errorf("container terminated unexpectedly: [%v] %v",
			terminationState.Reason, terminationState.Message)
	}

	if waitingState := containerStatus.State.Waiting; waitingState != nil {
		if strings.Compare("CrashLoopBackOff", waitingState.Reason) == 0 {
			return Failed, fmt.Errorf("container crashed: [%v] %v",
				waitingState.Reason, waitingState.Message)
		}
	}

	if containerStatus.State.Running != nil {
		return Ready, nil
	}

	return Unknown, nil
}

type PodWatchEvent struct {
	SessionName   string
	ComponentName string
	Pod           *corev1.Pod
	PodIP         string
	Health        Health
	Error         error
}
