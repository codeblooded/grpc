package orch

import (
	"testing"

	"k8s.io/apimachinery/pkg/watch"

	"github.com/grpc/grpc/testctrl/svc/types/test"
)

func TestPodEventDelegatorSet(t *testing.T) {
	del := Delegator{}

	ch := make(chan watch.Event, 1)
	session := test.NewSessionBuilder().Build()

	del.Set(session.Name(), ch)
	actualCh := del.Get(session.Name())
	if actualCh != ch {
		t.Errorf("PodEventDelegator Set did not set channel; expected %v but got %v", actualCh, ch)
	}
}

