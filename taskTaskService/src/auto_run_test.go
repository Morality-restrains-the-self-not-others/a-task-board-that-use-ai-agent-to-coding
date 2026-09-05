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

func configuredRunTemplate() map[string]interface{} {
	return map[string]interface{}{
		"region":            "cn-hangzhou",
		"cloud_platform_id": "plat-1",
		"vpc_id":            "vpc-1",
		"vswitch_id":        "vsw-1",
		"security_group_id": "sg-1",
		"platform":          "aliyun",
		"label":             "hangzhou · aliyun",
		"default_auto_run":  true,
		"hardware_config": map[string]interface{}{
			"cpu_cores":  "2",
			"memory_gb":  "4",
			"storage_gb": "40",
		},
	}
}

func startAutoRunMockServices(t *testing.T, withTemplate bool) (cloudCalls *[]map[string]interface{}, mu *sync.Mutex) {
	t.Helper()
	return startAutoRunMockServicesWithNested(t, withTemplate, "")
}

// nestedError: empty → nested-git-repos success with empty list; non-empty → error field.
func startAutoRunMockServicesWithNested(t *testing.T, withTemplate bool, nestedError string) (cloudCalls *[]map[string]interface{}, mu *sync.Mutex) {
	t.Helper()
	mu = &sync.Mutex{}
	calls := make([]map[string]interface{}, 0)
	cloudCalls = &calls

	projSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pathHasWorkspace(r, "ws1") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			_, _ = w.Write([]byte("[]"))
			return
		}
		if strings.Contains(r.URL.Path, "/api/internal/nested-git-repos") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"parent_repo_url": r.URL.Query().Get("repo_url"),
				"nested_repos":    []interface{}{},
				"error":           nestedError,
			})
			return
		}
		if pathHasProject(r, "p1") {
			tpl := map[string]interface{}{}
			if withTemplate {
				tpl = configuredRunTemplate()
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                  "p1",
				"git_repos":           []string{"https://git.example/repo.git"},
				"server_run_template": tpl,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(projSrv.Close)

	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		*cloudCalls = append(*cloudCalls, map[string]interface{}{
			"method":   r.Method,
			"path":     r.URL.Path,
			"trace_id": r.Header.Get("X-Trace-Id"),
			"body":     body,
		})
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{"status": "success"}
		if strings.Contains(r.URL.Path, "tenant-installed-images/lookup") {
			resp["id"] = r.URL.Query().Get("id")
			resp["name"] = "test-image"
			resp["external_image_id"] = "ext-img-12345"
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(cloudSrv.Close)

	cfg.ProjectServiceURL = projSrv.URL
	cfg.CloudServiceURL = cloudSrv.URL
	cfg.TaskBillURL = ""
	cfg.InternalSecret = "test-secret"

	prevSchedule := scheduleTaskAutoRunFn
	scheduleTaskAutoRunFn = func(p autoRunTriggerParams) {
		if err := triggerTaskAutoRunFn(p); err != nil {
			t.Logf("sync auto_run trigger err: %v", err)
		}
	}
	t.Cleanup(func() { scheduleTaskAutoRunFn = prevSchedule })

	prevEnsureAt := ensureAutoRunAtCommentFn
	ensureAutoRunAtCommentFn = func(p autoRunTriggerParams) (string, error) {
		return "cmt-mock-auto-run", nil
	}
	t.Cleanup(func() { ensureAutoRunAtCommentFn = prevEnsureAt })

	// Mock lookupInstalledImageFn to return a valid image with external_image_id.
	prevLookup := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return &InstalledImageLookup{ID: imageID, Name: "test-image", ExternalImageID: "ext-img-12345"}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prevLookup })

	return cloudCalls, mu
}

func TestTriggerTaskAutoRunPostsCloudWithTrace(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	err := triggerTaskAutoRun(autoRunTriggerParams{
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task_abc12345",
		UserID:      "u1",
		ImageID:     "img1",
		RunTemplate: configuredRunTemplate(),
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 1 {
		t.Fatalf("cloud calls=%d", len(*cloudCalls))
	}
	call := (*cloudCalls)[0]
	if !strings.Contains(call["path"].(string), "/cloud/compute/start-vm/") {
		t.Fatalf("path=%v", call["path"])
	}
	if call["trace_id"] != "task_abc12345" {
		t.Fatalf("trace_id=%v want task_abc12345", call["trace_id"])
	}
}

func TestTriggerTaskAutoRunInjectsClientPublicIP(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	tpl := configuredRunTemplate()
	tpl["security_group_id"] = "" // force start-vm-auto + auto_create_security_group
	err := triggerTaskAutoRun(autoRunTriggerParams{
		TenantID:       "t1",
		WorkspaceID:    "ws1",
		TaskID:         "task_sg_ip_01",
		UserID:         "u1",
		ImageID:        "img1",
		RunTemplate:    tpl,
		ClientPublicIP: "203.0.113.88",
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 1 {
		t.Fatalf("cloud calls=%d", len(*cloudCalls))
	}
	call := (*cloudCalls)[0]
	if !strings.Contains(call["path"].(string), "/cloud/compute/start-vm-auto/") {
		t.Fatalf("path=%v", call["path"])
	}
	body, _ := call["body"].(map[string]interface{})
	if body["client_public_ip"] != "203.0.113.88" {
		t.Fatalf("client_public_ip=%v body=%v", body["client_public_ip"], body)
	}
	if body["auto_create_security_group"] != true {
		t.Fatalf("auto_create_security_group=%v", body["auto_create_security_group"])
	}
}

func TestCreateTaskAutoRunTriggersStart(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	body := `{
		"title":"Auto run task",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatal("missing task id")
	}
	if created["auto_run"] != true {
		t.Fatalf("auto_run=%v", created["auto_run"])
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(*cloudCalls)
		mu.Unlock()
		if n >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for start-vm")
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	call := (*cloudCalls)[0]
	if call["trace_id"] != taskID {
		t.Fatalf("trace_id=%v want %s", call["trace_id"], taskID)
	}
}

func TestCreateTaskAutoRunQueuedDefersStart(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)
	setupWorkspaceRhythmAllDay(t, "t1", "ws1", true, 1)

	body := `{
		"title":"Auto run queued task",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"queued_auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	if created["auto_run"] != true {
		t.Fatalf("auto_run=%v", created["auto_run"])
	}
	if created["queued_auto_run"] != true {
		t.Fatalf("queued_auto_run=%v created=%v", created["queued_auto_run"], created)
	}
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	n := len(*cloudCalls)
	mu.Unlock()
	if n != 0 {
		t.Fatalf("queued create must not start-vm immediately, cloudCalls=%d", n)
	}
}

func TestCreateTaskAutoRunRejectsMissingGitIdentity(t *testing.T) {
	setupTestDB(t)
	startAutoRunMockServices(t, true)

	body := `{
		"title":"Auto run missing identity",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "repo_identities_required") && !strings.Contains(rec.Body.String(), errMsgRepoIdentitiesRequired) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestCreateTaskAutoRunForwardsClientPublicIPFromXFF(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	body := `{
		"title":"Auto run with xff",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Forwarded-For", "198.51.100.44, 10.0.0.1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(*cloudCalls)
		mu.Unlock()
		if n >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for start-vm")
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	call := (*cloudCalls)[0]
	cloudBody, _ := call["body"].(map[string]interface{})
	if cloudBody["client_public_ip"] != "198.51.100.44" {
		t.Fatalf("client_public_ip=%v want 198.51.100.44 body=%v", cloudBody["client_public_ip"], cloudBody)
	}
}

func TestCreateTaskAutoRunBodyClientPublicIPOverridesXFF(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	body := `{
		"title":"Auto run body ip",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"client_public_ip":"203.0.113.99",
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(*cloudCalls)
		mu.Unlock()
		if n >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for start-vm")
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	call := (*cloudCalls)[0]
	cloudBody, _ := call["body"].(map[string]interface{})
	if cloudBody["client_public_ip"] != "203.0.113.99" {
		t.Fatalf("client_public_ip=%v want 203.0.113.99", cloudBody["client_public_ip"])
	}
}

func TestCreateTaskAutoRunCloudUnavailable(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	cfg.CloudServiceURL = ""

	body := `{
		"title":"No cloud",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&out)
	if out["code"] != "AUTO_RUN_CLOUD_UNAVAILABLE" {
		t.Fatalf("code=%v body=%s", out["code"], rec.Body.String())
	}
}

func TestCreateTaskAutoRunTemplateRequired(t *testing.T) {
	setupTestDB(t)
	_, _ = startAutoRunMockServices(t, false)

	body := `{
		"title":"No template",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&out)
	if out["code"] != "AUTO_RUN_RUN_TEMPLATE_REQUIRED" {
		t.Fatalf("code=%v", out["code"])
	}
}

func TestCreateTaskAutoRunProjectNotAllowed(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)
	// Override project mock: template configured but allow flag off.
	projSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pathHasWorkspace(r, "ws1") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			_, _ = w.Write([]byte("[]"))
			return
		}
		if strings.Contains(r.URL.Path, "/api/internal/nested-git-repos") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"parent_repo_url": r.URL.Query().Get("repo_url"),
				"nested_repos":    []interface{}{},
			})
			return
		}
		if pathHasProject(r, "p1") {
			tpl := configuredRunTemplate()
			tpl["default_auto_run"] = false
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                  "p1",
				"git_repos":           []string{"https://git.example/repo.git"},
				"server_run_template": tpl,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(projSrv.Close)
	cfg.ProjectServiceURL = projSrv.URL

	body := `{
		"title":"Project disallows auto run",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&out)
	if out["code"] != "AUTO_RUN_PROJECT_NOT_ALLOWED" {
		t.Fatalf("code=%v body=%s", out["code"], rec.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 0 {
		t.Fatalf("start-vm must not be called when project disallows auto_run, got %d calls", len(*cloudCalls))
	}
}

func TestUpdateTaskAutoRunFalseToTrueTriggersStart(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	createBody := `{
		"title":"Later auto",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":false,
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	taskID := created["id"].(string)

	mu.Lock()
	*cloudCalls = nil
	mu.Unlock()

	upd := `{"auto_run":true,"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}]}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(upd))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec2.Code, rec2.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 1 {
		t.Fatalf("expected 1 start, got %d", len(*cloudCalls))
	}
	if (*cloudCalls)[0]["trace_id"] != taskID {
		t.Fatalf("trace=%v", (*cloudCalls)[0]["trace_id"])
	}
}

func TestUpdateTaskAutoRunAlreadyTrueNoForceSkipsStart(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	createBody := `{
		"title":"Already auto",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	taskID := created["id"].(string)

	mu.Lock()
	*cloudCalls = nil
	mu.Unlock()

	upd := `{"title":"Still auto","auto_run":true,"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}]}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(upd))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec2.Code, rec2.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 0 {
		t.Fatalf("expected no start, got %d", len(*cloudCalls))
	}
}

func TestUpdateTaskForceAutoRunTriggersStart(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	createBody := `{
		"title":"Force auto",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	taskID := created["id"].(string)

	mu.Lock()
	*cloudCalls = nil
	mu.Unlock()

	upd := `{"auto_run":true,"force_auto_run":true,"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}]}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(upd))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec2.Code, rec2.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 1 {
		t.Fatalf("expected 1 force start, got %d", len(*cloudCalls))
	}
}

func TestCreateTaskAutoRunSkipsStartWhenNestedGitNeedsAuth(t *testing.T) {
	setupTestDB(t)
	authMsg := "无法获取子 Git 仓库列表：未检测到可用授权。请先在个人资料完成 Git 网站绑定，或先登录本地 GitLab 后重试。"
	cloudCalls, mu := startAutoRunMockServicesWithNested(t, true, authMsg)

	var ensureMu sync.Mutex
	var ensureCalls []autoRunTriggerParams
	prevEnsureAt := ensureAutoRunAtCommentFn
	ensureAutoRunAtCommentFn = func(p autoRunTriggerParams) (string, error) {
		ensureMu.Lock()
		ensureCalls = append(ensureCalls, p)
		ensureMu.Unlock()
		return "cmt-mock-auto-run", nil
	}
	t.Cleanup(func() { ensureAutoRunAtCommentFn = prevEnsureAt })

	body := `{
		"title":"Auto run without git auth",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	if created["auto_run"] != true {
		t.Fatalf("auto_run=%v want true", created["auto_run"])
	}
	if created["auto_run_start_skipped"] != true {
		t.Fatalf("auto_run_start_skipped=%v", created["auto_run_start_skipped"])
	}
	if created["code"] != autoRunGitAccessSkippedCode {
		t.Fatalf("code=%v", created["code"])
	}
	reason, _ := created["auto_run_start_skip_reason"].(string)
	if !strings.Contains(reason, "未检测到可用授权") {
		t.Fatalf("skip_reason=%q", reason)
	}

	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	cloudN := len(*cloudCalls)
	mu.Unlock()
	if cloudN != 0 {
		t.Fatalf("expected 0 start-vm calls when git auth missing, got %d", cloudN)
	}
	ensureMu.Lock()
	nEnsure := len(ensureCalls)
	var gotSkip string
	if nEnsure > 0 {
		gotSkip = ensureCalls[0].StartSkipReason
	}
	ensureMu.Unlock()
	if nEnsure != 1 {
		t.Fatalf("expected 1 auto-run comment ensure on soft-skip, got %d", nEnsure)
	}
	if !strings.Contains(gotSkip, "未检测到可用授权") {
		t.Fatalf("StartSkipReason=%q want auth skip reason", gotSkip)
	}

	// Cold-open GET must still expose the persisted skip reason.
	taskID, _ := created["id"].(string)
	reqGet := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", nil)
	reqGet.Header.Set("X-Auth-Tenant-Id", "t1")
	reqGet.Header.Set("X-Auth-User-Id", "u1")
	recGet := httptest.NewRecorder()
	handleTaskRoutes(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get: %d %s", recGet.Code, recGet.Body.String())
	}
	var got map[string]interface{}
	_ = json.NewDecoder(recGet.Body).Decode(&got)
	if got["auto_run_start_skipped"] != true {
		t.Fatalf("GET auto_run_start_skipped=%v", got["auto_run_start_skipped"])
	}
	gotReason, _ := got["auto_run_start_skip_reason"].(string)
	if !strings.Contains(gotReason, "未检测到可用授权") {
		t.Fatalf("GET skip_reason=%q", gotReason)
	}
}

func TestCreateTaskAutoRunEmitsCommentWhenNestedGitProbeFails(t *testing.T) {
	setupTestDB(t)
	// Empty nestedError alone yields success; transport failure is simulated by shutting
	// project nested endpoint — use a dedicated project server that 500s nested probe.
	mu := &sync.Mutex{}
	calls := make([]map[string]interface{}, 0)
	cloudCalls := &calls

	projSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pathHasWorkspace(r, "ws1") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			_, _ = w.Write([]byte("[]"))
			return
		}
		if strings.Contains(r.URL.Path, "/api/internal/nested-git-repos") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"upstream timeout"}`))
			return
		}
		if pathHasProject(r, "p1") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                  "p1",
				"git_repos":           []string{"https://git.example/repo.git"},
				"server_run_template": configuredRunTemplate(),
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(projSrv.Close)

	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		*cloudCalls = append(*cloudCalls, map[string]interface{}{"path": r.URL.Path})
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{"status": "success"}
		if strings.Contains(r.URL.Path, "tenant-installed-images/lookup") {
			resp["id"] = r.URL.Query().Get("id")
			resp["name"] = "test-image"
			resp["external_image_id"] = "ext-img-12345"
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(cloudSrv.Close)

	cfg.ProjectServiceURL = projSrv.URL
	cfg.CloudServiceURL = cloudSrv.URL
	cfg.InternalSecret = "test-secret"

	prevSchedule := scheduleTaskAutoRunFn
	scheduleTaskAutoRunFn = func(p autoRunTriggerParams) {
		if err := triggerTaskAutoRunFn(p); err != nil {
			t.Logf("sync auto_run trigger err: %v", err)
		}
	}
	t.Cleanup(func() { scheduleTaskAutoRunFn = prevSchedule })

	var ensureMu sync.Mutex
	var ensureCalls []autoRunTriggerParams
	prevEnsureAt := ensureAutoRunAtCommentFn
	ensureAutoRunAtCommentFn = func(p autoRunTriggerParams) (string, error) {
		ensureMu.Lock()
		ensureCalls = append(ensureCalls, p)
		ensureMu.Unlock()
		return "cmt-probe-fail", nil
	}
	t.Cleanup(func() { ensureAutoRunAtCommentFn = prevEnsureAt })

	prevLookup := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return &InstalledImageLookup{ID: imageID, Name: "test-image", ExternalImageID: "ext-img-12345"}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prevLookup })

	body := `{
		"title":"写一个 hello world程序",
		"description":"用 js",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	if created["auto_run_start_skipped"] != true {
		t.Fatalf("auto_run_start_skipped=%v", created["auto_run_start_skipped"])
	}
	reason, _ := created["auto_run_start_skip_reason"].(string)
	if !strings.Contains(reason, "探测失败") {
		t.Fatalf("skip_reason=%q want 探测失败", reason)
	}

	ensureMu.Lock()
	nEnsure := len(ensureCalls)
	gotSkip := ""
	if nEnsure > 0 {
		gotSkip = ensureCalls[0].StartSkipReason
	}
	ensureMu.Unlock()
	if nEnsure != 1 {
		t.Fatalf("expected auto-run comment even on probe failure, ensureCalls=%d", nEnsure)
	}
	if !strings.Contains(gotSkip, "探测失败") {
		t.Fatalf("StartSkipReason=%q", gotSkip)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 0 {
		t.Fatalf("expected 0 start-vm on probe failure, got %d", len(*cloudCalls))
	}
}

func TestTriggerTaskAutoRunCommentOnlyWhenStartSkipReason(t *testing.T) {
	setupTestDB(t)
	var cloudHits int
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cloudHits++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(cloudSrv.Close)
	cfg.CloudServiceURL = cloudSrv.URL
	cfg.InternalSecret = "s"

	var ensureHits int
	var sawSkip string
	prevEnsureAt := ensureAutoRunAtCommentFn
	ensureAutoRunAtCommentFn = func(p autoRunTriggerParams) (string, error) {
		ensureHits++
		sawSkip = p.StartSkipReason
		return "cmt-skip-only", nil
	}
	t.Cleanup(func() { ensureAutoRunAtCommentFn = prevEnsureAt })

	err := triggerTaskAutoRun(autoRunTriggerParams{
		TenantID:        "t1",
		WorkspaceID:     "ws1",
		TaskID:          "task_skip_cmt",
		UserID:          "u1",
		ImageID:         "img1",
		RunTemplate:     configuredRunTemplate(),
		StartSkipReason: "无法获取子 Git 仓库列表：探测失败，已跳过自动启动服务器",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ensureHits != 1 {
		t.Fatalf("ensureHits=%d want 1", ensureHits)
	}
	if !strings.Contains(sawSkip, "探测失败") {
		t.Fatalf("sawSkip=%q", sawSkip)
	}
	if cloudHits != 0 {
		t.Fatalf("cloudHits=%d want 0 (comment-only path)", cloudHits)
	}
}

func TestUpdateTaskForceAutoRunSkipsWhenGitInaccessible(t *testing.T) {
	setupTestDB(t)
	cloudCalls, mu := startAutoRunMockServices(t, true)

	body := `{
		"title":"Force skip git",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	taskID := created["id"].(string)

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(*cloudCalls)
		mu.Unlock()
		if n >= 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for initial start-vm")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cloudCalls2, mu2 := startAutoRunMockServicesWithNested(t, true, "无法访问父仓库（远端返回不可见/无权限）")
	mu.Lock()
	*cloudCalls = nil
	mu.Unlock()

	var ensureMu sync.Mutex
	var ensureCalls []autoRunTriggerParams
	prevEnsureAt := ensureAutoRunAtCommentFn
	ensureAutoRunAtCommentFn = func(p autoRunTriggerParams) (string, error) {
		ensureMu.Lock()
		ensureCalls = append(ensureCalls, p)
		ensureMu.Unlock()
		return "cmt-force-skip", nil
	}
	t.Cleanup(func() { ensureAutoRunAtCommentFn = prevEnsureAt })

	upd := `{"auto_run":true,"force_auto_run":true,"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}]}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(upd))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec2.Code, rec2.Body.String())
	}
	var updated map[string]interface{}
	_ = json.NewDecoder(rec2.Body).Decode(&updated)
	if updated["auto_run_start_skipped"] != true {
		t.Fatalf("expected skip on force, body=%v", updated)
	}
	time.Sleep(50 * time.Millisecond)
	mu2.Lock()
	cloudN := len(*cloudCalls2)
	mu2.Unlock()
	if cloudN != 0 {
		t.Fatalf("expected 0 force start when git inaccessible, got %d", cloudN)
	}
	ensureMu.Lock()
	nEnsure := len(ensureCalls)
	gotSkip := ""
	if nEnsure > 0 {
		gotSkip = ensureCalls[0].StartSkipReason
	}
	ensureMu.Unlock()
	if nEnsure != 1 {
		t.Fatalf("expected comment ensure on force soft-skip, got %d", nEnsure)
	}
	if !strings.Contains(gotSkip, "无法访问父仓库") {
		t.Fatalf("StartSkipReason=%q", gotSkip)
	}
}

func TestProbeGitAccessForAutoRunEmptyProjects(t *testing.T) {
	if got := probeGitAccessForAutoRun(context.Background(), "u1", "t1", nil); got != "" {
		t.Fatalf("empty projects should not skip, got %q", got)
	}
}

func TestStartVMSetsTraceHeaders(t *testing.T) {
	var gotTrace string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTrace = r.Header.Get("X-Trace-Id")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	cfg.CloudServiceURL = srv.URL
	cfg.InternalSecret = "s"

	prevEnsureAt := ensureAutoRunAtCommentFn
	ensureAutoRunAtCommentFn = func(p autoRunTriggerParams) (string, error) {
		return "cmt-mock-auto-run", nil
	}
	t.Cleanup(func() { ensureAutoRunAtCommentFn = prevEnsureAt })

	ctx := context.Background()
	// Use Correlation so ApplyOutboundHeaders emits X-Trace-Id.
	// (startVM itself relies on caller-provided ctx; triggerTaskAutoRun sets this.)
	_ = ctx
	err := triggerTaskAutoRun(autoRunTriggerParams{
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task_trace01",
		UserID:      "u1",
		ImageID:     "img1",
		RunTemplate: configuredRunTemplate(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotTrace != "task_trace01" {
		t.Fatalf("X-Trace-Id=%q", gotTrace)
	}
}

// startAutoRunProbeMockServer serves project + validate-git-repos + nested-git-repos
// endpoints for OPT-20260818-047 probe tests.
// validateResults: nil → validate-git-repos returns 500 (transport-fail path).
func startAutoRunProbeMockServer(t *testing.T, autoClone bool, validateResults []map[string]interface{}) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/api/internal/nested-git-repos") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"parent_repo_url": r.URL.Query().Get("repo_url"),
				"nested_repos":    []interface{}{},
				"error":           "无法获取子 Git 仓库列表：探测失败",
			})
			return
		}
		if strings.Contains(r.URL.Path, "/validate-git-repos") {
			if validateResults == nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"boom"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": validateResults})
			return
		}
		if pathHasProject(r, "p1") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                      "p1",
				"auto_clone_nested_repos": autoClone,
				"git_repos":               []string{"https://git.example/repo.git"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
}

func TestProbeGitAccessForAutoRunAutoCloneOffNestedErrorStillProceeds(t *testing.T) {
	// auto_clone_nested_repos=false → nested-git error ignored; parent accessible → proceed.
	startAutoRunProbeMockServer(t, false, []map[string]interface{}{
		{"url": "https://git.example/repo.git", "is_accessible": true, "message": ""},
	})
	if got := probeGitAccessForAutoRun(context.Background(), "u1", "t1", []string{"p1"}); got != "" {
		t.Fatalf("auto_clone=false + nested error + parent ok should proceed, got %q", got)
	}
}

func TestProbeGitAccessForAutoRunAutoCloneOffParentInaccessibleSkips(t *testing.T) {
	startAutoRunProbeMockServer(t, false, []map[string]interface{}{
		{"url": "https://git.example/repo.git", "is_accessible": false, "message": "无法访问父仓库：无有效授权"},
	})
	got := probeGitAccessForAutoRun(context.Background(), "u1", "t1", []string{"p1"})
	if got == "" {
		t.Fatal("auto_clone=false + parent inaccessible should skip")
	}
	if !strings.Contains(got, "无有效授权") {
		t.Fatalf("reason=%q", got)
	}
}

func TestProbeGitAccessForAutoRunAutoCloneOffParentProbeUnavailableSkips(t *testing.T) {
	// validate-git-repos returns 500 → fail-closed skip.
	startAutoRunProbeMockServer(t, false, nil)
	got := probeGitAccessForAutoRun(context.Background(), "u1", "t1", []string{"p1"})
	if got == "" {
		t.Fatal("auto_clone=false + parent probe transport failure should skip")
	}
	if !strings.Contains(got, "父 Git 仓库") {
		t.Fatalf("reason=%q", got)
	}
}

func TestProbeGitAccessForAutoRunAutoCloneOnNestedErrorSkips(t *testing.T) {
	// auto_clone_nested_repos=true → keep legacy nested-git-repos probe; error skips.
	startAutoRunProbeMockServer(t, true, nil)
	got := probeGitAccessForAutoRun(context.Background(), "u1", "t1", []string{"p1"})
	if got == "" {
		t.Fatal("auto_clone=true + nested-git error should skip")
	}
	if !strings.Contains(got, "子 Git 仓库") {
		t.Fatalf("reason=%q", got)
	}
}

func createAutoRunTaskThenBreakImageLookup(t *testing.T) (taskID string, cloudCalls *[]map[string]interface{}, mu *sync.Mutex) {
	t.Helper()
	cloudCalls, mu = startAutoRunMockServices(t, true)
	createBody := `{
		"title":"Terminal skip auto_run",
		"workspace_id":"ws1",
		"container_image_id":"img1",
		"auto_run":true,
		"repo_identities":[{"repo_url":"https://git.example/repo.git","git_identity_id":"gid-auto"}],
		"projects":[{"project_id":"p1","repo_index":0,"base_branch":"main","target_branch":"feat/x"}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("missing task id")
	}
	mu.Lock()
	*cloudCalls = nil
	mu.Unlock()
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return nil, nil
	}
	return id, cloudCalls, mu
}

func patchTaskProgress(t *testing.T, taskID, columnID, columnName string) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"progress_column_id":"` + columnID + `"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	req.Header.Set("X-Task-Test-Progress-Column-Name", columnName)
	req.Header.Set("X-Task-Test-Allowed-Progress-Column-Ids", columnID)
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	return rec
}

func TestShouldValidateAutoRunOnTaskUpdate(t *testing.T) {
	if !shouldValidateAutoRunOnTaskUpdate(true, false) {
		t.Fatal("open progress with auto_run must validate")
	}
	if shouldValidateAutoRunOnTaskUpdate(true, true) {
		t.Fatal("terminal progress must skip auto_run prereq")
	}
	if shouldValidateAutoRunOnTaskUpdate(false, false) {
		t.Fatal("auto_run=false must not validate")
	}
}

func TestUpdateTaskProgressToCompletedSkipsAutoRunPrereqWhenImageGone(t *testing.T) {
	setupTestDB(t)
	taskID, cloudCalls, mu := createAutoRunTaskThenBreakImageLookup(t)
	rec := patchTaskProgress(t, taskID, "col-done", "已完成")
	if rec.Code != http.StatusOK {
		t.Fatalf("completed progress must ignore auto_run prereq, got %d %s", rec.Code, rec.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 0 {
		t.Fatalf("terminal progress must not start-vm, got %d calls", len(*cloudCalls))
	}
}

func TestUpdateTaskProgressToCancelledSkipsAutoRunPrereqWhenImageGone(t *testing.T) {
	setupTestDB(t)
	taskID, cloudCalls, mu := createAutoRunTaskThenBreakImageLookup(t)
	rec := patchTaskProgress(t, taskID, "col-cancel", "已取消")
	if rec.Code != http.StatusOK {
		t.Fatalf("cancelled progress must ignore auto_run prereq, got %d %s", rec.Code, rec.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*cloudCalls) != 0 {
		t.Fatalf("terminal progress must not start-vm, got %d calls", len(*cloudCalls))
	}
}

func TestUpdateTaskProgressToInProgressStillRequiresAutoRunPrereqWhenImageGone(t *testing.T) {
	setupTestDB(t)
	taskID, _, _ := createAutoRunTaskThenBreakImageLookup(t)
	rec := patchTaskProgress(t, taskID, "col-wip", "进行中")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("in-progress still requires auto_run prereq, got %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&out)
	if out["code"] != autoRunRuntimeEnvRequiredCode {
		t.Fatalf("code=%v body=%s", out["code"], rec.Body.String())
	}
}

func TestUpdateTaskProgressToCompletedPublishesStatusChangedWhenImageGone(t *testing.T) {
	setupTestDB(t)
	taskID, _, _ := createAutoRunTaskThenBreakImageLookup(t)
	var mu sync.Mutex
	var calls []map[string]interface{}
	prev := publishTaskStatusChangedFn
	publishTaskStatusChangedFn = func(ctx context.Context, tenantID, workspaceID, id string, prevColumnID, columnID string, prevCompleted, completed bool, progressColumnName string) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, map[string]interface{}{
			"task_id":              id,
			"column":               columnID,
			"progress_column_name": progressColumnName,
		})
		return nil
	}
	t.Cleanup(func() { publishTaskStatusChangedFn = prev })

	rec := patchTaskProgress(t, taskID, "col-done", "已完成")
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 1 {
		t.Fatalf("expected 1 TASK_STATUS_CHANGED, got %d", len(calls))
	}
	if calls[0]["column"] != "col-done" {
		t.Fatalf("column=%v", calls[0]["column"])
	}
	if calls[0]["progress_column_name"] != "已完成" {
		t.Fatalf("progress_column_name=%v", calls[0]["progress_column_name"])
	}
}
