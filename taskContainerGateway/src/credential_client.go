package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

type tokenInitResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    string
}

func credentialServiceURL() string {
	base := strings.TrimRight(strings.TrimSpace(cfg.CredentialServiceURL), "/")
	if base == "" {
		return "http://127.0.0.1:8015"
	}
	return base
}

func credentialTokenInitURL(sc scope) string {
	cid := strings.TrimSpace(sc.CommentID)
	if cid == "" || cid == "-" {
		return ""
	}
	return fmt.Sprintf(
		"%s/v1/token/init/tenant/%s/workspace/%s/task/%s/comment/%s",
		credentialServiceURL(),
		sc.TenantID,
		sc.WorkspaceID,
		sc.TaskID,
		cid,
	)
}

func credentialTokenInit(ctx context.Context, sc scope) (tokenInitResult, int, []byte) {
	url := credentialTokenInitURL(sc)
	if url == "" {
		body, _ := json.Marshal(map[string]string{
			"detail":     "comment_id is required",
			"error_code": "COMMENT_ID_REQUIRED",
		})
		return tokenInitResult{}, http.StatusBadRequest, body
	}
	body := []byte("{}")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		body, _ := json.Marshal(map[string]string{"detail": fmt.Sprintf("token-init request error: %v", err)})
		return tokenInitResult{}, http.StatusBadGateway, body
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		body, _ := json.Marshal(map[string]string{"detail": fmt.Sprintf("token-init unreachable: %v", err)})
		return tokenInitResult{}, http.StatusBadGateway, body
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		body, _ := json.Marshal(map[string]string{"detail": "failed to read token-init response"})
		return tokenInitResult{}, http.StatusBadGateway, body
	}
	if resp.StatusCode != http.StatusOK {
		if len(raw) == 0 {
			raw, _ = json.Marshal(map[string]string{"detail": "token-init failed"})
		}
		return tokenInitResult{}, resp.StatusCode, raw
	}
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresAt    string `json:"expires_at"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		body, _ := json.Marshal(map[string]string{"detail": "invalid token-init response"})
		return tokenInitResult{}, http.StatusBadGateway, body
	}
	access := strings.TrimSpace(parsed.AccessToken)
	if access == "" {
		body, _ := json.Marshal(map[string]string{"detail": "token-init returned empty access_token"})
		return tokenInitResult{}, http.StatusBadGateway, body
	}
	return tokenInitResult{
		AccessToken:  access,
		RefreshToken: strings.TrimSpace(parsed.RefreshToken),
		ExpiresAt:    strings.TrimSpace(parsed.ExpiresAt),
	}, http.StatusOK, nil
}

// credentialRepoCloneCredentials calls CRED repo-clone-credentials with a container access token.
func credentialRepoCloneCredentials(ctx context.Context, sc scope, accessToken string) (int, []byte) {
	cid := strings.TrimSpace(sc.CommentID)
	if cid == "" || cid == "-" {
		body, _ := json.Marshal(map[string]string{
			"detail":     "comment_id is required",
			"error_code": "COMMENT_ID_REQUIRED",
		})
		return http.StatusBadRequest, body
	}
	payload, _ := json.Marshal(map[string]string{"access_token": accessToken})
	url := fmt.Sprintf(
		"%s/api/tenant/%s/workspace/%s/task/%s/comment/%s/cloud/server-container-token/repo-clone-credentials/",
		credentialServiceURL(),
		sc.TenantID,
		sc.WorkspaceID,
		sc.TaskID,
		cid,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		body, _ := json.Marshal(map[string]string{"detail": fmt.Sprintf("repo-clone-credentials request error: %v", err)})
		return http.StatusBadGateway, body
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		body, _ := json.Marshal(map[string]string{"detail": fmt.Sprintf("repo-clone-credentials unreachable: %v", err)})
		return http.StatusBadGateway, body
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		body, _ := json.Marshal(map[string]string{"detail": "failed to read repo-clone-credentials response"})
		return http.StatusBadGateway, body
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	return resp.StatusCode, raw
}

const task2appAccessTokenPlaceholder = "__TASK2APP_ACCESS_TOKEN__"

func accessTokenNeedsServerIssue(token string) bool {
	value := strings.TrimSpace(token)
	if value == "" {
		return true
	}
	upper := strings.ToUpper(value)
	switch upper {
	case strings.ToUpper(task2appAccessTokenPlaceholder),
		"TASK2APP_ACCESS_TOKEN",
		"$TASK2APP_ACCESS_TOKEN",
		"${TASK2APP_ACCESS_TOKEN}",
		"$ACCESS_TOKEN",
		"${ACCESS_TOKEN}":
		return true
	default:
		return false
	}
}
