package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Router-level regression for OPT-20260807-019:
// POST on the collection URL /api/projects/tenant_id/{tid} must dispatch to
// handleCreateProject (201 + project detail), not silently fall through to
// handleListProjects (200 []). The refactor 6edee88 (2026-08-04) broke this:
// ParseConventionPath consumes "tenant_id/{tid}", leaving an empty rest which
// routed every HTTP method to handleListProjects.
func TestProjectsTenantIDRoutePostCreates(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	body := `{"name":"RouterCreateProj","description":"router-level create","git_repos":["https://github.com/example/router-create"],"tags":["router"],"server_run_template":{},"workspaces_ids":["ws_test"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST via router: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created["name"] != "RouterCreateProj" {
		t.Errorf("expected name=RouterCreateProj, got %v", created["name"])
	}
	if id, _ := created["id"].(string); id == "" {
		t.Errorf("expected created project to carry a non-empty id, got %v", created["id"])
	}
}

// Trailing-slash variant: frontend calls without trailing slash, some callers
// use one; both must behave identically.
func TestProjectsTenantIDRoutePostCreatesTrailingSlash(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	body := `{"name":"RouterCreateProjSlash","description":"desc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST via router (slash): expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// GET on the same collection URL must keep returning the 200 project list.
func TestProjectsTenantIDRouteGetLists(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET via router: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var projects []map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&projects); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if projects == nil {
		t.Fatal("expected a JSON array, got null")
	}
}

// Regression for 保存项目运行模版失败（2026-08-07）:
// PATCH /api/projects/tenant_id/{tid}/{pid}/ 必须分发到 handleUpdateProject，
// 不能因为 ParseConventionPath 吞掉 kv 对后的位置段 pid 而落入 collection 405。
func TestProjectsTenantIDRoutePatchUpdates(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	createBody := `{"name":"RouterPatchProj","description":"router-level patch","git_repos":["https://github.com/example/router-patch"],"tags":["router"],"server_run_template":{},"workspaces_ids":["ws_test"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatal("expected created project id")
	}

	// 前端 ProjectRunTemplatePanel 保存路径：/api/projects/tenant_id/{tid}/{pid}/ PATCH
	patchBody := `{"server_run_template":{"platform_type":"aliyun","region":"cn-hangzhou","hardware_config":{"cpu_cores":"2","memory_gb":"4","storage_gb":"40"}}}`
	req = httptest.NewRequest(http.MethodPatch, "/api/projects/tenant_id/t1/"+pid+"/", strings.NewReader(patchBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH tenant_id-first: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var detail map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	tmpl, _ := detail["server_run_template"].(map[string]interface{})
	if tmpl == nil || tmpl["region"] != "cn-hangzhou" {
		t.Fatalf("expected server_run_template persisted, got %v", detail["server_run_template"])
	}
}

// OPT-20260807-021 防御：裸 /api/projects 路径 POST 必须 405，
// 不能静默落入 handleListProjects 返回 200 []。
func TestProjectsBarePathPostMethodNotAllowed(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(`{"name":"X"}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /api/projects: expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "method not allowed" {
		t.Errorf("expected error=method not allowed, got %v", body)
	}
}

// GET 同路径必须返回项目详情（而非项目列表）—— tenant_id-first 位置段回归。
func TestProjectsTenantIDRouteGetDetail(t *testing.T) {
	setupTestDB(t)
	mux := http.NewServeMux()
	mountRoutes(mux)

	createBody := `{"name":"RouterGetDetail","description":"desc","workspaces_ids":["ws_test"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	pid, _ := created["id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1/"+pid+"/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET tenant_id-first detail: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var detail map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&detail); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if id, _ := detail["id"].(string); id != pid {
		t.Fatalf("expected project detail for %s, got object id=%v (silently returned list?)", pid, detail["id"])
	}
}

// OPT-20260820-015: /api/projects/ 分发层强制租户成员校验。
// 非成员 403、成员 200、internal 放行（内部调用旁路）。
func TestProjectsTenantIDRouteRequiresMembership(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/tenant/members/resolve" {
			if r.URL.Query().Get("user_id") == "u1" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"user_id":"u1"}`))
				return
			}
			http.NotFound(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	old := cfg.TaskTenantURL
	cfg.TaskTenantURL = srv.URL
	defer func() { cfg.TaskTenantURL = old }()

	mux := http.NewServeMux()
	mountRoutes(mux)

	// 成员 u1 → 200
	req := httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("member GET expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// 非成员 u2 → 403
	req = httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u2")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member GET expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	// internal 旁路 → 200
	req = httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "internal")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("internal GET expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// clone-run APISIX regenerated from empty conf/ (no conf-local merge) injects
// X-TaskGateway-Internal-Secret: ” while the service loads conf-local. Identity
// is not promoted → 401 请先登录 → SPA「加载项目失败」.
func TestProjectsListGatewayEmptySecretUnauthorized(t *testing.T) {
	setupTestDB(t)
	prev := cfg.GatewayInternalSecret
	cfg.GatewayInternalSecret = "test-gw-secret"
	t.Cleanup(func() { cfg.GatewayInternalSecret = prev })

	mux := http.NewServeMux()
	mountRoutes(mux)
	h := gatewayUserMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-TaskGateway-Internal-Secret", "")
	req.Header.Set("X-User-Id", "u1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("empty APISIX secret: expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "请先登录") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestProjectsListGatewayMatchingSecretOK(t *testing.T) {
	setupTestDB(t)
	prev := cfg.GatewayInternalSecret
	cfg.GatewayInternalSecret = "test-gw-secret"
	t.Cleanup(func() { cfg.GatewayInternalSecret = prev })

	mux := http.NewServeMux()
	mountRoutes(mux)
	h := gatewayUserMiddleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1", nil)
	req.Header.Set("X-Gateway-Auth-Verified", "1")
	req.Header.Set("X-TaskGateway-Internal-Secret", "test-gw-secret")
	req.Header.Set("X-User-Id", "u1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("matching gateway secret: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
