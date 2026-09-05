package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const errMsgRepoIdentitiesRequired = "请为每个关联仓库选择本次运行的提交身份与授权账号"

// RepoIdentitySelection is the per-repo identity chosen on a run comment.
type RepoIdentitySelection struct {
	RepoURL           string `json:"repo_url"`
	GitIdentityID     string `json:"git_identity_id"`
	GithubUserID      string `json:"github_user_id,omitempty"`
	OauthGitsite      string `json:"oauth_gitsite,omitempty"`
	OauthRemoteUserID string `json:"oauth_remote_user_id,omitempty"`
	OauthGrantedAt    string `json:"oauth_granted_at,omitempty"`
}

func isGithubRepoURL(repoURL string) bool {
	return strings.Contains(strings.ToLower(repoURL), "github.com")
}

func ParseRepoIdentities(raw interface{}) ([]RepoIdentitySelection, error) {
	if raw == nil {
		return nil, nil
	}
	switch typed := raw.(type) {
	case []RepoIdentitySelection:
		return typed, nil
	case []map[string]interface{}:
		out := make([]RepoIdentitySelection, 0, len(typed))
		for _, item := range typed {
			out = append(out, repoIdentityFromMap(item))
		}
		return out, nil
	case []interface{}:
		out := make([]RepoIdentitySelection, 0, len(typed))
		for _, item := range typed {
			m, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("repo_identities 格式无效")
			}
			out = append(out, repoIdentityFromMap(m))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("repo_identities 格式无效")
	}
}

func repoIdentityFromMap(m map[string]interface{}) RepoIdentitySelection {
	return RepoIdentitySelection{
		RepoURL:           jsonScalarString(m["repo_url"]),
		GitIdentityID:     jsonScalarString(m["git_identity_id"]),
		GithubUserID:      jsonScalarString(m["github_user_id"]),
		OauthGitsite:      jsonScalarString(m["oauth_gitsite"]),
		OauthRemoteUserID: jsonScalarString(m["oauth_remote_user_id"]),
		OauthGrantedAt:    jsonScalarString(m["oauth_granted_at"]),
	}
}

func jsonScalarString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return strings.TrimSpace(t.String())
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		if s == "<nil>" {
			return ""
		}
		return s
	}
}

func requiredRepoURLList(requiredRepoURLs []string) []string {
	required := make([]string, 0, len(requiredRepoURLs))
	for _, url := range requiredRepoURLs {
		url = strings.TrimSpace(url)
		if url != "" {
			required = append(required, url)
		}
	}
	return required
}

func repoIdentityByURL(selections []RepoIdentitySelection) map[string]RepoIdentitySelection {
	byURL := map[string]RepoIdentitySelection{}
	for _, row := range selections {
		url := strings.TrimSpace(row.RepoURL)
		if url == "" {
			continue
		}
		byURL[url] = row
	}
	return byURL
}

func ValidateRepoIdentitiesForRun(selections []RepoIdentitySelection, requiredRepoURLs []string) error {
	required := requiredRepoURLList(requiredRepoURLs)
	if len(required) == 0 {
		return nil
	}
	byURL := repoIdentityByURL(selections)
	for _, url := range required {
		row, ok := byURL[url]
		if !ok || strings.TrimSpace(row.GitIdentityID) == "" {
			return fmt.Errorf("%s：%s", errMsgRepoIdentitiesRequired, url)
		}
		if isGithubRepoURL(url) && strings.TrimSpace(row.GithubUserID) == "" {
			return fmt.Errorf("%s：%s", errMsgRepoIdentitiesRequired, url)
		}
	}
	return nil
}

// ValidateGitIdentitiesForCreateAutoRun requires git_identity_id per linked repo when auto_run starts at create/update.
// GitHub App 账号仍按评论运行路径校验，创建自动运行只强制提交署名身份。
func ValidateGitIdentitiesForCreateAutoRun(selections []RepoIdentitySelection, requiredRepoURLs []string) error {
	required := requiredRepoURLList(requiredRepoURLs)
	if len(required) == 0 {
		return nil
	}
	byURL := repoIdentityByURL(selections)
	for _, url := range required {
		row, ok := byURL[url]
		if !ok || strings.TrimSpace(row.GitIdentityID) == "" {
			return fmt.Errorf("%s：%s", errMsgRepoIdentitiesRequired, url)
		}
	}
	return nil
}

// validateGitIdentitiesOwnership (OPT-20260821-022) 校验每个 git_identity_id 属于当前用户/租户，
// 防止伪造他人身份 ID 写入自动运行的克隆/提交署名。
func validateGitIdentitiesOwnership(selections []RepoIdentitySelection, userID, tenantID string) error {
	userID = strings.TrimSpace(userID)
	seen := map[string]bool{}
	for _, sel := range selections {
		gid := strings.TrimSpace(sel.GitIdentityID)
		if gid == "" {
			continue
		}
		if seen[gid] {
			continue
		}
		seen[gid] = true
		row, err := loadGitIdentityByID(gid)
		if err != nil {
			return fmt.Errorf("校验 Git 身份失败: %v", err)
		}
		if row == nil {
			return fmt.Errorf("Git 身份不存在")
		}
		if userID != "" && row.UserID != userID {
			return fmt.Errorf("Git 身份不属于当前用户")
		}
		if strings.TrimSpace(tenantID) != "" && row.CompanyID != strings.TrimSpace(tenantID) {
			return fmt.Errorf("Git 身份不属于当前租户")
		}
	}
	return nil
}

func collectLinkedRepoURLsFromBody(ctx context.Context, tenantID string, body map[string]interface{}) []string {
	if body == nil {
		return nil
	}
	projs, ok := body["projects"].([]interface{})
	if !ok {
		return nil
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, raw := range projs {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		pid := strField(item, "project_id")
		if pid == "" {
			continue
		}
		repoIndex := intFromField(item, "repo_index")
		urls, _ := getProjectRepoURLs(ctx, tenantID, pid)
		addr := ""
		if len(urls) > 0 {
			if repoIndex >= 0 && repoIndex < len(urls) {
				addr = urls[repoIndex]
			} else {
				addr = urls[0]
			}
		}
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		if _, dup := seen[addr]; dup {
			continue
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
	}
	return out
}

func resolveAutoRunRepoIdentities(ctx context.Context, autoRun bool, userID, tenantID, taskID string, body map[string]interface{}) ([]RepoIdentitySelection, error) {
	if !autoRun {
		return nil, nil
	}
	required := collectLinkedRepoURLsFromBody(ctx, tenantID, body)
	if len(required) == 0 && strings.TrimSpace(taskID) != "" {
		required = loadTaskLinkedRepoURLs(taskID)
	}
	selections, err := ParseRepoIdentities(body["repo_identities"])
	if err != nil {
		return nil, err
	}
	if err := ValidateGitIdentitiesForCreateAutoRun(selections, required); err != nil {
		return nil, err
	}
	// OPT-20260821-022: 只校验非空不能阻止把他人署名写进自动运行克隆/提交。
	if err := validateGitIdentitiesOwnership(selections, userID, tenantID); err != nil {
		return nil, err
	}
	return selections, nil
}

func RepoIdentitiesJSON(selections []RepoIdentitySelection) string {
	if len(selections) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(selections)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func repoIdentitiesFromJSONColumn(raw string) []map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []map[string]interface{}{}
	}
	var out []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return []map[string]interface{}{}
	}
	return out
}

func resolveCommentRepoIdentitiesJSON(body map[string]interface{}, required []string, hasMentions bool) (string, error) {
	selections, err := ParseRepoIdentities(body["repo_identities"])
	if err != nil {
		return "", err
	}
	if hasMentions {
		if err := ValidateRepoIdentitiesForRun(selections, required); err != nil {
			return "", err
		}
	}
	return RepoIdentitiesJSON(selections), nil
}

func loadTaskLinkedRepoURLs(taskID string) []string {
	rows, err := db.Query(
		`SELECT COALESCE(repo_address,'') FROM task_projects WHERE task_id=? ORDER BY created_at, id`,
		taskID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []string{}
	seen := map[string]struct{}{}
	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err != nil {
			continue
		}
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		if _, ok := seen[addr]; ok {
			continue
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
	}
	return out
}

func loadCommentRepoIdentitiesJSON(taskID, commentID string) (jsonText string, found bool, err error) {
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return "", false, nil
	}
	err = db.QueryRow(
		`SELECT COALESCE(repo_identities_json,'') FROM task_comments WHERE id=? AND task_id=?`,
		commentID, taskID,
	).Scan(&jsonText)
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(jsonText), true, nil
}

func repoIdentityMapsFromJSON(raw string) []map[string]string {
	parsed := repoIdentitiesFromJSONColumn(raw)
	out := make([]map[string]string, 0, len(parsed))
	for _, item := range parsed {
		repoURL := jsonScalarString(item["repo_url"])
		gid := jsonScalarString(item["git_identity_id"])
		oauthSite := jsonScalarString(item["oauth_gitsite"])
		if repoURL == "" && gid == "" && oauthSite == "" {
			continue
		}
		row := map[string]string{
			"repo_url":        repoURL,
			"git_identity_id": gid,
		}
		if gh := jsonScalarString(item["github_user_id"]); gh != "" {
			row["github_user_id"] = gh
		}
		if oauthSite != "" {
			row["oauth_gitsite"] = oauthSite
		}
		if v := jsonScalarString(item["oauth_remote_user_id"]); v != "" {
			row["oauth_remote_user_id"] = v
		}
		if v := jsonScalarString(item["oauth_granted_at"]); v != "" {
			row["oauth_granted_at"] = v
		}
		out = append(out, row)
	}
	return out
}

func enrichRepoIdentityMapsWithGitUsers(maps []map[string]string) []map[string]string {
	if db == nil || len(maps) == 0 {
		return maps
	}
	for i, row := range maps {
		gid := strings.TrimSpace(row["git_identity_id"])
		if gid == "" {
			continue
		}
		var name, email, uid string
		err := db.QueryRow(
			`SELECT COALESCE(git_user_name,''), COALESCE(git_user_email,''), COALESCE(user_id,'')
			 FROM task_git_identities WHERE id=? LIMIT 1`,
			gid,
		).Scan(&name, &email, &uid)
		if err != nil {
			continue
		}
		if n := strings.TrimSpace(name); n != "" {
			maps[i]["user_name"] = n
		}
		if e := strings.TrimSpace(email); e != "" {
			maps[i]["user_email"] = e
		}
		if u := strings.TrimSpace(uid); u != "" {
			maps[i]["user_id"] = u
		}
	}
	return maps
}
