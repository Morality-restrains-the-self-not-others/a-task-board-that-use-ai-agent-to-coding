package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gatewayauth"

	"github.com/redis/go-redis/v9"
)

// tenantMembershipJWTSecret returns the HS256 signing key for membership JWTs.
// Priority: TENANT_MEMBERSHIP_JWT_SECRET env > cfg.InternalSecret > dev fallback.
func tenantMembershipJWTSecret() string {
	if s := strings.TrimSpace(os.Getenv("TENANT_MEMBERSHIP_JWT_SECRET")); s != "" {
		return s
	}
	if s := strings.TrimSpace(cfg.InternalSecret); s != "" {
		return s
	}
	return "task2app-local-tenant-membership-jwt-dev-do-not-use-in-prod"
}

// tenantMembershipsMaxBytes returns the max encoded payload size before truncation.
// Default 7000 bytes (fits within 8KB HTTP header limit with JWT overhead).
func tenantMembershipsMaxBytes() int {
	if v := strings.TrimSpace(os.Getenv("TENANT_MEMBERSHIPS_MAX_BYTES")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 7000
}

// tenantMembershipsEnabled returns whether membership JWT embedding is enabled.
// Default true; set TASKAUTH_TENANT_MEMBERSHIPS_ENABLED=0 to disable.
func tenantMembershipsEnabled() bool {
	v := strings.TrimSpace(os.Getenv("TASKAUTH_TENANT_MEMBERSHIPS_ENABLED"))
	if v == "" {
		return true
	}
	enabled, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return enabled
}

// membershipRedisClient is lazily initialised from cfg.  Nil when Redis is not
// configured — the rev check degrades gracefully (rev=0) in that case.
var membershipRedisClient *redis.Client

func initMembershipRedis() {
	if membershipRedisClient != nil {
		return
	}
	host := strings.TrimSpace(os.Getenv("TASKAUTH_REDIS_HOST"))
	if host == "" {
		host = strings.TrimSpace(os.Getenv("INFRA_HOST"))
	}
	if host == "" {
		return
	}
	port := strings.TrimSpace(os.Getenv("TASKAUTH_REDIS_PORT"))
	if port == "" {
		port = "6379"
	}
	db := 0
	if v := strings.TrimSpace(os.Getenv("TASKAUTH_REDIS_DB")); v != "" {
		db, _ = strconv.Atoi(v)
	}
	membershipRedisClient = redis.NewClient(&redis.Options{
		Addr:        host + ":" + port,
		DB:          db,
		DialTimeout: 2 * time.Second,
		ReadTimeout: 1 * time.Second,
	})
}

func getMembershipRev(ctx context.Context, userID string) (int64, error) {
	if membershipRedisClient == nil {
		return 0, fmt.Errorf("redis not configured")
	}
	key := "membership_rev:" + userID
	val, err := membershipRedisClient.Get(ctx, key).Int64()
	if err != nil {
		return 0, err
	}
	return val, nil
}

// fetchAndSignTenantClaims fetches the user's full membership list from
// taskTenantService, reads the current Redis revision, and signs a HS256 JWT.
// Returns "" on any error (graceful degradation — the auth response still
// succeeds; downstream services fall back to the existing HTTP path).
func fetchAndSignTenantClaims(userID string) string {
	if !tenantMembershipsEnabled() {
		return ""
	}
	base := strings.TrimRight(cfg.TenantServiceURL, "/")
	if base == "" {
		return ""
	}

	members, err := fetchTenantMemberships(userID, base)
	if err != nil {
		log.Printf("[taskAuth] event=forward_auth_memberships_fetch status=error user_id=%s err=%v", userID, err)
		return ""
	}

	rev := int64(0)
	if r, err := getMembershipRev(context.Background(), userID); err == nil {
		rev = r
	}

	claims := &gatewayauth.TenantClaims{
		Subject:  userID,
		Revision: rev,
		Members:  members,
	}
	secret := tenantMembershipJWTSecret()
	maxBytes := tenantMembershipsMaxBytes()
	token, err := gatewayauth.EncodeTenantClaimsJWT(claims, secret, maxBytes)
	if err != nil {
		log.Printf("[taskAuth] event=forward_auth_memberships_sign status=error user_id=%s err=%v", userID, err)
		return ""
	}
	log.Printf("[taskAuth] event=forward_auth_memberships_fetch status=ok user_id=%s members=%d rev=%d size=%d",
		userID, len(members), rev, len(token))
	return token
}

// fetchTenantMemberships calls GET /api/internal/tenant/members?user_id=X on
// taskTenantService.  Returns a minimal compact list suitable for JWT embedding.
func fetchTenantMemberships(userID, baseURL string) ([]gatewayauth.TenantMembership, error) {
	url := fmt.Sprintf("%s/api/internal/tenant/members?user_id=%s", baseURL, userID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tenant service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tenant service returned %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var list []map[string]any
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, fmt.Errorf("parse members: %w", err)
	}
	result := make([]gatewayauth.TenantMembership, 0, len(list))
	for _, m := range list {
		id, _ := m["id"].(string)
		cid, _ := m["company_id"].(string)
		isAdmin, _ := m["is_admin"].(bool)
		isActive, _ := m["is_active"].(bool)
		if cid == "" {
			continue
		}
		result = append(result, gatewayauth.TenantMembership{
			MemberID: id,
			TenantID: cid,
			IsAdmin:  isAdmin,
			IsActive: isActive,
		})
	}
	return result, nil
}

// handleTenantMemberships is the frontend endpoint that returns the membership
// JWT for the authenticated user.  GET /api/auth/tenant-memberships/
func handleTenantMemberships(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Resolve user from cookie/header (same logic as forward-auth, but with
	// IP binding since this is a direct user-facing endpoint).
	userID, err := resolveTokenUserIDFromRequest(r)
	if err != nil || userID == "" {
		writeErrorDetail(w, r, http.StatusUnauthorized, "请先登录")
		return
	}
	isActive, _, _, isArchived, err := loadUserAuthFlags(userID)
	if err != nil || !isActive || isArchived {
		writeErrorDetail(w, r, http.StatusUnauthorized, "账号未激活或已停用")
		return
	}

	hsJWT := fetchAndSignTenantClaims(userID)
	rsJWT := fetchAndSignTenantClaimsRS256(userID)
	writeJSON(w, http.StatusOK, map[string]any{
		"jwt":     hsJWT,
		"jwt_rsa": rsJWT,
	})
}

// fetchAndSignTenantClaimsRS256 signs a membership JWT with RS256 (asymmetric)
// so the frontend can verify it using the public key from /tenant-memberships/public-key.
func fetchAndSignTenantClaimsRS256(userID string) string {
	if !tenantMembershipsEnabled() {
		return ""
	}
	if signingKey == nil {
		return ""
	}
	base := strings.TrimRight(cfg.TenantServiceURL, "/")
	if base == "" {
		return ""
	}
	members, err := fetchTenantMemberships(userID, base)
	if err != nil {
		return ""
	}
	rev := int64(0)
	if r, err := getMembershipRev(context.Background(), userID); err == nil {
		rev = r
	}
	claims := &gatewayauth.TenantClaims{
		Subject:  userID,
		Revision: rev,
		Members:  members,
	}
	token, err := gatewayauth.EncodeTenantClaimsJWTRS256(claims, signingKey.Private, tenantMembershipsMaxBytes())
	if err != nil {
		log.Printf("[taskAuth] event=forward_auth_memberships_rs256_sign status=error user_id=%s err=%v", userID, err)
		return ""
	}
	return token
}

// handleTenantMembershipsPublicKey returns the public key PEM for verifying
// RS256 membership JWTs on the frontend.
// GET /api/auth/tenant-memberships/public-key
func handleTenantMembershipsPublicKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if signingKey == nil {
		writeErrorDetail(w, r, http.StatusServiceUnavailable, "signing key not initialized")
		return
	}
	pemBytes, err := gatewayauth.JWKSPublicKeyPEM(signingKey.Private)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "failed to export public key")
		return
	}
	w.Header().Set("Content-Type", "application/x-pem-file; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(pemBytes)
}
