package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultPersonalCompanyName 与 taskEvents saas.DefaultPersonalCompanyName 对齐：
// 空展示名时的中性默认公司名。新默认形态为「{user}的公司」，误种子还靠
// member_name == company.name 覆盖。它绝不是成员昵称。
const defaultPersonalCompanyName = "我的公司"

// fetchPersonalNicknamesFn 接缝：单测可替换，避免依赖真实 taskAuth。
var fetchPersonalNicknamesFn = fetchPersonalNicknamesFromAuth

var httpClientPersonalNickname = &http.Client{Timeout: 5 * time.Second}

// isMisSeededMemberName 判断 member_name 是否被误写成公司名/默认占位符。
// 历史路径（USER_CREATED / onboarding create）曾把公司名写入 member_name，
// 导致人员管理「公司成员名称」列显示「我的公司」而非租户个人昵称。
func isMisSeededMemberName(memberName, companyName string) bool {
	name := strings.TrimSpace(memberName)
	if name == "" {
		return true
	}
	if name == defaultPersonalCompanyName {
		return true
	}
	companyName = strings.TrimSpace(companyName)
	return companyName != "" && name == companyName
}

// resolveMemberDisplayName 返回对外展示的公司成员名称：
// 误种子 → 优先个人昵称；再无则回退 userID 前缀（不回显公司名）。
func resolveMemberDisplayName(memberName, companyName, personalNickname, userID string) string {
	if !isMisSeededMemberName(memberName, companyName) {
		return strings.TrimSpace(memberName)
	}
	if n := strings.TrimSpace(personalNickname); n != "" {
		return n
	}
	return displayNameFallback("", userID)
}

// fetchPersonalNicknamesFromAuth 批量取 auth_user_profile.username（个人昵称）。
// 失败 fail-open 返回空 map，调用方继续用本地回退。
func fetchPersonalNicknamesFromAuth(userIDs []string) map[string]string {
	out := make(map[string]string, len(userIDs))
	ids := make([]string, 0, len(userIDs))
	seen := map[string]bool{}
	for _, id := range userIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return out
	}
	base := strings.TrimRight(cfg.TaskAuthURL, "/")
	if base == "" {
		base = "http://127.0.0.1:8003"
	}
	body, err := json.Marshal(map[string]interface{}{"user_ids": ids})
	if err != nil {
		return out
	}
	req, err := http.NewRequest(http.MethodPost, base+"/api/internal/users/batch/details/", bytes.NewReader(body))
	if err != nil {
		return out
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := httpClientPersonalNickname.Do(req)
	if err != nil {
		logInfo("fetchPersonalNicknames call failed: "+err.Error(), "")
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logInfo("fetchPersonalNicknames non-OK status="+itoa(resp.StatusCode), "")
		return out
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return out
	}
	var parsed struct {
		Results map[string]struct {
			Username string `json:"username"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return out
	}
	for uid, info := range parsed.Results {
		if n := strings.TrimSpace(info.Username); n != "" {
			out[uid] = n
		}
	}
	return out
}

// healMisSeededMemberName 将误种子的 member_name 回填为个人昵称（幂等）。
func healMisSeededMemberName(memberID, personalNickname string) {
	personalNickname = strings.TrimSpace(personalNickname)
	memberID = strings.TrimSpace(memberID)
	if personalNickname == "" || memberID == "" || db == nil {
		return
	}
	_, _ = db.Exec(
		`UPDATE tenant_company_member SET member_name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		personalNickname, memberID,
	)
}
