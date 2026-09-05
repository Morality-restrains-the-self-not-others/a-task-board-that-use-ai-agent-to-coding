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

func createRevisionTestProject(t *testing.T, name, desc string) string {
	t.Helper()
	body := fmt.Sprintf(`{"name":%q,"description":%q}`, name, desc)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
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

func getProjectRevisionsRec(t *testing.T, tenantID, projectID, query string) *httptest.ResponseRecorder {
	t.Helper()
	url := "/api/projects/tenant_id/" + tenantID + "/" + projectID + "/revisions/"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, tenantID, []string{projectID, "revisions"})
	return rec
}

func decodeProjectRevisionList(t *testing.T, rec *httptest.ResponseRecorder) (results []map[string]interface{}, total float64) {
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
	id := createRevisionTestProject(t, "Alpha", "body")
	rec := getProjectRevisionsRec(t, "t1", id, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body.String())
	}
	rows, total := decodeProjectRevisionList(t, rec)
	if total != 1 || len(rows) != 1 {
		t.Fatalf("want 1, total=%v n=%d", total, len(rows))
	}
	if fmt.Sprint(rows[0]["version_num"]) != "1" {
		t.Fatalf("version=%v", rows[0]["version_num"])
	}
	if rows[0]["name"] != "Alpha" || rows[0]["description"] != "body" {
		t.Fatalf("snapshot=%v", rows[0])
	}
}

func TestRevisionNoContentChangeSkips(t *testing.T) {
	setupTestDB(t)
	id := createRevisionTestProject(t, "Alpha", "body")
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/tenant_id/t1/"+id+"/", strings.NewReader(`{"tags":["x"]}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Resource-Id", id)
	rec := httptest.NewRecorder()
	handleUpdateProject(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch tags: %d %s", rec.Code, rec.Body.String())
	}
	_, total := decodeProjectRevisionList(t, getProjectRevisionsRec(t, "t1", id, ""))
	if total != 1 {
		t.Fatalf("tags-only must not append, total=%v", total)
	}
}

func TestRevisionNameChangeAppendsV2(t *testing.T) {
	setupTestDB(t)
	id := createRevisionTestProject(t, "Alpha", "body")
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/tenant_id/t1/"+id+"/", strings.NewReader(`{"name":"Beta"}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Resource-Id", id)
	rec := httptest.NewRecorder()
	handleUpdateProject(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch name: %d %s", rec.Code, rec.Body.String())
	}
	rows, total := decodeProjectRevisionList(t, getProjectRevisionsRec(t, "t1", id, ""))
	if total != 2 {
		t.Fatalf("total=%v", total)
	}
	if fmt.Sprint(rows[0]["version_num"]) != "2" {
		t.Fatalf("DESC first=%v", rows[0]["version_num"])
	}
	cf, _ := rows[0]["changed_fields"].(string)
	if !strings.Contains(cf, "name") {
		t.Fatalf("changed_fields=%q", cf)
	}
}

func TestRevisionWrongTenantReturns404(t *testing.T) {
	setupTestDB(t)
	id := createRevisionTestProject(t, "Alpha", "body")
	req := httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t-other/"+id+"/revisions/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t-other")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t-other", []string{id, "revisions"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 cross-tenant, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestRevisionGetWrongProjectReturns404(t *testing.T) {
	setupTestDB(t)
	a := createRevisionTestProject(t, "A", "da")
	b := createRevisionTestProject(t, "B", "db")
	rows, _ := decodeProjectRevisionList(t, getProjectRevisionsRec(t, "t1", a, ""))
	revID, _ := rows[0]["id"].(string)
	req := httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1/"+b+"/revisions/"+revID+"/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t1", []string{b, "revisions", revID})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 IDOR, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestRevisionListPaginationIsolatesProject(t *testing.T) {
	setupTestDB(t)
	id := createRevisionTestProject(t, "P0", "d")
	other := createRevisionTestProject(t, "Other", "o")
	for i := 1; i <= 5; i++ {
		body := fmt.Sprintf(`{"name":"P%d"}`, i)
		req := httptest.NewRequest(http.MethodPatch, "/api/projects/tenant_id/t1/"+id+"/", strings.NewReader(body))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		req.Header.Set("X-Auth-User-Id", "u1")
		req.Header.Set("X-Resource-Id", id)
		rec := httptest.NewRecorder()
		handleUpdateProject(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("patch %d: %d %s", i, rec.Code, rec.Body.String())
		}
	}
	rec := getProjectRevisionsRec(t, "t1", id, "limit=3&offset=0")
	rows, total := decodeProjectRevisionList(t, rec)
	if total != 6 || len(rows) != 3 {
		t.Fatalf("total=%v n=%d", total, len(rows))
	}
	_, otherTotal := decodeProjectRevisionList(t, getProjectRevisionsRec(t, "t1", other, ""))
	if otherTotal != 1 {
		t.Fatalf("other leaked %v", otherTotal)
	}
}

func TestRevisionInsertFailureRollsBackName(t *testing.T) {
	setupTestDB(t)
	id := createRevisionTestProject(t, "Hello", "body")
	orig := insertProjectRevisionTxFn
	t.Cleanup(func() { insertProjectRevisionTxFn = orig })
	insertProjectRevisionTxFn = func(tx *sql.Tx, row projectRevisionRow) error {
		return fmt.Errorf("injected revision insert failure")
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/tenant_id/t1/"+id+"/", strings.NewReader(`{"name":"ShouldNotStick"}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Resource-Id", id)
	rec := httptest.NewRecorder()
	handleUpdateProject(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("expected fail-closed, got 200 %s", rec.Body.String())
	}
	var name string
	if err := db.QueryRow("SELECT name FROM project_entries WHERE id=?", id).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Hello" {
		t.Fatalf("hot name=%q despite revision failure", name)
	}
}

func TestRevisionPublishOmitsDescription(t *testing.T) {
	setupTestDB(t)
	var captured map[string]interface{}
	orig := publishProjectRevisionRecordedFn
	t.Cleanup(func() { publishProjectRevisionRecordedFn = orig })
	publishProjectRevisionRecordedFn = func(ctx context.Context, tenantID, projectID, revisionID string, versionNum int, actorUserID, changedFields string) error {
		captured = map[string]interface{}{
			"tenant_id": tenantID, "project_id": projectID, "revision_id": revisionID,
			"version_num": versionNum, "actor_user_id": actorUserID, "changed_fields": changedFields,
		}
		return nil
	}
	id := createRevisionTestProject(t, "Hello", "secret-body")
	if captured == nil || captured["project_id"] != id {
		t.Fatalf("create publish=%v", captured)
	}
	if _, ok := captured["description"]; ok {
		t.Fatal("payload must not include description")
	}
}
