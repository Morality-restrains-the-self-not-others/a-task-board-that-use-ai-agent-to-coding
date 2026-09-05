package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestTaskDetailEnrichesHumanComments(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	createBody := `{"title":"WithComments","workspace_id":"ws1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)

	_, err := db.Exec(
		`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,created_at) VALUES(?,?,?,?,?,NOW())`,
		"cmt-1", taskID, "u1", "human hello", "",
	)
	if err != nil {
		t.Fatalf("insert comment: %v", err)
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", nil)
	reqGet.Header.Set("X-Auth-Tenant-Id", "t1")
	reqGet.Header.Set("X-Auth-User-Id", "u1")
	recGet := httptest.NewRecorder()
	handleTaskRoutes(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get: %d %s", recGet.Code, recGet.Body.String())
	}
	var detail map[string]interface{}
	json.NewDecoder(recGet.Body).Decode(&detail)
	comments, ok := detail["comments"].([]interface{})
	if !ok || len(comments) != 1 {
		t.Fatalf("comments=%v", detail["comments"])
	}
	c0, _ := comments[0].(map[string]interface{})
	if c0["content"] != "human hello" {
		t.Fatalf("content=%v", c0["content"])
	}
}

func TestHumanCommentsPaginationLegacyArray(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	createBody := `{"title":"Paginate","workspace_id":"ws1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)

	for i, id := range []string{"c1", "c2"} {
		_, err := db.Exec(
			`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,created_at) VALUES(?,?,?,?,?,DATE_ADD(NOW(), INTERVAL ? SECOND))`,
			id, taskID, "u1", "msg"+id, "", -i,
		)
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	reqLegacy := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/"+taskID+"/comments/", nil)
	reqLegacy.Header.Set("X-Auth-Tenant-Id", "t1")
	reqLegacy.Header.Set("X-Auth-User-Id", "u1")
	recLegacy := httptest.NewRecorder()
	handleCommentRoutes(recLegacy, reqLegacy)
	if recLegacy.Code != http.StatusOK {
		t.Fatalf("legacy list: %d", recLegacy.Code)
	}
	var arr []interface{}
	json.NewDecoder(recLegacy.Body).Decode(&arr)
	if len(arr) != 2 {
		t.Fatalf("legacy array len=%d", len(arr))
	}

	reqPage := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/"+taskID+"/comments/?limit=1", nil)
	reqPage.Header.Set("X-Auth-Tenant-Id", "t1")
	reqPage.Header.Set("X-Auth-User-Id", "u1")
	recPage := httptest.NewRecorder()
	handleCommentRoutes(recPage, reqPage)
	var page map[string]interface{}
	json.NewDecoder(recPage.Body).Decode(&page)
	results, _ := page["results"].([]interface{})
	if len(results) != 1 {
		t.Fatalf("page results=%v", page)
	}
	if page["next_cursor"] == nil {
		t.Fatalf("expected next_cursor")
	}
	r0, _ := results[0].(map[string]interface{})
	if r0["id"] != "c1" {
		t.Fatalf("first page should be newest-first (DESC), got id=%v", r0["id"])
	}
}

func startMockAICommentService(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/ai-comments") {
			limit := 50
			if lim := strings.TrimSpace(r.URL.Query().Get("limit")); lim != "" {
				if n, err := strconv.Atoi(lim); err == nil && n > 0 {
					limit = n
				}
			}
			if r.URL.Query().Get("full") == "1" {
				t.Fatalf("ai-comments enrich must not request full=1 when limit is set")
			}
			results := make([]map[string]interface{}, 0, limit+1)
			for i := 0; i < limit+1; i++ {
				results = append(results, map[string]interface{}{
					"id":      fmt.Sprintf("ai-%d", i),
					"content": fmt.Sprintf("ai msg %d", i),
				})
			}
			hasMore := len(results) > limit
			if hasMore {
				results = results[:limit]
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results":     results,
				"has_more":    hasMore,
				"next_cursor": "ai-cur-1",
			})
			return
		}
		if strings.Contains(r.URL.Path, "/container-agent-comments") {
			if r.URL.Query().Get("full") == "1" {
				t.Fatalf("container-agent-comments enrich must not request full=1 when limit is set")
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results":     []map[string]interface{}{},
				"has_more":    false,
				"next_cursor": nil,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.AICommentServiceURL = srv.URL
}

func TestTaskDetailEnrichCommentsLimitedToFiftyWithHasMore(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	startMockAICommentService(t)

	createBody := `{"title":"EnrichLimit","workspace_id":"ws1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)

	for i := 0; i < 51; i++ {
		id := fmt.Sprintf("cmt-%02d", i)
		_, err := db.Exec(
			`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,created_at) VALUES(?,?,?,?,?,DATE_ADD(NOW(), INTERVAL ? SECOND))`,
			id, taskID, "u1", "msg"+id, "", -i,
		)
		if err != nil {
			t.Fatalf("insert comment %s: %v", id, err)
		}
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", nil)
	reqGet.Header.Set("X-Auth-Tenant-Id", "t1")
	reqGet.Header.Set("X-Auth-User-Id", "u1")
	recGet := httptest.NewRecorder()
	handleTaskRoutes(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("get: %d %s", recGet.Code, recGet.Body.String())
	}
	var detail map[string]interface{}
	json.NewDecoder(recGet.Body).Decode(&detail)

	comments, ok := detail["comments"].([]interface{})
	if !ok {
		t.Fatalf("comments type=%T", detail["comments"])
	}
	if len(comments) != detailCommentEnrichLimit {
		t.Fatalf("comments len=%d want %d", len(comments), detailCommentEnrichLimit)
	}
	if detail["comments_has_more"] != true {
		t.Fatalf("comments_has_more=%v", detail["comments_has_more"])
	}
	if detail["comments_next_cursor"] == nil {
		t.Fatalf("expected comments_next_cursor")
	}

	aiComments, ok := detail["ai_comments"].([]interface{})
	if !ok {
		t.Fatalf("ai_comments type=%T", detail["ai_comments"])
	}
	if len(aiComments) != detailCommentEnrichLimit {
		t.Fatalf("ai_comments len=%d want %d", len(aiComments), detailCommentEnrichLimit)
	}
	if detail["ai_comments_has_more"] != true {
		t.Fatalf("ai_comments_has_more=%v", detail["ai_comments_has_more"])
	}
	if detail["ai_comments_next_cursor"] == nil {
		t.Fatalf("expected ai_comments_next_cursor")
	}
}
