package saas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// EnsureCreatorMember 幂等确保创建者成员行存在。
// 自愈路径（OPT-20260806-013）：公司已存在但成员行缺失时由 user_created
// 处理器重试时调用，修复「用户有公司无成员」状态。
// 已有成员行时绝不覆盖 member_name（避免把正确昵称写回公司名）。
func (r *Repository) EnsureCreatorMember(companyID int64, userID string) error {
	exists, err := r.memberExistsInCompany(fmt.Sprint(companyID), userID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	memberName := r.fetchPersonalNickname(userID)
	return r.createCompanyMember(fmt.Sprint(companyID), userID, memberName, true)
}

// memberExistsInCompany checks whether user already has a membership row for company.
func (r *Repository) memberExistsInCompany(companyID, userID string) (bool, error) {
	tenantURL := taskTenantServiceURL()
	req, err := http.NewRequest(http.MethodGet,
		tenantURL+"/api/internal/tenant/members?user_id="+url.QueryEscape(userID), nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("X-Internal-Secret", tenantServiceSecret())
	resp, err := r.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return false, fmt.Errorf("members lookup status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var list []map[string]interface{}
	if err := json.Unmarshal(raw, &list); err != nil {
		return false, err
	}
	for _, m := range list {
		if fmt.Sprint(m["company_id"]) == companyID {
			return true, nil
		}
	}
	return false, nil
}

// fetchPersonalNickname reads auth_user_profile.username via taskAuth batch/details.
// Fail-open: errors / empty → ""（宁可空昵称，也不把公司名写入 member_name）。
func (r *Repository) fetchPersonalNickname(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	taskAuthURL := strings.TrimSpace(os.Getenv("TASK_AUTH_INTERNAL_URL"))
	if taskAuthURL == "" {
		taskAuthURL = "http://127.0.0.1:8003"
	}
	taskAuthURL = strings.TrimRight(taskAuthURL, "/")
	body, err := json.Marshal(map[string]interface{}{"user_ids": []string{userID}})
	if err != nil {
		return ""
	}
	req, err := http.NewRequest(http.MethodPost, taskAuthURL+"/api/internal/users/batch/details/", bytes.NewReader(body))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	sec := r.secret
	if sec == "" {
		sec = taskEventsSecret()
	}
	if sec != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", sec)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		log.Printf("[fetch_personal_nickname] call failed user_id=%s err=%v", userID, err)
		return ""
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("[fetch_personal_nickname] non-OK user_id=%s status=%d", userID, resp.StatusCode)
		return ""
	}
	var parsed struct {
		Results map[string]struct {
			Username string `json:"username"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ""
	}
	if info, ok := parsed.Results[userID]; ok {
		return strings.TrimSpace(info.Username)
	}
	return ""
}

// createCompanyMember calls taskTenantService POST /api/internal/tenant/members
// to insert an admin member row into tenant_company_member.
func (r *Repository) createCompanyMember(companyID, userID, memberName string, isAdmin bool) error {
	body := map[string]interface{}{
		"user_id":     userID,
		"company_id":  companyID,
		"is_admin":    isAdmin,
		"is_active":   true,
		"member_name": memberName,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	tenantURL := taskTenantServiceURL()
	req, err := http.NewRequest(http.MethodPost,
		tenantURL+"/api/internal/tenant/members", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", tenantServiceSecret())
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respRaw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(respRaw)))
	}
	return nil
}
