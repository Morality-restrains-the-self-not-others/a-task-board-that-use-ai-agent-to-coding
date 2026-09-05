package relaylifecycle

import (
	"context"
	"testing"

	"taskEvents/domain"
)

func TestWorkflowUpdateHandlerIsNoOpSuccess(t *testing.T) {
	h := &WorkflowUpdateHandler{}
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "RELAY_START_ACCEPTED",
		Envelope:  domain.EventEnvelope{EventType: "RELAY_START_ACCEPTED"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

func TestWorkflowUpdateHandlerUnknownEventPermanent(t *testing.T) {
	h := &WorkflowUpdateHandler{}
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "UNKNOWN",
		Envelope:  domain.EventEnvelope{EventType: "UNKNOWN"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v", out)
	}
}

func TestMapWorkflowTransition(t *testing.T) {
	cases := map[string]string{
		"RELAY_START_ACCEPTED":        "accept",
		"RELAY_START_ATTEMPTED":       "dispatch_attempted",
		"RELAY_START_DISPATCH_FAILED": "dispatch_failed",
	}
	for event, want := range cases {
		got, ok := mapWorkflowTransition(event)
		if !ok || got != want {
			t.Fatalf("%s: got %q ok=%v want %q", event, got, ok, want)
		}
	}
	if _, ok := mapWorkflowTransition("OTHER"); ok {
		t.Fatal("expected OTHER to fail")
	}
}
