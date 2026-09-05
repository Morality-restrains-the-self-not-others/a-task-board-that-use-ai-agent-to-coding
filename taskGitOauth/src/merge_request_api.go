package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

// applyGitLabAPIOrigin prefers the configured provider website origin (tenant
// BaseURL / YAML website) over html_url scheme. Self-hosted GitLab often
// advertises https://host (default port 443) while the instance only listens
// on http://host:80 — OAuth already uses Website; merge-request API must too.
func (a *App) applyGitLabAPIOrigin(ref *domain.MergeRequestRef) {
	if a == nil || ref == nil || ref.Provider != domain.MergeProviderGitLab {
		return
	}
	htmlOrigin := ref.Origin()
	pc := a.providerConfigForRepoURL(ref.HTMLURL)
	if pc == nil && ref.Host != "" {
		pc = a.providerConfigForRepoURL("http://" + ref.Host + "/")
		if pc == nil {
			pc = a.providerConfigForRepoURL("https://" + ref.Host + "/")
		}
	}
	if pc == nil {
		return
	}
	origin := infrastructure.OriginFromURL(pc.Website)
	if origin == "" || origin == htmlOrigin {
		return
	}
	ref.APIOrigin = origin
	logInfo("event=merge_request_api_origin host=%s html_origin=%s api_origin=%s",
		ref.Host, htmlOrigin, origin)
}

func gitMergeRequestStatusURL(ref domain.MergeRequestRef) string {
	if ref.Provider == domain.MergeProviderGitHub {
		return fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", ref.ProjectPath, ref.Number)
	}
	return fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d", ref.Origin(), url.PathEscape(ref.ProjectPath), ref.Number)
}

func gitMergeRequestMergeURL(ref domain.MergeRequestRef) string {
	return gitMergeRequestStatusURL(ref) + "/merge"
}

func (a *App) gitFetchMergeRequest(token string, ref domain.MergeRequestRef) (mergeRequestState, error) {
	req, err := http.NewRequest(http.MethodGet, gitMergeRequestStatusURL(ref), nil)
	if err != nil {
		return mergeRequestState{}, err
	}
	applyGitAPIAuth(req, ref, token)
	resp, err := a.gitDo(req)
	if err != nil {
		return mergeRequestState{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return mergeRequestState{}, fmt.Errorf("git status http %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return mergeRequestState{}, err
	}
	return parseMergeRequestState(ref, payload), nil
}

func gitJSONScalarPresent(v any) bool {
	if v == nil {
		return false
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	return s != "" && s != "<nil>" && !strings.EqualFold(s, "null")
}

func gitPayloadLooksMerged(payload map[string]any) bool {
	if merged, ok := payload["merged"].(bool); ok && merged {
		return true
	}
	st := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["state"])))
	if st == "merged" {
		return true
	}
	if gitJSONScalarPresent(payload["merged_at"]) {
		return true
	}
	return gitJSONScalarPresent(payload["merge_commit_sha"])
}

func parseMergeRequestState(ref domain.MergeRequestRef, payload map[string]any) mergeRequestState {
	title := strings.TrimSpace(fmt.Sprint(payload["title"]))
	if title == "<nil>" {
		title = ""
	}
	if gitPayloadLooksMerged(payload) {
		return mergeRequestState{State: "merged", Title: title}
	}
	st := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["state"])))
	if st == "closed" {
		return mergeRequestState{State: "closed", Title: title}
	}
	return mergeRequestState{State: "open", Title: title}
}

func (a *App) gitMergeMergeRequest(token string, ref domain.MergeRequestRef) error {
	var body io.Reader
	if ref.Provider == domain.MergeProviderGitHub {
		buf, _ := json.Marshal(map[string]any{"merge_method": "merge"})
		body = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(http.MethodPut, gitMergeRequestMergeURL(ref), body)
	if err != nil {
		return err
	}
	applyGitAPIAuth(req, ref, token)
	if ref.Provider == domain.MergeProviderGitHub {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.gitDoMerge(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("git merge http %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	return nil
}

func applyGitAPIAuth(req *http.Request, ref domain.MergeRequestRef, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if ref.Provider == domain.MergeProviderGitHub {
		req.Header.Set("Accept", "application/vnd.github+json")
	}
}
