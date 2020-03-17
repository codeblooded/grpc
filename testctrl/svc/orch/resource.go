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
			r.err = fmt.Errorf("resource %v: docker container has terminated, unable to recover", r)
			return
		}

		if waitingState := cstate.Waiting; waitingState != nil {
			if strings.Compare("CrashLoopBackOff", waitingState.Reason) == 0 {
				r.unhealthy = true
				r.err = fmt.Errorf("resource %v: docker container has entered crash loop", r)
				return
			}
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
		r.err = fmt.Errorf("resource %v: pod has entered an unknown phase: %v", r, status.Message)
	case v1.PodFailed:
		r.unhealthy = true
		r.err = fmt.Errorf("resource %v: pod has failed: %v", r, status.Message)
	}
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

