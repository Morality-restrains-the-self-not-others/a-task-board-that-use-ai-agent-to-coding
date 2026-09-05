package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// 各公司的设置页（/profile/company-settings/）成员操作桥接接口
//
// 前端 UserProfileCompanySettingsPanel 提供三个操作，直接调用 taskAuth：
//   1. 上传图片   → POST   /api/accounts/users/profile/company-avatar/
//   2. 移除头像   → DELETE /api/accounts/users/profile/company-avatar/?company_id=
//   3. 从个人昵称与头像复制 → POST /api/accounts/users/profile/copy-personal-to-company/
//
// 三个端点在此将请求转发至 taskTenantService 内部成员 API
// （/api/internal/tenant/members/*，成员数据/头像文件的实际 owner），
// 成功后将结果组装成与 GET /api/accounts/users/profile/ 相同结构的 profile
// 返回，前端 applyProfileData 直接复用。
// ---------------------------------------------------------------------------

// handleUploadCompanyAvatar uploads an avatar for the current user's company membership.
// POST /api/accounts/users/profile/company-avatar/
func handleUploadCompanyAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	// OPT-20260806-046: 与全局头像同一安全开关（默认关闭 = 上传开启），熔断能力保留。
	if avatarUploadDisabled {
		writeError(w, r, http.StatusForbidden, "头像上传已暂时禁用（安全维护中），请稍后再试")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize+1024) // small headroom for multipart overhead
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		writeError(w, r, http.StatusBadRequest, "文件过大或格式无效，最大支持 2MB")
		return
	}
	cid := strings.TrimSpace(r.FormValue("company_id"))
	if cid == "" {
		writeError(w, r, http.StatusBadRequest, "company_id 不能为空")
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "未找到上传的文件")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxAvatarSize+1))
	if err != nil {
		log.Printf("[taskAuth] company avatar read error for user %s: %v", userID, err)
		writeError(w, r, http.StatusInternalServerError, "读取文件失败")
		return
	}
	if len(data) > maxAvatarSize {
		writeError(w, r, http.StatusBadRequest, "文件过大，最大支持 2MB")
		return
	}

	// OPT-20260806-046: 与全局头像同一套安全加固（magic bytes + 解码 + 尺寸/压缩比）
	mimeType, _, err := sniffAndValidateAvatarImage(header.Header.Get("Content-Type"), data)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := tenantForwardMultipart("POST", "/api/internal/tenant/members/avatar",
		map[string]string{"user_id": userID, "company_id": cid},
		"avatar", mimeType, data)
	if err != nil {
		log.Printf("[taskAuth] forward company avatar upload user=%s company=%s: %v", userID, cid, err)
		writeError(w, r, http.StatusBadGateway, "服务暂不可用，请稍后重试")
		return
	}
	defer resp.Body.Close()
	writeTenantBridgeResponse(w, r, userID, resp)
}

// handleDeleteCompanyAvatar removes the avatar for the current user's company membership.
// DELETE /api/accounts/users/profile/company-avatar/?company_id=
func handleDeleteCompanyAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	cid := strings.TrimSpace(r.URL.Query().Get("company_id"))
	if cid == "" {
		writeError(w, r, http.StatusBadRequest, "company_id 不能为空")
		return
	}
	resp, err := tenantForwardRequest("DELETE",
		fmt.Sprintf("%s/api/internal/tenant/members/avatar?user_id=%s&company_id=%s",
			tenantServiceBase(), url.QueryEscape(userID), url.QueryEscape(cid)), nil)
	if err != nil {
		log.Printf("[taskAuth] forward company avatar delete user=%s company=%s: %v", userID, cid, err)
		writeError(w, r, http.StatusBadGateway, "服务暂不可用，请稍后重试")
		return
	}
	defer resp.Body.Close()
	writeTenantBridgeResponse(w, r, userID, resp)
}

// handleCopyPersonalToCompany copies the user's saved personal nickname and
// global avatar onto a company membership.
// POST /api/accounts/users/profile/copy-personal-to-company/
// Body: {"company_id": "...", "personal_nickname": "..."}
// 昵称优先取请求体（前端「个人昵称」输入框文字），缺省回退数据库已保存值；
// 头像只复制已保存的全局头像（无全局头像时保留该公司现有头像不动；
// avatarUploadDisabled 开启期间头像复制一并跳过）。
func handleCopyPersonalToCompany(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	cid := strings.TrimSpace(strField(body, "company_id"))
	if cid == "" {
		writeError(w, r, http.StatusBadRequest, "company_id 不能为空")
		return
	}

	nickname := strings.TrimSpace(strField(body, "personal_nickname"))
	dbNickname, avatar, _ := currentUserNicknameAndAvatar(userID)

	// 1. 昵称：请求体缺省时回退数据库已保存的昵称；两者皆空则跳过昵称更新
	if nickname == "" {
		nickname = dbNickname
	}
	if nickname != "" {
		resp, err := tenantForwardJSON("PATCH", "/api/internal/tenant/members/update", map[string]string{
			"user_id":     userID,
			"company_id":  cid,
			"member_name": nickname,
		})
		if err != nil {
			log.Printf("[taskAuth] copy personal → company nickname user=%s company=%s: %v", userID, cid, err)
			writeError(w, r, http.StatusBadGateway, "服务暂不可用，请稍后重试")
			return
		}
		resp.Body.Close()
	}

	// 2. 头像：仅当存在已保存的全局头像（base64 data URI）时复制；
	//    解码失败或传输失败不阻断昵称复制结果，记录日志降级。
	//    （OPT-20260806-046: 安全加固完成，头像复制已恢复；存储侧头像均为
	//    上传时经 sniffAndValidateAvatarImage 校验过的内容。
	//    安全开关开启期间头像复制一并跳过 — 与上传同熔断）
	if !avatarUploadDisabled && avatar != "" {
		avatarData, avatarCT, err := decodeAvatarDataURI(avatar)
		if err != nil {
			log.Printf("[taskAuth] copy personal → company avatar decode user=%s company=%s: %v", userID, cid, err)
		} else {
			resp, err := tenantForwardMultipart("POST", "/api/internal/tenant/members/avatar",
				map[string]string{"user_id": userID, "company_id": cid},
				"avatar", avatarCT, avatarData)
			if err != nil {
				log.Printf("[taskAuth] copy personal → company avatar user=%s company=%s: %v", userID, cid, err)
			} else {
				resp.Body.Close()
			}
		}
	}

	payload, err := buildUserProfileJSON(userID)
	if err != nil {
		log.Printf("[taskAuth] build profile after copy for user %s: %v", userID, err)
		writeError(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

// currentUserNicknameAndAvatar returns the user's saved nickname and avatar
// (base64 data URI, possibly empty) from auth_user_profile.
func currentUserNicknameAndAvatar(userID string) (nickname, avatar string, err error) {
	err = db.QueryRow(
		`SELECT COALESCE(username, ''), COALESCE(avatar, '') FROM auth_user_profile WHERE user_id = ?`,
		userID,
	).Scan(&nickname, &avatar)
	if err != nil {
		log.Printf("[taskAuth] read profile for copy user %s: %v", userID, err)
		return "", "", err
	}
	return nickname, avatar, nil
}

// decodeAvatarDataURI parses "data:<mime>;base64,<payload>" into raw bytes + mime type.
func decodeAvatarDataURI(dataURI string) ([]byte, string, error) {
	dataURI = strings.TrimSpace(dataURI)
	rest, ok := strings.CutPrefix(dataURI, "data:")
	if !ok {
		return nil, "", fmt.Errorf("not a data uri")
	}
	head, payload, ok := strings.Cut(rest, ",")
	if !ok {
		return nil, "", fmt.Errorf("malformed data uri")
	}
	mimeType := strings.TrimSuffix(head, ";base64")
	if !strings.Contains(head, ";base64") {
		return nil, "", fmt.Errorf("data uri not base64")
	}
	if _, ok := allowedAvatarTypes[mimeType]; !ok {
		return nil, "", fmt.Errorf("unsupported avatar type %q", mimeType)
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return nil, "", fmt.Errorf("base64 decode: %v", err)
	}
	if len(data) == 0 || len(data) > maxAvatarSize {
		return nil, "", fmt.Errorf("avatar payload out of range")
	}
	return data, mimeType, nil
}

// ——— taskTenantService 内部转发 helpers ———

func tenantServiceBase() string {
	return strings.TrimRight(cfg.TenantServiceURL, "/")
}

// tenantForwardRequest performs a JSON/empty-body request to the tenant service
// with the internal secret header. Returns the raw response (caller must close).
func tenantForwardRequest(method, rawURL string, payload interface{}) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	return client.Do(req)
}

// tenantForwardJSON is tenantForwardRequest for JSON payloads.
func tenantForwardJSON(method, path string, payload interface{}) (*http.Response, error) {
	return tenantForwardRequest(method, tenantServiceBase()+path, payload)
}

// tenantForwardMultipart forwards a multipart upload (file field + text fields)
// to the tenant service.
func tenantForwardMultipart(method, path string, fields map[string]string, fileField, fileCT string, fileData []byte) (*http.Response, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return nil, err
		}
	}
	fw, err := mw.CreateFormFile(fileField, fileField)
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(fileData); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, tenantServiceBase()+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	return client.Do(req)
}

// writeTenantBridgeResponse forwards the tenant service response to the caller:
// errors are passed through ({"error": ...} shape), success returns the rebuilt
// full profile so the frontend can applyProfileData directly.
func writeTenantBridgeResponse(w http.ResponseWriter, r *http.Request, userID string, resp *http.Response) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		var parsed map[string]interface{}
		msg := "操作失败"
		if json.Unmarshal(raw, &parsed) == nil {
			if m, _ := parsed["error"].(string); m != "" {
				msg = m
			}
		}
		writeError(w, r, resp.StatusCode, msg)
		return
	}
	payload, err := buildUserProfileJSON(userID)
	if err != nil {
		log.Printf("[taskAuth] build profile after company member op for user %s: %v", userID, err)
		writeError(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}
