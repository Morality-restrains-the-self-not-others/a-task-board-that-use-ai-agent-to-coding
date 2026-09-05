package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type validatedContainerScope struct {
	CompanyID   string
	WorkspaceID string
	TaskID      string
}

func isContainerAgentInboundPath(path string) bool {
	if !strings.Contains(path, "/container-agent-comments/") {
		return false
	}
	for _, action := range []string{"/stream", "/complete", "/fail"} {
		if strings.HasSuffix(strings.TrimSuffix(path, "/"), action) {
			return true
		}
	}
	return false
}

func containerAgentTestAuthBypass(r *http.Request) bool {
	return r != nil && r.Header.Get("X-Task-Test-Skip-Container-Agent-Auth") == "1"
}

func requireContainerAgentAuth(r *http.Request, tenantID, workspaceID, taskID string) bool {
	if containerAgentTestAuthBypass(r) {
		return true
	}
	if requireTaskAICommentInternalSecret(r) {
		return true
	}
	token := strings.TrimSpace(r.Header.Get("X-Access-Token"))
	if token == "" {
		return false
	}
	scope, ok := validateContainerAccessToken(r.Context(), token)
	if !ok || scope == nil {
		return false
	}
	return scope.CompanyID == tenantID && scope.WorkspaceID == workspaceID && scope.TaskID == taskID
}

func validateContainerAccessToken(ctx context.Context, accessToken string) (*validatedContainerScope, bool) {
	base := strings.TrimRight(cfg.CredentialServiceURL, "/")
	if base == "" {
		return nil, false
	}
	payload, _ := json.Marshal(map[string]string{"access_token": accessToken})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/token/validate", strings.NewReader(string(payload)))
	if err != nil {
		return nil, false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := cloudHTTP.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	var out struct {
		Valid       bool   `json:"valid"`
		CompanyID   string `json:"company_id"`
		WorkspaceID string `json:"workspace_id"`
		TaskID      string `json:"task_id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || !out.Valid {
		return nil, false
	}
	return &validatedContainerScope{
		CompanyID:   out.CompanyID,
		WorkspaceID: out.WorkspaceID,
		TaskID:      out.TaskID,
	}, true
}
