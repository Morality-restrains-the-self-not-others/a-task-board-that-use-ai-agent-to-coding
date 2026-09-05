package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dbload "dbload"
)

func TestBootstrapAdminCreatesSuperAdminRow(t *testing.T) {
	testDSN, cleanup, err := dbload.OpenTestMySQL("task-auth", repoRoot())
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	if err := ensureBootstrapAdminSeeded(testDSN, repoRoot()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := openDB(testDSN); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	defer db.Close()

	userID, err := lookupBootstrapAdminUserID()
	if err != nil {
		t.Fatalf("lookup user id: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_super_admin WHERE user_id = ?`, userID).Scan(&count); err != nil {
		t.Fatalf("count super_admin: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 super_admin row, got %d", count)
	}
}

func TestHandleGetUserRequiresToken(t *testing.T) {
	setupAuthTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/123/", nil)
	req.SetPathValue("user_id", "123")
	rec := httptest.NewRecorder()
	handleGetUser(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleGetUserAndBatchResolve(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("resolve@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/"+userID+"/", nil)
	req.SetPathValue("user_id", userID)
	req.Header.Set("Authorization", "Token "+token)
	rec := httptest.NewRecorder()
	handleGetUser(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET user expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var userPayload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &userPayload); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	if userPayload["email"] != "resolve@test.com" {
		t.Fatalf("email: %v", userPayload["email"])
	}

	cfg.InternalSecret = "test-secret"
	batchBody := strings.NewReader(`{"user_ids":["` + userID + `","missing-id"]}`)
	batchReq := httptest.NewRequest(http.MethodPost, "/api/internal/users/batch/resolve/", batchBody)
	batchReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	batchRec := httptest.NewRecorder()
	handleBatchResolveUsers(batchRec, batchReq)
	if batchRec.Code != http.StatusOK {
		t.Fatalf("batch-resolve expected 200, got %d body=%s", batchRec.Code, batchRec.Body.String())
	}
	var batchPayload struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.Unmarshal(batchRec.Body.Bytes(), &batchPayload); err != nil {
		t.Fatalf("decode batch: %v", err)
	}
	if len(batchPayload.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(batchPayload.Results))
	}
	if batchPayload.Results[0]["found"] != true {
		t.Fatalf("found user result: %+v", batchPayload.Results[0])
	}
	if batchPayload.Results[1]["found"] != false {
		t.Fatalf("missing user result: %+v", batchPayload.Results[1])
	}

	superReq := httptest.NewRequest(http.MethodGet, "/api/internal/users/"+userID+"/super-admin/", nil)
	superReq.SetPathValue("user_id", userID)
	superReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	superRec := httptest.NewRecorder()
	handleGetSuperAdmin(superRec, superReq)
	if superRec.Code != http.StatusOK {
		t.Fatalf("super-admin expected 200, got %d", superRec.Code)
	}
	var superPayload map[string]interface{}
	if err := json.Unmarshal(superRec.Body.Bytes(), &superPayload); err != nil {
		t.Fatalf("decode super-admin: %v", err)
	}
	if superPayload["is_super_admin"] != false {
		t.Fatalf("expected non-super admin, got %+v", superPayload)
	}

	tokenResolveBody := strings.NewReader(`{"token":"` + token + `"}`)
	tokenResolveReq := httptest.NewRequest(http.MethodPost, "/api/internal/token/resolve/", tokenResolveBody)
	tokenResolveReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	tokenResolveRec := httptest.NewRecorder()
	handleResolveToken(tokenResolveRec, tokenResolveReq)
	if tokenResolveRec.Code != http.StatusOK {
		t.Fatalf("token-resolve expected 200, got %d body=%s", tokenResolveRec.Code, tokenResolveRec.Body.String())
	}
	var tokenPayload map[string]interface{}
	if err := json.Unmarshal(tokenResolveRec.Body.Bytes(), &tokenPayload); err != nil {
		t.Fatalf("decode token-resolve: %v", err)
	}
	if tokenPayload["user_id"] != userID {
		t.Fatalf("token-resolve user_id: %v", tokenPayload["user_id"])
	}
	if tokenPayload["is_active"] != true {
		t.Fatalf("token-resolve is_active: %v", tokenPayload["is_active"])
	}
}

func TestSearchUserIDsUsernameAndBatchUserDetails(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("fuzzy@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := upsertUserProfile(userID, "fuzzy_user"); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	// 邮箱前缀搜索
	if ids := searchUserIDs("fuzzy@test"); len(ids) != 1 || ids[0] != userID {
		t.Fatalf("email search ids=%v", ids)
	}
	// 用户名前缀搜索（邮箱 identifier 无法匹配该前缀，必须走 auth_user_profile）
	if ids := searchUserIDs("fuzzy_user"); len(ids) != 1 || ids[0] != userID {
		t.Fatalf("username prefix search ids=%v", ids)
	}
	// 用户名中缀搜索（前缀无匹配时回退 infix，仅 username 命中）
	if ids := searchUserIDs("zzy_u"); len(ids) != 1 || ids[0] != userID {
		t.Fatalf("username infix search ids=%v", ids)
	}

	// handleBatchUserDetails 批量返回邮箱/用户名
	cfg.InternalSecret = "test-secret"
	batchBody := strings.NewReader(`{"user_ids":["` + userID + `","missing-id"]}`)
	batchReq := httptest.NewRequest(http.MethodPost, "/api/internal/users/batch/details/", batchBody)
	batchReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	batchRec := httptest.NewRecorder()
	handleBatchUserDetails(batchRec, batchReq)
	if batchRec.Code != http.StatusOK {
		t.Fatalf("batch-details expected 200, got %d body=%s", batchRec.Code, batchRec.Body.String())
	}
	var payload struct {
		Results map[string]struct {
			UserID   string `json:"user_id"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"results"`
	}
	if err := json.Unmarshal(batchRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode batch-details: %v", err)
	}
	info, ok := payload.Results[userID]
	if !ok {
		t.Fatalf("batch-details missing user %s: %+v", userID, payload.Results)
	}
	if info.Email != "fuzzy@test.com" || info.Username != "fuzzy_user" {
		t.Fatalf("batch-details info=%+v", info)
	}
	if _, ok := payload.Results["missing-id"]; !ok {
		t.Fatalf("missing-id should get empty entry: %+v", payload.Results)
	}
}

func TestBatchUserDetailsIncludesPhone(t *testing.T) {
	setupAuthTestDB(t)
	userID, err := createUserWithPhoneLogin("+86", "13900001111", "hash", "")
	if err != nil {
		t.Fatalf("create phone user: %v", err)
	}
	cfg.InternalSecret = "test-secret"
	batchBody := strings.NewReader(`{"user_ids":["` + userID + `"]}`)
	batchReq := httptest.NewRequest(http.MethodPost, "/api/internal/users/batch/details/", batchBody)
	batchReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	batchRec := httptest.NewRecorder()
	handleBatchUserDetails(batchRec, batchReq)
	if batchRec.Code != http.StatusOK {
		t.Fatalf("batch-details expected 200, got %d body=%s", batchRec.Code, batchRec.Body.String())
	}
	var payload struct {
		Results map[string]struct {
			Phone string `json:"phone"`
			Email string `json:"email"`
		} `json:"results"`
	}
	if err := json.Unmarshal(batchRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	info := payload.Results[userID]
	if info.Phone != "13900001111" {
		t.Fatalf("phone=%q want 13900001111 payload=%s", info.Phone, batchRec.Body.String())
	}
}
