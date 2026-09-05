package main

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// handleContainerLayerGitRepoIdentitiesSync resolves browser repos
// (repo_url + identity_id) via taskCloudService prepare, then forwards the
// container-shaped body (repo_match_key + user_name + user_email).
func handleContainerLayerGitRepoIdentitiesSync(
	w http.ResponseWriter,
	r *http.Request,
	sc scope,
	target containerTarget,
	session validateSessionResult,
	rawBody []byte,
) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed, use POST"})
		return
	}
	ctx := r.Context()
	body := parseJSONBody(rawBody)
	layerID := strField(body, "layer_id")
	if layerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "layer_id 必填"})
		return
	}

	rawRepos, ok := body["repos"].([]any)
	if !ok || len(rawRepos) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"detail": "repos 须为非空数组，每项含 repo_url 与 identity_id",
		})
		return
	}

	preparePayload := map[string]any{
		"tenant_id": sc.TenantID,
		"user_id":   session.UserID,
		"repos":     rawRepos,
	}
	prepStatus, prepBody := cloudPostJSON(
		ctx,
		"/api/internal/layer-git-repo-identities/prepare",
		preparePayload,
		"cloud_prepare_git_repo_identities",
	)
	if prepStatus != http.StatusOK {
		writeRawJSON(w, prepStatus, prepBody)
		return
	}

	var prepared struct {
		OK     bool             `json:"ok"`
		Repos  []map[string]any `json:"repos"`
		Detail string           `json:"detail"`
	}
	if err := json.Unmarshal(prepBody, &prepared); err != nil || !prepared.OK || len(prepared.Repos) == 0 {
		if prepared.Detail != "" {
			writeJSON(w, http.StatusBadGateway, map[string]string{"detail": prepared.Detail})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"detail": "invalid layer-git-repo-identities prepare response",
		})
		return
	}

	forwardBody, err := json.Marshal(map[string]any{"repos": prepared.Repos})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "failed to encode sync body"})
		return
	}

	upstreamURL := scopedAPIRoot(target.BaseURL, sc) + "/layers/" + url.PathEscape(layerID) + "/git/repo-identities/sync"
	upStatus, upBody, _ := forwardToOnlineService(ctx, http.MethodPost, upstreamURL, target.AccessToken, forwardBody)
	writeRawJSON(w, upStatus, upBody)
}
