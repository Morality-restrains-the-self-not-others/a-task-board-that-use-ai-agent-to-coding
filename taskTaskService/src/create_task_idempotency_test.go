package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func resetForkCreateGuard() {
	forkCreateGuardMu.Lock()
	defer forkCreateGuardMu.Unlock()
	forkCreateGuardMap = map[string]*forkCreateClaim{}
}

func postCreateTask(t *testing.T, body string, extraHeaders map[string]string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", bytes.NewReader([]byte(body)))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("Content-Type", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	var payload map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	return rec, payload
}

func countTaskRows(t *testing.T, workspaceID string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_tasks WHERE workspace_id=?`, workspaceID).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	return count
}

func TestCreateForkDedupShortWindow(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	resetForkCreateGuard()
	prevPublish := publishTaskCreatedFn
	publishTaskCreatedFn = func(ctx context.Context, tenantID, workspaceID, taskID, title, userID, postExpiresAt string, workspaceSeq int) error {
		return nil
	}
	t.Cleanup(func() { publishTaskCreatedFn = prevPublish })

	body := `{"title":"fork hello","workspace_id":"ws1","fork_from":"src-1","task_kind":"bug-fix"}`
	rec1, p1 := postCreateTask(t, body, nil)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first create status=%d body=%s", rec1.Code, rec1.Body.String())
	}
	id1 := fmt.Sprintf("%v", p1["id"])
	if id1 == "" || id1 == "<nil>" {
		t.Fatalf("first create id=%v", p1["id"])
	}

	// 完全相同 body 的第二次 POST → 返回首次已创建任务，不新建。
	rec2, p2 := postCreateTask(t, body, nil)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("second create status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	id2 := fmt.Sprintf("%v", p2["id"])
	if id1 != id2 {
		t.Fatalf("dedup failed: id1=%s id2=%s", id1, id2)
	}
	if countTaskRows(t, "ws1") != 1 {
		t.Fatalf("expected 1 task row after dedup, got %d", countTaskRows(t, "ws1"))
	}
}

func TestCreateForkConcurrentDedup(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	resetForkCreateGuard()
	prevPublish := publishTaskCreatedFn
	publishTaskCreatedFn = func(ctx context.Context, tenantID, workspaceID, taskID, title, userID, postExpiresAt string, workspaceSeq int) error {
		return nil
	}
	t.Cleanup(func() { publishTaskCreatedFn = prevPublish })

	body := `{"title":"concurrent fork","workspace_id":"ws1","fork_from":"src-2","task_kind":"bug-fix"}`
	const n = 5
	var wg sync.WaitGroup
	ids := make([]string, n)
	barrier := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier
			rec, payload := postCreateTask(t, body, nil)
			if rec.Code != http.StatusCreated {
				t.Errorf("concurrent create status=%d body=%s", rec.Code, rec.Body.String())
				return
			}
			ids[idx] = fmt.Sprintf("%v", payload["id"])
		}(i)
	}
	close(barrier)
	wg.Wait()
	for i := 1; i < n; i++ {
		if ids[i] == "" || ids[i] != ids[0] {
			t.Fatalf("concurrent dedup failed: ids=%v", ids)
		}
	}
	if countTaskRows(t, "ws1") != 1 {
		t.Fatalf("expected 1 task row after concurrent dedup, got %d", countTaskRows(t, "ws1"))
	}
}

func TestCreateIdempotencyKeyDedup(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	resetForkCreateGuard()
	prevPublish := publishTaskCreatedFn
	publishTaskCreatedFn = func(ctx context.Context, tenantID, workspaceID, taskID, title, userID, postExpiresAt string, workspaceSeq int) error {
		return nil
	}
	t.Cleanup(func() { publishTaskCreatedFn = prevPublish })

	body := `{"title":"plain idem","workspace_id":"ws1","task_kind":"bug-fix"}`
	rec1, p1 := postCreateTask(t, body, map[string]string{"Idempotency-Key": "client-key-1"})
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first status=%d body=%s", rec1.Code, rec1.Body.String())
	}
	id1 := fmt.Sprintf("%v", p1["id"])

	rec2, p2 := postCreateTask(t, body, map[string]string{"Idempotency-Key": "client-key-1"})
	if rec2.Code != http.StatusCreated {
		t.Fatalf("second status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	if id1 != fmt.Sprintf("%v", p2["id"]) {
		t.Fatalf("idempotency-key dedup failed: id1=%s id2=%v", id1, p2["id"])
	}
	if countTaskRows(t, "ws1") != 1 {
		t.Fatalf("expected 1 task row, got %d", countTaskRows(t, "ws1"))
	}

	// 不同 Idempotency-Key → 新建。
	rec3, _ := postCreateTask(t, body, map[string]string{"Idempotency-Key": "client-key-2"})
	if rec3.Code != http.StatusCreated {
		t.Fatalf("third status=%d body=%s", rec3.Code, rec3.Body.String())
	}
	if countTaskRows(t, "ws1") != 2 {
		t.Fatalf("expected 2 task rows, got %d", countTaskRows(t, "ws1"))
	}
}
