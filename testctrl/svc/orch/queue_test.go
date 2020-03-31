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

func TestQueueCount(t *testing.T) {
	n := 3 // number of sessions
	queue := NewQueue(alwaysAccomodater{})
	sessions := makeSessions(t, n)
	for _, session := range sessions {
		queue.Enqueue(session)
	}

	if count := queue.Count(); count != n {
		t.Errorf("count did not return accurate number of items, expected %v but got %v", n, count)
	}
}

func TestQueueEnqueue(t *testing.T) {
	n := 3 // number of sessions
	queue := NewQueue(alwaysAccomodater{})

	expectedSessions := makeSessions(t, n)
	for _, session := range expectedSessions {
		if err := queue.Enqueue(session); err != nil {
			t.Fatalf("encountered unexpected error during session creation: %v", err)
		}
	}

	for i, expected := range expectedSessions {
		actual := queue.items[i].session

		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("items were missed or added in an incorrect order: %v != %v", actual, expected)
		}
	}
}

func TestQueueDequeue(t *testing.T) {
	n := 3 // number of sessions
	var queue *Queue
	var sessions []*types.Session

	// test FIFO-order preserved when it can accomodate all sessions
	queue = NewQueue(alwaysAccomodater{})
	sessions = makeSessions(t, n)
	for _, session := range sessions {
		queue.items = append(queue.items, &item{session})
	}

	for i, expected := range sessions {
		got := queue.Dequeue()
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("dequeue (iteration %v of %v) was out of order, got %v but expected %v", i+1, n, got, expected)
		}
	}

	// test using availability
	poolName := "DequeuePool"
	availability := NewAvailability()
	availability.AddPool(&Pool{
		Name: poolName,
		Available: 3,
		Capacity: 5,
	})

	queue = NewQueue(availability)

	sessions = makeSessions(t, 3)

	sessions[0].Workers = makeWorkers(t, 5, &poolName)
	queue.Enqueue(sessions[0])

	sessions[1].Workers = makeWorkers(t, 3, &poolName)
	queue.Enqueue(sessions[1])

	sessions[2].Workers = makeWorkers(t, 1, &poolName)
	queue.Enqueue(sessions[2])

	got := queue.Dequeue()
	if !reflect.DeepEqual(got, sessions[1]) {
		t.Errorf("dequeue out of order (FIFO then availability), expected %v but got %v", sessions[1], got)
	}
}

type alwaysAccomodater struct{}

func (aa alwaysAccomodater) CanAccomodate(session *types.Session) (bool, error) {
	return true, nil
}

func makeSessions(t *testing.T, n int) []*types.Session {
	t.Helper()
	var sessions []*types.Session
	for i := 0; i < n; i++ {
		sessions = append(sessions, types.NewSession(nil, nil, nil))
	}
	return sessions
}

func makeWorkers(t *testing.T, n int, pool *string) []*types.Component {
	t.Helper()
	var components []*types.Component

	if n < 1 {
		return components
	}

	components = append(components, types.NewComponent(testContainerImage, types.ServerComponent))

	for i := n - 1; i > 0; i-- {
		components = append(components, types.NewComponent(testContainerImage, types.ClientComponent))
	}

	if pool != nil {
		for _, c := range components {
			c.PoolName = *pool
		}
	}

	return components
}
