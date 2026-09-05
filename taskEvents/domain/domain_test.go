package domain

import "testing"

func TestConsumerSession_AllowsEvent(t *testing.T) {
	s := ConsumerSession{SubscribedEvents: []string{"USER_CREATED", "COMPANY_CREATED"}}
	if !s.AllowsEvent("USER_CREATED") {
		t.Fatal("expected USER_CREATED allowed")
	}
	if s.AllowsEvent("SSE_MESSAGE") {
		t.Fatal("expected SSE_MESSAGE denied")
	}
}

func TestIdempotencyKeyForEvent(t *testing.T) {
	k := IdempotencyKeyForEvent("USER_CREATED", "42")
	if k != "USER_CREATED:42" {
		t.Fatalf("got %q", k)
	}
}

func TestDomainEventRouter_CommandFor(t *testing.T) {
	r := DomainEventRouter{
		Domain: "accounts",
		Routes: map[string]EventRoute{
			"USER_CREATED": {EventType: "USER_CREATED", Path: "/api/internal/task-events/accounts/dispatch/"},
		},
	}
	cmd, ok := r.CommandFor(EventEnvelope{EventType: "USER_CREATED", Data: []byte(`{"user_id":1}`)})
	if !ok || cmd.Path == "" {
		t.Fatal("expected route")
	}
}
