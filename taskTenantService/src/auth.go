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
			writeErrorMap(w, r, http.StatusUnauthorized, map[string]interface{}{
				"error":   "unauthorized",
				"message": err.Error(),
			})
			return
		}
		r.Header.Set(gatewayauth.HeaderAuthUserID, userID)
		r.Header.Set(gatewayauth.HeaderAuthTenantID, tenantID)
		next(w, r)
	}
}

func gatewayUserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gatewayauth.ApplyGatewayUser(r, cfg.GatewayInternalSecret)
		next.ServeHTTP(w, r)
	})
}

func getAuthUser(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get(gatewayauth.HeaderAuthUserID))
}

func getAuthTenant(r *http.Request) string {
	return r.Header.Get(gatewayauth.HeaderAuthTenantID)
}

func requireAuthUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid := getAuthUser(r)
	if uid == "" {
		writeError(w, r, http.StatusUnauthorized, "请先登录")
		return "", false
	}
	return uid, true
}
