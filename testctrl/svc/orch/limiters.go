package orch

import (
	"github.com/grpc/grpc/testctrl/svc/types"
)

type limitlessTracker struct{}

func (lt limitlessTracker) Reserve(session *types.Session) error {
	return nil
}

func (lt limitlessTracker) Unreserve(session *types.Session) error {
	return nil
}
