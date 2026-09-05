package taskstatuschanged

import (
	"context"
	"fmt"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/repository/cloudconfig"
)

type mockLister struct {
	list *cloudconfig.TaskRuntimeList
	err  error
}

func (m *mockLister) ListForTask(companyID int64, workspaceID, taskID string) (*cloudconfig.TaskRuntimeList, error) {
	return m.list, m.err
}

func TestDispatchNoRunningResourceSuccess(t *testing.T) {
	pub := &recordingPublisher{}
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{
			InstanceID: "", ServerURL: "", LastRuntimeStatus: "",
		}},
		Publisher: pub,
		Stopper:   &recordingStopper{},
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id": "task_empty", "tenant_id": "100", "company_id": float64(100),
		"workspace_id": "ws1", "terminal_kind": "completed",
	}))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v want success", outcome)
	}
	if len(pub.events) != 0 {
		t.Fatalf("expected no STOPPED, got %d", len(pub.events))
	}
}

func TestDispatchStartingNoInstanceNoop(t *testing.T) {
	pub := &recordingPublisher{}
	h := &Handler{
		Loader: &mockLoader{row: &cloudconfig.ConfigRow{
			CommentID:         "cmt-starting",
			LastRuntimeStatus: "Starting",
		}},
		Publisher: pub,
		Stopper:   &recordingStopper{},
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id": "task_start", "tenant_id": "100", "company_id": float64(100),
		"workspace_id": "ws1", "terminal_kind": "cancelled",
	}))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v", outcome)
	}
	if len(pub.events) != 0 {
		t.Fatalf("Starting without instance must no-op, got %d events", len(pub.events))
	}
}

// Progress → 已完成/已取消 must stop VMs that already have instance_id even while Starting/Pending
// (RunInstances returns id before status becomes Running).
func TestDispatchStartingWithInstancePublishesStop(t *testing.T) {
	pub := &recordingPublisher{}
	h := &Handler{
		Lister: &mockLister{list: &cloudconfig.TaskRuntimeList{
			RunningMachineCount: 0,
			Comments: []cloudconfig.ConfigRow{
				{
					CommentID:         "cmt-boot",
					InstanceID:        "i-starting-1",
					Region:            "cn-qingdao",
					Platform:          "aliyun",
					AuthorizationID:   "auth-1",
					LastRuntimeStatus: "Starting",
				},
			},
		}},
		Publisher: pub,
		Stopper:   &recordingStopper{},
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id": "task_boot", "tenant_id": "100", "company_id": float64(100),
		"workspace_id": "ws1", "terminal_kind": "completed",
		"progress_column_name": "已完成",
	}))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v", outcome)
	}
	if len(pub.events) != 1 {
		t.Fatalf("Starting+instance must publish STOPPED, got %d", len(pub.events))
	}
	ev := pub.events[0]
	if ev.EventType != "CLOUD_SERVER_STOPPED" {
		t.Fatalf("event_type=%s", ev.EventType)
	}
	if ev.Data["instance_id"] != "i-starting-1" || ev.Data["comment_id"] != "cmt-boot" {
		t.Fatalf("payload=%v", ev.Data)
	}
	if ev.Key != "task_boot:cmt-boot" {
		t.Fatalf("key=%s", ev.Key)
	}
}

func TestDispatchTwoCommentsPublishesTwoStops(t *testing.T) {
	pub := &recordingPublisher{}
	h := &Handler{
		Lister: &mockLister{list: &cloudconfig.TaskRuntimeList{
			RunningMachineCount: 2,
			Comments: []cloudconfig.ConfigRow{
				{CommentID: "cmt-a", InstanceID: "i-a", Region: "cn-hongkong", Platform: "aliyun", AuthorizationID: "1"},
				{CommentID: "cmt-b", InstanceID: "i-b", Region: "cn-qingdao", Platform: "aliyun", AuthorizationID: "1"},
			},
		}},
		Publisher: pub,
		Stopper:   &recordingStopper{},
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id": "t1", "tenant_id": "100", "company_id": float64(100),
		"workspace_id": "ws1", "terminal_kind": "completed",
	}))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v", outcome)
	}
	if len(pub.events) != 2 {
		t.Fatalf("expected 2 STOPPED, got %d", len(pub.events))
	}
	seen := map[string]string{}
	for _, ev := range pub.events {
		if ev.EventType != "CLOUD_SERVER_STOPPED" {
			t.Fatalf("event_type=%s", ev.EventType)
		}
		cid, _ := ev.Data["comment_id"].(string)
		iid, _ := ev.Data["instance_id"].(string)
		rid, _ := ev.Data["region_id"].(string)
		if cid == "" || iid == "" || rid == "" {
			t.Fatalf("missing self-contained fields: %+v", ev.Data)
		}
		seen[cid] = ev.Key
	}
	if seen["cmt-a"] != "t1:cmt-a" || seen["cmt-b"] != "t1:cmt-b" {
		t.Fatalf("keys=%v", seen)
	}
}

func TestDispatchLaunchRequestOnlyMarksCommentReleased(t *testing.T) {
	orig := markCommentTerminalReleasedFn
	defer func() { markCommentTerminalReleasedFn = orig }()
	var called []string
	markCommentTerminalReleasedFn = func(tenantID, workspaceID, taskID, commentID string) error {
		called = append(called, tenantID+"|"+workspaceID+"|"+taskID+"|"+commentID)
		return nil
	}
	pub := &recordingPublisher{}
	h := &Handler{
		Lister: &mockLister{list: &cloudconfig.TaskRuntimeList{
			RunningMachineCount: 0,
			Comments: []cloudconfig.ConfigRow{
				{CommentID: "cmt-lr", LaunchRequestID: "req-123"},
			},
		}},
		Publisher: pub,
		Stopper:   &recordingStopper{},
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id": "task_lr", "tenant_id": "100", "company_id": float64(100),
		"workspace_id": "ws1", "terminal_kind": "completed",
	}))
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if outcome != domain.DispatchSuccess {
		t.Fatalf("outcome=%v want success", outcome)
	}
	if len(called) != 1 || called[0] != "100|ws1|task_lr|cmt-lr" {
		t.Fatalf("markCommentTerminalReleased calls=%v", called)
	}
	if len(pub.events) != 0 {
		t.Fatalf("launch-request-only must not publish STOPPED, got %d", len(pub.events))
	}
}

func TestDispatchLaunchRequestOnlyMarkFailureRetryable(t *testing.T) {
	orig := markCommentTerminalReleasedFn
	defer func() { markCommentTerminalReleasedFn = orig }()
	markCommentTerminalReleasedFn = func(tenantID, workspaceID, taskID, commentID string) error {
		return fmt.Errorf("cloud down")
	}
	h := &Handler{
		Lister: &mockLister{list: &cloudconfig.TaskRuntimeList{
			RunningMachineCount: 0,
			Comments: []cloudconfig.ConfigRow{
				{CommentID: "cmt-lr", LaunchRequestID: "req-123"},
			},
		}},
		Publisher: &recordingPublisher{},
		Stopper:   &recordingStopper{},
	}
	outcome, err := h.Dispatch(context.Background(), cmdWithData(map[string]interface{}{
		"task_id": "task_lr", "tenant_id": "100", "company_id": float64(100),
		"workspace_id": "ws1", "terminal_kind": "completed",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	if outcome != domain.DispatchRetryable {
		t.Fatalf("outcome=%v want retryable", outcome)
	}
}
