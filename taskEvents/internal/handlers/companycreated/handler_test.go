package companycreated

import (
	"context"
	"encoding/json"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/repository/saas"
	"taskEvents/internal/saastest"
)

func TestHandleCompanyCreatedChain(t *testing.T) {
	mux := saastest.NewIntentMux()
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	var published map[string]interface{}
	h := &Handler{
		Repo: repo,
		Publisher: &stubPublisher{fn: func(_ context.Context, eventType string, data map[string]interface{}, _ string) error {
			if eventType != "WORKSPACE_CREATED" {
				t.Fatalf("unexpected event %s", eventType)
			}
			published = data
			return nil
		}},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"company_id": 100,
		"creator_id": 42,
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "COMPANY_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "COMPANY_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if published == nil {
		t.Fatal("WORKSPACE_CREATED not published")
	}
	if published["is_default"] != true {
		t.Fatalf("published: %+v", published)
	}
	if !mux.DeliverableTenants["100"] || !mux.ProgressTenants["100"] {
		t.Fatalf("defaults not set: deliverable=%v progress=%v", mux.DeliverableTenants, mux.ProgressTenants)
	}
	if _, ok := mux.Workspaces["100"]; !ok {
		t.Fatal("workspace not created")
	}
}

// TestHandleCompanyCreatedWorkspaceNameCompany（OPT-20260827-020）：
// 新公司默认工作空间名称用公司名（"{company} 的工作空间"），
// 而非固定「用户的工作空间」，避免跨租户下拉全部撞名。
func TestHandleCompanyCreatedWorkspaceNameCompany(t *testing.T) {
	mux := saastest.NewIntentMux()
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	h := &Handler{
		Repo:      repo,
		Publisher: &stubPublisher{},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"company_id":   100,
		"creator_id":   42,
		"company_name": "启元科技",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "COMPANY_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "COMPANY_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	ws, ok := mux.Workspaces["100"]
	if !ok {
		t.Fatal("workspace not created")
	}
	if ws.Name != "启元科技 的工作空间" {
		t.Fatalf("workspace name = %q, want %q", ws.Name, "启元科技 的工作空间")
	}
}

// TestHandleCompanyCreatedWorkspaceNameEmptyCompanyName 回归：
// 事件缺 company_name 时仍回退固定名「用户的工作空间」，不阻断建空间。
func TestHandleCompanyCreatedWorkspaceNameEmptyCompanyName(t *testing.T) {
	mux := saastest.NewIntentMux()
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: &stubPublisher{}}
	data, _ := json.Marshal(map[string]interface{}{"company_id": 200, "creator_id": 42})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "COMPANY_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "COMPANY_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	ws, ok := mux.Workspaces["200"]
	if !ok {
		t.Fatal("workspace not created")
	}
	if ws.Name != "用户的工作空间" {
		t.Fatalf("workspace name = %q, want fallback %q", ws.Name, "用户的工作空间")
	}
}

func TestHandleCompanyCreatedIdempotent(t *testing.T) {
	mux := saastest.NewIntentMux()
	mux.Workspaces["100"] = saastest.Workspace{ID: 999, Name: "existing"}
	srv := mux.Server()
	defer srv.Close()
	cleanup := mux.SetServiceEnv(srv.URL)
	defer cleanup()

	repo := saas.New("")
	pub := &stubPublisher{}
	h := &Handler{Repo: repo, Publisher: pub}
	data, _ := json.Marshal(map[string]interface{}{"company_id": 100, "creator_id": 42})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "COMPANY_CREATED",
		Envelope:  domain.EventEnvelope{EventType: "COMPANY_CREATED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if pub.calls != 1 {
		t.Fatalf("expected idempotent retry to republish WORKSPACE_CREATED once, got %d", pub.calls)
	}
}

type stubPublisher struct {
	calls int
	fn    func(context.Context, string, map[string]interface{}, string) error
}

func (s *stubPublisher) PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	s.calls++
	if s.fn != nil {
		return s.fn(ctx, eventType, data, key)
	}
	return nil
}
