package taskgracefulshutdownawait

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"taskEvents/domain"
	"taskEvents/internal/repository/cloudconfig"
)

type mockLoader struct {
	row *cloudconfig.ConfigRow
}

func (m *mockLoader) LoadForTask(companyID int64, workspaceID, taskID string) (*cloudconfig.ConfigRow, error) {
	return m.row, nil
}

type recordingPublisher struct {
	events []struct {
		Type string
		Data map[string]interface{}
	}
}

func (p *recordingPublisher) PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	copied := map[string]interface{}{}
	for k, v := range data {
		copied[k] = v
	}
	p.events = append(p.events, struct {
		Type string
		Data map[string]interface{}
	}{Type: eventType, Data: copied})
	return nil
}

func TestDecideAwait(t *testing.T) {
	if DecideAwait(nil, 1, 18) != DecisionReleased {
		t.Fatal("nil cfg => released")
	}
	if DecideAwait(&cloudconfig.ConfigRow{}, 1, 18) != DecisionReleased {
		t.Fatal("empty => released")
	}
	if DecideAwait(&cloudconfig.ConfigRow{ServerURL: "http://x"}, 1, 18) != DecisionWait {
		t.Fatal("busy => wait")
	}
	if DecideAwait(&cloudconfig.ConfigRow{ServerURL: "http://x"}, 18, 18) != DecisionExhausted {
		t.Fatal("last attempt busy => exhausted")
	}
}

func TestDispatchReleased(t *testing.T) {
	h := &Handler{
		Loader:    &mockLoader{row: &cloudconfig.ConfigRow{}},
		Publisher: &recordingPublisher{},
		Sleep:     func(time.Duration) {},
	}
	raw, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "tenant_id": "100", "company_id": float64(100),
		"workspace_id": "ws1", "attempt": 2,
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: EventType,
		Envelope:  domain.EventEnvelope{EventType: EventType, Data: raw},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("out=%v err=%v", out, err)
	}
}

func TestDispatchWaitSchedulesNext(t *testing.T) {
	pub := &recordingPublisher{}
	h := &Handler{
		Loader:      &mockLoader{row: &cloudconfig.ConfigRow{ServerURL: "http://127.0.0.1:9"}},
		Publisher:   pub,
		Sleep:       func(time.Duration) {},
		MaxAttempts: 5,
		RetryDelay:  time.Millisecond,
	}
	raw, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "tenant_id": "100", "company_id": float64(100),
		"workspace_id": "ws1", "attempt": 1, "server_url": "http://127.0.0.1:9",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: EventType,
		Envelope:  domain.EventEnvelope{EventType: EventType, Data: raw},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("out=%v err=%v", out, err)
	}
	if len(pub.events) != 1 || pub.events[0].Type != EventType {
		t.Fatalf("events=%v", pub.events)
	}
	if pub.events[0].Data["attempt"] != 2 {
		t.Fatalf("next attempt=%v", pub.events[0].Data["attempt"])
	}
}
