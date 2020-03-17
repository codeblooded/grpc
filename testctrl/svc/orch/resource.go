package orch

import (
	"fmt"
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
		for i := int32(0); i < component.Replicas(); i++ {
			resources = append(resources, &Resource{
				name:      component.Name(),
				component: component,
			})
		}
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

	for _, cstatus := range status.ContainerStatuses {
		cstate := cstatus.State
		if cstate.Terminated != nil {
			r.unhealthy = true
			r.err = fmt.Errorf("resource %v: docker container has terminated, unable to recover", r)
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

type Monitor struct {
	done	    bool
	unhealthy   bool
	resources   map[string]*Resource
	errResource *Resource
	mux         sync.Mutex
}

func NewMonitor() *Monitor {
	return &Monitor{
		resources: make(map[string]*Resource),
	}
}

func (m *Monitor) Get(resourceName string) *Resource {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.resources[resourceName]
}

func (m *Monitor) Add(r *Resource) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.resources[r.Name()] = r
}

func (m *Monitor) Remove(resourceName string) {
	m.mux.Lock()
	defer m.mux.Unlock()
	delete(m.resources, resourceName)
}

func (m *Monitor) Update(pod *v1.Pod) error {
	resourceName := pod.Labels["resource-name"]
	if len(resourceName) < 1 {
		return fmt.Errorf("monitor cannot update using pod named '%v', missing a resource-name label")
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	r := m.resources[resourceName]
	if r == nil {
		return fmt.Errorf("monitor does not manage resource named '%v'", resourceName)
	}

	r.Update(pod.Status)
	if r.Unhealthy() {
		m.unhealthy = true
		m.errResource = r
		return r.Error()
	}

	return nil
}

func (m *Monitor) Unhealthy() bool {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.unhealthy
}

func (m *Monitor) Error() error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.errResource != nil {
		return m.errResource.Error()
	}

	return nil
}

func (m *Monitor) Done() bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	for _, r := range m.resources {
		if !r.Done() {
			return false
		}
	}

	return true
}
