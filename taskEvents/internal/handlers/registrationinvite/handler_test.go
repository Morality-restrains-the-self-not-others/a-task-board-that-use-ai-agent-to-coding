package registrationinvite

import (
	"context"
	"encoding/json"
	"testing"

	"taskEvents/domain"
)

func TestDispatchPolicyUpdated(t *testing.T) {
	h := LocalHandler()
	raw, _ := json.Marshal(map[string]interface{}{
		"enabled":     true,
		"daily_quota": 10,
		"updated_by":  "u1",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "REGISTRATION_INVITE_POLICY_UPDATED",
		Envelope:  domain.EventEnvelope{Data: raw},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("outcome=%v err=%v", out, err)
	}
}

func TestDispatchIssuedAndRedeemed(t *testing.T) {
	h := LocalHandler()
	issued, _ := json.Marshal(map[string]interface{}{
		"code_id": "c1", "code": "ABCD2345", "issuer_user_id": "u2", "issued_day": "2026-07-22",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "REGISTRATION_INVITE_CODE_ISSUED",
		Envelope:  domain.EventEnvelope{Data: issued},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("issued outcome=%v err=%v", out, err)
	}
	redeemed, _ := json.Marshal(map[string]interface{}{
		"code": "ABCD2345", "redeemed_by_user_id": "u3", "redeemed_at": "2026-07-22T04:00:00Z",
	})
	out, err = h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "REGISTRATION_INVITE_CODE_REDEEMED",
		Envelope:  domain.EventEnvelope{Data: redeemed},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("redeemed outcome=%v err=%v", out, err)
	}
}

func TestDispatchUnsupported(t *testing.T) {
	h := LocalHandler()
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{EventType: "UNKNOWN"})
	if err == nil || out != domain.DispatchPermanent {
		t.Fatalf("expected permanent failure, got out=%v err=%v", out, err)
	}
}
