// Copyright 2020 gRPC authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orch

import (
	"fmt"

	"github.com/grpc/grpc/testctrl/svc/types"
)

// Pool describes a cluster of identical machines.
type Pool struct {
	// Name is an indentifier that uniquely distinguishes a pool instance.
	Name string

	// Available is the number of machines that are idle and able to be reserved.
	Available int

	// Capacity is the total number of machines in the pool.
	Capacity int
}

// Availability encapsulates a set of pools, determining whether the current number of machines can
// accomodate a session and, if so, mutates the number of available nodes in the pool.
//
// It is not thread-safe, and it should be instantiated with the NewAvailability constructor.
type Availability struct {
	// pools maps the name of a pool to an instance of the Pool type for quick lookups.
	pools map[string]Pool
}

// NewAvailability creates a new availability instance.
func NewAvailability() *Availability {
	return &Availability{
		pools: make(map[string]Pool),
	}
}

// AddPool adds the pool to the list of pools that are considered for running session components.
// When this function returns, the pool is considered "known" by the availability instance.
func (a *Availability) AddPool(pool Pool) {
	a.pools[pool.Name] = pool
}

// RemovePool removes the pool from the list of pools that are considered for running session
// components. If the pool was never added, it returns a PoolUnknownError.
func (a *Availability) RemovePool(pool Pool) error {
	name := pool.Name
	if _, ok := a.pools[name]; !ok {
		return PoolUnknownError{name}
	}

	delete(a.pools, name)
	return nil
}

// Reserve accepts a session that is going to reserve machines and decreases the number of available
// machines from the appropriate pools. It does not actually reserve any machine or provision any
// resources.
//
// If there are not enough machines available in the specified pools to accomodate the session, it
// returns a PoolAvailabilityError. If a component references a pool that was not added to the
// availability instance, it returns a PoolUnknownError. If the number of machines required by the
// session exheeds the capacity of the pools, it returns a PoolCapacityError.
func (a *Availability) Reserve(session *types.Session) error {
	components := sessionComponents(session)

	machineCounts, err := a.machineCounts(components)
	if err != nil {
		return err
	}

	fits, err := a.fits(machineCounts)
	if err != nil {
		return err
	}
	if !fits {
		return PoolAvailabilityError{}
	}

	for name, count := range machineCounts {
		pool := a.pools[name]
		pool.Available -= count
		a.pools[name] = pool
	}
	return nil
}

// Return accepts a session that has reserved machines that are no longer needed. It increases the
// number of available machines to allow other sessions to provision resources. It does not actually
// return, tear down or delete any machine or resources.
//
// This method makes an assumption that Reserve has been called on the session, performing no checks
// that the available machines in each pool do not exheed their capacities.
//
// If a component references a pool that was not added to the availability instance, it returns a
// PoolUnknownError.
func (a *Availability) Return(session *types.Session) error {
	components := sessionComponents(session)

	machineCounts, err := a.machineCounts(components)
	if err != nil {
		return err
	}

	for name, count := range machineCounts {
		pool := a.pools[name]
		pool.Available += count
		a.pools[name] = pool
	}
	return nil
}

// machineCounts returns a map with the name of each pool as the key and the number of machines
// required from the pool as the value. If a component references a pool that was not added to the
// availability instance, it returns a PoolUnknownError.
func (a *Availability) machineCounts(components []*types.Component) (map[string]int, error) {
	machines := make(map[string]int)

	for poolName, _ := range a.pools {
		machines[poolName] = 0
	}

	for _, component := range components {
		if c, ok := machines[component.PoolName]; ok {
			machines[component.PoolName] = c + 1
		} else {
			return nil, PoolUnknownError{component.PoolName}
		}
	}

	return machines, nil
}

// fits accepts a map with pool names as keys and number of required machines within the pool as
// values. It returns true if there are enough available resources to schedule. If the number of
// machines required exheeds the capacity of the pools, it returns a PoolCapacityError.
func (a *Availability) fits(machineCounts map[string]int) (bool, error) {
	for poolName, c := range machineCounts {
		pool := a.pools[poolName]

		if c > pool.Capacity {
			return false, PoolCapacityError{poolName, c, pool.Capacity}
		}

		if c > pool.Available {
			return false, nil
		}
	}

	return true, nil
}

// PoolAvailabilityError indicates Reserve was called with a session that required a number of
// machines that were not currently available. These machines may become available at another time.
type PoolAvailabilityError struct{}

// Error returns a string representation of the available error message.
func (pae PoolAvailabilityError) Error() string {
	return fmt.Sprintf("not enough machines are available to accomodate the reservation")
}

// PoolCapacityError indicates that a session requires a number of machines which is greater than
// the number of machines in the pool. The session can never be scheduled.
type PoolCapacityError struct {
	// Name is the string identifier for the pool.
	Name string

	// Requested is the number of machines that the session requires.
	Requested int

	// Capacity is the maximum number of machines that the pool can accomodate.
	Capacity int
}

// Error returns a string representation of the capacity error message.
func (pce PoolCapacityError) Error() string {
	return fmt.Sprintf("pool %v has only %v machines, but the session required %v machines",
		pce.Name, pce.Requested, pce.Capacity)
}

// PoolUnknownError indicates that a session component required a pool that was not added on the
// availability instance.
type PoolUnknownError struct {
	// Name is the string identifier for a un-added pool.
	Name string
}

// Error returns a string representation of the unknown error message.
func (pue PoolUnknownError) Error() string {
	return fmt.Sprintf("pool '%v' was referenced, but not added to the availability instance",
		pue.Name)
}

// sessionComponents collects and returns a slice with all of a session's components.
func sessionComponents(session *types.Session) []*types.Component {
	components := []*types.Component{}
	if session.Driver != nil {
		components = append(components, session.Driver)
	}
	for _, w := range session.Workers {
		components = append(components, w)
	}
	return components
}
