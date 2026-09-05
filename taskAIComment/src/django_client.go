package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type userProfile struct {
	ID         string
	Username   string
	Email      string
	MemberName string // workspace member display name (OPT-20260720-045)
}

func lookupUser(userID string) userProfile {
	out := userProfile{ID: userID, Username: userID, Email: ""}
	if userID == "" {
		return out
	}
	url := strings.TrimRight(cfg.TaskAuthURL, "/") + "/api/accounts/users/" + userID + "/"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return out
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := authHTTP.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return out
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	if json.Unmarshal(raw, &data) != nil {
		return out
	}
	if id, ok := data["id"].(string); ok && id != "" {
		out.ID = id
	}
	if u, ok := data["username"].(string); ok && u != "" {
		out.Username = u
	} else if login, ok := data["login"].(string); ok && login != "" {
		out.Username = login
	}
	if e, ok := data["email"].(string); ok {
		out.Email = e
	}
	if mn, ok := data["member_name"].(string); ok && mn != "" {
		out.MemberName = mn
	}
	return out
}

func lookupUsers(userIDs []string) map[string]userProfile {
	out := map[string]userProfile{}
	seen := map[string]struct{}{}
	for _, uid := range userIDs {
		if uid == "" {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		out[uid] = lookupUser(uid)
	}
	return out
}
