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

package svc

import (
	"context"
	"fmt"

	svcpb "github.com/grpc/grpc/testctrl/proto/scheduling/v1"
	"github.com/grpc/grpc/testctrl/svc/store"
	"github.com/grpc/grpc/testctrl/svc/types"

	lrpb "google.golang.org/genproto/googleapis/longrunning"
)

// Scheduler is a single method interface for queueing sessions.
type Scheduler interface {
	// Schedule enqueues a session, returning any immediate error.
	// Infrastructure and test runtime errors will not be
	// returned.
	Schedule(s *types.Session) error
}

// SchedulingServer implements the scheduling service.
type SchedulingServer struct {
	scheduler  Scheduler
	operations lrpb.OperationsServer
	store      store.Store
}

// NewSchedulingServer constructs a scheduling server from a scheduler,
// an operations server, and a store.
func NewSchedulingServer(scheduler Scheduler, operations lrpb.OperationsServer, store store.Store) *SchedulingServer {
	return &SchedulingServer{
		scheduler:  scheduler,
		operations: operations,
		store:      store,
	}
}

// StartTestSession implements the scheduling service interface for
// starting a test session.
func (s *SchedulingServer) StartTestSession(ctx context.Context, req *svcpb.StartTestSessionRequest) (operation *lrpb.Operation, err error) {
	driver := types.NewComponent(
		req.Driver.ContainerImage,
		types.DriverComponent,
	)
	workers := make([]*types.Component, len(req.Workers))
	for i, v := range req.Workers {
		workers[i] = types.NewComponent(
			v.ContainerImage,
			types.ComponentKindFromProto(v.Kind),
		)
	}
	session := types.NewSession(driver, workers, req.Scenario)
	err = s.store.StoreSession(session)
	if err != nil {
		err = fmt.Errorf("error storing new test session: %v", err)
		return
	}
	var event *types.Event
	event, err = s.store.GetLatestEvent(session.Name)
	if err != nil {
		err = fmt.Errorf(
			"error retrieving event for new test session: %v",
			err,
		)
		return
	}
	operation, err = NewOperation(event, session)
	if err != nil {
		err = fmt.Errorf(
			"Error creating operation for new test session: %v",
			err,
		)
		return
	}
	err = s.scheduler.Schedule(session)
	if err != nil {
		operation = nil
		s.store.DeleteSession(session.Name)
		err = fmt.Errorf(
			"error scheduling new test session: %v",
			err,
		)
		return
	}
	return
}
