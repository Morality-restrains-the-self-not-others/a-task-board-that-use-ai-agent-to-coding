package projectrevisionrecorded

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
		"revision_id": "rev1", "project_id": "proj_a", "tenant_id": "t1",
		"version_num": 1, "actor_user_id": "u1", "changed_fields": "name",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "PROJECT_REVISION_RECORDED",
		Envelope:  domain.EventEnvelope{EventType: "PROJECT_REVISION_RECORDED", Data: raw},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("outcome=%v err=%v", out, err)
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
		"revision_id": "rev-same", "project_id": "proj_a", "task_id": "should-not-key",
	})
	cmd := domain.DomainCommand{
		EventType: "PROJECT_REVISION_RECORDED",
		Envelope:  domain.EventEnvelope{EventType: "PROJECT_REVISION_RECORDED", Data: raw, Key: "proj_a"},
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
			"revision_id": rev, "project_id": "proj_same",
		})
		return domain.DomainCommand{
			EventType: "PROJECT_REVISION_RECORDED",
			Envelope:  domain.EventEnvelope{EventType: "PROJECT_REVISION_RECORDED", Data: raw, Key: "proj_same"},
		}
	}
	if _, err := svc.Handle(context.Background(), makeCmd("r1")); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Handle(context.Background(), makeCmd("r2")); err != nil {
		t.Fatal(err)
	}
	if commands.calls != 2 {
		t.Fatalf("calls=%d want 2", commands.calls)
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
