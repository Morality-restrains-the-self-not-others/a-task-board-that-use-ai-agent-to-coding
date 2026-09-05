package workpanelfanout

import (
	"context"
	"encoding/json"
	"testing"

	"taskEvents/domain"
)

func TestDispatchMissingWorkspace(t *testing.T) {
	h := &Handler{}
	data, _ := json.Marshal(map[string]interface{}{"task_id": "t1"})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_STATUS_CHANGED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_STATUS_CHANGED", Data: data},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v", out)
	}
}

func TestDispatchPublishesWorkspaceChannel(t *testing.T) {
	var gotChannel string
	var gotPayload map[string]interface{}
	h := &Handler{
		Publish: func(_ context.Context, channel string, payload []byte) error {
			gotChannel = channel
			if err := json.Unmarshal(payload, &gotPayload); err != nil {
				t.Fatal(err)
			}
			return nil
		},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id":                     "task-9",
		"tenant_id":                   "850",
		"workspace_id":                "ws-1",
		"previous_progress_column_id": "col-a",
		"progress_column_id":          "col-b",
		"previous_completed":          false,
		"completed":                   false,
		"progress_column_name":        "进行中",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_STATUS_CHANGED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_STATUS_CHANGED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if gotChannel != "sse:workspace:ws-1" {
		t.Fatalf("channel=%s", gotChannel)
	}
	if gotPayload["task_id"] != "workspace:ws-1" {
		t.Fatalf("hub task_id=%v", gotPayload["task_id"])
	}
	status, _ := gotPayload["status_data"].(map[string]interface{})
	if status["event_name"] != "task_status_changed" {
		t.Fatalf("event_name=%v", status["event_name"])
	}
	if status["task_id"] != "task-9" {
		t.Fatalf("real task_id=%v", status["task_id"])
	}
	if status["progress_column_id"] != "col-b" {
		t.Fatalf("progress_column_id=%v", status["progress_column_id"])
	}
}

func TestDispatchWrongEvent(t *testing.T) {
	h := &Handler{}
	_, err := h.Dispatch(context.Background(), domain.DomainCommand{EventType: "OTHER"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDispatchTaskCreated(t *testing.T) {
	var gotChannel string
	var gotPayload map[string]interface{}
	h := &Handler{
		Publish: func(_ context.Context, channel string, payload []byte) error {
			gotChannel = channel
			if err := json.Unmarshal(payload, &gotPayload); err != nil {
				t.Fatal(err)
			}
			return nil
		},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id":      "task-10",
		"tenant_id":    "850",
		"workspace_id": "ws-1",
		"title":        "新任务",
		"user_id":      "user-1",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if gotChannel != "sse:workspace:ws-1" {
		t.Fatalf("channel=%s", gotChannel)
	}
	if gotPayload["task_id"] != "workspace:ws-1" {
		t.Fatalf("hub task_id=%v", gotPayload["task_id"])
	}
	status, _ := gotPayload["status_data"].(map[string]interface{})
	if status["event_name"] != "task_created" {
		t.Fatalf("event_name=%v", status["event_name"])
	}
	if status["task_id"] != "task-10" {
		t.Fatalf("real task_id=%v", status["task_id"])
	}
	if status["title"] != "新任务" {
		t.Fatalf("title=%v", status["title"])
	}
}

func TestDispatchTaskDeleted(t *testing.T) {
	var gotChannel string
	var gotPayload map[string]interface{}
	h := &Handler{
		Publish: func(_ context.Context, channel string, payload []byte) error {
			gotChannel = channel
			if err := json.Unmarshal(payload, &gotPayload); err != nil {
				t.Fatal(err)
			}
			return nil
		},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id":      "task-11",
		"tenant_id":    "850",
		"workspace_id": "ws-1",
		"company_id":   "850",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_DELETED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_DELETED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if gotChannel != "sse:workspace:ws-1" {
		t.Fatalf("channel=%s", gotChannel)
	}
	status, _ := gotPayload["status_data"].(map[string]interface{})
	if status["event_name"] != "task_deleted" {
		t.Fatalf("event_name=%v", status["event_name"])
	}
	if status["task_id"] != "task-11" {
		t.Fatalf("real task_id=%v", status["task_id"])
	}
}
