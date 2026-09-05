package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

func TestNextWorkspaceSeqIncrements(t *testing.T) {
	setupTestDB(t)
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	s1, err := nextWorkspaceSeq(tx, "t1", "ws1")
	if err != nil {
		t.Fatal(err)
	}
	s2, err := nextWorkspaceSeq(tx, "t1", "ws1")
	if err != nil {
		t.Fatal(err)
	}
	sOther, err := nextWorkspaceSeq(tx, "t1", "ws2")
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if s1 != 1 || s2 != 2 {
		t.Fatalf("ws1 seq=%d,%d want 1,2", s1, s2)
	}
	if sOther != 1 {
		t.Fatalf("ws2 seq=%d want 1", sOther)
	}
}

func TestNextWorkspaceSeqConcurrentNoDup(t *testing.T) {
	setupTestDB(t)
	const n = 8
	got := make([]int, n)
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			tx, err := db.Begin()
			if err != nil {
				errCh <- err
				return
			}
			seq, err := nextWorkspaceSeq(tx, "t1", "ws-conc")
			if err != nil {
				_ = tx.Rollback()
				errCh <- err
				return
			}
			if err := tx.Commit(); err != nil {
				errCh <- err
				return
			}
			got[i] = seq
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	for _, s := range got {
		if s < 1 || s > n {
			t.Fatalf("seq out of range: %d", s)
		}
		if seen[s] {
			t.Fatalf("duplicate seq %d", s)
		}
		seen[s] = true
	}
}

func TestCreateTasksAssignsWorkspaceSeq(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	var published []map[string]interface{}
	prev := publishTaskCreatedFn
	publishTaskCreatedFn = func(ctx context.Context, tenantID, workspaceID, taskID, title, userID, postExpiresAt string, workspaceSeq int) error {
		published = append(published, map[string]interface{}{
			"task_id": taskID, "workspace_seq": workspaceSeq, "workspace_id": workspaceID,
		})
		return nil
	}
	t.Cleanup(func() { publishTaskCreatedFn = prev })

	create := func() map[string]interface{} {
		t.Helper()
		body := `{"title":"Seq","workspace_id":"ws1","workspace_seq":99}`
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
		return created
	}

	a := create()
	b := create()
	c := create()
	if jsonNumber(a["workspace_seq"]) != 1 || jsonNumber(b["workspace_seq"]) != 2 || jsonNumber(c["workspace_seq"]) != 3 {
		t.Fatalf("seq=%v,%v,%v want 1,2,3", a["workspace_seq"], b["workspace_seq"], c["workspace_seq"])
	}
	if len(published) != 3 || jsonNumber(published[1]["workspace_seq"]) != 2 {
		t.Fatalf("TASK_CREATED payload seq=%#v", published)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/search/?q="+url.QueryEscape("#2")+"&workspace_id=ws1&limit=20", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleSearchTasks(rec, req, "t1")
	if rec.Code != http.StatusOK {
		t.Fatalf("search: %d %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, hit := range payload.Results {
		if hit["id"] == b["id"] {
			found = true
			if jsonNumber(hit["workspace_seq"]) != 2 {
				t.Fatalf("search hit seq=%v", hit["workspace_seq"])
			}
		}
	}
	if !found {
		t.Fatalf("search #2 missing id=%v in %#v", b["id"], payload.Results)
	}

	bodyWS2 := `{"title":"OtherWS","workspace_id":"ws2"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws2/todos/", strings.NewReader(bodyWS2))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("create ws2: %d %s", rec2.Code, rec2.Body.String())
	}
	var createdWS2 map[string]interface{}
	if err := json.NewDecoder(rec2.Body).Decode(&createdWS2); err != nil {
		t.Fatal(err)
	}
	if jsonNumber(createdWS2["workspace_seq"]) != 1 {
		t.Fatalf("ws2 seq=%v want 1", createdWS2["workspace_seq"])
	}
	reqCross := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/tasks/search/?q="+url.QueryEscape("#1")+"&workspace_id=ws2&limit=20", nil)
	reqCross.Header.Set("X-Auth-Tenant-Id", "t1")
	reqCross.Header.Set("X-Auth-User-Id", "u1")
	recCross := httptest.NewRecorder()
	handleSearchTasks(recCross, reqCross, "t1")
	if recCross.Code != http.StatusOK {
		t.Fatalf("search ws2: %d %s", recCross.Code, recCross.Body.String())
	}
	var cross struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.NewDecoder(recCross.Body).Decode(&cross); err != nil {
		t.Fatal(err)
	}
	for _, hit := range cross.Results {
		if hit["id"] == a["id"] {
			t.Fatalf("ws2 search leaked ws1 task %v", a["id"])
		}
		if hit["id"] == createdWS2["id"] && jsonNumber(hit["workspace_seq"]) != 1 {
			t.Fatalf("ws2 #1 seq=%v", hit["workspace_seq"])
		}
	}
}

func jsonNumber(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}
