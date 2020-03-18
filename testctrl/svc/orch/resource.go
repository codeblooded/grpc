package orch

import (
	"fmt"
	"strings"
	"sync"

	"k8s.io/api/core/v1"

	"github.com/grpc/grpc/testctrl/svc/types"
)

type Resource struct {
	name      string
	component *types.Component
	podStatus v1.PodStatus
	ready     bool
	unhealthy bool
	done      bool
	mux       sync.Mutex
	err       error
}

func NewResources(cs ...*types.Component) []*Resource {
	var resources []*Resource

	for _, component := range cs {
		resources = append(resources, &Resource{
			name:      component.Name(),
			component: component,
		})
	}

	return resources
}

func (r *Resource) Name() string {
	return r.name
}

func (r *Resource) Component() *types.Component {
	return r.component
}

func (r *Resource) Update(status v1.PodStatus) {
	r.mux.Lock()
	defer r.mux.Unlock()

	r.podStatus = status

	for _, cstatus := range status.ContainerStatuses {
		cstate := cstatus.State
		if cstatus.State.Terminated != nil || cstatus.LastTerminationState.Terminated != nil {
			r.unhealthy = true
			r.err = fmt.Errorf("docker container has terminated for component %v, unable to recover", r.Name())
			return
		}

		if waitingState := cstate.Waiting; waitingState != nil {
			if strings.Compare("CrashLoopBackOff", waitingState.Reason) == 0 {
				r.unhealthy = true
				r.err = fmt.Errorf("docker container has entered a crash loop for component %v", r.Name())
				return
			}
		}

		if cstatus.State.Running != nil {
			r.ready = true
		}
	}

	switch status.Phase {
	case v1.PodPending:
		fallthrough
	case v1.PodRunning:
		fallthrough
	case v1.PodSucceeded:
		r.unhealthy = false
		r.done = true
	case v1.PodUnknown:
		r.unhealthy = true
		r.err = fmt.Errorf("pod for component %v has entered an unknown phase with message: %v", r.Name(), status.Message)
	case v1.PodFailed:
		r.unhealthy = true
		r.err = fmt.Errorf("pod for component %v has failed with message: %v", r.Name(), status.Message)
	}
}

func (r *Resource) Ready() bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.ready
}

func (r *Resource) Error() error {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.err
}

func (r *Resource) PodStatus() v1.PodStatus {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.podStatus
}

func (r *Resource) Unhealthy() bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.unhealthy
}

func (r *Resource) Done() bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.done
}

