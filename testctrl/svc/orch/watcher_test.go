package orch

import (
	"errors"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/watch"
)

func TestWatcherStart(t *testing.T) {
	cases := []struct {
		description string
		mock        *podWatcherMock
		shouldError bool
	}{
		{
			description: "kubernetes watch error",
			mock:        &podWatcherMock{err: errors.New("fake kubernetes error")},
			shouldError: true,
		},
		{
			description: "successful start",
			mock:        &podWatcherMock{},
			shouldError: false,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			w := NewWatcher(c.mock)

			defer w.Stop()
			err := w.Start()

			didError := err != nil
			if didError && !c.shouldError {
				t.Errorf("unexpectedly returned an error '%v'", err)
			}
			if !didError && c.shouldError {
				t.Errorf("unexpectedly returned <nil>, instead of an error")
			}
		})
	}
}

func TestWatcherStop(t *testing.T) {
	var err error

	wi := watch.NewRaceFreeFake()
	mock := &podWatcherMock{wi: wi}
	w := NewWatcher(mock)

	if err = w.Start(); err != nil {
		t.Fatalf("setup failed, Start returned error: %v", err)
	}

	sessionName := "stop-test-session"
	eventChan, err := w.Subscribe(sessionName)
	if err != nil {
		t.Fatalf("setup failed, Subscribe returned error: %v", err)
	}

	w.Stop()
	wi.Add(newPodWithSessionName(t, sessionName))
	select {
	case <-eventChan:
		t.Error("received pod event after Stop invoked")
	case <-time.After(100 * time.Millisecond):
		return
	}
}

func TestWatcherSubscribe(t *testing.T) {
	var err error
	var event *PodWatchEvent
	timeout := 100 * time.Millisecond * timeMultiplier

	w := NewWatcher(nil)
	sharedSessionName := "double-subscription"
	_, _ = w.Subscribe(sharedSessionName)
	if _, err = w.Subscribe(sharedSessionName); err == nil {
		t.Errorf("did not return error for overriding subscription")
	}

	wi := watch.NewRaceFreeFake()
	mock := &podWatcherMock{wi: wi}
	w = NewWatcher(mock)

	if err = w.Start(); err != nil {
		t.Fatalf("setup failed, Start returned error: %v", err)
	}

	sessionName1 := "session-one"
	eventChan1, err := w.Subscribe(sessionName1)
	if err != nil {
		t.Fatalf("subscribe unexpectedly returned error for %v: %v", sessionName1, err)
	}

	sessionName2 := "session-two"
	eventChan2, err := w.Subscribe(sessionName2)
	if err != nil {
		t.Fatalf("subscribe unexpectedly returned error for %v: %v", sessionName2, err)
	}

	wi.Add(newPodWithSessionName(t, sessionName1))
	wi.Add(newPodWithSessionName(t, sessionName2))

	cases := []struct {
		eventChan   <-chan *PodWatchEvent
		sessionName string
	}{
		{eventChan1, sessionName1},
		{eventChan2, sessionName2},
	}

	for _, c := range cases {
		select {
		case event = <-c.eventChan:
			if event.SessionName != c.sessionName {
				t.Errorf("an event for session %v was unexpectedly passed through a channel for session %v",
					event.SessionName, c.sessionName)
			}
		case <-time.After(timeout):
			t.Errorf("failed to receive event within time limit (%v)", timeout)
		}

		select {
		case event = <-c.eventChan:
			t.Errorf("received second event unexpectedly")
		case <-time.After(timeout):
			break // success
		}
	}
}

func TestWatcherUnsubscribe(t *testing.T) {
	var err error
	timeout := 100 * time.Millisecond * timeMultiplier

	// test an error is returned without subscription
	t.Run("no subscription", func(t *testing.T) {
		w := NewWatcher(nil)
		if err := w.Unsubscribe("non-existent"); err == nil {
			t.Errorf("did not return an error for Unsubscribe call without subscription")
		}
	})

	// test unsubscription prevents further events from being sent
	t.Run("prevents further events", func(t *testing.T) {
		wi := watch.NewRaceFreeFake()
		mock := &podWatcherMock{wi: wi}
		w := NewWatcher(mock)

		if err = w.Start(); err != nil {
			t.Fatalf("setup failed, Start returned error: %v", err)
		}

		sessionName := "session-one"
		eventChan, err := w.Subscribe(sessionName)
		if err != nil {
			t.Fatalf("subscribe unexpectedly returned error for %v: %v", sessionName, err)
		}

		if err = w.Unsubscribe(sessionName); err != nil {
			t.Fatalf("Unsubscribe unexpectedly returned error: %v", err)
		}

		wi.Add(newPodWithSessionName(t, sessionName))

		select {
		case event := <-eventChan:
			if event != nil {
				t.Errorf("received event unexpectedly: %v", event)
			}
		case <-time.After(timeout):
			break // never passed is also valid
		}
	})
}
