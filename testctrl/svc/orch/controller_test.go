package orch

import (
	"errors"
	"testing"
	"time"

	"github.com/grpc/grpc/testctrl/svc/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestControllerSchedule(t *testing.T) {
	cases := []struct {
		description string
		session     *types.Session
		start       bool
		shouldError bool
	}{
		{
			description: "session nil",
			session:     nil,
			start:       true,
			shouldError: true,
		},
		{
			description: "without controller start",
			session:     makeSessions(t, 1)[0],
			start:       false,
			shouldError: true,
		},
		{
			description: "valid session and controller start",
			session:     makeSessions(t, 1)[0],
			start:       true,
			shouldError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			controller := NewController(fake.NewSimpleClientset())
			executor := &executorMock{}
			controller.newExecutorFunc = func() Executor {
				return executor
			}

			if tc.start {
				controller.Start()
				defer controller.Stop(0)
			}

			err := controller.Schedule(tc.session)
			if tc.shouldError && err == nil {
				t.Errorf("did not return an expected error")
			} else if !tc.shouldError {
				if err != nil {
					t.Fatalf("unexpected error returned: %v", err)
				}

				time.Sleep(100 * time.Millisecond * timeMultiplier)
				got := executor.session()
				if got == nil {
					t.Fatalf("expected executor to receive session %v, but it did not", tc.session.Name)
				}
				if got.Name != tc.session.Name {
					t.Fatalf("expected executor to receive session %v, but got %v", tc.session.Name, got.Name)
				}
			}
		})
	}
}

func TestControllerStart(t *testing.T) {
	t.Run("sets running state", func(t *testing.T) {
		controller := NewController(fake.NewSimpleClientset())
		controller.Start()
		defer controller.Stop(0)
		if controller.Stopped() {
			t.Errorf("Stopped unexpectedly returned true after starting controller")
		}
	})

	cases := []struct {
		description string
		mockNL      *nodeListerMock
		mockPW      *podWatcherMock
		shouldError bool
	}{
		{
			description: "setup queue error",
			mockNL:      &nodeListerMock{err: errors.New("fake kubernetes error")},
			shouldError: true,
		},
		{
			description: "setup watcher error",
			mockPW:      &podWatcherMock{err: errors.New("fake kubernetes error")},
			shouldError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			controller := NewController(fake.NewSimpleClientset())
			controller.waitQueue = newQueue(limitlessTracker{})

			if tc.mockNL != nil {
				controller.nl = tc.mockNL
			}

			if tc.mockPW != nil {
				controller.pw = tc.mockPW
				controller.watcher = NewWatcher(tc.mockPW)
			}

			err := controller.Start()
			if tc.shouldError && err == nil {
				t.Errorf("did not return an expected error")
			} else if !tc.shouldError && err != nil {
				t.Errorf("unexpected error returned: %v", err)
			}
		})
	}
}

func TestControllerStop(t *testing.T) {
	cases := []struct {
		description          string
		runningExecutorCount int
		executorTimeout      time.Duration
		stopTimeout          time.Duration
		shouldError          bool
	}{
		{
			description:          "no executors",
			runningExecutorCount: 0,
			stopTimeout:          200 * time.Millisecond,
			shouldError:          false,
		},
		{
			description:          "one executor finishes",
			runningExecutorCount: 1,
			executorTimeout:      100 * time.Millisecond,
			stopTimeout:          1 * time.Second,
			shouldError:          false,
		},
		{
			description:          "one executor exheeds timeout",
			runningExecutorCount: 1,
			executorTimeout:      1 * time.Second,
			stopTimeout:          100 * time.Millisecond,
			shouldError:          true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			controller := NewController(fake.NewSimpleClientset())
			controller.running = true
			controller.waitQueue = newQueue(limitlessTracker{})

			var executors []Executor
			for i := 0; i < tc.runningExecutorCount; i++ {
				executors = append(executors, &executorMock{
					sideEffect: func() {
						time.Sleep(tc.executorTimeout * timeMultiplier)
					},
				})
			}

			index := -1
			controller.newExecutorFunc = func() Executor {
				index++
				return executors[index]
			}

			sessions := makeSessions(t, tc.runningExecutorCount)
			for _, session := range sessions {
				controller.Schedule(session)
			}

			go controller.loop()
			time.Sleep(200 * time.Millisecond * timeMultiplier)
			err := controller.Stop(tc.stopTimeout)
			if tc.shouldError && err == nil {
				t.Errorf("executors unexpectedly finished before timeout")
			} else if !tc.shouldError && err != nil {
				t.Errorf("timeout unexpectedly reachedbefore executors done signal")
			}

			// try to schedule session after stopping
			session := makeSessions(t, 1)[0]
			if err = controller.Schedule(session); err == nil {
				t.Errorf("scheduling a session did not return an error after stop invoked")
			}
		})
	}
}
