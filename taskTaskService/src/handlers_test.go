package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	dbload "dbload"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	testDSN, cleanup, err := dbload.OpenTestMySQLClonedFromDir(
		"task-task", repoRoot(), "dataMigrate/taskTaskService",
		func(dsn string) error {
			if err := openDB(dsn); err != nil {
				return err
			}
			defer db.Close()
			return runDataMigrate(repoRoot())
		},
	)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	// 与 taskCloudService setupCloudTestDB 一致：测试库需先应用 schema，
	// 否则 handler 测试报 task_tasks 表不存在（随机单测门禁 2026-08-05 暴露）。
	// OPT-20260821-022: auto-run 校验 git_identity_id 归属，seed 常用测试身份（u1/t1）。
	_, _ = db.Exec(`
		INSERT INTO task_git_identities (id, user_id, company_id, label, git_user_name, git_user_email, is_default, created_at, updated_at)
		VALUES
		('gid-auto', 'u1', 't1', '', 'Auto', 'auto@example.com', 0, NOW(), NOW()),
		('gid-1', 'u1', 't1', '', 'One', 'one@example.com', 0, NOW(), NOW())
		ON DUPLICATE KEY UPDATE label = VALUES(label)`)
}

func startMockProjectService(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// GET /api/tenant/{tid}/workspaces[/?mine=1]
		if pathIsWorkspaceList(r) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "ws1", "name": "WS1", "company_id": "t1"},
				{"id": "ws2", "name": "WS2", "company_id": "t1"},
			})
			return
		}
		if pathHasWorkspace(r, "ws1") {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if pathHasWorkspace(r, "ws2") {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws2", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			w.Write([]byte("[]"))
			return
		}
		if strings.Contains(r.URL.Path, "/projects/tenant_id/") {
			json.NewEncoder(w).Encode(map[string]interface{}{"git_repos": []string{"https://git.example/repo.git"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
	cfg.TaskBillURL = ""
}

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handleHealth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCreateAndListTasks(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	body := `{"title":"Hello","workspace_id":"ws1","task_kind":"bug-fix","code_lang":"go"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	if created["task_kind"] != "bug-fix" {
		t.Fatalf("create task_kind=%v", created["task_kind"])
	}
	if created["code_lang"] != "go" {
		t.Fatalf("create code_lang=%v", created["code_lang"])
	}
	if jsonNumber(created["workspace_seq"]) != 1 {
		t.Fatalf("create workspace_seq=%v want 1", created["workspace_seq"])
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/", nil)
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var list []map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list))
	}
	if list[0]["task_kind"] != "bug-fix" {
		t.Fatalf("list task_kind=%v", list[0]["task_kind"])
	}
	if list[0]["code_lang"] != "go" {
		t.Fatalf("list code_lang=%v", list[0]["code_lang"])
	}
	if jsonNumber(list[0]["workspace_seq"]) != 1 {
		t.Fatalf("list workspace_seq=%v want 1", list[0]["workspace_seq"])
	}
}

// TestFeatureParamsCreateUpdateGate — 恢复的 HTTP 门禁用例（UNIT_TEST_DEBT 2026-07-22 接线）。
// 仅当请求体显式携带 feature_params 相关键时执行 source 必填校验；
// 普通创建/更新（web UI、插件 "none" 任务省略字段）保持默认 none，不受影响。
func TestFeatureParamsCreateUpdateGate(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	create := func(body string) (*httptest.ResponseRecorder, map[string]interface{}) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		req.Header.Set("X-Auth-User-Id", "u1")
		rec := httptest.NewRecorder()
		handleTaskRoutes(rec, req)
		var payload map[string]interface{}
		bodyBytes := rec.Body.Bytes()
		json.Unmarshal(bodyBytes, &payload)
		rec.Body.Reset()
		rec.Body.Write(bodyBytes)
		return rec, payload
	}

	// 未携带 feature_params 键：默认 none，201（现状不回归）
	rec, payload := create(`{"title":"Plain","workspace_id":"ws1"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("plain create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if payload["feature_params_source"] != "none" {
		t.Fatalf("plain create source=%v, want none", payload["feature_params_source"])
	}

	// 显式 none → 400
	rec, _ = create(`{"title":"NoneExplicit","workspace_id":"ws1","feature_params_source":"none"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("explicit none create: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "FEATURE_PARAMS_SOURCE_REQUIRED") {
		t.Fatalf("explicit none create: missing gate code, got: %s", rec.Body.String())
	}

	// 显式空字符串 source → 400
	rec, _ = create(`{"title":"EmptySource","workspace_id":"ws1","feature_params_source":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty source create: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// personal 未带 config → 400
	rec, _ = create(`{"title":"PersonalNoCfg","workspace_id":"ws1","feature_params_source":"personal"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("personal without config create: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// 仅带 personal_feature_params_config_id（source 缺失）→ 400
	rec, _ = create(`{"title":"CfgOnly","workspace_id":"ws1","personal_feature_params_config_id":"cfg-9"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("config-only create: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// 合法 source 组合 → 201
	for _, body := range []string{
		`{"title":"Ws","workspace_id":"ws1","feature_params_source":"workspace"}`,
		`{"title":"Co","workspace_id":"ws1","feature_params_source":"company"}`,
		`{"title":"Pers","workspace_id":"ws1","feature_params_source":"personal","personal_feature_params_config_id":"cfg-1"}`,
	} {
		rec, payload = create(body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("valid create %s: expected 201, got %d: %s", body, rec.Code, rec.Body.String())
		}
		if payload["feature_params_source"] == "none" {
			t.Fatalf("valid create %s: source unexpectedly none", body)
		}
	}

	// 更新：显式 none → 400
	rec, payload = create(`{"title":"GateUpd","workspace_id":"ws1","feature_params_source":"company"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("gate-upd create: %d %s", rec.Code, rec.Body.String())
	}
	taskID, _ := payload["id"].(string)
	if taskID == "" {
		t.Fatalf("no task id in create payload: %v", payload)
	}
	reqPatch := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(`{"feature_params_source":"none"}`))
	reqPatch.Header.Set("X-Auth-Tenant-Id", "t1")
	reqPatch.Header.Set("X-Auth-User-Id", "u1")
	recPatch := httptest.NewRecorder()
	handleTaskRoutes(recPatch, reqPatch)
	if recPatch.Code != http.StatusBadRequest {
		t.Fatalf("update to none: expected 400, got %d: %s", recPatch.Code, recPatch.Body.String())
	}
	if !strings.Contains(recPatch.Body.String(), "FEATURE_PARAMS_SOURCE_REQUIRED") {
		t.Fatalf("update to none: missing gate code, got: %s", recPatch.Body.String())
	}

	// 更新：不带 feature_params 键 → 200 且 source 保持不变
	reqPatch2 := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(`{"title":"Renamed"}`))
	reqPatch2.Header.Set("X-Auth-Tenant-Id", "t1")
	reqPatch2.Header.Set("X-Auth-User-Id", "u1")
	recPatch2 := httptest.NewRecorder()
	handleTaskRoutes(recPatch2, reqPatch2)
	if recPatch2.Code != http.StatusOK {
		t.Fatalf("plain update: expected 200, got %d: %s", recPatch2.Code, recPatch2.Body.String())
	}
	var updated map[string]interface{}
	json.NewDecoder(recPatch2.Body).Decode(&updated)
	if updated["feature_params_source"] != "company" {
		t.Fatalf("plain update: source drifted to %v, want company", updated["feature_params_source"])
	}
}

func TestSearchTasks(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	create := func(title string) {
		t.Helper()
		body := `{"title":"` + title + `","workspace_id":"ws1"}`
		req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		req.Header.Set("X-Auth-User-Id", "u1")
		rec := httptest.NewRecorder()
		handleTaskRoutes(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %q: expected 201, got %d: %s", title, rec.Code, rec.Body.String())
		}
	}
	create("Billing Alpha")
	create("Other Beta")

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/search/?q=Alpha&limit=10", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleSearchTasks(rec, req, "t1")
	if rec.Code != http.StatusOK {
		t.Fatalf("search: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("expected 1 hit for q=Alpha, got %d: %#v", len(payload.Results), payload.Results)
	}
	if payload.Results[0]["title"] != "Billing Alpha" {
		t.Fatalf("unexpected title: %#v", payload.Results[0]["title"])
	}
	if payload.Results[0]["workspace_id"] != "ws1" {
		t.Fatalf("unexpected workspace_id: %#v", payload.Results[0]["workspace_id"])
	}

	reqWS := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/search/?workspace_id=ws1&limit=5", nil)
	reqWS.Header.Set("X-Auth-Tenant-Id", "t1")
	reqWS.Header.Set("X-Auth-User-Id", "u1")
	recWS := httptest.NewRecorder()
	handleSearchTasks(recWS, reqWS, "t1")
	if recWS.Code != http.StatusOK {
		t.Fatalf("search workspace: expected 200, got %d: %s", recWS.Code, recWS.Body.String())
	}
	var all struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.NewDecoder(recWS.Body).Decode(&all); err != nil {
		t.Fatalf("decode all: %v", err)
	}
	if len(all.Results) != 2 {
		t.Fatalf("expected 2 tasks in ws1, got %d", len(all.Results))
	}

	reqMissing := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/search/?workspace_id=missing", nil)
	reqMissing.Header.Set("X-Auth-Tenant-Id", "t1")
	reqMissing.Header.Set("X-Auth-User-Id", "u1")
	recMissing := httptest.NewRecorder()
	handleSearchTasks(recMissing, reqMissing, "t1")
	if recMissing.Code != http.StatusNotFound {
		t.Fatalf("missing workspace: expected 404, got %d", recMissing.Code)
	}
}

// TestSearchTasksConventionPath — FE navbar/billing 使用约定路径
// GET /api/tasks/search/tenant_id/{tid}/?q=...（非 /api/tenant/{tid}/tasks/search/）
func TestSearchTasksConventionPath(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	body := `{"title":"Navbar Search Hit","workspace_id":"ws1"}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	createReq.Header.Set("X-Auth-Tenant-Id", "t1")
	createReq.Header.Set("X-Auth-User-Id", "u1")
	createRec := httptest.NewRecorder()
	handleTaskRoutes(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/search/tenant_id/t1/?q=Navbar&limit=10", nil)
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := serveMux(req)
	if rec.Code != http.StatusOK {
		t.Fatalf("convention search: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Results) != 1 {
		t.Fatalf("expected 1 hit, got %d: %#v", len(payload.Results), payload.Results)
	}
	if payload.Results[0]["title"] != "Navbar Search Hit" {
		t.Fatalf("unexpected title: %#v", payload.Results[0]["title"])
	}

	// missing tenant_id must not be treated as task id "search"
	bad := httptest.NewRequest(http.MethodGet, "/api/tasks/search/?q=x", nil)
	bad.Header.Set("X-Auth-User-Id", "u1")
	badRec := serveMux(bad)
	if badRec.Code != http.StatusBadRequest && badRec.Code != http.StatusNotFound {
		t.Fatalf("search without tenant_id: expected 400/404, got %d: %s", badRec.Code, badRec.Body.String())
	}
}

// TestTaskConventionVerbNotRoutedAsTaskId — 约定动词（todos/search）须在 {taskId} 分支前拦截，
// 否则首段动词会被当作 taskId 走 handleGetTaskByID → 404「资源不存在」（OPT-20260811-058）。
// 表驱动覆盖 taskFE 现有全部 /api/tasks/<verb>/ 调用，防止下一个动词（renew/batch/…）再被误路由。
func TestTaskConventionVerbNotRoutedAsTaskId(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	cases := []struct {
		name     string
		method   string
		path     string
		expectOK bool // 约定路径应返回非 4xx（search/todos 均为正常约定语义）
	}{
		{"search", http.MethodGet, "/api/tasks/search/tenant_id/t1/?q=Navbar&limit=10", true},
		{"todos-family", http.MethodGet, "/api/tasks/todos/tenant_id/t1/workspace_id/ws1/", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("X-Auth-User-Id", "u1")
			rec := serveMux(req)
			if rec.Code == http.StatusNotFound && strings.Contains(rec.Body.String(), errMsgNotFound) {
				t.Fatalf("convention verb %q 被误当作 taskId：status=%d body=%s", tc.name, rec.Code, rec.Body.String())
			}
			if tc.expectOK && rec.Code >= http.StatusBadRequest {
				t.Fatalf("convention verb %q 期望非 4xx，got %d body=%s", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestSearchTasksByOwnerAndAssignee(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	createWithPeople := func(title, owner, operator string, assignees []string) string {
		t.Helper()
		bodyMap := map[string]interface{}{
			"title":        title,
			"workspace_id": "ws1",
			"owner":        owner,
			"assignees":    assignees,
		}
		if operator != "" {
			bodyMap["operator"] = operator
		}
		raw, _ := json.Marshal(bodyMap)
		req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(string(raw)))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		req.Header.Set("X-Auth-User-Id", "u1")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleTaskRoutes(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %q: expected 201, got %d: %s", title, rec.Code, rec.Body.String())
		}
		var created map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
			t.Fatalf("decode create: %v", err)
		}
		id, _ := created["id"].(string)
		if id == "" {
			t.Fatalf("missing id in create response: %#v", created)
		}
		return id
	}

	idOwner := createWithPeople("Owned By Alice", "member-alice", "", nil)
	idAssignee := createWithPeople("Assigned Bob", "member-other", "", []string{"member-bob"})
	idOperator := createWithPeople("Operated By Dave", "member-other", "member-dave", nil)
	_ = createWithPeople("Unrelated", "member-other", "", []string{"member-carol"})

	search := func(query string) []map[string]interface{} {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/search/?"+query, nil)
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		req.Header.Set("X-Auth-User-Id", "u1")
		rec := httptest.NewRecorder()
		handleSearchTasks(rec, req, "t1")
		if rec.Code != http.StatusOK {
			t.Fatalf("search %q: expected 200, got %d: %s", query, rec.Code, rec.Body.String())
		}
		var payload struct {
			Results []map[string]interface{} `json:"results"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return payload.Results
	}

	byOwner := search("q=member-alice&limit=20")
	if len(byOwner) != 1 || byOwner[0]["id"] != idOwner {
		t.Fatalf("owner LIKE: expected id=%s, got %#v", idOwner, byOwner)
	}
	if byOwner[0]["owner"] != "member-alice" {
		t.Fatalf("owner field missing: %#v", byOwner[0])
	}

	byAssigneeQ := search("q=member-bob&limit=20")
	if len(byAssigneeQ) != 1 || byAssigneeQ[0]["id"] != idAssignee {
		t.Fatalf("assignee LIKE: expected id=%s, got %#v", idAssignee, byAssigneeQ)
	}
	assignees, _ := byAssigneeQ[0]["assignees"].([]interface{})
	if len(assignees) == 0 {
		t.Fatalf("assignees field missing: %#v", byAssigneeQ[0])
	}

	byAssigneeIDs := search("assignee_ids=member-bob&limit=20")
	if len(byAssigneeIDs) != 1 || byAssigneeIDs[0]["id"] != idAssignee {
		t.Fatalf("assignee_ids: expected id=%s, got %#v", idAssignee, byAssigneeIDs)
	}

	byOperator := search("q=member-dave&limit=20")
	if len(byOperator) != 1 || byOperator[0]["id"] != idOperator {
		t.Fatalf("operator LIKE: expected id=%s, got %#v", idOperator, byOperator)
	}
	if byOperator[0]["operator"] != "member-dave" {
		t.Fatalf("operator field missing: %#v", byOperator[0])
	}

	byOperatorIDs := search("assignee_ids=member-dave&limit=20")
	if len(byOperatorIDs) != 1 || byOperatorIDs[0]["id"] != idOperator {
		t.Fatalf("assignee_ids as operator: expected id=%s, got %#v", idOperator, byOperatorIDs)
	}
}

func TestSearchTasksRespectsWorkspaceACL(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		userID := strings.TrimSpace(r.Header.Get("X-Auth-User-Id"))
		if pathIsWorkspaceList(r) {
			switch userID {
			case "u1":
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": "ws1", "name": "WS1", "company_id": "t1"},
				})
			case "u2":
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": "ws2", "name": "WS2", "company_id": "t1"},
				})
			default:
				json.NewEncoder(w).Encode([]map[string]interface{}{})
			}
			return
		}
		if pathHasWorkspace(r, "ws1") {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if pathHasWorkspace(r, "ws2") {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws2", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
	cfg.TaskBillURL = ""

	createIn := func(title, workspaceID, userID string) string {
		t.Helper()
		body := `{"title":"` + title + `","workspace_id":"` + workspaceID + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/"+workspaceID+"/todos/", strings.NewReader(body))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		req.Header.Set("X-Auth-User-Id", userID)
		rec := httptest.NewRecorder()
		handleTaskRoutes(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create %q in %s: %d %s", title, workspaceID, rec.Code, rec.Body.String())
		}
		var created map[string]interface{}
		_ = json.NewDecoder(rec.Body).Decode(&created)
		id, _ := created["id"].(string)
		return id
	}
	idWS1 := createIn("Secret In WS1", "ws1", "u1")
	idWS2 := createIn("Secret In WS2", "ws2", "u2")

	searchAs := func(userID, query string) (int, []map[string]interface{}) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/search/?"+query, nil)
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		if userID != "" {
			req.Header.Set("X-Auth-User-Id", userID)
		}
		rec := httptest.NewRecorder()
		handleSearchTasks(rec, req, "t1")
		var payload struct {
			Results []map[string]interface{} `json:"results"`
		}
		_ = json.NewDecoder(rec.Body).Decode(&payload)
		return rec.Code, payload.Results
	}

	code, results := searchAs("u2", "q=Secret&limit=20")
	if code != http.StatusOK {
		t.Fatalf("u2 search: expected 200, got %d", code)
	}
	if len(results) != 1 || results[0]["id"] != idWS2 {
		t.Fatalf("u2 must only see ws2 task, got %#v (ws1=%s ws2=%s)", results, idWS1, idWS2)
	}

	code, results = searchAs("u1", "q=Secret&limit=20")
	if code != http.StatusOK || len(results) != 1 || results[0]["id"] != idWS1 {
		t.Fatalf("u1 must only see ws1 task, got code=%d %#v", code, results)
	}

	code, _ = searchAs("u2", "workspace_id=ws1&limit=20")
	if code != http.StatusForbidden {
		t.Fatalf("u2 workspace_id=ws1: expected 403, got %d", code)
	}

	code, _ = searchAs("", "q=Secret&limit=20")
	if code != http.StatusUnauthorized {
		t.Fatalf("anonymous search: expected 401, got %d", code)
	}

	// Access lookup failure must fail closed (no cross-workspace leak).
	cfg.ProjectServiceURL = "http://127.0.0.1:1"
	code, results = searchAs("u2", "q=Secret&limit=20")
	if code != http.StatusServiceUnavailable {
		t.Fatalf("lookup failure: expected 503, got %d", code)
	}
	if len(results) != 0 {
		t.Fatalf("lookup failure must not return tasks, got %#v", results)
	}
}

func TestFeatureParamsSnapshot(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	createBody := `{"title":"FP","workspace_id":"ws1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)

	snapBody := `{"params":{"k":"v"}}`
	reqSnap := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/feature-params-snapshots/", strings.NewReader(snapBody))
	reqSnap.Header.Set("X-Task-Id", taskID)
	recSnap := httptest.NewRecorder()
	handleFeatureParamsSnapshots(recSnap, reqSnap)
	if recSnap.Code != http.StatusCreated {
		t.Fatalf("snapshot: expected 201, got %d: %s", recSnap.Code, recSnap.Body.String())
	}
}

func TestCreateTaskMultiRepoSameProject(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pathHasWorkspace(r, "ws1") {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			w.Write([]byte("[]"))
			return
		}
		if pathHasProject(r, "p1") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"git_repos": []string{
					"https://git.example/repo-a.git",
					"https://git.example/repo-b.git",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
	cfg.TaskBillURL = ""

	body := `{
		"title":"Multi repo task",
		"workspace_id":"ws1",
		"projects":[
			{"project_id":"p1","repo_index":0,"base_branch":"master","target_branch":"feature/x"},
			{"project_id":"p1","repo_index":1,"base_branch":"master","target_branch":"feature/x"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create multi-repo task: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatal("expected task id")
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", nil)
	reqGet.Header.Set("X-Auth-Tenant-Id", "t1")
	reqGet.Header.Set("X-Auth-User-Id", "u1")
	recGet := httptest.NewRecorder()
	handleTaskRoutes(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get task: expected 200, got %d: %s", recGet.Code, recGet.Body.String())
	}

	var detail map[string]interface{}
	json.NewDecoder(recGet.Body).Decode(&detail)
	projects, ok := detail["projects"].([]interface{})
	if !ok || len(projects) != 2 {
		t.Fatalf("expected 2 project links, got %#v", detail["projects"])
	}
}

func TestCreateTaskRejectsMultipleProjects(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	body := `{
		"title":"Two projects",
		"workspace_id":"ws1",
		"projects":[
			{"project_id":"p1","repo_index":0,"base_branch":"master","target_branch":"feature/x"},
			{"project_id":"p2","repo_index":0,"base_branch":"master","target_branch":"feature/y"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), errMsgOneProjectOnly) {
		t.Fatalf("expected project limit error, got: %s", rec.Body.String())
	}
}

func TestCreateTaskRejectsMissingBaseBranchInChinese(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	body := `{
		"title":"Missing base branch",
		"workspace_id":"ws1",
		"projects":[
			{"project_id":"p1","repo_index":0,"base_branch":"","target_branch":"feature/x"}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	want := "请为关联项目第 1 个仓库配置基准分支"
	if !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("expected Chinese base_branch error %q, got: %s", want, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "base_branch required") {
		t.Fatalf("did not expect English base_branch error, got: %s", rec.Body.String())
	}
}

func TestTranslateBranchTitleRetiredReturns501(t *testing.T) {
	body := `{"title":"Feature-123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/translate-branch-title/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTranslateBranchTitle(rec, req, "t1")
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "task-project-service") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

// Django task_client.get_task_by_id 使用 X-Auth-User-Id=internal；当 workspace-access
// 返回真实成员列表时，不得因非成员而 403（否则容器 task-detail 得到空 project_repos）。
func TestGetTaskByIDAllowsInternalUserWhenWorkspaceHasMembers(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pathHasWorkspace(r, "ws1") && !strings.Contains(r.URL.Path, "workspace-access") {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"user_id": "u-real-member"},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
	cfg.TaskBillURL = ""

	createBody := `{"title":"Internal read","workspace_id":"ws1"}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	reqCreate.Header.Set("X-Auth-Tenant-Id", "t1")
	reqCreate.Header.Set("X-Auth-User-Id", "u-real-member")
	recCreate := httptest.NewRecorder()
	handleTaskRoutes(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", recCreate.Code, recCreate.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(recCreate.Body).Decode(&created)
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatal("expected task id")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID+"/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "internal")
	req.Header.Set("X-Task-Id", taskID)
	rec := httptest.NewRecorder()
	handleGetTaskByID(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("internal get by id: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	reqDenied := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID+"/", nil)
	reqDenied.Header.Set("X-Auth-Tenant-Id", "t1")
	reqDenied.Header.Set("X-Auth-User-Id", "u-stranger")
	reqDenied.Header.Set("X-Task-Id", taskID)
	recDenied := httptest.NewRecorder()
	handleGetTaskByID(recDenied, reqDenied)
	if recDenied.Code != http.StatusForbidden {
		t.Fatalf("stranger get by id: expected 403, got %d: %s", recDenied.Code, recDenied.Body.String())
	}
}

func TestContainerSnapshotInternal(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/workspaces/") {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			w.Write([]byte(`[{"user_id":"u1"}]`))
			return
		}
		if strings.Contains(r.URL.Path, "/projects/tenant_id/") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":        "proj1",
				"name":      "Demo Project",
				"git_repos": []string{"https://git.example/a.git"},
				"server_run_template": map[string]interface{}{
					"region":            "cn-hangzhou",
					"cloud_platform_id": "plat-1",
					"vpc_id":            "vpc-1",
					"vswitch_id":        "vsw-1",
					"security_group_id": "sg-1",
					"label":             "demo",
					"default_auto_run":  true,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
	cfg.CloudServiceURL = "http://127.0.0.1:9" // satisfy auto_run gate; start is stubbed below
	cfg.TaskBillURL = ""
	cfg.InternalSecret = ""
	prevSchedule := scheduleTaskAutoRunFn
	scheduleTaskAutoRunFn = func(p autoRunTriggerParams) {}
	t.Cleanup(func() { scheduleTaskAutoRunFn = prevSchedule })
	prevLookup := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return &InstalledImageLookup{ID: imageID, Name: "snap-image", ExternalImageID: "ext-img-snap"}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prevLookup })

	createBody := `{
		"title":"Snap Task",
		"workspace_id":"ws1",
		"description":"desc",
		"auto_run":true,
		"container_image_id":"img-snap",
		"repo_identities":[{"repo_url":"https://git.example/a.git","git_identity_id":"gid-1"}],
		"projects":[{"project_id":"proj1","repo_index":0,"base_branch":"main","target_branch":"feat"}],
		"branch_strategy":{"work_branch_name":"work/x","merge_target_branch_name":"main","target_branch_name":"feat"}
	}`
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	reqCreate.Header.Set("X-Auth-Tenant-Id", "t1")
	reqCreate.Header.Set("X-Auth-User-Id", "u1")
	recCreate := httptest.NewRecorder()
	handleTaskRoutes(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", recCreate.Code, recCreate.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(recCreate.Body).Decode(&created)
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatal("expected task id")
	}

	patchBody := `{"repo_clone_git_identities":{"https://git.example/a.git":"gid-1"}}`
	reqPatch := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/repo-clone-git-identities/", strings.NewReader(patchBody))
	reqPatch.Header.Set("X-Auth-Tenant-Id", "t1")
	reqPatch.Header.Set("X-Auth-User-Id", "u1")
	reqPatch.Header.Set("X-Workspace-Id", "ws1")
	reqPatch.Header.Set("X-Resource-Id", taskID)
	recPatch := httptest.NewRecorder()
	handleTaskRoutes(recPatch, reqPatch)
	if recPatch.Code != http.StatusOK {
		t.Fatalf("patch identities: expected 200, got %d: %s", recPatch.Code, recPatch.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tasks/"+taskID+"/container-snapshot", nil)
	req.Header.Set("X-Task-Id", taskID)
	rec := httptest.NewRecorder()
	handleInternalContainerSnapshot(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("snapshot: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var snap map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&snap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if snap["title"] != "Snap Task" {
		t.Fatalf("title=%v", snap["title"])
	}
	if snap["auto_run"] != true {
		t.Fatalf("auto_run=%v", snap["auto_run"])
	}
	if snap["installed_image_id"] != "img-snap" {
		t.Fatalf("installed_image_id=%v want img-snap", snap["installed_image_id"])
	}
	if snap["owner_id"] != "u1" {
		t.Fatalf("owner_id=%v want task owner fallback for nested discovery", snap["owner_id"])
	}
	if snap["target_branch"] != "work/x" {
		t.Fatalf("target_branch=%v", snap["target_branch"])
	}
	projects, _ := snap["projects"].([]interface{})
	if len(projects) != 1 {
		t.Fatalf("projects=%v", snap["projects"])
	}
	p0 := projects[0].(map[string]interface{})
	if p0["project_name"] != "Demo Project" {
		t.Fatalf("project_name=%v", p0["project_name"])
	}
	idents, _ := snap["repo_identities"].([]interface{})
	if len(idents) != 1 {
		t.Fatalf("repo_identities=%v", snap["repo_identities"])
	}
	id0 := idents[0].(map[string]interface{})
	if id0["git_identity_id"] != "gid-1" {
		t.Fatalf("git_identity_id=%v", id0["git_identity_id"])
	}

	reqMiss := httptest.NewRequest(http.MethodGet, "/api/internal/tasks/missing/container-snapshot", nil)
	reqMiss.Header.Set("X-Task-Id", "missing")
	recMiss := httptest.NewRecorder()
	handleInternalContainerSnapshot(recMiss, reqMiss)
	if recMiss.Code != http.StatusNotFound {
		t.Fatalf("missing: expected 404, got %d", recMiss.Code)
	}
}

func TestMain(m *testing.M) {
	cfg.Host = "127.0.0.1"
	cfg.Port = 8017
	cfg.DBPath = ":memory:"
	os.Exit(m.Run())
}

// pathHasWorkspace 兼容新旧 project service 路径约定的 workspace 单查匹配：
//
//	旧: /api/tenant/{tid}/workspaces/{wid}
//	新: /api/projects/workspaces/tenant_id/{tid}/{wid}
func pathHasWorkspace(r *http.Request, wid string) bool {
	p := r.URL.Path
	return strings.Contains(p, "/workspaces/"+wid) || strings.HasSuffix(p, "/"+wid)
}

// pathHasProject 匹配项目条目查询新约定: /api/projects/tenant_id/{tid}/{pid}
func pathHasProject(r *http.Request, pid string) bool {
	p := strings.TrimSuffix(r.URL.Path, "/")
	return strings.Contains(p, "/projects/tenant_id/") && strings.HasSuffix(p, "/"+pid)
}

// pathIsWorkspaceList 匹配新旧约定的 workspaces 列表查询（?mine=1）。
// 单查 URL /api/projects/workspaces/tenant_id/{tid}/{wid} 的 tenant_id 后还有
// workspace 段，不属于列表，须排除。
func pathIsWorkspaceList(r *http.Request) bool {
	p := strings.TrimSuffix(strings.SplitN(r.URL.Path, "?", 2)[0], "/")
	if strings.HasSuffix(p, "/workspaces") {
		return true // 旧约定: /api/tenant/{tid}/workspaces[/?mine=1]
	}
	idx := strings.Index(p, "/tenant_id/")
	if idx < 0 {
		return false
	}
	return !strings.Contains(p[idx+len("/tenant_id/"):], "/")
}
