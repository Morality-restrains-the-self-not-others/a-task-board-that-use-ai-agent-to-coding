package domain

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type stubStore struct {
	seen  map[IdempotencyKey]struct{}
	marks []IdempotencyKey
}

func newStubStore() *stubStore { return &stubStore{seen: map[IdempotencyKey]struct{}{}} }
func (s *stubStore) Seen(k IdempotencyKey) bool {
	_, ok := s.seen[k]
	return ok
}
func (s *stubStore) Mark(k IdempotencyKey) {
	s.seen[k] = struct{}{}
	s.marks = append(s.marks, k)
}

type countingCommands struct{ calls int }

func (c *countingCommands) Dispatch(ctx context.Context, cmd DomainCommand) (DispatchOutcome, error) {
	c.calls++
	return DispatchSuccess, nil
}

func keyOf(env EventEnvelope) IdempotencyKey { return IdempotencyKey("k:" + env.Key) }

// 同一幂等键的事件第二次到达时不得再次调用 handler（at-most-once 语义）。
func TestIdempotentDispatchSkipRepeatedKey(t *testing.T) {
	store := newStubStore()
	commands := &countingCommands{}
	svc := IdempotentDispatchService{Commands: commands, Idempotency: store, KeyFor: keyOf}

	data, _ := json.Marshal(map[string]interface{}{"a": 1})
	cmd := DomainCommand{EventType: "EMAIL_SENT", Envelope: EventEnvelope{EventType: "EMAIL_SENT", Data: data, Key: "same"}}

	if out, err := svc.Handle(context.Background(), cmd); err != nil || out != DispatchSuccess {
		t.Fatalf("first handle: out=%v err=%v", out, err)
	}
	if commands.calls != 1 {
		t.Fatalf("handler calls after first = %d, want 1", commands.calls)
	}
	if out, err := svc.Handle(context.Background(), cmd); err != nil || out != DispatchSuccess {
		t.Fatalf("second handle must succeed silently: out=%v err=%v", out, err)
	}
	if commands.calls != 1 {
		t.Fatalf("handler calls after redelivery = %d, want 1 (deduped)", commands.calls)
	}
	if len(store.marks) != 1 {
		t.Fatalf("marks = %d, want 1", len(store.marks))
	}
}

// OPT-20260807-018 安全护栏: 幂等键含密码重置/激活 token 明文，可观测性输出必须为指纹。
// 指纹确定性（同键同指纹）+ 区分性（不同键不同指纹）+ 不泄漏（指纹不含键明文）。
func TestIdempotencyKeyFingerprintHidesSecret(t *testing.T) {
	secret := "EMAIL_SENT:password_reset:EkYq-SecretResetToken-9f3:contact@daydaymoney.com"
	fp := idempotencyKeyFingerprint(secret)
	if fp == "" || len(fp) != 32 {
		t.Fatalf("fingerprint len = %d, want 32 hex chars", len(fp))
	}
	if fp != idempotencyKeyFingerprint(secret) {
		t.Fatal("fingerprint must be deterministic for the same key")
	}
	if idempotencyKeyFingerprint(secret) == idempotencyKeyFingerprint("EMAIL_SENT:password_reset:OtherToken:contact@daydaymoney.com") {
		t.Fatal("different keys must produce different fingerprints")
	}
	for _, secretPart := range []string{"EkYq-SecretResetToken-9f3", "contact@daydaymoney.com", "password_reset"} {
		if strings.Contains(fp, secretPart) {
			t.Fatalf("fingerprint leaks key material %q", secretPart)
		}
	}
}

// 不同幂等键的事件互不干扰（EMAIL_SENT 键碰撞事故的回归护栏）。
func TestIdempotentDispatchDistinctKeysPass(t *testing.T) {
	store := newStubStore()
	commands := &countingCommands{}
	svc := IdempotentDispatchService{Commands: commands, Idempotency: store, KeyFor: keyOf}

	data, _ := json.Marshal(map[string]interface{}{"a": 1})
	mk := func(key string) DomainCommand {
		return DomainCommand{EventType: "EMAIL_SENT", Envelope: EventEnvelope{EventType: "EMAIL_SENT", Data: data, Key: key}}
	}
	for _, key := range []string{"reset-token-A", "reset-token-B", "verification-code-1"} {
		if _, err := svc.Handle(context.Background(), mk(key)); err != nil {
			t.Fatalf("handle %s: %v", key, err)
		}
	}
	if commands.calls != 3 {
		t.Fatalf("handler calls = %d, want 3 (all distinct keys delivered)", commands.calls)
	}
}
