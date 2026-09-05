package taskstatuschanged

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/repository/cloudconfig"
)

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

type recordingStopper struct {
	mu    sync.Mutex
	calls []localStopCall
}

type localStopCall struct {
	TenantID    string
	WorkspaceID string
	TaskID      string
	ServerURL   string
	StopReason  string
}

func (s *recordingStopper) StopLocal(ctx context.Context, tenantID, workspaceID, taskID, serverURL, stopReason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, localStopCall{
		TenantID: tenantID, WorkspaceID: workspaceID, TaskID: taskID,
		ServerURL: serverURL, StopReason: stopReason,
	})
	return nil
}

func (s *recordingStopper) StopContainerOnly(ctx context.Context, taskID, stopReason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, localStopCall{TaskID: taskID, StopReason: stopReason})
	return nil
}

func cmdWithData(data map[string]interface{}) domain.DomainCommand {
	raw, _ := json.Marshal(data)
	return domain.DomainCommand{
		EventType: "TASK_STATUS_CHANGED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_STATUS_CHANGED", Data: raw},
	}
}

func TestDispatchNonTerminal(t *testing.T) {
	pub := &recordingPublisher{}
	stopper := &recordingStopper{}
	h := &Handler{
		Loader:    &mockLoader{row: &cloudconfig.ConfigRow{InstanceID: "i-1"}},
		Publisher: pub,
		Stopper:   stopper,
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id":              "t1",
		"tenant_id":            "100",
		"company_id":           "100",
		"workspace_id":         "ws1",
		"progress_column_name": "进行中",
		"previous_completed":   false,
		"completed":            false,
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v want success", outcome)
	}
	if len(pub.events) != 0 {
		t.Fatalf("expected no publish, got %d", len(pub.events))
	}
	if len(stopper.calls) != 0 {
		t.Fatalf("expected no local stop, got %d", len(stopper.calls))
	}
}

// TestDispatchNoConfigRetry: 无 CSC / 仅空模板 → success no-op（不再 retry 以免 DLT）。
func TestDispatchNoConfigRetry(t *testing.T) {
	pub := &recordingPublisher{}
	h := &Handler{
		Loader:    &mockLoader{row: nil},
		Publisher: pub,
		Stopper:   &recordingStopper{},
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id":              "t1",
		"tenant_id":            "100",
		"company_id":           float64(100),
		"workspace_id":         "ws1",
		"progress_column_name": "已完成",
		"previous_completed":   false,
		"completed":            false,
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v want success", outcome)
	}
	if len(pub.events) != 0 {
		t.Fatalf("expected no publish, got %d", len(pub.events))
	}
}

func TestDispatchInstancePublishesCloudServerStopped(t *testing.T) {
	pub := &recordingPublisher{}
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{
			InstanceID:      "i-abc",
			Region:          "cn-hongkong",
			Platform:        "aliyun",
			AuthorizationID: "42",
			WorkspaceID:     "ws1",
		}},
		Publisher: pub,
		Stopper:   &recordingStopper{},
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id":              "t1",
		"tenant_id":            "100",
		"company_id":           float64(100),
		"workspace_id":         "ws1",
		"progress_column_name": "Completed",
		"previous_completed":   false,
		"completed":            false,
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v want success", outcome)
	}
	if len(pub.events) != 1 {
		t.Fatalf("expected 1 publish, got %d", len(pub.events))
	}
	ev := pub.events[0]
	if ev.EventType != "CLOUD_SERVER_STOPPED" {
		t.Fatalf("event_type=%s", ev.EventType)
	}
	if ev.Data["instance_id"] != "i-abc" {
		t.Fatalf("instance_id=%v", ev.Data["instance_id"])
	}
	if ev.Data["stop_reason"] != "task_status_completed" {
		t.Fatalf("stop_reason=%v", ev.Data["stop_reason"])
	}
	if ev.Key != "t1" {
		t.Fatalf("key=%s", ev.Key)
	}
}

func TestDispatchGracefulNotifySchedulesAwait(t *testing.T) {
	pub := &recordingPublisher{}
	stopper := &recordingStopper{}
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{
			ServerURL: "http://127.0.0.1:19001",
		}},
		Publisher: pub,
		Stopper:   stopper,
		Notifier:  stubNotifier{err: nil},
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id":       "t2",
		"tenant_id":     "100",
		"company_id":    float64(100),
		"workspace_id":  "ws1",
		"terminal_kind": "cancelled",
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v want success", outcome)
	}
	if len(stopper.calls) != 0 {
		t.Fatalf("expected no local stop on graceful notify, got %d", len(stopper.calls))
	}
	if len(pub.events) != 1 || pub.events[0].EventType != GracefulShutdownAwaitEvent {
		t.Fatalf("events=%v", pub.events)
	}
}

type stubNotifier struct{ err error }

func (s stubNotifier) NotifyShutdown(context.Context, string, string, string, string) error {
	return s.err
}

func TestDispatchServerURLCallsLocalStopper(t *testing.T) {
	stopper := &recordingStopper{}
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{
			ServerURL: "http://127.0.0.1:19001",
		}},
		Publisher: &recordingPublisher{},
		Stopper:   stopper,
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id":       "t2",
		"tenant_id":     "100",
		"company_id":    float64(100),
		"workspace_id":  "ws1",
		"terminal_kind": "cancelled",
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v want success", outcome)
	}
	if len(stopper.calls) != 1 {
		t.Fatalf("expected 1 local stop, got %d", len(stopper.calls))
	}
	c := stopper.calls[0]
	if c.ServerURL != "http://127.0.0.1:19001" {
		t.Fatalf("server_url=%s", c.ServerURL)
	}
	if c.StopReason != "task_status_cancelled" {
		t.Fatalf("stop_reason=%s", c.StopReason)
	}
	if c.TaskID != "t2" {
		t.Fatalf("task_id=%s", c.TaskID)
	}
}

func TestDispatchMigratesForeignBeforeStop(t *testing.T) {
	pub := &recordingPublisher{}
	mig := &recordingMigrator{
		foreign: []BusyForeignBinding{{TaskID: "task-b", ServerURL: "http://10.0.0.1:9"}},
	}
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{
			InstanceID:      "i-shared",
			Region:          "cn-hongkong",
			Platform:        "aliyun",
			AuthorizationID: "42",
			WorkspaceID:     "ws1",
		}},
		Publisher: pub,
		Stopper:   &recordingStopper{},
		Migrator:  mig,
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id":              "task-a",
		"tenant_id":            "100",
		"company_id":           float64(100),
		"workspace_id":         "ws1",
		"progress_column_name": "已完成",
	}))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v", outcome)
	}
	if len(mig.migrateCalls) != 1 || mig.migrateCalls[0] != "task-b" {
		t.Fatalf("migrateCalls=%v", mig.migrateCalls)
	}
	if len(mig.clearCalls) != 1 || mig.clearCalls[0] != "i-shared" {
		t.Fatalf("clearCalls=%v", mig.clearCalls)
	}
	if len(mig.flagCalls) != 1 || mig.flagCalls[0] != "task-a" {
		t.Fatalf("flagCalls=%v", mig.flagCalls)
	}
	stopper := h.Stopper.(*recordingStopper)
	if len(stopper.calls) < 1 || stopper.calls[0].TaskID != "task-b" {
		t.Fatalf("expected foreign container-only stop first, calls=%v", stopper.calls)
	}
	if len(pub.events) != 1 || pub.events[0].EventType != "CLOUD_SERVER_STOPPED" {
		t.Fatalf("events=%v", pub.events)
	}
	if len(mig.markCalls) != 1 || mig.markCalls[0] != "task-a" {
		t.Fatalf("markCalls=%v", mig.markCalls)
	}
}

func TestDispatchMigrateFailureDoesNotStop(t *testing.T) {
	pub := &recordingPublisher{}
	mig := &recordingMigrator{
		foreign:    []BusyForeignBinding{{TaskID: "task-b", ServerURL: "http://x"}},
		migrateErr: fmt.Errorf("migrate boom"),
	}
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{
			InstanceID: "i-shared", Region: "cn-hongkong", AuthorizationID: "1", WorkspaceID: "ws1",
		}},
		Publisher: pub,
		Stopper:   &recordingStopper{},
		Migrator:  mig,
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id": "task-a", "tenant_id": "100", "company_id": float64(100),
		"workspace_id": "ws1", "terminal_kind": "cancelled",
	}))
	if err == nil {
		t.Fatal("expected migrate error")
	}
	if outcome != domain.DispatchRetryable {
		t.Fatalf("outcome=%v", outcome)
	}
	if len(pub.events) != 0 {
		t.Fatalf("must not publish STOPPED on migrate failure, got %d", len(pub.events))
	}
}

type recordingMigrator struct {
	foreign      []BusyForeignBinding
	listErr      error
	migrateErr   error
	markErr      error
	migrateCalls []string
	markCalls    []string
	flagCalls    []string
	clearCalls   []string
}

func (m *recordingMigrator) ListBusyForeign(context.Context, int64, string, string, string) ([]BusyForeignBinding, error) {
	return m.foreign, m.listErr
}
func (m *recordingMigrator) MigrateOff(_ context.Context, _, _, ownerTaskID, _ string) error {
	m.migrateCalls = append(m.migrateCalls, ownerTaskID)
	return m.migrateErr
}
func (m *recordingMigrator) ClearIdleSiblings(_ context.Context, _, _, instanceID, _ string) error {
	m.clearCalls = append(m.clearCalls, instanceID)
	return nil
}
func (m *recordingMigrator) SetTerminalReleasedFlag(_ context.Context, _, _, taskID string) error {
	m.flagCalls = append(m.flagCalls, taskID)
	return nil
}
func (m *recordingMigrator) MarkTerminalReleased(_ context.Context, _, _, taskID string) error {
	m.markCalls = append(m.markCalls, taskID)
	return m.markErr
}

func TestDispatchCompletedBecameTrue(t *testing.T) {
	pub := &recordingPublisher{}
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{
			InstanceID:      "i-legacy",
			Region:          "cn-hongkong",
			AuthorizationID: "1",
		}},
		Publisher: pub,
		Stopper:   &recordingStopper{},
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id":            "t3",
		"company_id":         float64(100),
		"tenant_id":          "100",
		"workspace_id":       "ws1",
		"previous_completed": false,
		"completed":          true,
	}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v", outcome)
	}
	if len(pub.events) != 1 {
		t.Fatalf("expected publish, got %d", len(pub.events))
	}
	if pub.events[0].Data["stop_reason"] != "task_status_completed" {
		t.Fatalf("stop_reason=%v", pub.events[0].Data["stop_reason"])
	}
}
