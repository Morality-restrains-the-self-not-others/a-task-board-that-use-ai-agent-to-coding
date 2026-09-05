package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
)

// tenantMock records what the taskTenantService internal API receives during a test.
type tenantMock struct {
	membersJSON       string
	secretHeader      string
	updateCalled      int
	updateBodies      []map[string]interface{}
	avatarUploaded    int
	avatarFiles       [][]byte
	avatarDeleted     int
	avatarDeleteQuery string
	avatarStatus      int // non-zero: simulated failure for avatar endpoints
}

func newTenantMock(t *testing.T, m *tenantMock) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/tenant/members", func(w http.ResponseWriter, r *http.Request) {
		m.secretHeader = r.Header.Get("X-Internal-Secret")
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, m.membersJSON)
	})
	mux.HandleFunc("/api/internal/tenant/members/avatar", func(w http.ResponseWriter, r *http.Request) {
		m.secretHeader = r.Header.Get("X-Internal-Secret")
		switch r.Method {
		case http.MethodDelete:
			m.avatarDeleted++
			m.avatarDeleteQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"m1","user_id":"u","company_id":"c-1","member_avatar":""}`)
		case http.MethodPost:
			if m.avatarStatus != 0 {
				writeMockJSON(w, m.avatarStatus, `{"error":"mock failure"}`)
				return
			}
			if err := r.ParseMultipartForm(maxAvatarSize + 1<<20); err != nil {
				writeMockJSON(w, http.StatusBadRequest, `{"error":"invalid multipart"}`)
				return
			}
			m.avatarUploaded++
			f, _, err := r.FormFile("avatar")
			if err != nil {
				writeMockJSON(w, http.StatusBadRequest, `{"error":"avatar file required"}`)
				return
			}
			defer f.Close()
			data, _ := io.ReadAll(f)
			m.avatarFiles = append(m.avatarFiles, data)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"m1","user_id":"u","company_id":"c-1","member_avatar":"company_member_avatars/c-1/m1_x.png"}`)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/internal/tenant/members/update", func(w http.ResponseWriter, r *http.Request) {
		m.secretHeader = r.Header.Get("X-Internal-Secret")
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]interface{}
		_ = json.Unmarshal(body, &parsed)
		m.updateCalled++
		m.updateBodies = append(m.updateBodies, parsed)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"m1","user_id":"u","company_id":"c-1","member_name":"ok"}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeMockJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprint(w, body)
}

// withTenantConfig points cfg.TenantServiceURL at the mock and restores on cleanup.
func withTenantConfig(t *testing.T, srv *httptest.Server) {
	t.Helper()
	prev := cfg
	cfg = Config{TenantServiceURL: srv.URL, InternalSecret: "test-secret"}
	t.Cleanup(func() { cfg = prev })
}

// withAvatarUploadDisabled temporarily enables the avatar-upload safety switch so
// tests can exercise the fail-closed 403 path (restored on cleanup).
// OPT-20260806-046: 安全加固完成后默认开关为 false（上传开启）；此 helper 仅用于
// 验证熔断能力保留。
func withAvatarUploadDisabled(t *testing.T) {
	t.Helper()
	prev := avatarUploadDisabled
	avatarUploadDisabled = true
	t.Cleanup(func() { avatarUploadDisabled = prev })
}

const testMemberList = `[{"id":"m1","user_id":"%s","company_id":"c-1","is_admin":true,"is_creator":true,"is_active":true,"workspace_id":"","member_name":"软刀","member_avatar":"","member_avatar_url":"","created_at":"2026-08-06 10:00:00","company_name":"软刀"}]`

// TestHandleUploadCompanyAvatar — 上传公司头像：转发 multipart 到 tenant 内部 API，
// 返回与 GET profile 同构的完整 profile。临时安全开关关闭状态下验证上传链路本身。
func TestHandleUploadCompanyAvatar(t *testing.T) {
	setupAuthTestDB(t)

	userID := mustCreateWechatTestUser(t, "u-company-avatar-001")
	upsertUserProfile(userID, "全局昵称")
	mock := &tenantMock{membersJSON: fmt.Sprintf(testMemberList, userID)}
	srv := newTenantMock(t, mock)
	withTenantConfig(t, srv)

	avatarBytes := makeTestPNG(t, 32, 32)
	body, contentType := multipartBody(t, map[string]string{"company_id": "c-1"}, "avatar", avatarBytes)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/company-avatar/", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleUploadCompanyAvatar(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if mock.avatarUploaded != 1 {
		t.Errorf("tenant avatar POST count = %d, want 1", mock.avatarUploaded)
	}
	if len(mock.avatarFiles) != 1 || !bytes.Equal(mock.avatarFiles[0], avatarBytes) {
		t.Errorf("forwarded avatar bytes mismatch: %v", mock.avatarFiles)
	}
	if mock.secretHeader != "test-secret" {
		t.Errorf("internal secret header = %q, want test-secret", mock.secretHeader)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response not json: %v", err)
	}
	if resp["user_id"] != userID {
		t.Errorf("profile user_id = %v, want %s", resp["user_id"], userID)
	}
	if got := resp["company_nicknames"]; got == nil {
		t.Errorf("profile company_nicknames missing")
	}
}

// TestHandleUploadCompanyAvatar_BadInputs — 缺 company_id / 非法图片类型 → 400，不转发。
func TestHandleUploadCompanyAvatar_BadInputs(t *testing.T) {
	setupAuthTestDB(t)

	userID := mustCreateWechatTestUser(t, "u-company-avatar-002")
	mock := &tenantMock{membersJSON: "[]"}
	srv := newTenantMock(t, mock)
	withTenantConfig(t, srv)

	// 缺 company_id
	body, contentType := multipartBody(t, nil, "avatar", []byte{1, 2, 3})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/company-avatar/", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleUploadCompanyAvatar(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing company_id: status = %d, want 400", w.Code)
	}
	if mock.avatarUploaded != 0 {
		t.Errorf("missing company_id must not forward, got %d", mock.avatarUploaded)
	}

	// 非法图片类型
	body, contentType = multipartBodyWithCT(t, map[string]string{"company_id": "c-1"}, "avatar", []byte("plain text"), "text/plain")
	req = httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/company-avatar/", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-Id", userID)
	w = httptest.NewRecorder()
	handleUploadCompanyAvatar(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad mime: status = %d, want 400", w.Code)
	}
	if mock.avatarUploaded != 0 {
		t.Errorf("bad mime must not forward, got %d", mock.avatarUploaded)
	}
}

// TestHandleUploadCompanyAvatar_TenantError — tenant 返回 404 → 桥接透传错误与状态码。
func TestHandleUploadCompanyAvatar_TenantError(t *testing.T) {
	setupAuthTestDB(t)

	userID := mustCreateWechatTestUser(t, "u-company-avatar-003")
	mock := &tenantMock{membersJSON: "[]", avatarStatus: http.StatusNotFound}
	srv := newTenantMock(t, mock)
	withTenantConfig(t, srv)

	body, contentType := multipartBody(t, map[string]string{"company_id": "c-1"}, "avatar", makeTestPNG(t, 32, 32))
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/company-avatar/", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleUploadCompanyAvatar(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "mock failure" {
		t.Errorf("error passthrough = %v, want mock failure", resp["error"])
	}
}

// TestHandleDeleteCompanyAvatar — 移除公司头像：携带 user_id/company_id 转发 DELETE。
func TestHandleDeleteCompanyAvatar(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "u-company-avatar-004")
	mock := &tenantMock{membersJSON: fmt.Sprintf(testMemberList, userID)}
	srv := newTenantMock(t, mock)
	withTenantConfig(t, srv)

	req := httptest.NewRequest(http.MethodDelete, "/api/accounts/users/profile/company-avatar/?company_id=c-1", nil)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleDeleteCompanyAvatar(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if mock.avatarDeleted != 1 {
		t.Errorf("tenant avatar DELETE count = %d, want 1", mock.avatarDeleted)
	}
	if !strings.Contains(mock.avatarDeleteQuery, "user_id="+userID) || !strings.Contains(mock.avatarDeleteQuery, "company_id=c-1") {
		t.Errorf("delete query = %q, want user_id+company_id", mock.avatarDeleteQuery)
	}
}

// TestHandleDeleteCompanyAvatar_MissingCompanyID — 缺 company_id → 400。
func TestHandleDeleteCompanyAvatar_MissingCompanyID(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "u-company-avatar-005")
	mock := &tenantMock{membersJSON: "[]"}
	srv := newTenantMock(t, mock)
	withTenantConfig(t, srv)

	req := httptest.NewRequest(http.MethodDelete, "/api/accounts/users/profile/company-avatar/", nil)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleDeleteCompanyAvatar(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// TestHandleCopyPersonalToCompany — 复制昵称+全局头像：PATCH 昵称 + POST 解码后的头像字节。
// 临时安全开关关闭状态下验证完整复制链路。
func TestHandleCopyPersonalToCompany(t *testing.T) {
	setupAuthTestDB(t)

	userID := mustCreateWechatTestUser(t, "u-copy-company-001")
	upsertUserProfile(userID, "全局昵称")
	avatarBytes := makeTestPNG(t, 32, 32)
	upsertUserAvatar(userID, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(avatarBytes))
	mock := &tenantMock{membersJSON: fmt.Sprintf(testMemberList, userID)}
	srv := newTenantMock(t, mock)
	withTenantConfig(t, srv)

	body := `{"company_id":"c-1","personal_nickname":"输入框昵称"}`
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/copy-personal-to-company/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleCopyPersonalToCompany(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if mock.updateCalled != 1 {
		t.Fatalf("nickname update count = %d, want 1", mock.updateCalled)
	}
	if got := mock.updateBodies[0]["member_name"]; got != "输入框昵称" {
		t.Errorf("member_name = %v, want 输入框昵称", got)
	}
	if mock.avatarUploaded != 1 {
		t.Fatalf("avatar upload count = %d, want 1", mock.avatarUploaded)
	}
	if len(mock.avatarFiles) != 1 || !bytes.Equal(mock.avatarFiles[0], avatarBytes) {
		t.Errorf("forwarded avatar bytes mismatch: %v", mock.avatarFiles)
	}
}

// TestHandleCopyPersonalToCompany_AvatarUploadDisabled — 安全开关开启（默认）：
// 只复制昵称，跳过头像上传；端点本身仍正常返回 profile。
func TestHandleCopyPersonalToCompany_AvatarUploadDisabled(t *testing.T) {
	setupAuthTestDB(t)
	withAvatarUploadDisabled(t)
	userID := mustCreateWechatTestUser(t, "u-copy-company-004")
	upsertUserProfile(userID, "全局昵称")
	avatarBytes := makeTestPNG(t, 32, 32)
	upsertUserAvatar(userID, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(avatarBytes))
	mock := &tenantMock{membersJSON: fmt.Sprintf(testMemberList, userID)}
	srv := newTenantMock(t, mock)
	withTenantConfig(t, srv)

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/copy-personal-to-company/", strings.NewReader(`{"company_id":"c-1","personal_nickname":"仅昵称"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleCopyPersonalToCompany(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if mock.updateCalled != 1 {
		t.Errorf("nickname update count = %d, want 1", mock.updateCalled)
	}
	if got := mock.updateBodies[0]["member_name"]; got != "仅昵称" {
		t.Errorf("member_name = %v, want 仅昵称", got)
	}
	if mock.avatarUploaded != 0 {
		t.Errorf("avatar upload count = %d, want 0 (avatar upload disabled)", mock.avatarUploaded)
	}
}

// TestHandleUploadCompanyAvatar_Disabled — 安全开关开启：403 + 明确文案，不转发。
func TestHandleUploadCompanyAvatar_Disabled(t *testing.T) {
	setupAuthTestDB(t)
	withAvatarUploadDisabled(t)
	userID := mustCreateWechatTestUser(t, "u-company-avatar-006")
	mock := &tenantMock{membersJSON: "[]"}
	srv := newTenantMock(t, mock)
	withTenantConfig(t, srv)

	body, contentType := multipartBody(t, map[string]string{"company_id": "c-1"}, "avatar", []byte{1})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/company-avatar/", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleUploadCompanyAvatar(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !strings.Contains(fmt.Sprint(resp["error"]), "禁用") {
		t.Errorf("error = %v, want 禁用文案", resp["error"])
	}
	if mock.avatarUploaded != 0 {
		t.Errorf("disabled upload must not forward, got %d", mock.avatarUploaded)
	}
}

// TestHandleUploadAvatar_GlobalDisabled — 全局头像上传同样被禁用：403 + 明确文案。
func TestHandleUploadAvatar_GlobalDisabled(t *testing.T) {
	setupAuthTestDB(t)
	withAvatarUploadDisabled(t)
	userID := mustCreateWechatTestUser(t, "u-global-avatar-001")
	upsertUserProfile(userID, "全局昵称")

	body, contentType := multipartBody(t, nil, "avatar", []byte{1})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/avatar/", body)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleUploadAvatar(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !strings.Contains(fmt.Sprint(resp["error"]), "禁用") {
		t.Errorf("error = %v, want 禁用文案", resp["error"])
	}
}

// TestHandleCopyPersonalToCompany_FallbackNickname — 请求体缺昵称 → 回退数据库已保存昵称。
func TestHandleCopyPersonalToCompany_FallbackNickname(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "u-copy-company-002")
	upsertUserProfile(userID, "数据库昵称")
	mock := &tenantMock{membersJSON: fmt.Sprintf(testMemberList, userID)}
	srv := newTenantMock(t, mock)
	withTenantConfig(t, srv)

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/copy-personal-to-company/", strings.NewReader(`{"company_id":"c-1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleCopyPersonalToCompany(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if mock.updateCalled != 1 {
		t.Fatalf("nickname update count = %d, want 1", mock.updateCalled)
	}
	if got := mock.updateBodies[0]["member_name"]; got != "数据库昵称" {
		t.Errorf("member_name = %v, want 数据库昵称", got)
	}
	if mock.avatarUploaded != 0 {
		t.Errorf("avatar upload count = %d, want 0 (no saved global avatar)", mock.avatarUploaded)
	}
}

// TestHandleCopyPersonalToCompany_MissingCompanyID — 缺 company_id → 400。
func TestHandleCopyPersonalToCompany_MissingCompanyID(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "u-copy-company-003")
	mock := &tenantMock{membersJSON: "[]"}
	srv := newTenantMock(t, mock)
	withTenantConfig(t, srv)

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/profile/copy-personal-to-company/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userID)
	w := httptest.NewRecorder()
	handleCopyPersonalToCompany(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if mock.updateCalled != 0 {
		t.Errorf("must not forward on missing company_id, got %d updates", mock.updateCalled)
	}
}

// TestDecodeAvatarDataURI — data URI 解码边界。
func TestDecodeAvatarDataURI(t *testing.T) {
	raw := []byte{0x89, 0x50, 0x4e, 0x47, 0x01}
	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)
	data, ct, err := decodeAvatarDataURI(uri)
	if err != nil {
		t.Fatalf("valid uri: %v", err)
	}
	if !bytes.Equal(data, raw) || ct != "image/png" {
		t.Errorf("decoded = (%v, %s), want (%v, image/png)", data, ct, raw)
	}
	if _, _, err := decodeAvatarDataURI("not-a-data-uri"); err == nil {
		t.Error("non-data-uri must fail")
	}
	if _, _, err := decodeAvatarDataURI("data:image/png;base64,%%%"); err == nil {
		t.Error("invalid base64 must fail")
	}
	if _, _, err := decodeAvatarDataURI("data:application/pdf;base64," + base64.StdEncoding.EncodeToString(raw)); err == nil {
		t.Error("unsupported mime must fail")
	}
	big := bytes.Repeat([]byte{1}, maxAvatarSize+1)
	if _, _, err := decodeAvatarDataURI("data:image/png;base64," + base64.StdEncoding.EncodeToString(big)); err == nil {
		t.Error("oversized payload must fail")
	}
}

// ——— helpers ———

func multipartBody(t *testing.T, fields map[string]string, fileField string, fileData []byte) (*bytes.Buffer, string) {
	t.Helper()
	return multipartBodyWithCT(t, fields, fileField, fileData, "image/png")
}

func multipartBodyWithCT(t *testing.T, fields map[string]string, fileField string, fileData []byte, fileCT string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="avatar.png"`, fileField))
	hdr.Set("Content-Type", fileCT)
	fw, _ := mw.CreatePart(hdr)
	_, _ = fw.Write(fileData)
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}
