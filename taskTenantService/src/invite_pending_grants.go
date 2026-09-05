package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"authz"
)

// pendingGrant is the invite-time page/region grant persisted on tenant_invitation.
type pendingGrant struct {
	GroupKey string `json:"group_key"`
	Effect   string `json:"effect"`
}

const maxPendingGrants = 200

// parseInviteGrants extracts optional grants from invite JSON body.
// Invalid effect / empty group_key / oversize → error.
func parseInviteGrants(body map[string]interface{}) ([]pendingGrant, error) {
	raw, ok := body["grants"]
	if !ok || raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("grants 必须为数组")
	}
	if len(arr) > maxPendingGrants {
		return nil, fmt.Errorf("grants 最多 %d 条", maxPendingGrants)
	}
	out := make([]pendingGrant, 0, len(arr))
	seen := map[string]string{}
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("grants 元素必须为对象")
		}
		key := strings.TrimSpace(strField(m, "group_key"))
		if key == "" {
			return nil, fmt.Errorf("grants.group_key 不能为空")
		}
		effect := authz.NormalizeGrantEffect(strField(m, "effect"))
		if prev, ok := seen[key]; ok {
			seen[key] = authz.MaxGrantEffect(prev, effect)
			continue
		}
		seen[key] = effect
	}
	for k, e := range seen {
		out = append(out, pendingGrant{GroupKey: k, Effect: e})
	}
	return out, nil
}

func encodePendingGrantsJSON(grants []pendingGrant) (interface{}, error) {
	if len(grants) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(grants)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

const maxPendingRoleNames = 20

// parseInviteRoleNames extracts optional reusable-role names from invite JSON body.
// role_names must be a JSON array of non-empty strings; duplicates are deduped
// (trimmed). Missing/null → nil. Any structural violation is an explicit error
// (no silent fallback).
func parseInviteRoleNames(body map[string]interface{}) ([]string, error) {
	raw, ok := body["role_names"]
	if !ok || raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("role_names 必须为数组")
	}
	if len(arr) > maxPendingRoleNames {
		return nil, fmt.Errorf("role_names 最多 %d 个", maxPendingRoleNames)
	}
	out := make([]string, 0, len(arr))
	seen := map[string]bool{}
	for _, item := range arr {
		s, ok := item.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return nil, fmt.Errorf("role_names 元素必须为非空字符串")
		}
		s = strings.TrimSpace(s)
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out, nil
}

func encodePendingRoleNamesJSON(names []string) (interface{}, error) {
	if len(names) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(names)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func decodePendingRoleNamesJSON(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || raw == "[]" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	cleaned := make([]string, 0, len(out))
	seen := map[string]bool{}
	for _, n := range out {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		cleaned = append(cleaned, n)
	}
	return cleaned, nil
}

func decodePendingGrantsJSON(raw string) ([]pendingGrant, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" || raw == "[]" {
		return nil, nil
	}
	var out []pendingGrant
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	cleaned := make([]pendingGrant, 0, len(out))
	for _, g := range out {
		key := strings.TrimSpace(g.GroupKey)
		if key == "" {
			continue
		}
		cleaned = append(cleaned, pendingGrant{
			GroupKey: key,
			Effect:   authz.NormalizeGrantEffect(g.Effect),
		})
	}
	return cleaned, nil
}

// applyPendingGrantsViaAuth creates a custom access role in taskAuth and returns role name.
// Failures are returned so join can log and continue without blocking membership.
func applyPendingGrantsViaAuth(companyID, displayName string, grants []pendingGrant) (roleName string, err error) {
	if len(grants) == 0 {
		return "", nil
	}
	base := strings.TrimRight(authServiceURL(), "/")
	if base == "" {
		return "", fmt.Errorf("auth service not configured")
	}
	payload := map[string]interface{}{
		"company_id":   companyID,
		"display_name": displayName,
		"grants":       grants,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, base+"/api/internal/authz/apply-member-grants/", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := httpClientRoleCheck.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("auth apply-member-grants status=%d body=%s", resp.StatusCode, string(raw))
	}
	var out struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Name == "" {
		return "", fmt.Errorf("auth apply-member-grants missing name")
	}
	return out.Name, nil
}

func accessRoleDisplayNameForInvite(memberName string) string {
	label := strings.TrimSpace(memberName)
	if label == "" {
		label = "未命名"
	}
	name := "访问·" + label
	if len([]rune(name)) > 64 {
		runes := []rune(name)
		name = string(runes[:64])
	}
	return name
}
