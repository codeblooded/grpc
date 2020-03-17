package orch

import (
	"fmt"
	"sync"

	"github.com/grpc/grpc/testctrl/svc/types"
)

type Resource struct {
	Name      string
	Component *types.Component
	PodStatus v1.PodStatus
	Unhealthy bool
	Error     error
}

func NewResources(cs ...*types.Component) []Resource {
	var resources []Resource

	for _, component := range cs {
		for i := 0; i < component.Replicas(); i++ {
			resources = append(resources, Resource{
				Name:      component.Name(),
				Component: component,
			})
		}
	}

	return resources
}

type Monitor struct {
	unhealthy   bool
	resources   map[string]Resource
	errResource Resource
	mux         sync.Mutex
}

func (m *Monitor) Get(resourceName string) Resource {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.resources[resourceName]
}

func (m *Monitor) Add(r Resource) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.resources[r.Name] = r
}

func (m *Monitor) Remove(resourceName string) Resource {
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

	r.PodStatus = pod.Status
	return m.setHealth(r)
}

func (m *Monitor) Unhealthy() bool {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.unhealthy
}

func (m *Monitor) setHealth(r Resource) error {
	status := r.PodStatus

	for _, cstatus := range status.ContainerStatuses {
		cstate := cstatus.State
		if cstate.Terminated != nil {
			r.Unhealthy = true
			r.Error = fmt.Errorf("resource %v: docker container has terminated, unable to recover", r)
		}
	}

	switch status.Phase {
	case v1.PodPending:
		fallthrough
	case v1.PodRunning:
		fallthrough
	case v1.PodSucceeded:
		r.Unhealthy = false
	case v1.PodUnknown:
		r.Unhealthy = true
		r.Error = fmt.Errorf("resource %v: pod has entered an unknown phase: %v", r, status.Message)
	case v1.PodFailed:
		r.Unhealthy = true
		r.Error = fmt.Errorf("resource %v: pod has failed: %v", r, status.Message)
	}

	if !m.unhealthy && r.Unhealthy {
		m.unhealthy = true
		m.errResource = r
	}
}
