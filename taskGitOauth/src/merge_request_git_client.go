package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	gitStatusHTTPTimeout        = 30 * time.Second
	gitStatusResponseHeaderWait = 15 * time.Second
	gitMergeHTTPTimeout         = 60 * time.Second
	gitMergeResponseHeaderWait  = 45 * time.Second
)

func isGitAPITimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "client.timeout") ||
		strings.Contains(msg, "timeout exceeded while awaiting headers") ||
		strings.Contains(msg, "context deadline exceeded")
}

func gitAPITimeoutUserError(refHost, kind string) string {
	host := strings.TrimSpace(refHost)
	if kind == "merge" {
		if host == "" {
			return "Git 站点合并未在时限内完成，请打开 MR 确认是否已合并后再试"
		}
		return fmt.Sprintf("Git 站点合并未在时限内完成（%s），请打开 MR 确认是否已合并后再试", host)
	}
	if host == "" {
		return "查询 Git 合并请求状态超时，请稍后重试"
	}
	return fmt.Sprintf("查询 Git 合并请求状态超时（%s），请稍后重试", host)
}

func writeGitAPIError(w http.ResponseWriter, host string, err error, kind string) {
	if isGitAPITimeout(err) {
		writeJSON(w, http.StatusGatewayTimeout, map[string]any{
			"detail": gitAPITimeoutUserError(host, kind),
		})
		return
	}
	writeJSON(w, http.StatusBadGateway, map[string]any{"detail": truncate(err.Error(), 300)})
}

func (a *App) gitDo(req *http.Request) (*http.Response, error) {
	return a.gitDoWithTimeout(req, gitStatusHTTPTimeout, gitStatusResponseHeaderWait)
}

func (a *App) gitDoMerge(req *http.Request) (*http.Response, error) {
	return a.gitDoWithTimeout(req, gitMergeHTTPTimeout, gitMergeResponseHeaderWait)
}

func (a *App) gitDoWithTimeout(req *http.Request, total, header time.Duration) (*http.Response, error) {
	if a.GitAPIDoFn != nil {
		return a.GitAPIDoFn(req)
	}
	started := time.Now()
	client := &http.Client{
		Timeout: total,
		Transport: &http.Transport{
			Proxy:                 nil,
			TLSHandshakeTimeout:   12 * time.Second,
			ResponseHeaderTimeout: header,
			ForceAttemptHTTP2:     false,
		},
	}
	resp, err := client.Do(req)
	elapsed := time.Since(started).Milliseconds()
	host := ""
	scheme := ""
	if req != nil && req.URL != nil {
		host = req.URL.Host
		scheme = req.URL.Scheme
	}
	method := ""
	if req != nil {
		method = req.Method
	}
	if err != nil {
		logWarn("event=merge_request_stage stage=git_http method=%s scheme=%s host=%s elapsed_ms=%d err=%v",
			method, scheme, host, elapsed, err)
		return nil, err
	}
	logInfo("event=merge_request_stage stage=git_http method=%s scheme=%s host=%s status=%d elapsed_ms=%d",
		method, scheme, host, resp.StatusCode, elapsed)
	return resp, nil
}
