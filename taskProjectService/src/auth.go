package main

import (
	"net/http"
	"strings"

	"gatewayauth"
)

// verifyJWT calls taskAuth to validate the token and returns user_id + tenant_id.
func verifyJWT(r *http.Request) (userID string, tenantID string, err error) {
	return gatewayauth.VerifyWithTaskAuth(r, cfg.TaskAuthURL)
}

// authMiddleware wraps a handler with JWT verification.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, tenantID, err := verifyJWT(r)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, err.Error())
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
	return r.Header.Get(gatewayauth.HeaderAuthTenantID)
}

// requireAuthUser ensures the request carries a verified user identity.
// Returns the user ID and true; writes 401 and returns false on failure.
func requireAuthUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid := getAuthUser(r)
	if uid == "" || uid == "internal" {
		writeError(w, r, http.StatusUnauthorized, "请先登录")
		return "", false
	}
	return uid, true
}

// isInternalCall returns true when the request originates from a trusted
// internal service (X-Auth-User-Id == "internal").
func isInternalCall(r *http.Request) bool {
	return strings.TrimSpace(r.Header.Get(gatewayauth.HeaderAuthUserID)) == "internal"
}

// requireTenantMember verifies that the user is a member of the given tenant
// by calling taskTenantService's internal resolve endpoint. Internal service-
// to-service calls bypass this check. When TaskTenantURL is not configured
// (e.g. test/dev environments), the check is skipped with a warning.
// Returns true if the user is a member; writes 401/403/502 and returns false otherwise.
func requireTenantMember(w http.ResponseWriter, r *http.Request, userID, tenantID string) bool {
	if isInternalCall(r) {
		return true
	}
	if userID == "" {
		if r.Header.Get(gatewayauth.HeaderGatewayVerified) == "1" {
			logWarn(
				"gateway identity not promoted; X-TaskGateway-Internal-Secret does not match conf-local gatewayInternalSecret",
				r.Header.Get("X-Trace-Id"),
			)
		}
		writeError(w, r, http.StatusUnauthorized, "请先登录")
		return false
	}
	// Fast path: gateway-embedded membership JWT (avoids HTTP round-trip)
	if claims, ok := gatewayauth.ParseTenantClaims(r, cfg.GatewayInternalSecret); ok {
		if claims.Subject == userID && gatewayauth.IsTenantMember(claims, tenantID) {
			return true
		}
	}

	// Skip membership check when tenant service URL is not configured (tests / dev)
	if strings.TrimSpace(cfg.TaskTenantURL) == "" {
		return true
	}
	m, err := tenantResolveMember(tenantID, userID)
	if err != nil {
		writeError(w, r, http.StatusBadGateway, "租户成员校验失败")
		return false
	}
	if m == nil {
		writeError(w, r, http.StatusForbidden, "您不是该公司的成员")
		return false
	}
	return true
}
