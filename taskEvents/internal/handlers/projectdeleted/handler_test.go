package projectdeleted

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"taskEvents/consumer"
	"taskEvents/domain"
)

type mockDetach struct {
	calls int
	last  string
	err   error
}

func (m *mockDetach) DetachByProject(ctx context.Context, projectID string) error {
	m.calls++
	m.last = projectID
	return m.err
}

func TestHandler_Success(t *testing.T) {
	mc := &mockDetach{}
	h := &Handler{Client: mc}
	data, _ := json.Marshal(map[string]interface{}{"project_id": "proj_1", "tenant_id": "t1"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "PROJECT_DELETED",
		Envelope:  domain.EventEnvelope{Data: data},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("out=%v err=%v", out, err)
	}
	if mc.calls != 1 || mc.last != "proj_1" {
		t.Fatalf("calls=%d last=%s", mc.calls, mc.last)
	}
}

func TestHandler_MissingProjectIDPermanent(t *testing.T) {
	h := &Handler{Client: &mockDetach{}}
	data, _ := json.Marshal(map[string]interface{}{"tenant_id": "t1"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "PROJECT_DELETED",
		Envelope:  domain.EventEnvelope{Data: data},
	})
	if out != domain.DispatchPermanent || err == nil {
		t.Fatalf("out=%v err=%v", out, err)
	}
}

func TestHandler_RetryableDetach(t *testing.T) {
	h := &Handler{Client: &mockDetach{err: errors.New("connection refused")}}
	data, _ := json.Marshal(map[string]interface{}{"project_id": "proj_1"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "PROJECT_DELETED",
		Envelope:  domain.EventEnvelope{Data: data},
	})
	if out != domain.DispatchRetryable || err == nil {
		t.Fatalf("out=%v err=%v", out, err)
	}
}

type memStore struct {
	seen map[domain.IdempotencyKey]struct{}
}

func (s *memStore) Seen(k domain.IdempotencyKey) bool {
	_, ok := s.seen[k]
	return ok
}
func (s *memStore) Mark(k domain.IdempotencyKey) { s.seen[k] = struct{}{} }

type countingHandler struct {
	inner *Handler
	calls int
}

func (c *countingHandler) Dispatch(ctx context.Context, cmd domain.DomainCommand) (domain.DispatchOutcome, error) {
	c.calls++
	return c.inner.Dispatch(ctx, cmd)
}

func TestReplaySameProjectIDSkipsSecondDispatch(t *testing.T) {
	store := &memStore{seen: map[domain.IdempotencyKey]struct{}{}}
	inner := &Handler{Client: &mockDetach{}}
	commands := &countingHandler{inner: inner}
	svc := domain.IdempotentDispatchService{
		Commands:    commands,
		Idempotency: store,
		KeyFor:      consumer.IdempotencyKeyFromEnvelope,
	}
	raw, _ := json.Marshal(map[string]interface{}{"project_id": "proj_same", "tenant_id": "t1"})
	cmd := domain.DomainCommand{
		EventType: "PROJECT_DELETED",
		Envelope:  domain.EventEnvelope{EventType: "PROJECT_DELETED", Data: raw, Key: "proj_same"},
	}
	if _, err := svc.Handle(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Handle(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if commands.calls != 1 {
		t.Fatalf("calls=%d want 1", commands.calls)
	}
}

func TestDifferentProjectIDsDoNotCollapse(t *testing.T) {
	store := &memStore{seen: map[domain.IdempotencyKey]struct{}{}}
	inner := &Handler{Client: &mockDetach{}}
	commands := &countingHandler{inner: inner}
	svc := domain.IdempotentDispatchService{
		Commands:    commands,
		Idempotency: store,
		KeyFor:      consumer.IdempotencyKeyFromEnvelope,
	}
	makeCmd := func(pid string) domain.DomainCommand {
		raw, _ := json.Marshal(map[string]interface{}{"project_id": pid, "tenant_id": "t-same"})
		return domain.DomainCommand{
			EventType: "PROJECT_DELETED",
			Envelope:  domain.EventEnvelope{EventType: "PROJECT_DELETED", Data: raw, Key: pid},
		}
	}
	if _, err := svc.Handle(context.Background(), makeCmd("p1")); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Handle(context.Background(), makeCmd("p2")); err != nil {
		t.Fatal(err)
	}
	if commands.calls != 2 {
		t.Fatalf("calls=%d want 2", commands.calls)
	}
}
