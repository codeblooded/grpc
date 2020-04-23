package orch

import (
	"fmt"
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	corev1Fake "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/grpc/grpc/testctrl/svc/types"
)

// testContainerImage is a default container image supplied during component construction in tests.
const testContainerImage = "example:latest"

// limitlessTracker allows queue to operate as only a FIFO-queue for easier testing.
type limitlessTracker struct{}

func (lt limitlessTracker) Reserve(session *types.Session) error {
	return nil
}

func (lt limitlessTracker) Unreserve(session *types.Session) error {
	return nil
}

// makeSessions creates the specified number of Session instances. These instances do not have
// components or a scenario.
func makeSessions(t *testing.T, n int) []*types.Session {
	t.Helper()
	var sessions []*types.Session
	for i := 0; i < n; i++ {
		sessions = append(sessions, types.NewSession(nil, nil, nil))
	}
	return sessions
}

// makeWorkers creates a slice of Component instances. The slice will contain exactly 1 server and
// n-1 clients. If a pool is specified, their PoolName field will be assigned to it.
func makeWorkers(t *testing.T, n int, pool *string) []*types.Component {
	t.Helper()
	var components []*types.Component

	if n < 1 {
		return components
	}

	components = append(components, types.NewComponent(testContainerImage, types.ServerComponent))

	for i := n - 1; i > 0; i-- {
		components = append(components, types.NewComponent(testContainerImage, types.ClientComponent))
	}

	if pool != nil {
		for _, c := range components {
			c.PoolName = *pool
		}
	}

	return components
}

// newPodWithSessionName creates an empty kubernetes Pod object. This object is assigned a
// "session-name" label with the specified value.
func newPodWithSessionName(t *testing.T, name string) *corev1.Pod {
	t.Helper()
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"session-name": name,
			},
		},
	}
}

// strUnwrap unwraps a string pointer, returning the string if it is not nil. Otherwise, it returns
// a string with the text "<nil>". This avoids dereference issues in tests.
func strUnwrap(str *string) string {
	val := "<nil>"
	if str != nil {
		return *str
	}

	return val
}

type fakeWatcher struct {
	events chan watch.Event
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{
		events: make(chan watch.Event),
	}
}

func (fw *fakeWatcher) Stop() {
	close(fw.events)
}

func (fw *fakeWatcher) ResultChan() <-chan watch.Event {
	return fw.events
}

func (fw *fakeWatcher) send(pod *corev1.Pod, eventType watch.EventType) {
	fw.events <- watch.Event{
		Type:   eventType,
		Object: pod,
	}
}

type kubeSimulator struct {
	*corev1Fake.FakePods

	// mux guards the pods from multiple threads
	mux sync.Mutex

	// pods is the map of pods that should be running. Name is the key.
	pods map[string]*corev1.Pod

	// fakeWatcher is a fake that implements watch.Interface, allowing a watcher to be tested.
	watcher *fakeWatcher
}

func newKubeSimulator() *kubeSimulator {
	return &kubeSimulator{
		pods:    make(map[string]*corev1.Pod),
		watcher: newFakeWatcher(),
	}
}

func (ks *kubeSimulator) Create(pod *corev1.Pod) (*corev1.Pod, error) {
	ks.mux.Lock()
	defer ks.mux.Unlock()

	ks.pods[pod.ObjectMeta.Name] = pod
	ks.watcher.send(pod, watch.Added)
	return pod, nil
}

func (ks *kubeSimulator) DeleteCollection(_ *metav1.DeleteOptions, _ metav1.ListOptions) error {
	ks.mux.Lock()
	defer ks.mux.Unlock()

	for k, v := range ks.pods {
		ks.watcher.send(v, watch.Deleted)
		delete(ks.pods, k)
	}
	return nil
}

func (ks *kubeSimulator) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	return ks.watcher, nil
}

func (ks *kubeSimulator) setStatus(component *types.Component, status corev1.ContainerStatus) error {
	ks.mux.Lock()
	defer ks.mux.Unlock()

	pod, ok := ks.pods[component.Name]
	if !ok {
		return fmt.Errorf("pod %v is not running on cluster", component.Name)
	}

	pod.Status.ContainerStatuses = []corev1.ContainerStatus{status}
	if status.State.Running != nil {
		pod.Status.PodIP = "127.0.0.1"
	}

	ks.watcher.send(pod, watch.Modified)
	return nil
}

type containerStatus struct{}

var ContainerStatus = containerStatus{}

func (cs containerStatus) terminated(exitCode int32) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		LastTerminationState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: exitCode,
			},
		},
	}
}

func (cs containerStatus) terminating(exitCode int32) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		State: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: exitCode,
			},
		},
	}
}

func (cs containerStatus) crashed(exitCode int32) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		State: corev1.ContainerState{
			Waiting: &corev1.ContainerStateWaiting{
				Reason: "CrashLoopBackOff",
			},
		},
	}
}

func (cs containerStatus) running() corev1.ContainerStatus {
	return corev1.ContainerStatus{
		State: corev1.ContainerState{
			Running: &corev1.ContainerStateRunning{},
		},
	}
}
