package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// insertMpSubscribePendingRow 插入一条带指定 created_at 的服务号挂起行。
func insertMpSubscribePendingRow(t *testing.T, unionID string, createdAt time.Time) {
	t.Helper()
	ts := createdAt.UTC().Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		INSERT INTO auth_wechat_mp_subscribe_pending (unionid, openid, app_id, created_at)
		VALUES (?, '', '', ?)`, unionID, ts)
	if err != nil {
		t.Fatalf("insert auth_wechat_mp_subscribe_pending: %v", err)
	}
}

// insertMpFollowTicketRow 插入一条带指定 expire_at 的票据行。
func insertMpFollowTicketRow(t *testing.T, id int64, userID int64, expireAt time.Time) {
	t.Helper()
	exp := expireAt.UTC().Format("2006-01-02 15:04:05.000000")
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		INSERT INTO auth_wechat_mp_follow_ticket
			(id, user_id, status, wechat_ticket, expire_at, conflict_code, mp_openid, unionid, created_at, updated_at)
		VALUES (?, ?, 'pending', 'wx', ?, '', '', '', ?, ?)`, id, userID, exp, now, now)
	if err != nil {
		t.Fatalf("insert auth_wechat_mp_follow_ticket: %v", err)
	}
}

func mpPendingRowCount(t *testing.T, unionID string) int {
	t.Helper()
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM auth_wechat_mp_subscribe_pending WHERE unionid = ?`, unionID).Scan(&n)
	if err != nil {
		t.Fatalf("count auth_wechat_mp_subscribe_pending: %v", err)
	}
	return n
}

func mpTicketRowCount(t *testing.T, id int64) int {
	t.Helper()
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM auth_wechat_mp_follow_ticket WHERE id = ?`, id).Scan(&n)
	if err != nil {
		t.Fatalf("count auth_wechat_mp_follow_ticket: %v", err)
	}
	return n
}

// TestCleanupWechatMpStaleRowsEmptyTable 回归 OPT-20260826-003：空表清理为 0 不报错。
func TestCleanupWechatMpStaleRowsEmptyTable(t *testing.T) {
	setupAuthTestDB(t)
	pending, tickets, err := cleanupWechatMpStaleRows(30, 7, 100)
	if err != nil {
		t.Fatalf("cleanupWechatMpStaleRows: %v", err)
	}
	if pending != 0 || tickets != 0 {
		t.Fatalf("expected 0/0 on empty table, got %d/%d", pending, tickets)
	}
}

// TestCleanupWechatMpStaleRowsDeletesOnlyStaleRows 回归：挂起行按 created_at 30d、
// 票据行按 expire_at 7d 清理，新鲜行保留。
func TestCleanupWechatMpStaleRowsDeletesOnlyStaleRows(t *testing.T) {
	setupAuthTestDB(t)
	now := time.Now().UTC()

	stalePending := "stale-pending-" + now.Format("150405.000000000")
	freshPending := "fresh-pending-" + now.Format("150405.000000000")
	insertMpSubscribePendingRow(t, stalePending, now.AddDate(0, 0, -31))
	insertMpSubscribePendingRow(t, freshPending, now.AddDate(0, 0, -1))

	staleTicket := int64(9000001)
	freshTicket := int64(9000002)
	insertMpFollowTicketRow(t, staleTicket, 1001, now.AddDate(0, 0, -8))
	insertMpFollowTicketRow(t, freshTicket, 1002, now.AddDate(0, 0, -1))

	pending, tickets, err := cleanupWechatMpStaleRows(30, 7, 100)
	if err != nil {
		t.Fatalf("cleanupWechatMpStaleRows: %v", err)
	}
	if pending != 1 {
		t.Fatalf("expected 1 stale pending deleted, got %d", pending)
	}
	if tickets != 1 {
		t.Fatalf("expected 1 stale ticket deleted, got %d", tickets)
	}
	if got := mpPendingRowCount(t, stalePending); got != 0 {
		t.Fatalf("stale pending should be deleted, still present %d", got)
	}
	if got := mpPendingRowCount(t, freshPending); got != 1 {
		t.Fatalf("fresh pending should be retained, got %d", got)
	}
	if got := mpTicketRowCount(t, staleTicket); got != 0 {
		t.Fatalf("stale ticket should be deleted, still present %d", got)
	}
	if got := mpTicketRowCount(t, freshTicket); got != 1 {
		t.Fatalf("fresh ticket should be retained, got %d", got)
	}
}

// TestCleanupWechatMpStaleRowsBatchLoop 回归分批循环：单批 limit 小则多轮删除至整批 < limit。
func TestCleanupWechatMpStaleRowsBatchLoop(t *testing.T) {
	setupAuthTestDB(t)
	now := time.Now().UTC()
	// 6 条超期挂起行，limit=2 → 3 轮。
	for i := 0; i < 6; i++ {
		insertMpSubscribePendingRow(t, "batch-pending-"+now.Format("150405.000000000")+"-"+string(rune('a'+i)), now.AddDate(0, 0, -40))
	}
	pending, tickets, err := cleanupWechatMpStaleRows(30, 7, 2)
	if err != nil {
		t.Fatalf("cleanupWechatMpStaleRows: %v", err)
	}
	if pending != 6 {
		t.Fatalf("expected 6 batch pending deleted, got %d", pending)
	}
	if tickets != 0 {
		t.Fatalf("expected 0 tickets deleted, got %d", tickets)
	}
}

// TestInternalWechatMpCleanupEndpoint 回归内部端点：带内部密钥可清理、无密钥 403、
// 响应含 pending_deleted / ticket_deleted 计数。
func TestInternalWechatMpCleanupEndpoint(t *testing.T) {
	setupAuthTestDB(t)
	now := time.Now().UTC()
	insertMpSubscribePendingRow(t, "ep-pending-"+now.Format("150405.000000000"), now.AddDate(0, 0, -31))
	insertMpFollowTicketRow(t, 9000010, 2001, now.AddDate(0, 0, -8))

	origSecret := cfg.InternalSecret
	cfg.InternalSecret = "test-internal-secret"
	t.Cleanup(func() { cfg.InternalSecret = origSecret })

	// 无内部密钥 → 403
	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskauth/wechat-mp-cleanup/", nil)
	rec := httptest.NewRecorder()
	handleInternalWechatMpCleanup(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without secret, got %d", rec.Code)
	}

	// 带内部密钥 → 200 + 计数
	req = httptest.NewRequest(http.MethodPost, "/api/internal/taskauth/wechat-mp-cleanup/?pending_max_age_days=30&ticket_max_age_days=7&limit=10", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	rec = httptest.NewRecorder()
	handleInternalWechatMpCleanup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if deleted, ok := body["pending_deleted"].(float64); !ok || deleted < 1 {
		t.Fatalf("expected pending_deleted >= 1, got %v", body["pending_deleted"])
	}
	if deleted, ok := body["ticket_deleted"].(float64); !ok || deleted < 1 {
		t.Fatalf("expected ticket_deleted >= 1, got %v", body["ticket_deleted"])
	}
}
