package main

import (
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

// djangoHTTP is a shared HTTP client for internal service-to-service calls.
// Originally defined in django_client.go (removed, OPT-052). Retained because
// container_gateway_proxy, server_release_reconcile, container_inbound_actions,
// and container_inbound still use it for outbound requests.
var djangoHTTP = &http.Client{Timeout: 30 * time.Second}

// copyResponseHeaders copies non-Content-Length headers from resp to w.
// Used by container gateway proxy (originally shared with django_client.go, OPT-052).
func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	for k, vals := range resp.Header {
		if strings.EqualFold(k, "Content-Length") {
			continue
		}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
}

func isContainerOutboundComputeSub(sub string) bool {
	// container-job-*（含 edit-run L1 复合编排）与 layer/git-push 均代理至 taskContainerGateway。
	prefixes := []string{
		"compute/container-layer-",
		"compute/container-clone-log",
		"compute/container-bootstrap-clone-log",
		"compute/container-layers-empty-root",
		"compute/container-auto-run-steps",
		"compute/container-job-",
		"compute/container-git-identity-sync",
		"compute/container-task-lifecycle-",
	}
	for _, prefix := range prefixes {
		if sub == strings.TrimSuffix(prefix, "-") || strings.HasPrefix(sub, prefix) {
			return true
		}
	}
	return false
}

func proxyContainerGatewayRequest(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimRight(cfg.ContainerGatewayURL, "/")
	if base == "" {
		writeErrorMapJSON(w, r, http.StatusBadGateway, map[string]interface{}{
			"status":  "error",
			"message": "taskContainerGateway not configured",
		})
		return
	}
	target := base + r.URL.RequestURI()
	req, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	copyProxyHeaders(r, req)
	resp, err := djangoHTTP.Do(req)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusBadGateway, map[string]interface{}{
			"status":  "error",
			"message": "container gateway proxy failed: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func copyProxyHeaders(src *http.Request, dst *http.Request) {
	for _, key := range []string{
		"Authorization",
		"X-User-Id",
		"X-Auth-User-Id",
		"X-Auth-Tenant-Id",
		"X-Gateway-Auth-Verified",
		"X-TaskGateway-Internal-Secret",
		"Content-Type",
		"Accept",
	} {
		if v := strings.TrimSpace(src.Header.Get(key)); v != "" {
			dst.Header.Set(key, v)
		}
	}
	userID := getAuthUser(src)
	if userID != "" && dst.Header.Get("X-User-Id") == "" {
		dst.Header.Set("X-User-Id", userID)
	}
	if cfg.GatewayInternalSecret != "" && dst.Header.Get("X-TaskGateway-Internal-Secret") == "" {
		dst.Header.Set("X-Gateway-Auth-Verified", "1")
		dst.Header.Set("X-TaskGateway-Internal-Secret", cfg.GatewayInternalSecret)
	}
	// Container Gateway 内部旁路头与 APISIX 的 X-TaskGateway-Internal-Secret 不是同一把钥匙。
	// 漏发此头时 authorizeContainerRequest 会走 taskAuth validate-session（Token IP 绑定），
	// 浏览器 GET 层图被 401，ztree 无法水合。
	if sec := strings.TrimSpace(cfg.ContainerGatewayInternalSecret); sec != "" {
		if dst.Header.Get("X-TaskContainerGateway-Internal-Secret") == "" {
			dst.Header.Set("X-TaskContainerGateway-Internal-Secret", sec)
		}
	}
	// Propagate full span context so taskContainerGateway strict middleware
	// does not reject X-Trace-Id-only upstream requests with HTTP 400.
	tracelog.ApplyOutboundHeaders(dst, src.Context())
}
