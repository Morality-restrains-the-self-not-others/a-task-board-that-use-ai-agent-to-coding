package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func createRevisionTestTask(t *testing.T, title, desc string) string {
	t.Helper()
	body := fmt.Sprintf(`{"title":%q,"description":%q,"workspace_id":"ws1"}`, title, desc)
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("missing id")
	}
	return id
}

func getTaskRevisions(t *testing.T, taskID, userID string, query string) *httptest.ResponseRecorder {
	t.Helper()
	url := "/api/tenant/t1/workspace/ws1/todos/" + taskID + "/revisions/"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", userID)
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	return rec
}

func decodeRevisionList(t *testing.T, rec *httptest.ResponseRecorder) (results []map[string]interface{}, total float64) {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	raw, _ := payload["results"].([]interface{})
	for _, item := range raw {
		m, _ := item.(map[string]interface{})
		results = append(results, m)
	}
	total, _ = payload["total"].(float64)
	return results, total
}

func TestRevisionCreateWritesV1(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	id := createRevisionTestTask(t, "Hello", "body")
	rec := getTaskRevisions(t, id, "u1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	rows, total := decodeRevisionList(t, rec)
	if total != 1 || len(rows) != 1 {
		t.Fatalf("want 1 revision, total=%v rows=%d", total, len(rows))
	}
	if jsonNumber(rows[0]["version_num"]) != 1 {
		t.Fatalf("version=%v", rows[0]["version_num"])
	}
	if rows[0]["title"] != "Hello" || rows[0]["description"] != "body" {
		t.Fatalf("snapshot=%v", rows[0])
	}
}

func TestRevisionNoContentChangeSkips(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	id := createRevisionTestTask(t, "Hello", "body")
	req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+id+"/", strings.NewReader(`{"priority":"high"}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	list := getTaskRevisions(t, id, "u1", "")
	_, total := decodeRevisionList(t, list)
	if total != 1 {
		t.Fatalf("priority-only update must not append, total=%v", total)
	}
}

func TestRevisionTitleChangeAppendsV2(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	id := createRevisionTestTask(t, "Hello", "body")
	req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+id+"/", strings.NewReader(`{"title":"World"}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	list := getTaskRevisions(t, id, "u1", "")
	rows, total := decodeRevisionList(t, list)
	if total != 2 {
		t.Fatalf("want 2 revisions, total=%v", total)
	}
	if jsonNumber(rows[0]["version_num"]) != 2 {
		t.Fatalf("list must be version_num DESC, first=%v", rows[0]["version_num"])
	}
	cf, _ := rows[0]["changed_fields"].(string)
	if !strings.Contains(cf, "title") {
		t.Fatalf("changed_fields=%q", cf)
	}
	if rows[0]["title"] != "World" {
		t.Fatalf("v2 title=%v", rows[0]["title"])
	}
}

func TestRevisionListForbiddenWithoutWorkspaceAccess(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pathIsWorkspaceList(r) {
			json.NewEncoder(w).Encode([]map[string]interface{}{{"id": "ws1", "name": "WS1", "company_id": "t1"}})
			return
		}
		if pathHasWorkspace(r, "ws1") {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ws1", "company_id": "t1"})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			json.NewEncoder(w).Encode([]map[string]interface{}{{"user": "u1"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
	cfg.TaskBillURL = ""

	id := createRevisionTestTask(t, "Secret", "x")
	rec := getTaskRevisions(t, id, "u2", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestRevisionGetWrongTaskReturns404(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	a := createRevisionTestTask(t, "A", "da")
	b := createRevisionTestTask(t, "B", "db")
	list := getTaskRevisions(t, a, "u1", "")
	rows, _ := decodeRevisionList(t, list)
	revID, _ := rows[0]["id"].(string)
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/"+b+"/revisions/"+revID+"/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 IDOR, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestRevisionListPaginationIsolatesTask(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	id := createRevisionTestTask(t, "P0", "d")
	other := createRevisionTestTask(t, "Other", "o")
	for i := 1; i <= 5; i++ {
		body := fmt.Sprintf(`{"title":"P%d"}`, i)
		req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+id+"/", strings.NewReader(body))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		req.Header.Set("X-Auth-User-Id", "u1")
		rec := httptest.NewRecorder()
		handleTaskRoutes(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("patch %d: %d %s", i, rec.Code, rec.Body.String())
		}
	}
	rec := getTaskRevisions(t, id, "u1", "limit=3&offset=0")
	rows, total := decodeRevisionList(t, rec)
	if total != 6 {
		t.Fatalf("total=%v want 6", total)
	}
	if len(rows) != 3 {
		t.Fatalf("page size=%d", len(rows))
	}
	otherList := getTaskRevisions(t, other, "u1", "")
	_, otherTotal := decodeRevisionList(t, otherList)
	if otherTotal != 1 {
		t.Fatalf("other task leaked, total=%v", otherTotal)
	}
}

func TestRevisionInsertFailureRollsBackTitle(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	id := createRevisionTestTask(t, "Hello", "body")
	orig := insertTaskRevisionTxFn
	t.Cleanup(func() { insertTaskRevisionTxFn = orig })
	insertTaskRevisionTxFn = func(tx *sql.Tx, row taskRevisionRow) error {
		return fmt.Errorf("injected revision insert failure")
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+id+"/", strings.NewReader(`{"title":"ShouldNotStick"}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("expected fail-closed update, got 200 %s", rec.Body.String())
	}
	loaded, err := loadTask(id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Title != "Hello" {
		t.Fatalf("hot title changed to %q despite revision insert failure", loaded.Title)
	}
}

func TestRevisionPublishOmitsDescription(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	var captured map[string]interface{}
	orig := publishTaskRevisionRecordedFn
	t.Cleanup(func() { publishTaskRevisionRecordedFn = orig })
	publishTaskRevisionRecordedFn = func(ctx context.Context, tenantID, workspaceID, taskID, revisionID string, versionNum int, actorUserID, changedFields string) error {
		captured = map[string]interface{}{
			"tenant_id": tenantID, "workspace_id": workspaceID, "task_id": taskID,
			"revision_id": revisionID, "version_num": versionNum,
			"actor_user_id": actorUserID, "changed_fields": changedFields,
		}
		return nil
	}
	id := createRevisionTestTask(t, "Hello", "secret-body")
	if captured == nil {
		t.Fatal("expected create publish")
	}
	if captured["task_id"] != id {
		t.Fatalf("key/task_id=%v", captured["task_id"])
	}
	if _, ok := captured["description"]; ok {
		t.Fatal("payload must not include description")
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+id+"/", strings.NewReader(`{"description":"new-secret"}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch desc: %d %s", rec.Code, rec.Body.String())
	}
	cf, _ := captured["changed_fields"].(string)
	if !strings.Contains(cf, "description") {
		t.Fatalf("changed_fields=%q", cf)
	}
	if _, ok := captured["description"]; ok {
		t.Fatal("update payload must not include description")
	}
}

func TestRevisionListVersionDesc(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	id := createRevisionTestTask(t, "V1", "d")
	for _, title := range []string{"V2", "V3"} {
		req := httptest.NewRequest(http.MethodPatch, "/api/tenant/t1/workspace/ws1/todos/"+id+"/", strings.NewReader(`{"title":"`+title+`"}`))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		req.Header.Set("X-Auth-User-Id", "u1")
		rec := httptest.NewRecorder()
		handleTaskRoutes(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("patch %s: %d", title, rec.Code)
		}
	}
	rows, _ := decodeRevisionList(t, getTaskRevisions(t, id, "u1", ""))
	if len(rows) != 3 {
		t.Fatalf("rows=%d", len(rows))
	}
	if jsonNumber(rows[0]["version_num"]) != 3 || jsonNumber(rows[2]["version_num"]) != 1 {
		t.Fatalf("order %+v", []interface{}{rows[0]["version_num"], rows[1]["version_num"], rows[2]["version_num"]})
	}
}
