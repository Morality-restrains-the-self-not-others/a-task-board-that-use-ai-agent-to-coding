package billing

import (
	"context"
	"encoding/json"
	"testing"

	"taskEvents/domain"
)

func TestDispatchMissingTransactionID(t *testing.T) {
	h := &Handler{}
	data, _ := json.Marshal(map[string]interface{}{})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "BILLING_TRANSACTION_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "BILLING_TRANSACTION_CREATED", Data: data},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v", out)
	}
}

func TestDispatchOKWithoutRechargeSSE(t *testing.T) {
	h := &Handler{}
	data, _ := json.Marshal(map[string]interface{}{"transaction_id": "tx-1"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "BILLING_TRANSACTION_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "BILLING_TRANSACTION_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}

func TestDispatchRechargeWithoutUserIDSkipsSSE(t *testing.T) {
	h := &Handler{}
	data, _ := json.Marshal(map[string]interface{}{
		"transaction_id":   "wechat:WX1",
		"transaction_type": "recharge",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "BILLING_TRANSACTION_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "BILLING_TRANSACTION_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
}
