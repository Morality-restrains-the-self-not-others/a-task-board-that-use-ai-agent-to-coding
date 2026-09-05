package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func seedWechatLinkedLoginMethod(t *testing.T, userID, methodType, identifier string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO auth_login_method (id, content_type_id, object_id, method_type, identifier, password_hash, is_verified, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, '', 1, NOW(), NOW())`,
		generateSnowflakeID(), cfg.UserContentTypeID, userID, methodType, identifier); err != nil {
		t.Fatalf("seed login method: %v", err)
	}
}

func TestResolveWechatLinkedUserIDsByNicknameOpenIDUnionID(t *testing.T) {
	setupAuthTestDB(t)
	uNick := mustCreateWechatTestUser(t, "u-wa-union-nick")
	upsertWechatIdentity(uNick, "web", "wx123", "o-wa-openid-nick", "u-wa-union-nick", "微信昵称甲", "", timeNowUTC())
	uOpen := mustCreateWechatTestUser(t, "u-wa-union-open")
	upsertWechatIdentity(uOpen, "web", "wx123", "o-wa-openid-only", "u-wa-union-open", "other", "", timeNowUTC())

	ids, err := resolveWechatLinkedUserIDs("微信昵称甲")
	if err != nil {
		t.Fatalf("nickname: %v", err)
	}
	if len(ids) != 1 || ids[0] != uNick {
		t.Fatalf("nickname ids=%v want [%s]", ids, uNick)
	}

	ids, err = resolveWechatLinkedUserIDs("o-wa-openid-only")
	if err != nil {
		t.Fatalf("openid: %v", err)
	}
	if len(ids) != 1 || ids[0] != uOpen {
		t.Fatalf("openid ids=%v want [%s]", ids, uOpen)
	}

	ids, err = resolveWechatLinkedUserIDs("u-wa-union-nick")
	if err != nil {
		t.Fatalf("unionid: %v", err)
	}
	if len(ids) != 1 || ids[0] != uNick {
		t.Fatalf("unionid ids=%v want [%s]", ids, uNick)
	}
}

func TestResolveWechatLinkedUserIDsByBoundPhoneEmail(t *testing.T) {
	setupAuthTestDB(t)
	linked := mustCreateWechatTestUser(t, "u-wa-union-phone")
	upsertWechatIdentity(linked, "web", "wx123", "o-wa-phone", "u-wa-union-phone", "n1", "", timeNowUTC())
	seedWechatLinkedLoginMethod(t, linked, "phone", "13800138000")
	seedWechatLinkedLoginMethod(t, linked, "email", "wx-user@example.com")

	unlinked := mustCreateWechatTestUser(t, "u-wa-union-nophone")
	if _, err := db.Exec(`DELETE FROM wechat_identity WHERE user_id = ?`, unlinked); err != nil {
		t.Fatalf("delete identity: %v", err)
	}
	seedWechatLinkedLoginMethod(t, unlinked, "phone", "13900139000")

	ids, err := resolveWechatLinkedUserIDs("13800138000")
	if err != nil {
		t.Fatalf("phone: %v", err)
	}
	if len(ids) != 1 || ids[0] != linked {
		t.Fatalf("phone ids=%v want [%s]", ids, linked)
	}

	ids, err = resolveWechatLinkedUserIDs("wx-user@example.com")
	if err != nil {
		t.Fatalf("email: %v", err)
	}
	if len(ids) != 1 || ids[0] != linked {
		t.Fatalf("email ids=%v want [%s]", ids, linked)
	}

	ids, err = resolveWechatLinkedUserIDs("13900139000")
	if err != nil {
		t.Fatalf("unlinked phone: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("unlinked phone should miss, ids=%v", ids)
	}
}

func TestResolveWechatLinkedUserIDsMiss(t *testing.T) {
	setupAuthTestDB(t)
	ids, err := resolveWechatLinkedUserIDs("nobody-here")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids=%v", ids)
	}
}

func TestHandleInternalWechatLinkedAccount(t *testing.T) {
	setupAuthTestDB(t)
	prev := cfg.InternalSecret
	cfg.InternalSecret = "wa-secret"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	u := mustCreateWechatTestUser(t, "u-wa-http")
	upsertWechatIdentity(u, "web", "wx123", "o-wa-http", "u-wa-http", "HTTP昵称", "", timeNowUTC())

	req := httptest.NewRequest(http.MethodGet, "/api/internal/users/wechat-linked-account/?q=HTTP昵称", nil)
	rec := httptest.NewRecorder()
	handleInternalWechatLinkedAccount(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("no secret status=%d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/internal/users/wechat-linked-account/?q=HTTP昵称", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", "wa-secret")
	rec = httptest.NewRecorder()
	handleInternalWechatLinkedAccount(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	raw, _ := body["user_ids"].([]interface{})
	if len(raw) != 1 || raw[0] != u {
		t.Fatalf("user_ids=%v want [%s]", raw, u)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/internal/users/wechat-linked-account/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", "wa-secret")
	rec = httptest.NewRecorder()
	handleInternalWechatLinkedAccount(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty q status=%d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/internal/users/wechat-linked-account/?q="+strings.Repeat("你", 129), nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", "wa-secret")
	rec = httptest.NewRecorder()
	handleInternalWechatLinkedAccount(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("too long status=%d", rec.Code)
	}
}
