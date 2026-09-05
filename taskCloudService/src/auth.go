package main

import (
	"net/http"
	"strings"

	"gatewayauth"
)

func verifyJWT(r *http.Request) (userID string, tenantID string, err error) {
	return gatewayauth.VerifyWithTaskAuth(r, cfg.TaskAuthURL)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, tenantID, err := verifyJWT(r)
		if err != nil {
			writeErrorMapJSON(w, r, http.StatusUnauthorized, map[string]interface{}{"error": "unauthorized", "message": err.Error()})
			return
		}
		r.Header.Set(gatewayauth.HeaderAuthUserID, userID)
		r.Header.Set(gatewayauth.HeaderAuthTenantID, tenantID)
		next(w, r)
	}
}

// gatewayUserMiddleware promotes verified APISIX forward-auth identity
// (X-User-Id + gateway secret) into X-Auth-User-Id. It does not reject
// unauthenticated requests — handlers decide when a user is required.
//
// 回归：github-credential-status 等依赖 getAuthUser 的路径若缺少本中间件，
// 网关只注入 X-User-Id 时会得到空 github_connections，任务详情 GitHub 账号下拉为空。
func gatewayUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gatewayauth.ApplyGatewayUser(r, cfg.GatewayInternalSecret)
		next.ServeHTTP(w, r)
	})
}

// getAuthUser returns the canonical user id after gatewayUserMiddleware
// (or tests / JWT middleware that set X-Auth-User-Id directly).
func getAuthUser(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get(gatewayauth.HeaderAuthUserID))
}

func getAuthTenant(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get(gatewayauth.HeaderAuthTenantID))
}
