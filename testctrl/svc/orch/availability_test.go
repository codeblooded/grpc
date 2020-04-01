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
	"reflect"
	"testing"

	"github.com/grpc/grpc/testctrl/svc/types"
)

func TestAvailabilityAddPool(t *testing.T) {
	availability := NewAvailability()
	poolName := "TestPool"
	pool := Pool{Name: poolName}

	availability.AddPool(pool)
	found, exists := availability.pools[poolName]
	if !exists {
		t.Fatalf("did not add pool to internal map")
	}
	if !reflect.DeepEqual(pool, found) {
		t.Fatalf("unexpectedly mutated pool during addition")
	}
}

func TestAvailabilityRemovePool(t *testing.T) {
	availability := NewAvailability()
	pool := Pool{
		Name:      "TestPool",
		Available: 10,
		Capacity:  100,
	}

	// test remove with original object
	availability.pools[pool.Name] = pool
	availability.RemovePool(pool)
	if _, exists := availability.pools[pool.Name]; exists {
		t.Errorf("did not remove original pool from internal map")
	}

	// test remove with identically-named object
	availability.pools[pool.Name] = pool
	availability.RemovePool(Pool{Name: pool.Name})
	if _, exists := availability.pools[pool.Name]; exists {
		t.Errorf("did not remove pool with identical name from internal map")
	}

	// test error raised if pool does not exist
	fakePool := Pool{Name: "WaymoPool"}
	if err := availability.RemovePool(fakePool); err == nil {
		t.Errorf("did not error for non-existent pool")
	}
}

func TestAvailabilityReserve(t *testing.T) {
	cases := []struct {
		description      string
		workerCount      int
		workerPoolBefore Pool
		workerPoolAfter  Pool
		err              error
	}{
		{
			description:      "capacity exheeded",
			workerCount:      5,
			workerPoolBefore: Pool{Available: 4, Capacity: 4},
			workerPoolAfter:  Pool{Available: 4, Capacity: 4},
			err:              PoolCapacityError{},
		},
		{
			description:      "not available",
			workerCount:      7,
			workerPoolBefore: Pool{Available: 5, Capacity: 7},
			workerPoolAfter:  Pool{Available: 5, Capacity: 7},
			err:              PoolAvailabilityError{},
		},
		{
			description:      "exact availability match",
			workerCount:      3,
			workerPoolBefore: Pool{Available: 3, Capacity: 3},
			workerPoolAfter:  Pool{Available: 0, Capacity: 3},
			err:              nil,
		},
		{
			description:      "availability match greater",
			workerCount:      3,
			workerPoolBefore: Pool{Available: 8, Capacity: 9},
			workerPoolAfter:  Pool{Available: 5, Capacity: 9},
			err:              nil,
		},
	}

	for _, c := range cases {
		a := NewAvailability()

		driverPool := Pool{
			Name:      "DriverPool",
			Available: 1,
			Capacity:  1,
		}
		a.AddPool(driverPool)

		workerPool := c.workerPoolBefore
		workerPool.Name = "WorkerPool"
		a.AddPool(workerPool)

		driver := types.NewComponent(testContainerImage, types.DriverComponent)
		driver.PoolName = driverPool.Name

		var workers []*types.Component
		for i := 0; i < c.workerCount; i++ {
			component := types.NewComponent(testContainerImage, types.ClientComponent)
			component.PoolName = workerPool.Name
			workers = append(workers, component)
		}

		session := types.NewSession(driver, workers, nil)

		err := a.Reserve(session)
		expectedErrType := reflect.TypeOf(c.err)
		actualErrType := reflect.TypeOf(err)
		if c.err != nil {
			// check for the proper error
			if expectedErrType != actualErrType {
				t.Errorf("expected %v error for case %v, but got %v: %v",
					expectedErrType.Name(), c.description, actualErrType.Name(), err)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error returned: %v", err)
			}
		}

		got := a.pools[workerPool.Name]
		if got.Available != c.workerPoolAfter.Available {
			t.Errorf("expected %v machines remaining after reserve, but got %v",
				c.workerPoolAfter.Available, got.Available)
		}
		if got.Capacity != c.workerPoolAfter.Capacity {
			t.Errorf("expected %v machine capacity after reserve, but got %v",
				c.workerPoolAfter.Capacity, got.Capacity)
		}
	}

	// check error returned for unknown pool
	a := NewAvailability()
	a.AddPool(Pool{Name: "KnownPool"})

	driver := types.NewComponent(testContainerImage, types.DriverComponent)
	driver.PoolName = "UnknownPool"

	session := types.NewSession(driver, nil, nil)

	err := a.Reserve(session)
	errType := reflect.TypeOf(err)
	if errType == nil || errType != reflect.TypeOf(PoolUnknownError{}) {
		t.Errorf("expected pool unknown error for un-added pool, but got %v", errType.Name())
	}
}

func TestAvailabilityReturn(t *testing.T) {
	cases := []struct {
		description      string
		workerCount      int
		workerPoolBefore Pool
		workerPoolAfter  Pool
	}{
		{
			description:      "default",
			workerCount:      3,
			workerPoolBefore: Pool{Available: 4, Capacity: 7},
			workerPoolAfter:  Pool{Available: 7, Capacity: 7},
		},
	}

	for _, c := range cases {
		a := NewAvailability()

		driverPool := Pool{
			Name:      "DriverPool",
			Available: 1,
			Capacity:  1,
		}
		a.AddPool(driverPool)

		workerPool := c.workerPoolBefore
		workerPool.Name = "WorkerPool"
		a.AddPool(workerPool)

		driver := types.NewComponent(testContainerImage, types.DriverComponent)
		driver.PoolName = driverPool.Name

		var workers []*types.Component
		for i := 0; i < c.workerCount; i++ {
			component := types.NewComponent(testContainerImage, types.ClientComponent)
			component.PoolName = workerPool.Name
			workers = append(workers, component)
		}

		session := types.NewSession(driver, workers, nil)

		err := a.Return(session)
		if err != nil {
			t.Errorf("unexpected error returned: %v", err)
		}

		got := a.pools[workerPool.Name]
		if got.Available != c.workerPoolAfter.Available {
			t.Errorf("expected %v machines remaining after return, but got %v",
				c.workerPoolAfter.Available, got.Available)
		}
		if got.Capacity != c.workerPoolAfter.Capacity {
			t.Errorf("expected %v machine capacity after return, but got %v",
				c.workerPoolAfter.Capacity, got.Capacity)
		}
	}

	// check error returned for unknown pool
	a := NewAvailability()
	a.AddPool(Pool{Name: "KnownPool"})

	driver := types.NewComponent(testContainerImage, types.DriverComponent)
	driver.PoolName = "UnknownPool"

	session := types.NewSession(driver, nil, nil)

	err := a.Return(session)
	errType := reflect.TypeOf(err)
	if errType == nil || errType != reflect.TypeOf(PoolUnknownError{}) {
		t.Errorf("expected pool unknown error for un-added pool, but got %v", errType.Name())
	}
}
