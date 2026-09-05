package jobstreampersist

import (
	"context"
	"encoding/json"
	"testing"

	"taskEvents/domain"
)

func TestDispatchSkipsNonJobStream(t *testing.T) {
	called := false
	h := &Handler{Persist: func(ctx context.Context, rec JobStreamPersistRequest) error {
		called = true
		return nil
	}}
	data, _ := json.Marshal(map[string]any{
		"task_id":     "t1",
		"status_data": map[string]any{"status": "container_heartbeat"},
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "SSE_MESSAGE",
		Envelope:  domain.EventEnvelope{EventType: "SSE_MESSAGE", Data: data},
	})
	if err != nil || out != domain.DispatchSuccess || called {
		t.Fatalf("out=%v err=%v called=%v", out, err, called)
	}
}

func TestDispatchSkipsChunk(t *testing.T) {
	called := false
	h := &Handler{Persist: func(ctx context.Context, rec JobStreamPersistRequest) error {
		called = true
		return nil
	}}
	data, _ := json.Marshal(map[string]any{
		"task_id": "task1",
		"status_data": map[string]any{
			"status": "container_job_stream", "job_id": "J1", "phase": "chunk", "message": "out",
		},
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "SSE_MESSAGE",
		Envelope:  domain.EventEnvelope{EventType: "SSE_MESSAGE", Data: data},
	})
	if err != nil || out != domain.DispatchSuccess || called {
		t.Fatalf("out=%v err=%v called=%v", out, err, called)
	}
}

func TestDispatchPersistsStep(t *testing.T) {
	var got JobStreamPersistRequest
	h := &Handler{Persist: func(ctx context.Context, rec JobStreamPersistRequest) error {
		got = rec
		return nil
	}}
	data, _ := json.Marshal(map[string]any{
		"task_id": "task1",
		"status_data": map[string]any{
			"status": "container_job_stream", "job_id": "J1", "phase": "step",
			"message": "step 2: ls", "seq": 4, "step_number": 2,
			"comment_id": "cmt-a", "company_id": "t1", "workspace_id": "w1",
			"delivery_summary": "ls", "job_status": "running",
		},
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "SSE_MESSAGE",
		Envelope:  domain.EventEnvelope{EventType: "SSE_MESSAGE", Data: data},
	})
	if err != nil || out != domain.DispatchSuccess {
		t.Fatalf("out=%v err=%v", out, err)
	}
	if got.TaskID != "task1" || got.JobID != "J1" || got.Phase != "step" || got.Seq != 4 || got.StepNumber != 2 {
		t.Fatalf("got=%+v", got)
	}
	if got.CommentID != "cmt-a" || got.CompanyID != "t1" {
		t.Fatalf("scope=%+v", got)
	}
}

func TestShouldPersistPhase(t *testing.T) {
	if !ShouldPersistPhase("step") || ShouldPersistPhase("chunk") {
		t.Fatal("phase filter")
	}
}
