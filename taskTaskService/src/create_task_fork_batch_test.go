package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// TestCreateForkBatchCreatesN 单请求批量派生 N 个副本：201 + ids/created_count/first_id，
// 库内恰好 N 行且均带 fork_from。
func TestCreateForkBatchCreatesN(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	resetForkCreateGuard()
	prevPublish := publishTaskCreatedFn
	publishTaskCreatedFn = func(ctx context.Context, tenantID, workspaceID, taskID, title, userID, postExpiresAt string, workspaceSeq int) error {
		return nil
	}
	t.Cleanup(func() { publishTaskCreatedFn = prevPublish })

	body := `{"title":"fork batch","workspace_id":"ws1","fork_from":"src-batch","task_kind":"bug-fix","fork_count":3}`
	rec, payload := postCreateTask(t, body, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("batch create status=%d body=%s", rec.Code, rec.Body.String())
	}
	ids, createdCount := batchResult(payload)
	if createdCount != 3 {
		t.Fatalf("created_count=%d want 3 (payload=%s)", createdCount, payloadJSON(payload))
	}
	if len(ids) != 3 {
		t.Fatalf("ids len=%d want 3", len(ids))
	}
	if firstID, _ := payload["first_id"].(string); firstID != ids[0] {
		t.Fatalf("first_id=%v want ids[0]=%s", payload["first_id"], ids[0])
	}
	// ids 必须互不相同。
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate task id in batch: %s", id)
		}
		seen[id] = true
	}
	if got := countTaskRows(t, "ws1"); got != 3 {
		t.Fatalf("task rows=%d want 3", got)
	}
	var forkCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM task_tasks WHERE workspace_id='ws1' AND fork_from_id='src-batch'`).Scan(&forkCount); err != nil {
		t.Fatalf("fork_from count: %v", err)
	}
	if forkCount != 3 {
		t.Fatalf("fork_from rows=%d want 3", forkCount)
	}
}

// TestCreateForkBatchIdempotentRetry 同一批请求（相同 Idempotency-Key）重试时返回既有 ids，
// 不再重复创建/扣配额。
func TestCreateForkBatchIdempotentRetry(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	resetForkCreateGuard()
	prevPublish := publishTaskCreatedFn
	publishTaskCreatedFn = func(ctx context.Context, tenantID, workspaceID, taskID, title, userID, postExpiresAt string, workspaceSeq int) error {
		return nil
	}
	t.Cleanup(func() { publishTaskCreatedFn = prevPublish })

	body := `{"title":"fork batch retry","workspace_id":"ws1","fork_from":"src-retry","task_kind":"bug-fix","fork_count":3}`
	headers := map[string]string{"Idempotency-Key": "batch-key-retry"}
	rec1, p1 := postCreateTask(t, body, headers)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first batch status=%d body=%s", rec1.Code, rec1.Body.String())
	}
	ids1, _ := batchResult(p1)

	// 完全相同的第二次请求（相同 Idempotency-Key）→ 返回相同 ids。
	rec2, p2 := postCreateTask(t, body, headers)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("retry batch status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	ids2, createdCount2 := batchResult(p2)
	if createdCount2 != 3 {
		t.Fatalf("retry created_count=%d want 3", createdCount2)
	}
	if len(ids1) != len(ids2) {
		t.Fatalf("ids length mismatch: %v vs %v", ids1, ids2)
	}
	for i := range ids1 {
		if ids1[i] != ids2[i] {
			t.Fatalf("retry ids mismatch at %d: %s vs %s", i, ids1[i], ids2[i])
		}
	}
	if got := countTaskRows(t, "ws1"); got != 3 {
		t.Fatalf("task rows after retry=%d want 3 (no duplicate)", got)
	}
}

// TestCreateForkBatchPartialQuotaFailure 配额在第 3 副本耗尽：返回 200 partial + created_count=2，
// 库内仅 2 行，且首副本即失败时走错误状态。
func TestCreateForkBatchPartialQuotaFailure(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	resetForkCreateGuard()
	prevPublish := publishTaskCreatedFn
	publishTaskCreatedFn = func(ctx context.Context, tenantID, workspaceID, taskID, title, userID, postExpiresAt string, workspaceSeq int) error {
		return nil
	}
	t.Cleanup(func() { publishTaskCreatedFn = prevPublish })

	prevQuota := consumeTaskPostQuota
	var calls int32
	consumeTaskPostQuota = func(tenantID, taskID, workspaceID, userID, projectID string) (string, error) {
		if atomic.AddInt32(&calls, 1) >= 3 {
			return "", &insufficientBalanceError{Message: "任务帖配额不足", BalancePoints: 0, RequiredPoints: 1}
		}
		return time.Now().UTC().AddDate(0, 12, 0).Format("2006-01-02 15:04:05.000000"), nil
	}
	t.Cleanup(func() { consumeTaskPostQuota = prevQuota })

	body := `{"title":"fork batch quota","workspace_id":"ws1","fork_from":"src-quota","task_kind":"bug-fix","fork_count":3}`
	rec, payload := postCreateTask(t, body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("partial batch status=%d want 200 body=%s", rec.Code, rec.Body.String())
	}
	ids, createdCount := batchResult(payload)
	if createdCount != 2 {
		t.Fatalf("partial created_count=%d want 2 (payload=%s)", createdCount, payloadJSON(payload))
	}
	if len(ids) != 2 {
		t.Fatalf("partial ids len=%d want 2", len(ids))
	}
	if partial, _ := payload["partial"].(bool); !partial {
		t.Fatalf("partial flag=false want true")
	}
	errObj, ok := payload["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("partial error missing: %s", payloadJSON(payload))
	}
	if code, _ := errObj["code"].(string); code != "INSUFFICIENT_TASK_POST_QUOTA" {
		t.Fatalf("partial error.code=%v want INSUFFICIENT_TASK_POST_QUOTA", errObj["code"])
	}
	if got := countTaskRows(t, "ws1"); got != 2 {
		t.Fatalf("task rows after partial=%d want 2", got)
	}
}

// TestCreateForkBatchFirstCopyFails 首副本即配额不足：返回 402 而非 partial。
func TestCreateForkBatchFirstCopyFails(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	resetForkCreateGuard()
	prevPublish := publishTaskCreatedFn
	publishTaskCreatedFn = func(ctx context.Context, tenantID, workspaceID, taskID, title, userID, postExpiresAt string, workspaceSeq int) error {
		return nil
	}
	t.Cleanup(func() { publishTaskCreatedFn = prevPublish })

	prevQuota := consumeTaskPostQuota
	consumeTaskPostQuota = func(tenantID, taskID, workspaceID, userID, projectID string) (string, error) {
		return "", &insufficientBalanceError{Message: "任务帖配额不足", BalancePoints: 0, RequiredPoints: 1}
	}
	t.Cleanup(func() { consumeTaskPostQuota = prevQuota })

	body := `{"title":"fork batch quota0","workspace_id":"ws1","fork_from":"src-quota0","task_kind":"bug-fix","fork_count":3}`
	rec, payload := postCreateTask(t, body, nil)
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("first-fail status=%d want 402 body=%s", rec.Code, rec.Body.String())
	}
	if code, _ := payload["code"].(string); code != "INSUFFICIENT_TASK_POST_QUOTA" {
		t.Fatalf("first-fail code=%v want INSUFFICIENT_TASK_POST_QUOTA", payload["code"])
	}
	if got := countTaskRows(t, "ws1"); got != 0 {
		t.Fatalf("task rows=%d want 0", got)
	}
}

// batchResult 从批量响应中提取 ids 与 created_count。
func batchResult(payload map[string]interface{}) (ids []string, createdCount int) {
	if raw, ok := payload["ids"].([]interface{}); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				ids = append(ids, s)
			}
		}
	}
	createdCount = intOf(payload["created_count"])
	return ids, createdCount
}

func intOf(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}

func payloadJSON(payload map[string]interface{}) string {
	raw, _ := json.Marshal(payload)
	return string(raw)
}
