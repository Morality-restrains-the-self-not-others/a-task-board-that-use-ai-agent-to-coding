package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type creatorContact struct {
	Email string
	Phone string
}

// fetchCreatorContactsFn 接缝：单测可替换，避免依赖真实 taskAuth。
var fetchCreatorContactsFn = fetchCreatorContactsFromAuth

// searchCreatorIDsFn 接缝：按邮箱/手机/用户名搜创建者。
var searchCreatorIDsFn = searchCreatorIDsFromAuth

func creatorIDsFromCompanies(companies []companyRow) []string {
	ids := make([]string, 0, len(companies))
	seen := map[string]bool{}
	for i := range companies {
		id := strings.TrimSpace(companies[i].CreatorID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func mergeCompanyRows(base, extra []companyRow, limit int) []companyRow {
	if limit <= 0 {
		limit = 80
	}
	out := make([]companyRow, 0, limit)
	seen := map[string]bool{}
	appendUnique := func(rows []companyRow) {
		for i := range rows {
			id := strings.TrimSpace(rows[i].ID)
			if id == "" || seen[id] || len(out) >= limit {
				continue
			}
			seen[id] = true
			out = append(out, rows[i])
		}
	}
	appendUnique(base)
	appendUnique(extra)
	return out
}

func companiesByCreators(creatorIDs []string) []companyRow {
	out := make([]companyRow, 0)
	seen := map[string]bool{}
	for _, uid := range creatorIDs {
		uid = strings.TrimSpace(uid)
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		rows, err := listCompaniesByCreator(uid)
		if err != nil {
			continue
		}
		out = append(out, rows...)
	}
	return out
}

func attachCreatorContacts(item map[string]interface{}, creatorID string, contacts map[string]creatorContact) {
	c := contacts[strings.TrimSpace(creatorID)]
	item["email"] = strings.TrimSpace(c.Email)
	item["phone"] = strings.TrimSpace(c.Phone)
}

func fetchCreatorContactsFromAuth(userIDs []string) map[string]creatorContact {
	out := make(map[string]creatorContact, len(userIDs))
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
		logWarn("fetchCreatorContacts call failed: "+err.Error(), "")
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logWarn("fetchCreatorContacts non-OK status="+itoa(resp.StatusCode), "")
		return out
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return out
	}
	var parsed struct {
		Results map[string]struct {
			Email string `json:"email"`
			Phone string `json:"phone"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return out
	}
	for uid, info := range parsed.Results {
		out[uid] = creatorContact{
			Email: strings.TrimSpace(info.Email),
			Phone: strings.TrimSpace(info.Phone),
		}
	}
	return out
}

func searchCreatorIDsFromAuth(query string) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	base := strings.TrimRight(cfg.TaskAuthURL, "/")
	if base == "" {
		base = "http://127.0.0.1:8003"
	}
	u := base + "/api/internal/users/?q=" + url.QueryEscape(query) + "&limit=50"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := httpClientPersonalNickname.Do(req)
	if err != nil {
		logWarn("searchCreatorIDs call failed: "+err.Error(), "")
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logWarn("searchCreatorIDs non-OK status="+itoa(resp.StatusCode), "")
		return nil
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	var parsed struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil
	}
	out := make([]string, 0, len(parsed.Users))
	seen := map[string]bool{}
	for _, urow := range parsed.Users {
		id := strings.TrimSpace(urow.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
