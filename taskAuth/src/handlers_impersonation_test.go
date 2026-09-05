package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"taskAuth/domain"
)

func impersonateReq(method, path, actorID, roles, token, idemKey string) *http.Request {
	return impersonateReqBody(method, path, actorID, roles, token, idemKey, "")
}

func impersonateReqBody(method, path, actorID, roles, token, idemKey, body string) *http.Request {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if actorID != "" {
		req.Header.Set("X-User-Id", actorID)
	}
	if roles != "" {
		req.Header.Set("X-User-Roles", roles)
	}
	if token != "" {
		req.Header.Set("Authorization", "Token "+token)
	}
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	return req
}

const testImpersonationReasonJSON = `{"reason":"排查线上工单问题"}`

func postImpersonate(actorID, roles, token, targetID, idemKey string) *httptest.ResponseRecorder {
	return postImpersonateBody(actorID, roles, token, targetID, idemKey, testImpersonationReasonJSON)
}

func postImpersonateBody(actorID, roles, token, targetID, idemKey, body string) *httptest.ResponseRecorder {
	req := impersonateReqBody(http.MethodPost,
		"/api/system-admin/users/"+targetID+"/impersonate/",
		actorID, roles, token, idemKey, body)
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	return rec
}

func TestImpersonateRequiresPermission(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-noperm@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	rec := postImpersonate("bootstrap-admin", "", "", targetID, "k1")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImpersonateMissingIdempotencyKey(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-nokey@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	tok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	rec := postImpersonate("bootstrap-admin", "super_admin", tok, targetID, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImpersonateSuccessIssuesIndependentToken(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-ok@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	targetTok, err := getOrCreateToken(targetID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("target token: %v", err)
	}
	adminTok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("admin token: %v", err)
	}
	rec := postImpersonate("bootstrap-admin", "super_admin", adminTok, targetID, "idem-ok")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	token, _ := body["token"].(string)
	if !strings.HasPrefix(token, "imp_") {
		t.Fatalf("token prefix: %q", token)
	}
	if token == targetTok {
		t.Fatal("must not reuse target auth_customtoken")
	}
	user, _ := body["user"].(map[string]interface{})
	if user["id"] != targetID {
		t.Fatalf("user.id=%v want %s", user["id"], targetID)
	}
	imp, _ := body["impersonation"].(map[string]interface{})
	if imp["actor_user_id"] != "bootstrap-admin" || imp["target_user_id"] != targetID {
		t.Fatalf("impersonation=%v", imp)
	}
	foundRestore := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == impersonatorRestoreCookie && c.Value == adminTok {
			foundRestore = true
		}
	}
	if !foundRestore {
		t.Fatal("missing impersonatorRestore cookie")
	}
	resolved, err := resolveTokenUserIDWithIP(token, "1.2.3.4")
	if err != nil || resolved != targetID {
		t.Fatalf("resolve impersonation token: %q %v", resolved, err)
	}
}

func TestImpersonateIdempotentReplay(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-idem@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	adminTok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	r1 := postImpersonate("bootstrap-admin", "super_admin", adminTok, targetID, "same-key")
	r2 := postImpersonate("bootstrap-admin", "super_admin", adminTok, targetID, "same-key")
	if r1.Code != http.StatusOK || r2.Code != http.StatusOK {
		t.Fatalf("codes %d %d", r1.Code, r2.Code)
	}
	var a, b map[string]interface{}
	_ = json.Unmarshal(r1.Body.Bytes(), &a)
	_ = json.Unmarshal(r2.Body.Bytes(), &b)
	if a["token"] != b["token"] {
		t.Fatalf("replay token mismatch %v vs %v", a["token"], b["token"])
	}
}

func TestImpersonateSelfConflict(t *testing.T) {
	setupAuthTestDB(t)
	tok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	rec := postImpersonate("bootstrap-admin", "super_admin", tok, "bootstrap-admin", "k-self")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImpersonateInactiveUnprocessable(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-off@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_active = 0 WHERE id = ?`, targetID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	tok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	rec := postImpersonate("bootstrap-admin", "super_admin", tok, targetID, "k-off")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImpersonateMissingUser(t *testing.T) {
	setupAuthTestDB(t)
	tok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	rec := postImpersonate("bootstrap-admin", "super_admin", tok, "999999999999999999", "k-404")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImpersonateEmployeeCannotEscalate(t *testing.T) {
	setupAuthTestDB(t)
	empID, _, err := createUserWithEmailLogin("imp-emp@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	tok, err := getOrCreateToken(empID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	rec := postImpersonate(empID, "employee", tok, "bootstrap-admin", "k-esc")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImpersonateEmployeeCanImpersonateRegular(t *testing.T) {
	setupAuthTestDB(t)
	empID, _, err := createUserWithEmailLogin("imp-emp2@test.com", "hash")
	if err != nil {
		t.Fatalf("create emp: %v", err)
	}
	targetID, _, err := createUserWithEmailLogin("imp-reg@test.com", "hash")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	tok, err := getOrCreateToken(empID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	rec := postImpersonate(empID, "employee", tok, targetID, "k-emp-ok")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImpersonateNestedConflict(t *testing.T) {
	setupAuthTestDB(t)
	t1, _, err := createUserWithEmailLogin("imp-n1@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t2, _, err := createUserWithEmailLogin("imp-n2@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	adminTok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	r1 := postImpersonate("bootstrap-admin", "super_admin", adminTok, t1, "k-n1")
	if r1.Code != http.StatusOK {
		t.Fatalf("start: %d %s", r1.Code, r1.Body.String())
	}
	var body map[string]interface{}
	_ = json.Unmarshal(r1.Body.Bytes(), &body)
	impTok, _ := body["token"].(string)
	r2 := postImpersonate("bootstrap-admin", "super_admin", impTok, t2, "k-n2")
	if r2.Code != http.StatusConflict {
		t.Fatalf("expected 409 nested, got %d: %s", r2.Code, r2.Body.String())
	}
	got := r2.Body.String()
	if !strings.Contains(got, domain.NestedImpersonationClientMessage) {
		t.Fatalf("nested 409 must be Chinese client copy, got %s", got)
	}
	if strings.Contains(got, "already impersonating") {
		t.Fatalf("nested 409 must not leak English sentinel, got %s", got)
	}
}

func TestImpersonateResumeSameTargetDifferentIdempotencyKey(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-resume-key@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	adminTok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	r1 := postImpersonate("bootstrap-admin", "super_admin", adminTok, targetID, "k-resume-a")
	if r1.Code != http.StatusOK {
		t.Fatalf("start: %d %s", r1.Code, r1.Body.String())
	}
	r2 := postImpersonate("bootstrap-admin", "super_admin", adminTok, targetID, "k-resume-b")
	if r2.Code != http.StatusOK {
		t.Fatalf("resume with new idempotency key: got %d %s", r2.Code, r2.Body.String())
	}
	var a, b map[string]interface{}
	_ = json.Unmarshal(r1.Body.Bytes(), &a)
	_ = json.Unmarshal(r2.Body.Bytes(), &b)
	if a["token"] != b["token"] {
		t.Fatalf("expected same session token, got %v vs %v", a["token"], b["token"])
	}
	redirect, _ := b["redirect_url"].(string)
	if strings.HasPrefix(redirect, "/system-admin") {
		t.Fatalf("impersonation must land on target frontend, got %q", redirect)
	}
	listReq := impersonateReq(http.MethodGet, "/api/auth/inbox/", targetID, "", "", "")
	listRec := httptest.NewRecorder()
	handleListInbox(listRec, listReq)
	if strings.Count(listRec.Body.String(), `"kind":"impersonation_notice"`) != 1 {
		t.Fatalf("resume must not write a second inbox notice: %s", listRec.Body.String())
	}
}

func TestImpersonateResumeSameTargetUsingImpersonationToken(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-resume-tok@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	adminTok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	r1 := postImpersonate("bootstrap-admin", "super_admin", adminTok, targetID, "k-resume-tok-1")
	if r1.Code != http.StatusOK {
		t.Fatalf("start: %d %s", r1.Code, r1.Body.String())
	}
	var started map[string]interface{}
	_ = json.Unmarshal(r1.Body.Bytes(), &started)
	impTok, _ := started["token"].(string)
	r2 := postImpersonate(targetID, "super_admin", impTok, targetID, "k-resume-tok-2")
	if r2.Code != http.StatusOK {
		t.Fatalf("resume while already impersonating same target: got %d %s", r2.Code, r2.Body.String())
	}
	var resumed map[string]interface{}
	_ = json.Unmarshal(r2.Body.Bytes(), &resumed)
	if resumed["token"] != impTok {
		t.Fatalf("expected same impersonation token, got %v", resumed["token"])
	}
}

func TestImpersonationStatusAndStop(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-stop@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	adminTok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	start := postImpersonate("bootstrap-admin", "super_admin", adminTok, targetID, "k-stop")
	if start.Code != http.StatusOK {
		t.Fatalf("start: %d %s", start.Code, start.Body.String())
	}
	var body map[string]interface{}
	_ = json.Unmarshal(start.Body.Bytes(), &body)
	impTok, _ := body["token"].(string)

	stReq := impersonateReq(http.MethodGet, "/api/auth/impersonation/status/", targetID, "", impTok, "")
	stRec := httptest.NewRecorder()
	handleImpersonationStatus(stRec, stReq)
	if stRec.Code != http.StatusOK {
		t.Fatalf("status: %d %s", stRec.Code, stRec.Body.String())
	}
	var st map[string]interface{}
	_ = json.Unmarshal(stRec.Body.Bytes(), &st)
	if st["impersonating"] != true {
		t.Fatalf("status=%v", st)
	}
	redir, _ := st["redirect_url"].(string)
	if redir == "" || strings.HasPrefix(redir, "/system-admin") {
		t.Fatalf("status redirect_url=%v", st["redirect_url"])
	}

	stopReq := impersonateReq(http.MethodPost, "/api/auth/impersonation/stop/", targetID, "", impTok, "")
	stopReq.AddCookie(&http.Cookie{Name: impersonatorRestoreCookie, Value: adminTok})
	stopRec := httptest.NewRecorder()
	handleStopImpersonation(stopRec, stopReq)
	if stopRec.Code != http.StatusOK {
		t.Fatalf("stop: %d %s", stopRec.Code, stopRec.Body.String())
	}
	var stopped map[string]interface{}
	_ = json.Unmarshal(stopRec.Body.Bytes(), &stopped)
	if stopped["token"] != adminTok {
		t.Fatalf("restore token=%v", stopped["token"])
	}
	if stopped["redirect_url"] != "/system-admin/users/" {
		t.Fatalf("redirect=%v", stopped["redirect_url"])
	}

	again := httptest.NewRecorder()
	handleStopImpersonation(again, impersonateReq(http.MethodPost, "/api/auth/impersonation/stop/", targetID, "", impTok, ""))
	if again.Code != http.StatusOK {
		t.Fatalf("stop no-op: %d %s", again.Code, again.Body.String())
	}
}

func TestImpersonationStatusNotImpersonating(t *testing.T) {
	setupAuthTestDB(t)
	tok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := impersonateReq(http.MethodGet, "/api/auth/impersonation/status/", "bootstrap-admin", "super_admin", tok, "")
	rec := httptest.NewRecorder()
	handleImpersonationStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"impersonating":false`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestGatewayForwardAuthImpersonatorHeader(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"
	targetID, _, err := createUserWithEmailLogin("imp-fwd@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	adminTok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	start := postImpersonate("bootstrap-admin", "super_admin", adminTok, targetID, "k-fwd")
	if start.Code != http.StatusOK {
		t.Fatalf("start: %d %s", start.Code, start.Body.String())
	}
	var body map[string]interface{}
	_ = json.Unmarshal(start.Body.Bytes(), &body)
	impTok, _ := body["token"].(string)

	req := httptest.NewRequest(http.MethodGet, "/api/internal/gateway/forward-auth/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	req.Header.Set("Authorization", "Token "+impTok)
	rec := httptest.NewRecorder()
	handleGatewayForwardAuth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("forward-auth: %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-User-Id") != targetID {
		t.Fatalf("X-User-Id=%s want %s", rec.Header().Get("X-User-Id"), targetID)
	}
	if rec.Header().Get("X-Impersonator-Id") != "bootstrap-admin" {
		t.Fatalf("X-Impersonator-Id=%s", rec.Header().Get("X-Impersonator-Id"))
	}
	if rec.Header().Get("X-Impersonation-Session-Id") == "" {
		t.Fatal("missing X-Impersonation-Session-Id")
	}
	if rec.Header().Get("X-Impersonating") != "1" {
		t.Fatalf("X-Impersonating=%s", rec.Header().Get("X-Impersonating"))
	}
}

func TestImpersonateMissingReason(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-noreason@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	tok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	rec := postImpersonateBody("bootstrap-admin", "super_admin", tok, targetID, "k-noreason", `{"reason":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestImpersonateRejectsModalPromptAsReason(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-prompt@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	tok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	for _, reason := range []string{
		"请填写本次模拟登录的理由。该理由会写入审计并通知被模拟用户。",
		"例如：排查线上工单 T-12345",
	} {
		rec := postImpersonateBody("bootstrap-admin", "super_admin", tok, targetID, "k-prompt", `{"reason":`+strconv.Quote(reason)+`}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reason %q: expected 400, got %d: %s", reason, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "must not be the modal prompt") {
			t.Fatalf("reason %q: error body=%s", reason, rec.Body.String())
		}
	}
	listReq := impersonateReq(http.MethodGet, "/api/auth/inbox/", targetID, "", "", "")
	listRec := httptest.NewRecorder()
	handleListInbox(listRec, listReq)
	if strings.Contains(listRec.Body.String(), `"kind":"impersonation_notice"`) {
		t.Fatalf("rejected reason must not write an inbox notice: %s", listRec.Body.String())
	}
}

func TestImpersonateWritesInboxNotice(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("imp-inbox@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	adminTok, err := getOrCreateToken("bootstrap-admin", cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	rec := postImpersonate("bootstrap-admin", "super_admin", adminTok, targetID, "k-inbox")
	if rec.Code != http.StatusOK {
		t.Fatalf("start: %d %s", rec.Code, rec.Body.String())
	}
	replay := postImpersonate("bootstrap-admin", "super_admin", adminTok, targetID, "k-inbox")
	if replay.Code != http.StatusOK {
		t.Fatalf("replay: %d %s", replay.Code, replay.Body.String())
	}

	listReq := impersonateReq(http.MethodGet, "/api/auth/inbox/", targetID, "", "", "")
	listRec := httptest.NewRecorder()
	handleListInbox(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("inbox: %d %s", listRec.Code, listRec.Body.String())
	}
	if strings.Count(listRec.Body.String(), `"kind":"impersonation_notice"`) != 1 {
		t.Fatalf("expected one notice, got %s", listRec.Body.String())
	}
	if !strings.Contains(listRec.Body.String(), "排查线上工单问题") {
		t.Fatalf("missing reason in inbox: %s", listRec.Body.String())
	}
	var listedBody struct {
		Results []struct {
			Body   string `json:"body"`
			Reason string `json:"reason"`
		} `json:"results"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listedBody); err != nil {
		t.Fatalf("decode inbox body: %v", err)
	}
	if len(listedBody.Results) != 1 {
		t.Fatalf("need one inbox row, got %+v", listedBody.Results)
	}
	if listedBody.Results[0].Reason != "排查线上工单问题" {
		t.Fatalf("reason field=%q", listedBody.Results[0].Reason)
	}
	if strings.Contains(listedBody.Results[0].Body, "理由") {
		t.Fatalf("inbox body must not embed reason: %q", listedBody.Results[0].Body)
	}

	otherID, _, err := createUserWithEmailLogin("imp-inbox-other@test.com", "hash")
	if err != nil {
		t.Fatalf("create other: %v", err)
	}
	otherReq := impersonateReq(http.MethodGet, "/api/auth/inbox/", otherID, "", "", "")
	otherRec := httptest.NewRecorder()
	handleListInbox(otherRec, otherReq)
	if otherRec.Code != http.StatusOK {
		t.Fatalf("other inbox: %d %s", otherRec.Code, otherRec.Body.String())
	}
	if strings.Contains(otherRec.Body.String(), `"kind":"impersonation_notice"`) {
		t.Fatalf("IDOR leak: %s", otherRec.Body.String())
	}

	var listed struct {
		Results []struct {
			ID   string `json:"id"`
			Read bool   `json:"read"`
		} `json:"results"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode inbox: %v", err)
	}
	if len(listed.Results) != 1 || listed.Results[0].ID == "" {
		t.Fatalf("need one inbox id, got %+v", listed.Results)
	}
	mux := http.NewServeMux()
	mountRoutes(mux)
	markReq := impersonateReq(http.MethodPost, "/api/auth/inbox/"+listed.Results[0].ID+"/read/", targetID, "", "", "")
	markRec := httptest.NewRecorder()
	mux.ServeHTTP(markRec, markReq)
	if markRec.Code != http.StatusOK {
		t.Fatalf("mark read: %d %s", markRec.Code, markRec.Body.String())
	}
	listAfter := httptest.NewRecorder()
	handleListInbox(listAfter, impersonateReq(http.MethodGet, "/api/auth/inbox/", targetID, "", "", ""))
	if !strings.Contains(listAfter.Body.String(), `"read":true`) {
		t.Fatalf("expected read after mark: %s", listAfter.Body.String())
	}
}

func TestImpersonationEventTopics(t *testing.T) {
	if eventTopicMap["UserImpersonationStarted"] != "user-impersonation-started" {
		t.Fatalf("started topic=%q", eventTopicMap["UserImpersonationStarted"])
	}
	if eventTopicMap["UserImpersonationStopped"] != "user-impersonation-stopped" {
		t.Fatalf("stopped topic=%q", eventTopicMap["UserImpersonationStopped"])
	}
	if eventTopicMap["UserInboxMessageCreated"] != "user-inbox-message-created" {
		t.Fatalf("inbox topic=%q", eventTopicMap["UserInboxMessageCreated"])
	}
}
