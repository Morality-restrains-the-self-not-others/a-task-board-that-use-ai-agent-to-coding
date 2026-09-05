package main

import (
	"net/http"
	"strings"
	"time"

	"gatewayauth"
)

var authHTTP = &http.Client{Timeout: 10 * time.Second}

func verifyJWT(r *http.Request) (userID string, tenantID string, err error) {
	return gatewayauth.VerifyWithTaskAuth(r, cfg.TaskAuthURL)
}

func resolveGatewayAuthUser(r *http.Request) string {
	return gatewayauth.UserFromGatewayHeaders(r, cfg.TaskGatewayInternalSecret)
}

func getAuthUser(r *http.Request) string   { return r.Header.Get(gatewayauth.HeaderAuthUserID) }
func getAuthTenant(r *http.Request) string { return r.Header.Get(gatewayauth.HeaderAuthTenantID) }

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/health") || strings.HasPrefix(r.URL.Path, "/api/internal/") {
			next.ServeHTTP(w, r)
			return
		}
		if isContainerAgentInboundPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if gatewayauth.ApplyGatewayUser(r, cfg.TaskGatewayInternalSecret) {
			next.ServeHTTP(w, r)
			return
		}

		userID := strings.TrimSpace(r.Header.Get(gatewayauth.HeaderAuthUserID))
		if userID != "" {
			next.ServeHTTP(w, r)
			return
		}
		var tenantID string
		var err error
		userID, tenantID, err = verifyJWT(r)
		if err != nil {
			writeErrorMap(w, r, http.StatusUnauthorized, map[string]interface{}{
				"error":   "unauthorized",
				"message": err.Error(),
			})
			return
		}
		r.Header.Set(gatewayauth.HeaderAuthUserID, userID)
		if tenantID != "" {
			r.Header.Set(gatewayauth.HeaderAuthTenantID, tenantID)
		}
		next.ServeHTTP(w, r)
	})
}

func requireTaskAICommentInternalSecret(r *http.Request) bool {
	if cfg.TaskAICommentInternalSecret == "" {
		return true
	}
	return r.Header.Get("X-TaskAIComment-Internal-Secret") == cfg.TaskAICommentInternalSecret
}
