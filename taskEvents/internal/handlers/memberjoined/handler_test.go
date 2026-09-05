package memberjoined

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"taskEvents/domain"
)

type mockClient struct {
	calls int
	err   error
	last  [4]string
}

func (m *mockClient) EnsureDefault(ctx context.Context, userID, companyID, memberID, memberName string) error {
	m.calls++
	m.last = [4]string{userID, companyID, memberID, memberName}
	return m.err
}

func TestHandler_Success(t *testing.T) {
	mc := &mockClient{}
	h := &Handler{Client: mc}
	data, _ := json.Marshal(map[string]interface{}{
		"member_id": "m1", "user_id": "u1", "company_id": "c1", "member_name": "Bob",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "MEMBER_JOINED",
		Envelope:  domain.EventEnvelope{Data: data},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("out=%v err=%v", out, err)
	}
	if mc.calls != 1 || mc.last[3] != "Bob" {
		t.Fatalf("calls=%d last=%v", mc.calls, mc.last)
	}
}

func TestHandler_MissingMemberID(t *testing.T) {
	h := &Handler{Client: &mockClient{}}
	data, _ := json.Marshal(map[string]interface{}{"user_id": "u1", "company_id": "c1"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "MEMBER_JOINED",
		Envelope:  domain.EventEnvelope{Data: data},
	})
	if out != domain.DispatchPermanent || err == nil {
		t.Fatalf("want permanent, got out=%v err=%v", out, err)
	}
}

func TestHandler_Retryable(t *testing.T) {
	mc := &mockClient{err: errors.New("retryable ensure-default status 502: x")}
	h := &Handler{Client: mc}
	data, _ := json.Marshal(map[string]interface{}{
		"member_id": "m1", "user_id": "u1", "company_id": "c1",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "MEMBER_JOINED",
		Envelope:  domain.EventEnvelope{Data: data},
	})
	if out != domain.DispatchRetryable || err == nil {
		t.Fatalf("want retryable, got out=%v err=%v", out, err)
	}
}
