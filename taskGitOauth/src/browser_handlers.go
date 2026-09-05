package main

import (
	"fmt"
	"net/http"
	"strings"

	"taskGitOauth/domain"
)

const (
	githubSessionKey = "gitoauth_github_ctx"
	gitlabSessionKey = "gitoauth_gitlab_ctx"
	githubProvider   = "github"
	gitlabProvider   = "gitlab"
)

func wantsJSONOAuthStart(r *http.Request) bool {
	accept := strings.TrimSpace(r.Header.Get("Accept"))
	if accept == "" {
		return true
	}
	return strings.Contains(accept, "application/json")
}

func writeOAuthStartResponse(w http.ResponseWriter, r *http.Request, authorizeURL string) {
	if wantsJSONOAuthStart(r) {
		writeJSON(w, http.StatusOK, map[string]any{
			"authorize_url": authorizeURL,
		})
		return
	}
	http.Redirect(w, r, authorizeURL, http.StatusFound)
}

func (a *App) handleServiceProviderCallback(w http.ResponseWriter, r *http.Request, sp string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	spNorm := strings.TrimSpace(strings.ToLower(sp))
	if spNorm == "tenant-gitlab" {
		a.handleGitlabCallbackWithSP(w, r, "tenant-gitlab")
		return
	}
	if _, ok := domain.ParseTenantCompanyID(spNorm); ok {
		a.handleGitlabCallbackWithSP(w, r, spNorm)
		return
	}
	cfg, err := a.resolveProviderByServiceProvider(sp)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"detail":           "service_provider 在多个 provider 下重复，无法唯一路由",
			"service_provider": sp,
		})
		return
	}
	if cfg == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"detail":           "未找到 service_provider 对应的 OAuth 配置",
			"service_provider": sp,
		})
		return
	}
	switch cfg.Provider {
	case "github":
		a.handleGithubCallbackWithSP(w, r, sp)
	case "gitlab":
		a.handleGitlabCallbackWithSP(w, r, sp)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{
			"detail":           fmt.Sprintf("provider '%s' 暂不支持 OAuth 回调", cfg.Provider),
			"service_provider": sp,
			"provider":         cfg.Provider,
		})
	}
}

func (a *App) gatewayStartError(w http.ResponseWriter, provider, code, reason string) {
	detail := fmt.Sprintf("无法启动 %s 授权（%s）", provider, code)
	if a.Cfg.Debug && reason != "" {
		detail = detail + " [debug: " + reason + "]"
	}
	writeJSON(w, http.StatusServiceUnavailable, map[string]any{"detail": detail})
}

func parsePositiveInt(s string) (int64, error) {
	n, err := strconvParseInt(s)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("bad")
	}
	return n, nil
}

func strconvParseInt(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("bad")
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}
