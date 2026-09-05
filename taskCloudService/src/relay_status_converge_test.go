package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryRelayClient struct {
	mu   sync.Mutex
	data map[string]string
}

func newMemoryRelayClient() *memoryRelayClient {
	return &memoryRelayClient{data: map[string]string{}}
}

func (m *memoryRelayClient) Get(_ context.Context, key string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	return v, ok, nil
}

func (m *memoryRelayClient) SetEX(_ context.Context, key string, value string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *memoryRelayClient) Del(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func setupRelayMemoryStore(t *testing.T) *memoryRelayClient {
	t.Helper()
	client := newMemoryRelayClient()
	setRelaySessionStoreForTest(&relaySessionStore{
		client: client,
		ttl:    2 * time.Hour,
	})
	t.Cleanup(func() { setRelaySessionStoreForTest(nil) })
	return client
}

func seedRelaySession(t *testing.T, client *memoryRelayClient, session *RelayStartupSession) {
	t.Helper()
	raw, err := serializeRelaySession(session)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	_ = client.SetEX(context.Background(), workflowKey(session.WorkflowID), raw, 0)
	_ = client.SetEX(context.Background(), scopeKey(session.Scope), session.WorkflowID, 0)
}

func TestConvergeRelayStatusPushByScopeUpdatesSession(t *testing.T) {
	client := setupRelayMemoryStore(t)
	now := time.Date(2026, 7, 10, 1, 0, 0, 0, time.UTC)
	reqID := "req-1"
	seedRelaySession(t, client, &RelayStartupSession{
		WorkflowID:       "wf-1",
		Scope:            RelayTaskScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Phase:            "start_dispatching",
		RequestID:        &reqID,
		TokenInitialized: true,
		CreatedAt:        now.Add(-time.Minute),
		UpdatedAt:        now.Add(-time.Minute),
		CreatedAtRaw:     now.Add(-time.Minute).Format(time.RFC3339Nano),
		UpdatedAtRaw:     now.Add(-time.Minute).Format(time.RFC3339Nano),
	})

	seq := 7
	result := convergeRelayStatusPushByScope(
		context.Background(),
		relaySessions,
		RelayTaskScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		&seq,
		RelayStatusSnapshot{Running: true, OnlineServiceUp: true},
		"trace-x",
		now,
	)
	if !result.Converged {
		t.Fatalf("result=%+v", result)
	}
	if result.WorkflowID != "wf-1" || result.TraceID != "trace-x" {
		t.Fatalf("result=%+v", result)
	}

	raw, ok, err := client.Get(context.Background(), workflowKey("wf-1"))
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	loaded, err := deserializeRelaySession(raw)
	if err != nil {
		t.Fatalf("deserialize: %v", err)
	}
	if loaded.Phase != relayPhaseRunning {
		t.Fatalf("phase=%q", loaded.Phase)
	}
	if loaded.LastStatusSeq == nil || *loaded.LastStatusSeq != 7 {
		t.Fatalf("seq=%v", loaded.LastStatusSeq)
	}
	if loaded.LastStatus == nil || !loaded.LastStatus.Running || !loaded.LastStatus.OnlineServiceUp {
		t.Fatalf("last_status=%v", loaded.LastStatus)
	}
}

func TestConvergeRelayStatusPushByScopeMissingSession(t *testing.T) {
	_ = setupRelayMemoryStore(t)
	seq := 3
	result := convergeRelayStatusPushByScope(
		context.Background(),
		relaySessions,
		RelayTaskScope{TenantID: "t", WorkspaceID: "w", TaskID: "missing"},
		&seq,
		RelayStatusSnapshot{Running: true, OnlineServiceUp: true},
		"trace-miss",
		time.Now().UTC(),
	)
	if result.Converged {
		t.Fatal("expected not converged")
	}
	if result.ErrorCode != "status_converge_session_missing" {
		t.Fatalf("code=%q", result.ErrorCode)
	}
	if result.TraceID != "trace-miss" {
		t.Fatalf("trace=%q", result.TraceID)
	}
}

func TestConvergeStatusPushRejectsSeqRegression(t *testing.T) {
	prev := 10
	session := &RelayStartupSession{LastStatusSeq: &prev, Phase: "running"}
	seq := 5
	err := convergeStatusPushOnSession(session, &seq, RelayStatusSnapshot{Running: true, OnlineServiceUp: true}, time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "回退") {
		t.Fatalf("err=%v", err)
	}
}

func TestHandleRelayStatusPushReturnsAckAndConvergesLocally(t *testing.T) {
	client := setupRelayMemoryStore(t)
	now := time.Now().UTC()
	seedRelaySession(t, client, &RelayStartupSession{
		WorkflowID:       "wf-ack",
		Scope:            RelayTaskScope{TenantID: "t1", WorkspaceID: "w1", TaskID: "task1"},
		Phase:            "start_accepted",
		TokenInitialized: true,
		CreatedAt:        now,
		UpdatedAt:        now,
		CreatedAtRaw:     now.Format(time.RFC3339Nano),
		UpdatedAtRaw:     now.Format(time.RFC3339Nano),
	})

	prevKafka := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = "" // publishSSEMessage no-ops without kafka
	defer func() {
		cfg.KafkaBootstrapServers = prevKafka
	}()

	rec := httptest.NewRecorder()
	handleRelayStatusPush(
		rec, nil,
		&CloudServerConfig{ID: "cfg1", CompanyID: "t1", WorkspaceID: "w1", TaskID: "task1", AuthorizationID: "a1"},
		map[string]any{"seq": float64(7), "status": map[string]any{"running": true, "online_service_up": true}},
		"t1", "w1", "task1", "tok", "trace-r",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["status"] != "ok" || out["task_id"] != "task1" {
		t.Fatalf("out=%v", out)
	}
	if out["ack"] != float64(7) {
		t.Fatalf("ack=%v", out["ack"])
	}
	flushRelayStatusSSEDebounceForTest("task1")

	deadline := time.Now().Add(2 * time.Second)
	for {
		raw, ok, err := client.Get(context.Background(), workflowKey("wf-ack"))
		if err == nil && ok {
			loaded, derr := deserializeRelaySession(raw)
			if derr == nil && loaded.Phase == relayPhaseRunning {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("session not converged; last err=%v ok=%v", err, ok)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestPublishRelayStatusSSEBuildsDjangoCompatibleStatusData(t *testing.T) {
	got := buildRelayToTraeStatusData(map[string]any{
		"running":           true,
		"online_service_up": true,
		"ui_url":            "http://example/ui",
	})
	if got["status"] != relayToTraeSSEStatus {
		t.Fatalf("status=%v", got["status"])
	}
	if got["event_name"] != "server_status_update" {
		t.Fatalf("event_name=%v", got["event_name"])
	}
	if got["message"] != "http://example/ui" {
		t.Fatalf("message=%v", got["message"])
	}
	payload, _ := got["relay_payload"].(map[string]any)
	if payload == nil || payload["running"] != true {
		t.Fatalf("relay_payload=%v", got["relay_payload"])
	}
}

func TestScheduleRelayStatusSSEDebouncedCoalesces(t *testing.T) {
	prevKafka := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = ""
	defer func() { cfg.KafkaBootstrapServers = prevKafka }()

	scheduleRelayStatusSSEDebounced("task-debounce", map[string]any{"running": false}, "t1")
	scheduleRelayStatusSSEDebounced("task-debounce", map[string]any{"running": true, "ui_url": "u"}, "t2")

	relayStatusSSEMu.Lock()
	pending := relayStatusSSEByTaskID["task-debounce"]
	relayStatusSSEMu.Unlock()
	if pending == nil {
		t.Fatal("expected pending debounce entry")
	}
	if pending.payload["running"] != true {
		t.Fatalf("expected latest payload, got %v", pending.payload)
	}
	if pending.traceID != "t2" {
		t.Fatalf("traceID=%q", pending.traceID)
	}
	flushRelayStatusSSEDebounceForTest("task-debounce")
	relayStatusSSEMu.Lock()
	_, still := relayStatusSSEByTaskID["task-debounce"]
	relayStatusSSEMu.Unlock()
	if still {
		t.Fatal("pending should be cleared after flush")
	}
}

func TestHandleRelayStatusPushAckWithoutSessionStill200(t *testing.T) {
	_ = setupRelayMemoryStore(t)

	rec := httptest.NewRecorder()
	handleRelayStatusPush(
		rec, nil,
		&CloudServerConfig{ID: "cfg1", CompanyID: "t1", WorkspaceID: "w1", TaskID: "task-miss"},
		map[string]any{"seq": float64(1), "status": map[string]any{"running": false}},
		"t1", "w1", "task-miss", "tok", "",
	)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	time.Sleep(50 * time.Millisecond)
}
