package orch

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/watch"
)

// DelegateMap allows a single channel to listen to all pod events that are affiliated with a
// session. It is a thread-safe.
type Delegator struct {
	eventMap sync.Map
}

// SetDelegate sets the specified channel to receive pod events for a specific session.
func (d *Delegator) Set(sessionName string, denoiser *Denoiser) {
	d.eventMap.Store(sessionName, denoiser)
}

// UnsetDelegate closes a specific channel and removes its affiliation with a specific session.
func (d *Delegator) Unset(sessionName string) {
	d.eventMap.Delete(sessionName)
}

func (d *Delegator) Get(sessionName string) *Denoiser {
	eci, exists := d.eventMap.Load(sessionName)
	if !exists {
		return nil
	}

	return eci.(*Denoiser)
}

// Denoiser combats excessive channel writes by accumulating events over a certain timeout. Once
// the timeout is reached, only the final event is passed through the channel.
//
// An instance should be created using a struct literal with a channel. Once instantiated, a
// denoiser should never be copied.
//
// Denoisers rely on a singular, short-lived go-routine that spawns during the first call to Send
// after a timeout expiration. This go-routine lives for the length of the timeout, then it
// terminates after sending the final event of the same type through the channel.
type Denoiser struct {
	// Channel is the event channel where noise should be reduced.
	Channel chan watch.Event

	// Timeout is the amount of time that the denoiser should wait for a new event before
	// sending the most recent one over the channel.
	Timeout time.Duration

	// heldEvents is the stack of events that have passed since the last timeout.
	heldEvents []watch.Event

	// heldEventsMux guards the heldEvents stack, making it thread-safe.
	heldEventsMux sync.Mutex

	// waiting indicates that there is an active go-routine.
	waiting bool

	// waitingMux guards the waiting boolean if Send is called from multiple goroutines.
	waitingMux sync.Mutex
}

// Send lossily sends an event through a channel. If another event is passed to Send before the
// timeout is reached, the prior event is dropped. Calls to Send are thread-safe.
func (den *Denoiser) Send(e watch.Event) {
	den.push(e)

	if !den.Waiting() {
		go func() {
			den.setWaiting(true)
			defer den.setWaiting(false)

			if den.Timeout == 0 {
				den.Timeout = 3*time.Second
			}

			time.Sleep(den.Timeout)

			den.Channel <- den.pop()
			den.clear()
		}()
	}
}

func (den *Denoiser) push(e watch.Event) {
	den.heldEventsMux.Lock()
	defer den.heldEventsMux.Unlock()
	den.heldEvents = append(den.heldEvents, e)
}

func (den *Denoiser) pop() watch.Event {
	den.heldEventsMux.Lock()
	defer den.heldEventsMux.Unlock()
	return den.heldEvents[len(den.heldEvents)-1]
}

func (den *Denoiser) clear() {
	den.heldEventsMux.Lock()
	defer den.heldEventsMux.Unlock()
	den.heldEvents = []watch.Event{}
}

func (den *Denoiser) setWaiting(b bool) {
	den.waitingMux.Lock()
	defer den.waitingMux.Unlock()
	den.waiting = b
}

func (den *Denoiser) Waiting() bool {
	den.waitingMux.Lock()
	defer den.waitingMux.Unlock()
	return den.waiting
}

