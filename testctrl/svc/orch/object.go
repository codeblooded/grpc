package orch

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/api/core/v1"

	"github.com/golang/glog"

	"github.com/grpc/grpc/testctrl/svc/types"
)

var ErrorContainerTerminated error
var ErrorContainerTerminating error
var ErrorContainerCrashed error
var ErrorPodFailed error

func init() {
	ErrorContainerTerminated = fmt.Errorf("container terminated")
	ErrorContainerTerminating = fmt.Errorf("container terminating")
	ErrorContainerCrashed = fmt.Errorf("container crashed")
	ErrorPodFailed = fmt.Errorf("pod failed")
}

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
	var err error
	phase := status.Phase
	if phase == v1.PodFailed {
		err = ErrorPodFailed
	}

	var cstate containerState
	if cstatuses := status.ContainerStatuses; len(cstatuses) > 0 {
		cstatus := &status.ContainerStatuses[0]
		if wcstate := cstatus.State.Waiting; wcstate != nil {
			if strings.Compare("CrashLoopBackOff", wcstate.Reason) == 0 {
				cstate = containerStateCrashWaiting
				err = ErrorContainerCrashed
			} else {
				cstate = containerStateWaiting
			}
		} else if cstatus.State.Terminated != nil {
			cstate = containerStateTerminated
			err = ErrorContainerTerminated
		} else if cstatus.LastTerminationState.Terminated != nil {
			cstate = containerStateTerminating
			err = ErrorContainerTerminating
		} else if cstatus.State.Running != nil {
			cstate = containerStateRunning
		} else {
			cstate = containerStateUnknown
		}
	}

	o.mux.Lock()
	defer o.mux.Unlock()

	psm, ok := phaseStateMap[phase]
	if !ok {
		o.health = Unknown
	}

	health, ok := psm[cstate]
	if ok {
		o.health = health
	} else {
		o.health = Unknown
	}

	o.podStatus = status
	o.err = err

	if o.component.Kind() == types.DriverComponent {
	glog.V(5).Infof("driver phase=%v health=%v err=%v", phase, o.health, o.err)
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

type containerState int

const (
	containerStateUnknown containerState = iota
	containerStateTerminated
	containerStateTerminating
	containerStateWaiting
	containerStateCrashWaiting
	containerStateRunning
)

var phaseStateMap = map[v1.PodPhase]map[containerState]Health {
	v1.PodPending: map[containerState]Health{
		containerStateUnknown: Unknown,
		containerStateCrashWaiting: Unhealthy,
		containerStateTerminated: Unhealthy,
		containerStateTerminating: Unhealthy,
	},

	v1.PodRunning: map[containerState]Health{
		containerStateUnknown: Unhealthy,
		containerStateRunning: Healthy,
		containerStateWaiting: Unhealthy,
		containerStateCrashWaiting: Unhealthy,
		containerStateTerminated: Unhealthy,
		containerStateTerminating: Unhealthy,
	},

	v1.PodSucceeded: map[containerState]Health{
		containerStateUnknown: Done,
		containerStateRunning: Done,
		containerStateWaiting: Done,
		containerStateCrashWaiting: Failed,
		containerStateTerminated: Done,
		containerStateTerminating: Done,
	},

	v1.PodFailed: map[containerState]Health{
		containerStateUnknown: Failed,
		containerStateRunning: Failed,
		containerStateWaiting: Failed,
		containerStateCrashWaiting: Failed,
		containerStateTerminated: Failed,
		containerStateTerminating: Failed,
	},

	v1.PodUnknown: map[containerState]Health{
		containerStateUnknown: Unknown,
		containerStateRunning: Unhealthy,
		containerStateWaiting: Unhealthy,
		containerStateCrashWaiting: Unhealthy,
		containerStateTerminated: Unhealthy,
		containerStateTerminating: Unhealthy,
	},
}

