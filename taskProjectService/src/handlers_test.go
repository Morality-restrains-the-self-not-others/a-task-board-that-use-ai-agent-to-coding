package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	setupProjectTestDB(t)
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", body["status"])
	}
	if body["service"] != "taskProjectService" {
		t.Errorf("expected service=taskProjectService, got %s", body["service"])
	}
}

func TestCreateAndListProjects(t *testing.T) {
	setupTestDB(t)

	// Create
	body := `{"name":"Test Project","description":"desc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	if created["name"] != "Test Project" {
		t.Errorf("expected name=Test Project, got %v", created["name"])
	}

	// List
	req2 := httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1/", nil)
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	rec2 := httptest.NewRecorder()
	handleListProjects(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec2.Code)
	}
	var list []interface{}
	json.NewDecoder(rec2.Body).Decode(&list)
	if len(list) != 1 {
		t.Errorf("expected 1 project, got %d", len(list))
	}
}

func TestCreateAndGetProject(t *testing.T) {
	setupTestDB(t)

	// Create
	body := `{"name":"Get Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pid := created["id"].(string)

	// Get
	req2 := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	req2.Header.Set("X-Resource-Id", pid)
	rec2 := httptest.NewRecorder()
	handleGetProject(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec2.Code)
	}
	var got map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&got)
	if got["name"] != "Get Test" {
		t.Errorf("expected name=Get Test, got %v", got["name"])
	}
}

func TestUpdateProject(t *testing.T) {
	setupTestDB(t)

	// Create
	body := `{"name":"Original"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pid := created["id"].(string)

	// Update
	body2 := `{"name":"Updated"}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(body2))
	req2.Header.Set("X-Resource-Id", pid)
	rec2 := httptest.NewRecorder()
	handleUpdateProject(rec2, req2)

	var updated map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&updated)
	if updated["name"] != "Updated" {
		t.Errorf("expected name=Updated, got %v", updated["name"])
	}
}

func TestDeleteProject(t *testing.T) {
	setupTestDB(t)

	// Create
	body := `{"name":"To Delete"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pid := created["id"].(string)

	// Delete
	req2 := httptest.NewRequest(http.MethodDelete, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	req2.Header.Set("X-Resource-Id", pid)
	rec2 := httptest.NewRecorder()
	handleDeleteProject(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("delete: expected 200, got %d", rec2.Code)
	}

	// Verify deleted
	req3 := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	req3.Header.Set("X-Resource-Id", pid)
	rec3 := httptest.NewRecorder()
	handleGetProject(rec3, req3)
	if rec3.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rec3.Code)
	}
}

func TestDeleteProjectPublishesProjectDeleted(t *testing.T) {
	setupTestDB(t)
	var gotTenant, gotPID string
	old := publishProjectDeletedFn
	publishProjectDeletedFn = func(ctx context.Context, tenantID, projectID string) error {
		gotTenant, gotPID = tenantID, projectID
		return nil
	}
	t.Cleanup(func() { publishProjectDeletedFn = old })

	body := `{"name":"To Delete Event"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	pid := created["id"].(string)

	req2 := httptest.NewRequest(http.MethodDelete, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	req2.Header.Set("X-Resource-Id", pid)
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	rec2 := httptest.NewRecorder()
	handleDeleteProject(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	if gotPID != pid {
		t.Fatalf("published project_id=%q want %q", gotPID, pid)
	}
	if gotTenant != "t1" {
		t.Fatalf("published tenant_id=%q want t1", gotTenant)
	}
}

func TestCORSHeaders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://localhost:4000")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	acao := rec.Header().Get("Access-Control-Allow-Origin")
	if acao != "http://localhost:4000" {
		t.Errorf("expected CORS origin, got %s", acao)
	}
}

func TestTranslateBranchTitleNonChineseViaProjectsRoute(t *testing.T) {
	body := `{"title":"Feature-123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/translate-branch-title/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t1", []string{"translate-branch-title"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["translated_title"] != "feature-123" {
		t.Fatalf("translated_title=%v", payload["translated_title"])
	}
	if payload["used_ai"] != false {
		t.Fatalf("used_ai=%v", payload["used_ai"])
	}
}

func TestTranslateBranchTitleWrongMethodIs405(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1/translate-branch-title/", nil)
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t1", []string{"translate-branch-title"})
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMain(m *testing.M) {
	// Ensure data directory
	os.MkdirAll("../../data", 0755)
	initLogging("test-task-project-service")
	code := m.Run()
	os.Exit(code)
}
