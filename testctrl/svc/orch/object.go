package orch

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/api/core/v1"

	"github.com/grpc/grpc/testctrl/svc/types"
)

// Object is a Component coupled with its status, health, and kubernetes pod information. It is
// designed to be used internally. It should be instantiated with the NewObjects function. All
// methods are thread-safe.
type Object struct {
	component *types.Component
	podStatus v1.PodStatus
	health    Health
	mux       sync.Mutex
	err       error
}

// NewObjects is a constructor for an object that takes components as variadic arguments. For each
// component, it will return exactly one Object instance in the returned slice.
func NewObjects(cs ...*types.Component) []*Object {
	var objects []*Object

	for _, component := range cs {
		objects = append(objects, &Object{
			component: component,
		})
	}

	return objects
}

// Name provides convenient access to the component name.
func (o *Object) Name() string {
	return o.component.Name()
}

// Component is the component instance that the object wraps.
func (o *Object) Component() *types.Component {
	return o.component
}

// Update modifies a component's health and status by looking for errors in a kubernetes PodStatus.
func (o *Object) Update(status v1.PodStatus) {
	o.mux.Lock()
	defer o.mux.Unlock()

	o.podStatus = status

	for _, cstatus := range status.ContainerStatuses {
		cstate := cstatus.State
		if cstatus.State.Terminated != nil || cstatus.LastTerminationState.Terminated != nil {
			o.health = Unhealthy
			o.err = fmt.Errorf("docker container has terminated for component %v, unable to recover", o.Name())
			return
		}

		if waitingState := cstate.Waiting; waitingState != nil {
			if strings.Compare("CrashLoopBackOff", waitingState.Reason) == 0 {
				o.health = Unhealthy
				o.err = fmt.Errorf("docker container has entered a crash loop for component %v", o.Name())
				return
			}
		}

		if cstatus.State.Running != nil {
			o.health = Healthy
		}
	}

	switch status.Phase {
	case v1.PodPending:
		o.health = Unknown
	case v1.PodRunning:
		o.health = Healthy
	case v1.PodSucceeded:
		o.health = Done
	case v1.PodUnknown:
		o.health = Unhealthy
		o.err = fmt.Errorf("pod for component %v has entered an unknown phase with message: %v", o.Name(), status.Message)
	case v1.PodFailed:
		o.health = Unhealthy
		o.err = fmt.Errorf("pod for component %v has failed with message: %v", o.Name(), status.Message)
	}
}

// Health returns the health value that is currently affiliated with the object.
func (o *Object) Health() Health {
	o.mux.Lock()
	defer o.mux.Unlock()
	return o.health
}

// Error returns any error that caused the object to be unhealthy.
func (o *Object) Error() error {
	o.mux.Lock()
	defer o.mux.Unlock()
	return o.err
}

// PodStatus returns the kubernetes PodStatus object which was last supplied to the Update function.
func (o *Object) PodStatus() v1.PodStatus {
	o.mux.Lock()
	defer o.mux.Unlock()
	return o.podStatus
}

