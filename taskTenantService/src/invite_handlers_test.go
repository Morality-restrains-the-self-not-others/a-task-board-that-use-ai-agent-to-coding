package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInviteLink(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","company_member_name":"New Member","role":"member","workspace_id":"ws1","expiration_days":7}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	token, _ := out["invite_token"].(string)
	if token == "" {
		t.Fatal("expected invite_token in response")
	}
	if out["message"] != "邀请链接已生成" {
		t.Fatalf("unexpected message: %v", out)
	}
}

func TestInviteExpirationDaysMaxAccepted(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","company_member_name":"Year","role":"member","workspace_id":"ws1","expiration_days":365}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["expiration_days"] != float64(365) {
		t.Fatalf("expiration_days=%v", out["expiration_days"])
	}
}

func TestInviteExpirationDaysOverMaxRejected(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","company_member_name":"TooLong","role":"member","workspace_id":"ws1","expiration_days":366}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInviteEmail(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"email","email":"test@example.com","company_member_name":"Email User","role":"admin","workspace_id":"ws1","expiration_days":14,"message":"Welcome!"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["invite_token"] == nil {
		t.Fatal("expected invite_token")
	}
}

func TestInviteEmailUnsubscribedSkipsSend(t *testing.T) {
	mux := setupTestService(t)
	prev := checkEmailUnsubscribedFn
	checkEmailUnsubscribedFn = func(string) (bool, string) { return true, "http://example/unsub" }
	t.Cleanup(func() { checkEmailUnsubscribedFn = prev })
	body := `{"invite_method":"email","email":"optout@example.com","company_member_name":"Out","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["email_skipped"] != true {
		t.Fatalf("expected email_skipped: %v", out)
	}
	if out["code"] != "email_unsubscribed" {
		t.Fatalf("code=%v", out["code"])
	}
	if out["invitation_url"] == nil || out["invitation_url"] == "" {
		t.Fatal("expected invitation_url")
	}
	msg, _ := out["message"].(string)
	if !strings.Contains(msg, "手动复制") {
		t.Fatalf("message=%v", out["message"])
	}
}

func TestInviteMissingWorkspace(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","company_member_name":"No WS","role":"member"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing workspace, got %d", rec.Code)
	}
}

func TestInviteMissingEmailForEmailMethod(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"email","company_member_name":"No Email","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing email, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInviteInvalidMethod(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"sms","company_member_name":"Bad Method","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for invalid method, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInviteNonAdmin(t *testing.T) {
	mux := setupTestService(t)
	_, _ = db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active) VALUES ('m8','u7','c1',0,1)`)

	body := `{"invite_method":"link","company_member_name":"Test","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "u7")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("expected 403 for non-admin invite, got %d", rec.Code)
	}
}

func TestValidateAndJoin(t *testing.T) {
	mux := setupTestService(t)
	// Create invite as admin
	body := `{"invite_method":"link","company_member_name":"Joiner","role":"member","workspace_id":"ws1","expiration_days":7}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	var inviteOut map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &inviteOut)
	token, _ := inviteOut["invite_token"].(string)

	// Validate
	req2 := httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/validate-invite/?token="+token, nil)
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("expected 200 for validate, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var valOut map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &valOut)
	if valOut["valid"] != true {
		t.Fatalf("expected valid=true, got %v", valOut)
	}

	// Join
	joinBody := `{"token":"` + token + `"}`
	req3 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/join/", bytes.NewBufferString(joinBody)), "newuser")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 201 {
		t.Fatalf("expected 201 for join, got %d body=%s", rec3.Code, rec3.Body.String())
	}

	// Re-join should fail
	req4 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/join/", bytes.NewBufferString(joinBody)), "newuser")
	rec4 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec4, req4)
	if rec4.Code != 400 {
		t.Fatalf("expected 400 for re-join, got %d body=%s", rec4.Code, rec4.Body.String())
	}
}

func TestValidateInviteInvalidToken(t *testing.T) {
	mux := setupTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/validate-invite/?token=invalid", nil)
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for invalid token, got %d", rec.Code)
	}
}

func TestValidateInviteMissingToken(t *testing.T) {
	mux := setupTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/validate-invite/", nil)
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing token, got %d", rec.Code)
	}
}

func TestPendingInvitations(t *testing.T) {
	mux := setupTestService(t)
	// Create an invite
	body := `{"invite_method":"link","company_member_name":"Pending","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)

	// List pending invitations
	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var list []interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &list)
	if len(list) < 1 {
		t.Fatalf("expected at least 1 pending invitation, got %v", string(rec2.Body.Bytes()))
	}

	// Non-member cannot list pending (v63: authz 403, 原 400 已由权限判定替代)
	req3 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "stranger")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 403 {
		t.Fatalf("expected 403 for non-member, got %d body=%s", rec3.Code, rec3.Body.String())
	}
}

func TestRevokeInvitation(t *testing.T) {
	mux := setupTestService(t)
	// Create invite
	body := `{"invite_method":"link","company_member_name":"ToRevoke","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)

	// Get pending list to find invite ID
	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	var list []interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &list)
	if len(list) == 0 {
		t.Fatal("no pending invitation found")
	}
	inv, _ := list[0].(map[string]interface{})
	invID, _ := inv["id"].(string)

	// Revoke
	req3 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/"+invID+"/revoke-invitation/", nil), "admin1")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec3.Code, rec3.Body.String())
	}

	// Non-existent invite
	req4 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/nonexist/revoke-invitation/", nil), "admin1")
	rec4 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec4, req4)
	if rec4.Code != 404 {
		t.Fatalf("expected 404 for non-existent, got %d", rec4.Code)
	}
}

func TestResendInvitation(t *testing.T) {
	mux := setupTestService(t)
	// Create invite
	body := `{"invite_method":"link","company_member_name":"ToResend","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	oldToken := out["invite_token"].(string)

	// Get invite ID from pending list
	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	var list []interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &list)
	inv, _ := list[0].(map[string]interface{})
	invID, _ := inv["id"].(string)

	// Resend
	resendBody := `{"expiration_days":14}`
	req3 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/"+invID+"/resend-invitation-link/", bytes.NewBufferString(resendBody)), "admin1")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec3.Code, rec3.Body.String())
	}
	var resendOut map[string]interface{}
	_ = json.Unmarshal(rec3.Body.Bytes(), &resendOut)
	newToken := resendOut["invite_token"].(string)
	if newToken == oldToken {
		t.Fatal("expected new token after resend")
	}
	if !strings.Contains(resendOut["message"].(string), "重新生成") {
		t.Fatalf("expected resend message, got %v", resendOut)
	}

	// Resend with invalid expiration_days
	badBody := `{"expiration_days":0}`
	req4 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/"+invID+"/resend-invitation-link/", bytes.NewBufferString(badBody)), "admin1")
	rec4 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec4, req4)
	if rec4.Code != 400 {
		t.Fatalf("expected 400, got %d", rec4.Code)
	}
}

// 投递状态：email/phone 邀请初始为 queued（事件已发布待异步投递）
func TestInviteEmailDeliveryStatusQueued(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"email","email":"deliver@example.com","company_member_name":"D","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("invite %d body=%s", rec.Code, rec.Body.String())
	}

	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	var list []interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &list)
	if len(list) < 1 {
		t.Fatalf("no pending invites: %s", rec2.Body.String())
	}
	inv := list[0].(map[string]interface{})
	if inv["delivery_status"] != "queued" {
		t.Fatalf("expected delivery_status=queued, got %v", inv["delivery_status"])
	}
}

// 投递状态：link 邀请无投递动作（none）
func TestInviteLinkDeliveryStatusNone(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"link","company_member_name":"L","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("invite %d body=%s", rec.Code, rec.Body.String())
	}
	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	var list []interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &list)
	inv := list[0].(map[string]interface{})
	if inv["delivery_status"] != "none" {
		t.Fatalf("expected delivery_status=none, got %v", inv["delivery_status"])
	}
}

// companyNameForEvent：从 tenant_company 取显示名；缺行时回退 companyID
func TestCompanyNameForEvent(t *testing.T) {
	setupTestService(t) // 已 seed tenant_company c1 → 'Test Co'
	if got := companyNameForEvent("c1"); got != "Test Co" {
		t.Fatalf("companyNameForEvent(c1)=%q want 'Test Co'", got)
	}
	if got := companyNameForEvent("no-such-company"); got != "no-such-company" {
		t.Fatalf("companyNameForEvent(missing)=%q want fallback to companyID", got)
	}
}

// 回调端点：delivered / failed 状态回写，非法状态 400
func TestInvitationDeliveryCallback(t *testing.T) {
	mux := setupTestService(t)
	body := `{"invite_method":"email","email":"cb@example.com","company_member_name":"C","workspace_id":"ws1"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("invite %d body=%s", rec.Code, rec.Body.String())
	}

	// 取邀请 ID（pending 列表）
	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	var list []interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &list)
	inv := list[0].(map[string]interface{})
	invID := inv["id"].(string)

	// 回调 delivered
	cbBody := `{"status":"delivered"}`
	req3 := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/invitations/delivery-callback/"+invID+"/", bytes.NewBufferString(cbBody))
	rec3 := httptest.NewRecorder()
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("callback delivered %d body=%s", rec3.Code, rec3.Body.String())
	}

	// pending 列表应反映 delivered
	req4 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "admin1")
	rec4 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec4, req4)
	_ = json.Unmarshal(rec4.Body.Bytes(), &list)
	inv = list[0].(map[string]interface{})
	if inv["delivery_status"] != "delivered" {
		t.Fatalf("expected delivered after callback, got %v", inv["delivery_status"])
	}
	// OPT-20260809-027：delivered 回调后 pending 列表带出 email_sent_at（非空投递时间）
	if sentAt, _ := inv["email_sent_at"].(string); sentAt == "" {
		t.Fatalf("expected email_sent_at after delivered callback, got %q", inv["email_sent_at"])
	}
	_, _ = db.Exec(`UPDATE tenant_invitation SET delivery_status='queued', delivery_error=NULL WHERE id=?`, invID)

	// 回调 failed
	cbBody2 := `{"status":"failed","error_message":"SMTP 550 rejected"}`
	req5 := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/invitations/delivery-callback/"+invID+"/", bytes.NewBufferString(cbBody2))
	rec5 := httptest.NewRecorder()
	mux.ServeHTTP(rec5, req5)
	if rec5.Code != 200 {
		t.Fatalf("callback failed %d body=%s", rec5.Code, rec5.Body.String())
	}
	req6 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/members/pending-invitations/", nil), "admin1")
	rec6 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec6, req6)
	_ = json.Unmarshal(rec6.Body.Bytes(), &list)
	inv = list[0].(map[string]interface{})
	if inv["delivery_status"] != "failed" {
		t.Fatalf("expected failed after callback, got %v", inv["delivery_status"])
	}
	if inv["delivery_error"] != "SMTP 550 rejected" {
		t.Fatalf("expected delivery_error recorded, got %v", inv["delivery_error"])
	}

	// 非法状态 400
	cbBody3 := `{"status":"bogus"}`
	req7 := httptest.NewRequest(http.MethodPost, "/api/internal/tenant/invitations/delivery-callback/"+invID+"/", bytes.NewBufferString(cbBody3))
	rec7 := httptest.NewRecorder()
	mux.ServeHTTP(rec7, req7)
	if rec7.Code != 400 {
		t.Fatalf("expected 400 for invalid status, got %d", rec7.Code)
	}
}
