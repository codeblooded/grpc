package orch

import (
	"fmt"
	"sync"

	"k8s.io/api/core/v1"
)

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
	resourceName := pod.Labels["component-name"]
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

