package orch

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

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

type podWatcherMock struct {
	wi  watch.Interface
	err error
}

func (pwm *podWatcherMock) Watch(listOpts metav1.ListOptions) (watch.Interface, error) {
	if pwm.err != nil {
		return nil, pwm.err
	}

	if pwm.wi == nil {
		return watch.NewRaceFreeFake(), nil
	}

	return pwm.wi, nil
}
