package taskrevisionrecorded

import (
	"context"
	"encoding/json"
	"testing"

	"taskEvents/consumer"
	"taskEvents/domain"
)

func TestDispatchSuccess(t *testing.T) {
	h := LocalHandler()
	raw, _ := json.Marshal(map[string]interface{}{
		"revision_id": "rev1", "task_id": "task_a", "tenant_id": "t1",
		"version_num": 2, "actor_user_id": "u1", "changed_fields": "title",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_REVISION_RECORDED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_REVISION_RECORDED", Data: raw},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("outcome=%v err=%v", out, err)
	}
}

func TestDispatchUnsupported(t *testing.T) {
	h := LocalHandler()
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{EventType: "UNKNOWN"})
	if err == nil || out != domain.DispatchPermanent {
		t.Fatalf("expected permanent, got out=%v err=%v", out, err)
	}
}

func TestReplaySameRevisionIDSkipsSecondDispatch(t *testing.T) {
	store := &memStore{seen: map[domain.IdempotencyKey]struct{}{}}
	commands := &countingHandler{inner: LocalHandler()}
	svc := domain.IdempotentDispatchService{
		Commands:    commands,
		Idempotency: store,
		KeyFor:      consumer.IdempotencyKeyFromEnvelope,
	}
	raw, _ := json.Marshal(map[string]interface{}{
		"revision_id": "rev-same", "task_id": "task_a", "tenant_id": "t1",
	})
	cmd := domain.DomainCommand{
		EventType: "TASK_REVISION_RECORDED",
		Envelope: domain.EventEnvelope{
			EventType: "TASK_REVISION_RECORDED",
			Data:      raw,
			Key:       "task_a",
		},
	}
	if out, err := svc.Handle(context.Background(), cmd); err != nil || out != domain.DispatchSuccess {
		t.Fatalf("first: out=%v err=%v", out, err)
	}
	if out, err := svc.Handle(context.Background(), cmd); err != nil || out != domain.DispatchSuccess {
		t.Fatalf("replay: out=%v err=%v", out, err)
	}
	if commands.calls != 1 {
		t.Fatalf("handler calls=%d want 1", commands.calls)
	}
}

func TestDifferentRevisionIDsDoNotCollapse(t *testing.T) {
	store := &memStore{seen: map[domain.IdempotencyKey]struct{}{}}
	commands := &countingHandler{inner: LocalHandler()}
	svc := domain.IdempotentDispatchService{
		Commands:    commands,
		Idempotency: store,
		KeyFor:      consumer.IdempotencyKeyFromEnvelope,
	}
	makeCmd := func(rev string) domain.DomainCommand {
		raw, _ := json.Marshal(map[string]interface{}{
			"revision_id": rev, "task_id": "task_same", "tenant_id": "t1",
		})
		return domain.DomainCommand{
			EventType: "TASK_REVISION_RECORDED",
			Envelope: domain.EventEnvelope{
				EventType: "TASK_REVISION_RECORDED",
				Data:      raw,
				Key:       "task_same",
			},
		}
	}
	if _, err := svc.Handle(context.Background(), makeCmd("rev-1")); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Handle(context.Background(), makeCmd("rev-2")); err != nil {
		t.Fatal(err)
	}
	if commands.calls != 2 {
		t.Fatalf("different revision_id must not collapse, calls=%d", commands.calls)
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
