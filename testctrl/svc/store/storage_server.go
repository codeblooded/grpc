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

package store

import (
	"fmt"
	"sync"

	"github.com/grpc/grpc/testctrl/svc/types"
)

// Store is the interface for a data store.
// The data store stores sessions and associated events.
type Store interface {
	// Adds a Session object to the Store.
	StoreSession(session *types.Session) error
	// Records an event and associates it to an existing session.
	StoreEvent(sessionName string, event *types.Event) error
	// Gets the latest event associated to an existing session.
	GetLatestEvent(sessionName string) *types.Event
	// Deletes a Session object and associated events from the Store.
	DeleteSession(sessionName string) error
}

// StorageServer is an in-memory implementation of a data store.
type StorageServer struct {
	mutex      sync.Mutex
	sessionMap map[string]types.Session
	eventMap   map[string][]types.Event
}

// NewStorageServer constructs a new instance of StorageServer.
func NewStorageServer() *StorageServer {
	return &StorageServer{
		sessionMap: make(map[string]types.Session),
		eventMap:   make(map[string][]types.Event),
	}
}

// StoreSession stores a session in the StorageServer.
func (s *StorageServer) StoreSession(session *types.Session) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	sessionName := session.Name
	_, sessionExists := s.sessionMap[sessionName]
	if sessionExists {
		return fmt.Errorf("Duplicate session name: %s", sessionName)
	}
	s.sessionMap[sessionName] = *session
	s.eventMap[sessionName] = make([]types.Event, 0)
	return nil
}

// StoreEvent stores an event associated with an existing session.
func (s *StorageServer) StoreEvent(sessionName string, event *types.Event) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	eventStore, sessionExists := s.eventMap[sessionName]
	if sessionExists {
		s.eventMap[sessionName] = append(eventStore, *event)
		return nil
	}
	return fmt.Errorf("Unknown session name: %s", sessionName)
}

// GetLatestEvent returns the latest event associated with an existing session.
func (s *StorageServer) GetLatestEvent(sessionName string) *types.Event {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	eventStore, sessionExists := s.eventMap[sessionName]
	if !(sessionExists && len(eventStore) > 0) {
		return nil
	}
	event := eventStore[len(eventStore)-1]
	return &event
}

// DeleteSession deletes a session from the StorageServer.
func (s *StorageServer) DeleteSession(sessionName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, sessionExists := s.sessionMap[sessionName]
	if sessionExists {
		delete(s.eventMap, sessionName)
		delete(s.sessionMap, sessionName)
	}
}
