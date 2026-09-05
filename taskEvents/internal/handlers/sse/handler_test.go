package sse

import (
	"context"
	"encoding/json"
	"testing"

	"taskEvents/domain"
)

func TestDispatchMissingFields(t *testing.T) {
	h := &Handler{}
	data, _ := json.Marshal(map[string]interface{}{"task_id": "t1"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "SSE_MESSAGE",
		Envelope:  domain.EventEnvelope{EventType: "SSE_MESSAGE", Data: data},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v", out)
	}
}

func TestDispatchWrongEvent(t *testing.T) {
	h := &Handler{}
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{EventType: "OTHER"})
	if err == nil {
		t.Fatal("expected error")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v", out)
	}
}
