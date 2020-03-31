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
	"sync"

	"github.com/grpc/grpc/testctrl/svc/types"
)

// Queue provides a thread-safe way to limit the number of running sessions. It uses the
// availability instance to know the number of machines available and capacities. Then, it only
// dequeues sessions when they are able to be scheduled.
type Queue struct {
	// rr is the bookkeeper for the number of available machines.
	rr reserveReturner

	// items is the slice with sessions and metadata that are enqueued/dequeued.
	items []*item

	// mux guards against parallel mutations and reads occur across threads.
	mux sync.Mutex
}

func NewQueue(rr reserveReturner) *Queue {
	return &Queue{
		rr: rr,
	}
}

// Enqueue
func (q *Queue) Enqueue(session *types.Session) error {
	q.mux.Lock()
	defer q.mux.Unlock()
	q.items = append(q.items, &item{session})
	return nil
}

// Dequeue
func (q *Queue) Dequeue() *types.Session {
	q.mux.Lock()
	defer q.mux.Unlock()

	for _, i := range q.items {
		session := i.session

		if err := q.rr.Reserve(session); err == nil {
			q.items = q.items[1:]
			return session
		}
	}

	return nil
}

func (q *Queue) Done(session *types.Session) {
	q.mux.Lock()
	defer q.mux.Unlock()
	q.rr.Return(session)
}

func (q *Queue) Count() int {
	q.mux.Lock()
	defer q.mux.Unlock()
	return len(q.items)
}

// item is an internal type which provides additional metadata and statistics for a session waiting
// in the queue.
type item struct {
	// session is the waiting session.
	session *types.Session
}

// reserveReturner is an internal interface that handles bookkeeping for machine availability. This
// interface is implemented by the Availability type for production.
type reserveReturner interface {
	Reserve(*types.Session) error
	Return(*types.Session) error
}
