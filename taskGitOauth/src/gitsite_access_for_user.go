package main

import (
	"net/http"
	"net/url"
	"strings"
)

// headerResolvedProviderKey 内部 gitsite 换票路径把 site 解析出的 provider_key 挂到
// 请求头，供 handleAccessForUser 复用既有换取逻辑。
const headerResolvedProviderKey = "X-GitOauth-Resolved-Provider-Key"

// handleGitsiteAccessForUser 内部换票路径按 Git site 寻址（OPT-20260822-036）。
// 路径契约：/api/internal/gitsite/{site}/oauth/access-for-user/；{site} URL 编码
// （host[:port]，如 localhost%3A8012），经 ResolveProviderByGitsite 匹配 provider
// YAML website / match origin 后换取凭据。旧 github|gitlab|git-oauth 别名路径保留。
func (a *App) handleGitsiteAccessForUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/internal/gitsite/")
	parts := strings.Split(rest, "/")
	if len(parts) < 3 || parts[0] == "" || parts[1] != "oauth" || parts[2] != "access-for-user" {
		http.NotFound(w, r)
		return
	}
	site, err := url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(site) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "bad gitsite"})
		return
	}
	site = strings.ToLower(strings.TrimSpace(site))
	pc, err := a.Cfg.ResolveProviderByGitsiteWithDB(a.DB, a.Fernet, site)
	if err != nil {
		logWarn("gitsite access-for-user resolve site=%s: %v", site, err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "provider resolve failed"})
		return
	}
	if pc == nil {
		// ResolveProviderByGitsite 按 hostname 匹配；带端口的 site（localhost:8012）
		// 先退化为 hostname 再试，保证内网/测试站点命中。
		if h := gitsiteHostname(site); h != "" && h != site {
			pc, err = a.Cfg.ResolveProviderByGitsiteWithDB(a.DB, a.Fernet, h)
			if err != nil {
				logWarn("gitsite access-for-user resolve host=%s: %v", h, err)
				writeJSON(w, http.StatusInternalServerError, map[string]any{"detail": "provider resolve failed"})
				return
			}
		}
	}
	if pc == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"detail":    "not_found",
			"errDetail": "未识别的 Git 站点，请核对站点地址后重新授权",
		})
		return
	}
	r.Header.Set(headerResolvedProviderKey, pc.ProviderKey)
	a.handleAccessForUser(w, r)
}

// gitsiteHostname 从 site（host[:port] 或完整 URL）提取 hostname，供带端口站点回退匹配。
func gitsiteHostname(site string) string {
	raw := strings.TrimSpace(site)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Hostname() != "" {
		return strings.ToLower(u.Hostname())
	}
	if !strings.Contains(raw, "://") {
		if u2, err2 := url.Parse("http://" + raw); err2 == nil {
			return strings.ToLower(u2.Hostname())
		}
	}
	return ""
}
