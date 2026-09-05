package saas_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"taskEvents/internal/repository/saas"
)

// TestCreateWorkspaceAccessUsesSetPermission pins the workspace-access create
// endpoint. taskProjectService accepts creation only via POST
// /api/projects/workspace-access/set-permission/... (workspace-permissions is
// GET-only list; POST there returns 405). Regression for the 0110e94 URL
// migration, which pointed the create call at the GET-only list endpoint —
// with a faithful mock the admin access row was never created.
func TestCreateWorkspaceAccessUsesSetPermission(t *testing.T) {
	var createPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/internal/workspaces/"):
			// workspace exists
			writeJSONTest(w, 200, map[string]interface{}{"id": strings.TrimPrefix(r.URL.Path, "/api/internal/workspaces/"), "name": "ws"})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/projects/workspace-access/workspace-permissions/"):
			// no existing access rows
			writeJSONTest(w, 200, []map[string]interface{}{})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/projects/workspace-access/set-permission/"):
			createPaths = append(createPaths, r.URL.Path)
			writeJSONTest(w, 200, map[string]interface{}{"id": "wa_mock_1"})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/projects/workspaces/"):
			// progress system not set yet
			writeJSONTest(w, 404, map[string]string{"error": "progress system not set"})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/projects/settings/"):
			// no tenant default progress system
			writeJSONTest(w, 200, map[string]interface{}{"progress_system": nil})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("TASK_PROJECT_SERVICE_URL", srv.URL)
	defer os.Unsetenv("TASK_PROJECT_SERVICE_URL")

	repo := saas.New("")
	if err := repo.HandleWorkspaceCreated("ws_1", "用户的工作空间", "200", "42"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(createPaths) != 1 {
		t.Fatalf("expected exactly 1 workspace-access create call, got %d: %v", len(createPaths), createPaths)
	}
	if !strings.HasPrefix(createPaths[0], "/api/projects/workspace-access/set-permission/tenant_id/200/") {
		t.Fatalf("create call must target set-permission endpoint, got %s", createPaths[0])
	}
}

func writeJSONTest(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
