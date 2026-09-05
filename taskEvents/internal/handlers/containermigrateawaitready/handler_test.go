package containermigrateawaitready

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"taskEvents/domain"
	"taskEvents/internal/repository/cloudconfig"
)

func TestDecideAwaitReady(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		cfg      *cloudconfig.ConfigRow
		attempt  int
		max      int
		want     Decision
	}{
		{
			name:    "running_with_heartbeat",
			cfg:     &cloudconfig.ConfigRow{LastRuntimeStatus: "Running", ServerURL: "http://10.0.0.1:8080", InstanceID: "i-1"},
			attempt: 1, max: 5, want: DecisionReady,
		},
		{
			name:    "starting_wait",
			cfg:     &cloudconfig.ConfigRow{LastRuntimeStatus: "Starting", InstanceID: "i-1"},
			attempt: 1, max: 5, want: DecisionWait,
		},
		{
			name:    "pending_instance_wait",
			cfg:     &cloudconfig.ConfigRow{LastRuntimeStatus: "Running", ServerURL: "http://x", InstanceID: "pending-start-abc"},
			attempt: 1, max: 5, want: DecisionWait,
		},
		{
			name:    "running_no_heartbeat_retrigger",
			cfg:     &cloudconfig.ConfigRow{LastRuntimeStatus: "Running", ServerURL: "", InstanceID: "i-1"},
			attempt: 2, max: 5, want: DecisionTriggerStartWait,
		},
		{
			name:    "exhausted",
			cfg:     &cloudconfig.ConfigRow{LastRuntimeStatus: "Starting"},
			attempt: 5, max: 5, want: DecisionExhausted,
		},
		{
			name:    "nil_cfg_wait",
			cfg:     nil,
			attempt: 1, max: 5, want: DecisionWait,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := DecideAwaitReady(tc.cfg, tc.attempt, tc.max)
			if got != tc.want {
				t.Fatalf("DecideAwaitReady=%s want=%s", got, tc.want)
			}
		})
	}
}

type mockLoader struct {
	row *cloudconfig.ConfigRow
	err error
}

func (m *mockLoader) LoadForTask(companyID int64, workspaceID, taskID string) (*cloudconfig.ConfigRow, error) {
	return m.row, m.err
}

type recordingPublisher struct {
	mu     sync.Mutex
	events []publishedEvent
}

type publishedEvent struct {
	EventType string
	Data      map[string]interface{}
	Key       string
}

func (p *recordingPublisher) PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	copied := map[string]interface{}{}
	for k, v := range data {
		copied[k] = v
	}
	p.events = append(p.events, publishedEvent{EventType: eventType, Data: copied, Key: key})
	return nil
}

type recordingStarter struct {
	calls int
}

func (s *recordingStarter) Start(ctx context.Context, tenantID, workspaceID, taskID, imageID, imageURL string) error {
	s.calls++
	return nil
}

func makeCmd(data map[string]interface{}) domain.DomainCommand {
	raw, _ := json.Marshal(data)
	return domain.DomainCommand{
		EventType: EventType,
		Envelope:  domain.EventEnvelope{EventType: EventType, Data: raw, Key: "t1"},
	}
}

func TestHandlerReadySuccess(t *testing.T) {
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{
			LastRuntimeStatus: "Running", ServerURL: "http://10.0.0.2:9000", InstanceID: "i-ok",
		}},
		Publisher: &recordingPublisher{},
		Sleep:     func(time.Duration) {},
	}
	out, err := h.Dispatch(context.Background(), makeCmd(map[string]interface{}{
		"company_id": "1", "workspace_id": "ws", "task_id": "t1", "attempt": 1,
	}))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome=%v want Success", out)
	}
}

func TestHandlerNotReadyRetryableWithoutPublisher(t *testing.T) {
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{LastRuntimeStatus: "Starting", InstanceID: "i-1"}},
		Sleep:  func(time.Duration) {},
		MaxAttempts: 10,
	}
	out, err := h.Dispatch(context.Background(), makeCmd(map[string]interface{}{
		"company_id": float64(1), "workspace_id": "ws", "task_id": "t1", "attempt": 1,
	}))
	if out != domain.DispatchRetryable {
		t.Fatalf("outcome=%v want Retryable err=%v", out, err)
	}
	if err == nil {
		t.Fatal("expected retryable error")
	}
}

func TestHandlerSchedulesNextAttemptWithPublisher(t *testing.T) {
	pub := &recordingPublisher{}
	h := &Handler{
		Loader:      &mockLoader{row: &cloudconfig.ConfigRow{LastRuntimeStatus: "Starting", InstanceID: "i-1"}},
		Publisher:   pub,
		Sleep:       func(time.Duration) {},
		MaxAttempts: 10,
	}
	out, err := h.Dispatch(context.Background(), makeCmd(map[string]interface{}{
		"company_id": float64(1), "workspace_id": "ws", "task_id": "t1",
		"attempt": 3, "container_image_id": "img-1",
	}))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome=%v want Success (scheduled)", out)
	}
	pub.mu.Lock()
	defer pub.mu.Unlock()
	var found bool
	for _, e := range pub.events {
		if e.EventType != EventType {
			continue
		}
		found = true
		if fmt.Sprint(e.Data["attempt"]) != "4" {
			t.Fatalf("next attempt=%v want 4", e.Data["attempt"])
		}
	}
	if !found {
		t.Fatal("expected republish of CONTAINER_MIGRATE_AWAIT_READY")
	}
}

func TestHandlerRunningNoHeartbeatRetriggersStart(t *testing.T) {
	starter := &recordingStarter{}
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{
			LastRuntimeStatus: "Running", ServerURL: "", InstanceID: "i-1",
		}},
		Starter:     starter,
		Sleep:       func(time.Duration) {},
		MaxAttempts: 10,
	}
	out, err := h.Dispatch(context.Background(), makeCmd(map[string]interface{}{
		"company_id": float64(1), "workspace_id": "ws", "task_id": "t1", "attempt": 1,
		"container_image_id": "img-x",
	}))
	if out != domain.DispatchRetryable {
		t.Fatalf("outcome=%v want Retryable err=%v", out, err)
	}
	if starter.calls != 1 {
		t.Fatalf("starter calls=%d want 1", starter.calls)
	}
}

func TestHandlerMaxAttemptsPermanent(t *testing.T) {
	h := &Handler{
		Loader:      &mockLoader{row: &cloudconfig.ConfigRow{LastRuntimeStatus: "Starting"}},
		Sleep:       func(time.Duration) {},
		MaxAttempts: 5,
	}
	out, err := h.Dispatch(context.Background(), makeCmd(map[string]interface{}{
		"company_id": float64(1), "workspace_id": "ws", "task_id": "t1", "attempt": 5,
	}))
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome=%v want Permanent err=%v", out, err)
	}
	if err == nil {
		t.Fatal("expected permanent error")
	}
}
