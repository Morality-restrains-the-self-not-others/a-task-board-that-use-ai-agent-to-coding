package main

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"taskGitOauth/domain"
)

const gitlabReachabilityTimeout = 4 * time.Second

func (a *App) handleTenantGitlabReachability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tid := extractTenantIDFromPath(r.URL.Path)
	if tid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "missing tenant id"})
		return
	}
	if !a.ensureTenantMember(w, r, tid) {
		return
	}
	writeJSON(w, http.StatusOK, a.tenantGitlabReachabilityJSON(r, tid))
}

func (a *App) tenantGitlabReachabilityJSON(r *http.Request, tenantID string) map[string]any {
	row, err := a.DB.GetTenantGitLabConnection(tenantID)
	if err != nil {
		logWarn("event=gitlab_reachability stage=load tenant_id=%s err=%v", tenantID, err)
		return map[string]any{
			"configured": false,
			"intranet":   false,
			"status":     domain.ReachabilityUnconfigured,
			"reachable":  nil,
			"reason":     "load_failed",
		}
	}
	configured := row != nil
	intranet := row != nil && row.Intranet
	baseURL := ""
	if row != nil {
		baseURL = row.BaseURL
	}
	skip, status := domain.ClassifyReachability(configured, intranet)
	out := map[string]any{
		"configured":  configured,
		"intranet":    intranet,
		"base_url":    baseURL,
		"status":      status,
		"reachable":   nil,
		"reason":      "",
		"http_status": 0,
		"latency_ms":  0,
	}
	if skip {
		if status == domain.ReachabilitySkippedIntranet {
			out["reason"] = "intranet"
		}
		logInfo("event=gitlab_reachability tenant_id=%s status=%s intranet=%v skip=true", tenantID, status, intranet)
		return out
	}
	// SSRF: never probe a client-supplied URL query; only the stored base_url.
	_ = r.URL.Query().Get("url")
	result := a.probeStoredGitLab(baseURL)
	out["status"] = result.status
	out["reason"] = result.reason
	out["http_status"] = result.httpStatus
	out["latency_ms"] = result.latencyMS
	if result.status == domain.ReachabilityReachable {
		out["reachable"] = true
	} else {
		out["reachable"] = false
	}
	if result.status == domain.ReachabilityUnreachable {
		logWarn("event=gitlab_reachability tenant_id=%s status=%s reason=%s latency_ms=%d", tenantID, result.status, result.reason, result.latencyMS)
	} else {
		logInfo("event=gitlab_reachability tenant_id=%s status=%s reason=%s http_status=%d latency_ms=%d", tenantID, result.status, result.reason, result.httpStatus, result.latencyMS)
	}
	return out
}

type gitlabProbeResult struct {
	status     string
	reason     string
	httpStatus int
	latencyMS  int64
}

func (a *App) probeStoredGitLab(baseURL string) gitlabProbeResult {
	target := domain.ProbeTargetURL(baseURL)
	if target == "" {
		return gitlabProbeResult{status: domain.ReachabilityUnreachable, reason: "empty_base_url"}
	}
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return gitlabProbeResult{status: domain.ReachabilityUnreachable, reason: "bad_url"}
	}
	req.Header.Set("User-Agent", "daydaymoney-gitlab-reachability/1")
	req.Header.Set("Accept", "*/*")
	started := time.Now()
	resp, err := a.probeDo(req)
	latency := time.Since(started).Milliseconds()
	if err != nil {
		return gitlabProbeResult{
			status:    domain.ReachabilityUnreachable,
			reason:    classifyProbeNetError(err),
			latencyMS: latency,
		}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return gitlabProbeResult{
		status:     domain.ReachabilityReachable,
		reason:     "http",
		httpStatus: resp.StatusCode,
		latencyMS:  latency,
	}
}

func (a *App) probeDo(req *http.Request) (*http.Response, error) {
	if a.GitAPIDoFn != nil {
		return a.GitAPIDoFn(req)
	}
	client := &http.Client{
		Timeout: gitlabReachabilityTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			Proxy: nil,
			DialContext: (&net.Dialer{
				Timeout: 3 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   3 * time.Second,
			ResponseHeaderTimeout: 3 * time.Second,
			ForceAttemptHTTP2:     false,
		},
	}
	return client.Do(req)
}

func classifyProbeNetError(err error) string {
	if err == nil {
		return ""
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "timeout"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "connection refused"):
		return "refused"
	case strings.Contains(msg, "no such host"):
		return "dns"
	default:
		return "network"
	}
}
