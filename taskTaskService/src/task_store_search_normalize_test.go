package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNormalizeTaskSearchQuery(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"  Alpha  ", "Alpha"},
		{"#task_13838391081043321882", "task_13838391081043321882"},
		{"task_13838391081043321882", "task_13838391081043321882"},
		{"task-task_13838391081043321882", "task_13838391081043321882"},
		{"#task-task_13838391081043321882", "task_13838391081043321882"},
		{"321882", "321882"},
		{"task-task_partial", "task_partial"},
		{"task_877071722828820480_cmt_877071748669927424", "task_877071722828820480"},
		{"#task_877071722828820480_cmt_877071748669927424", "task_877071722828820480"},
		{"task-task_877071722828820480_cmt_877071748669927424", "task_877071722828820480"},
		{"task_task_877071722828820480_cmt_877071748669927424", "task_877071722828820480"},
		{"cmt_877071748669927424", "cmt_877071748669927424"},
		{"Alpha_cmt_not_an_id", "Alpha_cmt_not_an_id"},
	}
	for _, tc := range cases {
		got := normalizeTaskSearchQuery(tc.in)
		if got != tc.want {
			t.Fatalf("normalizeTaskSearchQuery(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestCommentIDFromSearchQuery(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"Alpha", ""},
		{"cmt_877071748669927424", "cmt_877071748669927424"},
		{"#cmt_877071748669927424", "cmt_877071748669927424"},
		{"task_877071722828820480_cmt_877071748669927424", "cmt_877071748669927424"},
		{"task-task_877071722828820480_cmt_877071748669927424", "cmt_877071748669927424"},
		{"cmt_heal", "cmt_heal"},
		{"cmt_877071748669927424 extra", "cmt_877071748669927424"},
	}
	for _, tc := range cases {
		got := commentIDFromSearchQuery(tc.in)
		if got != tc.want {
			t.Fatalf("commentIDFromSearchQuery(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestSearchTasksAcceptsServicePrefixedTaskID(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	body := `{"title":"Prefixed ID Hit","workspace_id":"ws1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" || !strings.HasPrefix(id, "task_") {
		t.Fatalf("unexpected created id: %#v", created["id"])
	}
	digits := strings.TrimPrefix(id, "task_")
	pasted := "task-task_" + digits

	search := func(q string) []map[string]interface{} {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/search/?q="+url.QueryEscape(q)+"&limit=20", nil)
		r.Header.Set("X-Auth-Tenant-Id", "t1")
		r.Header.Set("X-Auth-User-Id", "u1")
		rr := httptest.NewRecorder()
		handleSearchTasks(rr, r, "t1")
		if rr.Code != http.StatusOK {
			t.Fatalf("search q=%q: expected 200, got %d: %s", q, rr.Code, rr.Body.String())
		}
		var payload struct {
			Results []map[string]interface{} `json:"results"`
		}
		if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return payload.Results
	}

	hits := search(pasted)
	if len(hits) != 1 || hits[0]["id"] != id {
		t.Fatalf("prefixed id search: expected id=%s, got %#v", id, hits)
	}
	suffix := digits
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	bySuffix := search(suffix)
	if len(bySuffix) == 0 || bySuffix[0]["id"] != id {
		t.Fatalf("suffix search: expected id=%s, got %#v", id, bySuffix)
	}
}

func searchTasksForTest(t *testing.T, q string) []map[string]interface{} {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/search/?q="+url.QueryEscape(q)+"&limit=20", nil)
	r.Header.Set("X-Auth-Tenant-Id", "t1")
	r.Header.Set("X-Auth-User-Id", "u1")
	rr := httptest.NewRecorder()
	handleSearchTasks(rr, r, "t1")
	if rr.Code != http.StatusOK {
		t.Fatalf("search q=%q: expected 200, got %d: %s", q, rr.Code, rr.Body.String())
	}
	var payload struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return payload.Results
}

func createSearchTestTask(t *testing.T, title string) string {
	t.Helper()
	body := `{"title":"` + title + `","workspace_id":"ws1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" || !strings.HasPrefix(id, "task_") {
		t.Fatalf("unexpected created id: %#v", created["id"])
	}
	return id
}

func TestSearchTasksAcceptsCommentContainerName(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	id := createSearchTestTask(t, "Comment Container Hit")
	digits := strings.TrimPrefix(id, "task_")
	pasted := id + "_cmt_877071748669927424"

	hits := searchTasksForTest(t, pasted)
	if len(hits) != 1 || hits[0]["id"] != id {
		t.Fatalf("container name search: expected id=%s, got %#v", id, hits)
	}
	if got, _ := hits[0]["comment_id"].(string); got != "cmt_877071748669927424" {
		t.Fatalf("container name search: expected comment_id, got %#v", hits[0]["comment_id"])
	}
	legacyDouble := "task_task_" + digits + "_cmt_877071748669927424"
	hits = searchTasksForTest(t, legacyDouble)
	if len(hits) != 1 || hits[0]["id"] != id {
		t.Fatalf("legacy double-prefix container search: expected id=%s, got %#v", id, hits)
	}
	servicePrefixed := "task-task_" + digits + "_cmt_877071748669927424"
	hits = searchTasksForTest(t, servicePrefixed)
	if len(hits) != 1 || hits[0]["id"] != id {
		t.Fatalf("service-prefixed container search: expected id=%s, got %#v", id, hits)
	}
}

func TestSearchTasksAcceptsCommentID(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	id := createSearchTestTask(t, "Comment ID Hit")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/tasks/"+id+"/comments/", strings.NewReader(`{"content":"search me"}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	handleCommentRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create comment: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode comment: %v", err)
	}
	commentID, _ := created["id"].(string)
	if commentID == "" || !strings.HasPrefix(commentID, "cmt_") {
		t.Fatalf("unexpected comment id: %#v", created["id"])
	}

	hits := searchTasksForTest(t, commentID)
	if len(hits) != 1 || hits[0]["id"] != id {
		t.Fatalf("comment id search: expected id=%s, got %#v", id, hits)
	}
	if got, _ := hits[0]["comment_id"].(string); got != commentID {
		t.Fatalf("comment id search: expected comment_id=%s, got %#v", commentID, hits[0]["comment_id"])
	}

	// 普通标题查询不应携带 comment_id
	hits = searchTasksForTest(t, "Comment ID Hit")
	if len(hits) != 1 || hits[0]["id"] != id {
		t.Fatalf("title search: expected id=%s, got %#v", id, hits)
	}
	if got, ok := hits[0]["comment_id"]; ok && got != nil && got != "" {
		t.Fatalf("title search: unexpected comment_id=%#v", got)
	}
}
