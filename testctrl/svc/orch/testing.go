package orch

import (
	"github.com/grpc/grpc/testctrl/svc/types"
)

// testContainerImage is a default container image supplied during component construction in tests.
const testContainerImage = "example:latest"

// limitlessTracker allows queue to operate as only a FIFO-queue for easier testing.
type limitlessTracker struct{}

func (lt limitlessTracker) Reserve(session *types.Session) error {
	return nil
}

func (lt limitlessTracker) Unreserve(session *types.Session) error {
	return nil
}
