package svc

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	pb "github.com/grpc/grpc/testctrl/genproto/grpc/testing"
)

// TestSession is an immutable set of tests, required resources and their configuration.
type TestSession struct {
	name       string
	driver     interface{}
	workers    []interface{}
	scenarios  []*pb.Scenario
	createTime time.Time
}

// NewTestSession creates a TestSession, assigning it a unique name.
func NewTestSession(driver interface{}, workers []interface{}, scenarios []*pb.Scenario) *TestSession {
	return &TestSession{
		name:       fmt.Sprintf("testSessions/%s", uuid.New().String()),
		driver:     driver,
		workers:    workers,
		scenarios:  scenarios,
		createTime: time.Now(),
	}
}

// Name returns the globally unique name that identifies this test session instance.
func (t *TestSession) Name() string {
	return t.name
}

// Driver returns the test session's driver component.
func (t *TestSession) Driver() interface{} {
	return t.driver
}

// Workers returns the slice of the test session's worker components.
func (t *TestSession) Workers() []interface{} {
	return t.workers
}

// Scenarios returns the raw test scenario protobufs in a slice. The scenarios are not parsed
// into implementation-specific types, because the service is designed to know nothing about
// the tests.
func (t *TestSession) Scenarios() []*pb.Scenario {
	return t.scenarios
}
