package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// unarchiveSystemAdminUser 以 bootstrap-admin 调用 PATCH/POST 解档端点。
func unarchiveSystemAdminUser(t *testing.T, userID string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/users/"+userID+"/unarchive/", nil)
	req.Header.Set("X-User-Id", "bootstrap-admin")
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	return rec.Code, rec.Body.String()
}

func archiveUserByID(t *testing.T, userID string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE auth_user SET is_archived = 1 WHERE id = ?`, userID); err != nil {
		t.Fatalf("archive user %s: %v", userID, err)
	}
}

// OPT-20260824-089: 解档时若该用户仍持有的未作废标识已被活跃用户占用，应 409。
// 模拟系统管理端直接建号场景（不走注册回收逻辑）：归档用户 A 的手机号未被作废，
// 活跃用户 B 也绑同一手机号 → 解档 A 会恢复双活绑定，须拒绝。
func TestSystemAdminUnarchiveConflictsWithActiveUserPhone(t *testing.T) {
	setupAuthTestDB(t)

	userA, err := createUserWithPhoneLogin("86", "13800138000", "password123", "")
	if err != nil {
		t.Fatalf("userA: %v", err)
	}
	archiveUserByID(t, userA)

	// 活跃用户 B 绑同一手机号（未走 voidStale 回收，A 的绑定仍未作废）。
	userB, err := createUserWithPhoneLogin("86", "13800138000", "password456", "")
	if err != nil {
		t.Fatalf("userB: %v", err)
	}
	if userB == userA {
		t.Fatal("userB should differ from userA")
	}

	code, body := unarchiveSystemAdminUser(t, userA)
	if code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d body=%s", code, body)
	}
	if !strings.Contains(body, "手机号") || !strings.Contains(body, "13800138000") {
		t.Fatalf("conflict message should mention 手机号 and identifier, got %s", body)
	}
}

// 无冲突时解档应 200 并清除归档标记。
func TestSystemAdminUnarchiveNoConflictSucceeds(t *testing.T) {
	setupAuthTestDB(t)

	userA, err := createUserWithPhoneLogin("86", "13900139000", "password123", "")
	if err != nil {
		t.Fatalf("userA: %v", err)
	}
	archiveUserByID(t, userA)

	// 活跃用户 B 绑不同手机号，不构成冲突。
	if _, err := createUserWithPhoneLogin("86", "13800138000", "password456", ""); err != nil {
		t.Fatalf("userB: %v", err)
	}

	code, body := unarchiveSystemAdminUser(t, userA)
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, body)
	}
	var archived int
	if err := db.QueryRow(`SELECT is_archived FROM auth_user WHERE id = ?`, userA).Scan(&archived); err != nil {
		t.Fatalf("select is_archived: %v", err)
	}
	if archived != 0 {
		t.Fatalf("expected is_archived=0 after unarchive, got %d", archived)
	}
}

// 冲突标识被作废（如注册回收路径已 voidStale）后，解档应放行。
func TestSystemAdminUnarchiveAfterVoidedConflictSucceeds(t *testing.T) {
	setupAuthTestDB(t)

	userA, err := createUserWithPhoneLogin("86", "13800138000", "password123", "")
	if err != nil {
		t.Fatalf("userA: %v", err)
	}
	archiveUserByID(t, userA)

	// 活跃用户 B 注册同号，走真实回收逻辑：voidStalePhoneLoginMethods 作废 A 的绑定。
	if _, err := createUserWithPhoneLogin("86", "13800138000", "password456", ""); err != nil {
		t.Fatalf("userB: %v", err)
	}
	if _, err := voidStalePhoneLoginMethods("86", "13800138000"); err != nil {
		t.Fatalf("voidStale: %v", err)
	}

	code, body := unarchiveSystemAdminUser(t, userA)
	if code != http.StatusOK {
		t.Fatalf("expected 200 after voiding conflict, got %d body=%s", code, body)
	}
}
