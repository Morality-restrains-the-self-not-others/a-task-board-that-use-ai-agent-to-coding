package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// insertExpiringTask 直接插入带指定 post_expires_at 的任务（v15 存续期模式）。
// 与既有 insertTreeTask 约定一致：传 time.Time（driver 按 loc=Local 转北京墙钟存库）。
func insertExpiringTask(t *testing.T, id, title string, expiresAt time.Time) {
	t.Helper()
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,post_expires_at,created_at,updated_at)
		VALUES(?,?,?,?,0,'medium',0,'ws1','u1','','','','','',0,'company','','','',?,?,?)`,
		id, "t1", title, "", expiresAt, now, now)
	if err != nil {
		t.Fatal(err)
	}
}

func expiredReq(method, path string, body string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.Header.Set("X-Auth-Tenant-Id", "t1")
	r.Header.Set("X-Auth-User-Id", "u1")
	r.Header.Set("X-Task-Test-Skip-Django-Validate", "1")
	r.Header.Set("X-Task-Test-Progress-Column-Name", "进行中")
	r.Header.Set("X-Task-Test-Allowed-Progress-Column-Ids", "col-wip")
	return r
}

// serveMux 走真实 mountRoutes（与生产一致的 mux 分发），端到端验证路径解析。
func serveMux(req *http.Request) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mountRoutes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestExpiredPostRefusesEditAndComment(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	insertExpiringTask(t, "exp1", "Expired", time.Now().UTC().Add(-time.Hour))

	// 编辑过期帖子 → 402 TASK_POST_EXPIRED（规范键值对路径 /api/tasks/todos/tenant_id/.../workspace_id/...）
	rec := serveMux(expiredReq(http.MethodPatch, "/api/tasks/todos/tenant_id/t1/workspace_id/ws1/exp1/", `{"title":"try"}`))
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("edit: expected 402, got %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	if body["code"] != "TASK_POST_EXPIRED" {
		t.Fatalf("edit code=%v", body["code"])
	}

	// 评论过期帖子 → 402
	rec = serveMux(expiredReq(http.MethodPost, "/api/tenant/t1/tasks/exp1/comments/", `{"content":"hi"}`))
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("comment: expected 402, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestRenewExpiredPostExtendsTwelveMonths(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	expiredAt := time.Now().UTC().Add(-time.Hour)
	insertExpiringTask(t, "exp2", "Expired", expiredAt)

	// 规范路径: POST /api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/{taskId}/renew/
	rec := serveMux(expiredReq(http.MethodPost, "/api/tasks/todos/tenant_id/t1/workspace_id/ws1/exp2/renew/", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("renew: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&payload)
	if payload["post_expired"] == true {
		t.Fatalf("renew: post still expired: %v", payload["post_expires_at"])
	}
	expiresAtRaw, _ := payload["post_expires_at"].(string)
	if expiresAtRaw == "" {
		t.Fatalf("renew: missing post_expires_at: %v", payload)
	}
	newExpiry, err := time.Parse(time.RFC3339Nano, expiresAtRaw)
	if err != nil {
		t.Fatalf("renew: bad expires_at %q: %v", expiresAtRaw, err)
	}
	// 过期帖续存 = now + 12 个月（过期日早于 now，不采用）
	expectMin := time.Now().UTC().AddDate(0, 12, 0).Add(-time.Hour)
	expectMax := time.Now().UTC().AddDate(0, 12, 0).Add(time.Hour)
	if newExpiry.Before(expectMin) || newExpiry.After(expectMax) {
		t.Fatalf("renew: expiry=%v not within +12M window", newExpiry)
	}

	// 续存后编辑恢复可用
	rec = serveMux(expiredReq(http.MethodPatch, "/api/tasks/todos/tenant_id/t1/workspace_id/ws1/exp2/", `{"title":"ok"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("edit after renew: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
}

// TestRenewLegacyPathCompat — 存量位置参数路径（/api/tenant/{tid}/workspace/{wid}/todos/{taskId}/renew/）仍可续存（存量兼容）。
func TestRenewLegacyPathCompat(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	insertExpiringTask(t, "exp3", "ExpiredLegacy", time.Now().UTC().Add(-time.Hour))

	rec := serveMux(expiredReq(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/exp3/renew/", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("legacy renew: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&payload)
	if payload["post_expired"] == true {
		t.Fatalf("legacy renew: post still expired: %v", payload["post_expires_at"])
	}
}

func TestActivePostShowsExpiryFields(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	future := time.Now().UTC().AddDate(0, 12, 0)
	insertExpiringTask(t, "act1", "Active", future)

	rec := serveMux(expiredReq(http.MethodGet, "/api/tasks/todos/tenant_id/t1/workspace_id/ws1/act1/", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&payload)
	if payload["post_expires_at"] == nil {
		t.Fatalf("get: missing post_expires_at")
	}
	if payload["post_expired"] != false {
		t.Fatalf("get: post_expired=%v", payload["post_expired"])
	}
}

func TestExpirePostsScanPublishesExpiredOnly(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	insertExpiringTask(t, "exp-scan-1", "ExpiredScan", time.Now().UTC().Add(-2*time.Hour))
	insertExpiringTask(t, "exp-scan-2", "ActiveScan", time.Now().UTC().AddDate(0, 1, 0))

	published := []string{}
	publishTaskPostExpiredFn = func(ctx context.Context, tenantID, workspaceID, taskID string) error {
		published = append(published, taskID)
		return nil
	}
	defer func() { publishTaskPostExpiredFn = publishTaskPostExpired }()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/internal/tasks/expire-posts/", nil)
	req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	handleInternalExpirePosts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expire-posts: expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	if body["expired_count"] != float64(1) {
		t.Fatalf("expired_count=%v, want 1 (only the expired post)", body["expired_count"])
	}
	if len(published) != 1 || published[0] != "exp-scan-1" {
		t.Fatalf("published=%v, want [exp-scan-1]", published)
	}
}
