package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// insertOidcSsoIdempotencyRow 插入一条带指定 applied_at 的幂等行，返回 idempotency_key。
func insertOidcSsoIdempotencyRow(t *testing.T, companyID, operation, key string, appliedAt time.Time) {
	t.Helper()
	ts := appliedAt.UTC().Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		INSERT INTO auth_oidc_sso_idempotency (company_id, operation, idempotency_key, applied_at)
		VALUES (?, ?, ?, ?)`, companyID, operation, key, ts)
	if err != nil {
		t.Fatalf("insert auth_oidc_sso_idempotency: %v", err)
	}
}

func oidcSsoIdempotencyRowCount(t *testing.T, companyID, operation, key string) int {
	t.Helper()
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM auth_oidc_sso_idempotency
		WHERE company_id = ? AND operation = ? AND idempotency_key = ?`,
		companyID, operation, key).Scan(&n)
	if err != nil {
		t.Fatalf("count auth_oidc_sso_idempotency: %v", err)
	}
	return n
}

// TestCleanupOidcSsoIdempotencyDeletesOnlyStaleRows 回归 OPT-20260825-030：
// 热窗口内（7d）的行必须保留，超过窗口的行按批删除。
func TestCleanupOidcSsoIdempotencyDeletesOnlyStaleRows(t *testing.T) {
	setupAuthTestDB(t)
	now := time.Now().UTC()

	staleKey := fmt.Sprintf("stale-%d", now.UnixNano())
	freshKey := fmt.Sprintf("fresh-%d", now.UnixNano())
	insertOidcSsoIdempotencyRow(t, "comp-stale", "enable", staleKey, now.AddDate(0, 0, -30))
	insertOidcSsoIdempotencyRow(t, "comp-fresh", "enable", freshKey, now.AddDate(0, 0, -1))

	n, err := cleanupOidcSsoIdempotency(7, 100)
	if err != nil {
		t.Fatalf("cleanupOidcSsoIdempotency: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 stale row deleted, got %d", n)
	}
	if got := oidcSsoIdempotencyRowCount(t, "comp-stale", "enable", staleKey); got != 0 {
		t.Fatalf("stale row should be deleted, still present %d", got)
	}
	if got := oidcSsoIdempotencyRowCount(t, "comp-fresh", "enable", freshKey); got != 1 {
		t.Fatalf("fresh row should be retained, got count %d", got)
	}
}

// TestCleanupOidcSsoIdempotencyBatchLoop 回归分批循环：超过单批 limit 时多轮删除直至整批 < limit。
func TestCleanupOidcSsoIdempotencyBatchLoop(t *testing.T) {
	setupAuthTestDB(t)
	now := time.Now().UTC()
	const total = 5
	for i := 0; i < total; i++ {
		insertOidcSsoIdempotencyRow(t, "comp-batch", "rotate", fmt.Sprintf("key-%d-%d", i, now.UnixNano()), now.AddDate(0, 0, -30))
	}
	// 单批 limit=2 → 需要 3 轮 DELETE。
	n, err := cleanupOidcSsoIdempotency(7, 2)
	if err != nil {
		t.Fatalf("cleanupOidcSsoIdempotency: %v", err)
	}
	if n != total {
		t.Fatalf("expected %d rows deleted, got %d", total, n)
	}
	var left int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_oidc_sso_idempotency WHERE company_id = 'comp-batch'`).Scan(&left); err != nil {
		t.Fatalf("count remaining: %v", err)
	}
	if left != 0 {
		t.Fatalf("expected 0 remaining, got %d", left)
	}
}

// TestInternalOidcSsoIdempotencyCleanupEndpoint 回归内部端点：
// 带内部密钥可清理、无密钥 403、响应含 deleted 计数。
func TestInternalOidcSsoIdempotencyCleanupEndpoint(t *testing.T) {
	setupAuthTestDB(t)
	now := time.Now().UTC()
	insertOidcSsoIdempotencyRow(t, "comp-endpoint", "disable", fmt.Sprintf("ep-%d", now.UnixNano()), now.AddDate(0, 0, -30))

	// 测试环境 cfg.InternalSecret 为空时 requireInternalSecret fail-open；
	// 先注入测试密钥再断言 403/200，测试结束恢复原值。
	origSecret := cfg.InternalSecret
	cfg.InternalSecret = "test-internal-secret"
	t.Cleanup(func() { cfg.InternalSecret = origSecret })

	// 无内部密钥 → 403
	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskauth/oidc-sso-idempotency/cleanup/", nil)
	rec := httptest.NewRecorder()
	handleInternalOidcSsoIdempotencyCleanup(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without secret, got %d", rec.Code)
	}

	// 带内部密钥 → 200 + deleted
	req = httptest.NewRequest(http.MethodPost, "/api/internal/taskauth/oidc-sso-idempotency/cleanup/?max_age_days=7&limit=10", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	rec = httptest.NewRecorder()
	handleInternalOidcSsoIdempotencyCleanup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if deleted, ok := body["deleted"].(float64); !ok || deleted < 1 {
		t.Fatalf("expected deleted >= 1, got %v", body["deleted"])
	}
}
