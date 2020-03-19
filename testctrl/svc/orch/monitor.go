package orch

import (
	"fmt"
	"sync"

	"k8s.io/api/core/v1"
)

// Monitor manages a list of objects, providing facilities to update them, check the aggregate
// health and consider errors across them. A monitor must be instantiated using the NewMonitor
// constructor.
type Monitor struct {
	objects   map[string]*Object
	errObject *Object
	mux       sync.Mutex
}

// NewMonitor creates a new monitor instance, allocating and initializing its internal data
// structures which do not work when zeroed.
func NewMonitor() *Monitor {
	return &Monitor{
		objects: make(map[string]*Object),
	}
}

// Get returns the object who's name is supplied.
func (m *Monitor) Get(name string) *Object {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.objects[name]
}

// Add sets responsibility of an object on the monitor.
func (m *Monitor) Add(o *Object) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.objects[o.Name()] = o
}

// Remove revokes responsibility of an object from the monitor.
func (m *Monitor) Remove(name string) {
	m.mux.Lock()
	defer m.mux.Unlock()
	delete(m.objects, name)
}

// Update uses a supplied kubernetes pod to update the managed object.
func (m *Monitor) Update(pod *v1.Pod) error {
	componentName := pod.Labels["component-name"]
	if len(componentName) < 1 {
		return fmt.Errorf("monitor cannot update using pod named '%v', missing a component-name label", componentName)
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	o := m.objects[componentName]
	if o == nil {
		return fmt.Errorf("monitor does not manage the component %v", componentName)
	}

	o.Update(pod.Status)
	if o.Health() == Unhealthy {
		return o.Error()
	}

	return nil
}

// Error is an convenience function which returns the error from the most recent unhealthy object.
func (m *Monitor) Error() error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if m.errObject != nil {
		return m.errObject.Error()
	}

	return nil
}

// ErrObject returns the most recent unhealthy object.
func (m *Monitor) ErrObject() *Object {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.errObject
}

// Unhealthy returns true if any object has a health value of Unhealthy.
func (m *Monitor) Unhealthy() bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	for _, o := range m.objects {
		if o.Health() == Unhealthy {
			m.errObject = o
			return true
		}
	}

	return false
}

// Done returns true if all objects have a health value of Done.
func (m *Monitor) Done() bool {
	m.mux.Lock()
	defer m.mux.Unlock()

	for _, o := range m.objects {
		if o.Health() != Done {
			return false
		}
	}

	return true
}
