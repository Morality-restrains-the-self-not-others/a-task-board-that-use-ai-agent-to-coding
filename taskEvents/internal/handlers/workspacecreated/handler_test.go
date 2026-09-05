package workspacecreated

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/repository/saas"
	"taskEvents/internal/saastest"
)

func TestHandleWorkspaceCreated(t *testing.T) {
	mux := saastest.NewIntentMux()
	srv := mux.Server()
	defer srv.Close()

	// Point taskProjectService client at the mock server.
	os.Setenv("TASK_PROJECT_SERVICE_URL", srv.URL)
	defer os.Unsetenv("TASK_PROJECT_SERVICE_URL")

	repo := saas.New("")
	h := &Handler{Repo: repo}

	// Test 1: string workspace_id (taskProjectService Go service format)
	data, _ := json.Marshal(map[string]interface{}{
		"workspace_id":   "ws_-3847919998110487740",
		"workspace_name": "用户的工作空间",
		"company_id":     "200",
		"created_by":     "42",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "WORKSPACE_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "WORKSPACE_CREATED", Data: data},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("expected DispatchSuccess, got %v", out)
	}
	// Verify workspace access was created for the creator
	found := false
	for _, a := range mux.WorkspaceAccesses {
		if a.WorkspaceID == "ws_-3847919998110487740" && a.UserID == "42" && a.Permission == "admin" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected admin workspace access for user 42, got %+v", mux.WorkspaceAccesses)
	}

	// Test 2: numeric workspace_id (legacy format, still supported via StrField)
	mux2 := saastest.NewIntentMux()
	srv2 := mux2.Server()
	defer srv2.Close()

	os.Setenv("TASK_PROJECT_SERVICE_URL", srv2.URL)
	repo2 := saas.New("")
	h2 := &Handler{Repo: repo2}

	data2, _ := json.Marshal(map[string]interface{}{
		"workspace_id":   999,
		"workspace_name": "ws",
		"company_id":     "100",
		"created_by":     "42",
	})
	out2, err2 := h2.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "WORKSPACE_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "WORKSPACE_CREATED", Data: data2},
	})
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}
	if out2 != domain.DispatchSuccess {
		t.Fatalf("expected DispatchSuccess, got %v", out2)
	}
	found2 := false
	for _, a := range mux2.WorkspaceAccesses {
		if a.WorkspaceID == "999" && a.UserID == "42" && a.Permission == "admin" {
			found2 = true
			break
		}
	}
	if !found2 {
		t.Fatalf("expected admin workspace access for user 42, got %+v", mux2.WorkspaceAccesses)
	}
}

func TestHandleWorkspaceCreatedWrongEvent(t *testing.T) {
	h := &Handler{Repo: saas.New("")}
	_, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "OTHER_EVENT",
		Envelope:  domain.EventEnvelope{EventType: "OTHER_EVENT", Data: []byte(`{}`)},
	})
	if err == nil {
		t.Fatal("expected error for unsupported event type")
	}
}

func TestHandleWorkspaceCreatedMissingWorkspaceID(t *testing.T) {
	h := &Handler{Repo: saas.New("")}
	data, _ := json.Marshal(map[string]interface{}{
		"workspace_name": "ws",
		"company_id":     "100",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "WORKSPACE_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "WORKSPACE_CREATED", Data: data},
	})
	if err == nil {
		t.Fatal("expected error for missing workspace_id")
	}
	if out != domain.DispatchPermanent {
		t.Fatalf("expected DispatchPermanent, got %v", out)
	}
}
