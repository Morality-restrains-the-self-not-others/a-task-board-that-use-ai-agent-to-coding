package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// insertOrphanLoginMethod 插入一条 auth_user 不存在的孤儿 login_method，返回其 object_id。
func insertOrphanLoginMethod(t *testing.T) string {
	t.Helper()
	lmID := generateSnowflakeID()
	orphanUserID := fmt.Sprintf("%d", generateSnowflakeID()) // auth_user 中不存在
	now := "2026-01-01 00:00:00.000000"
	_, err := db.Exec(`
		INSERT INTO auth_login_method (
			id, content_type_id, object_id, method_type, identifier,
			phone_country_calling_code, password_hash, is_verified, created_at, updated_at
		) VALUES (?, ?, ?, 'phone', ?, ?, '', 0, ?, ?)`,
		lmID, cfg.UserContentTypeID, orphanUserID, "13800000000", "86", now, now)
	if err != nil {
		t.Fatalf("insert orphan login_method: %v", err)
	}
	return orphanUserID
}

func loginMethodVoided(t *testing.T, objectID string) bool {
	t.Helper()
	var voided sql.NullTime
	err := db.QueryRow(`SELECT binding_voided_at FROM auth_login_method WHERE object_id = ?`, objectID).Scan(&voided)
	if err != nil {
		t.Fatalf("select binding_voided_at: %v", err)
	}
	return voided.Valid
}

func TestVoidOrphanLoginMethodsVoidsOnlyMissingUserRows(t *testing.T) {
	setupAuthTestDB(t)

	// 孤儿行：auth_user 无对应用户。
	orphanUserID := insertOrphanLoginMethod(t)
	if loginMethodVoided(t, orphanUserID) {
		t.Fatal("orphan login_method should start non-voided")
	}

	// 活跃用户：正常注册的 phone 绑定不应被作废。
	activeUserID, err := createUserWithPhoneLogin("86", "13900139000", "password123", "")
	if err != nil {
		t.Fatalf("active user: %v", err)
	}
	if loginMethodVoided(t, activeUserID) {
		t.Fatal("active user login_method should start non-voided")
	}

	n, err := voidOrphanLoginMethods(100)
	if err != nil {
		t.Fatalf("voidOrphanLoginMethods: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected at least 1 orphan voided, got %d", n)
	}

	if !loginMethodVoided(t, orphanUserID) {
		t.Fatal("orphan login_method should be voided")
	}
	if loginMethodVoided(t, activeUserID) {
		t.Fatal("active user login_method must NOT be voided")
	}
}

func TestInternalVoidOrphanLoginMethodsEndpoint(t *testing.T) {
	setupAuthTestDB(t)
	insertOrphanLoginMethod(t)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskauth/login-methods/void-orphans/?limit=10", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	rec := httptest.NewRecorder()
	handleInternalVoidOrphanLoginMethods(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"voided":1`) {
		t.Fatalf("expected voided=1, body=%s", rec.Body.String())
	}
}
