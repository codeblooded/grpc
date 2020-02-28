package svc

import (
	"time"

	pb "github.com/grpc/grpc/testctrl/genproto/testctrl/svc"
)

// Event represents an action or state that occurred at a specific point in time on a component
// or the entire test session. Events are designed to be chained together to create a log of all
// significant points in time during a test session.
//
// Most of the time, developers do not need to see the events for individual components. They will
// likely refer to the events of individual components to find the culprit of an error event on the
// entire session.
type Event struct {
	// Kind is the type of event.
	Kind EventKind

	// Time is the timestamp when the event was recorded.
	Time time.Time

	// Component is a pointer to the test component that this event describes. Nil indicates the
	// that this event describes the test session itself.
	Component interface{}

	// Error allows the event to attach a specific implementation of the error interface. It is
	// only included when Kind is Error.
	Error error
}

// EventKind specifies the type of event. For example, an event could be created for a fatal error
// or the system is waiting in the queue.
type EventKind int32

const (
	// Unknown signifies a state that was not foreseen. This state may irrecoverable and may
	// require manual cleanup. It should never be seen in production.
	Unknown EventKind = 0

	// Queue indicates that its subject is waiting to reserve resources.
	Queue = 1

	// Provision indicates that the system is reserving the required resources and configuring the
	// runtime for its subject.
	Provision = 2

	// Run indicates that no anomalies have been detected in the provision, and the health checks
	// are returning standard values.
	Run = 3

	// Error indicates that there was a fatal error detected during the run or provision.
	Error = 4
)

// EventKindFromProto takes the generated protobuf enum type and safely converts it into an
// EventKind. It ensures the string representations are equivalent, even if values change.
func EventKindFromProto(p pb.Event_Kind) EventKind {
	return eventKindStringToConstMap[p.String()]
}

// String returns the string representation of the EventKind.
func (e EventKind) String() string {
	return eventKindConstToStringMap[e]
}

// Proto converts the EventKind enum into the generated protobuf enum. It ensures the string
// representations are equivalent, even if values change.
func (e EventKind) Proto() pb.Event_Kind {
	return pb.Event_Kind(pb.Event_Kind_value[e.String()])
}

var eventKindStringToConstMap = map[string]EventKind{
	"UNKNOWN":   Unknown,
	"QUEUE":     Queue,
	"PROVISION": Provision,
	"RUN":       Run,
	"ERROR":     Error,
}

var eventKindConstToStringMap = map[EventKind]string{
	Unknown:   "UNKNOWN",
	Queue:     "QUEUE",
	Provision: "PROVISION",
	Run:       "RUN",
	Error:     "ERROR",
}

// EventRecorder implementations save events.
type EventRecorder interface {
	// Record affiliates an event with a particular test session and saves it.
	Record(t TestSession, e Event)
}
