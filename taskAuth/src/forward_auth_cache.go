package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type forwardAuthCacheEntry struct {
	userID                 string
	userEmail              string // 主邮箱（resolveUserEmail），供下游服务做邮箱匹配（如厂商门户资格）
	isActive               bool
	isSuperuser            bool
	isStaff                bool
	isTester               bool
	phoneVerified          bool
	tenantClaimsJWT        string
	platformRoles          []string
	tenantPerms            map[string][]string
	impersonatorID         string
	impersonationSessionID string
	cacheStatus            string
	expiresAt              time.Time
}

var (
	forwardAuthCacheMu sync.Mutex
	forwardAuthCache   = map[string]forwardAuthCacheEntry{}
)

func forwardAuthCacheTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("TASKAUTH_FORWARD_AUTH_CACHE_TTL_MS"))
	if raw == "" {
		return 2 * time.Second
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		return 2 * time.Second
	}
	if ms == 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func credentialFingerprintForForwardAuth(r *http.Request) string {
	parts := make([]string, 0, 4)
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); auth != "" {
		parts = append(parts, "a:"+auth)
	}
	if c, err := r.Cookie("token"); err == nil && c.Value != "" {
		parts = append(parts, "t:"+c.Value)
	}
	if c, err := r.Cookie("sessionid"); err == nil && c.Value != "" {
		parts = append(parts, "s:"+c.Value)
	}
	if c, err := r.Cookie("userId"); err == nil && c.Value != "" {
		parts = append(parts, "u:"+c.Value)
	}
	if len(parts) == 0 {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func getForwardAuthCache(key string) (forwardAuthCacheEntry, bool) {
	ttl := forwardAuthCacheTTL()
	if ttl <= 0 || key == "" {
		return forwardAuthCacheEntry{}, false
	}
	forwardAuthCacheMu.Lock()
	defer forwardAuthCacheMu.Unlock()
	ent, ok := forwardAuthCache[key]
	if !ok || time.Now().After(ent.expiresAt) {
		if ok {
			delete(forwardAuthCache, key)
		}
		return forwardAuthCacheEntry{}, false
	}
	return ent, true
}

func putForwardAuthCache(key string, userID string, userEmail string, isActive, isSuperuser, isStaff, isTester, phoneVerified bool, tenantClaimsJWT string, platformRoles []string, tenantPerms map[string][]string, impersonatorID, impersonationSessionID string) {
	ttl := forwardAuthCacheTTL()
	if ttl <= 0 || key == "" || userID == "" {
		return
	}
	forwardAuthCacheMu.Lock()
	defer forwardAuthCacheMu.Unlock()
	forwardAuthCache[key] = forwardAuthCacheEntry{
		userID:                 userID,
		userEmail:              userEmail,
		isActive:               isActive,
		isSuperuser:            isSuperuser,
		isStaff:                isStaff,
		isTester:               isTester,
		phoneVerified:          phoneVerified,
		tenantClaimsJWT:        tenantClaimsJWT,
		platformRoles:          platformRoles,
		tenantPerms:            tenantPerms,
		impersonatorID:         impersonatorID,
		impersonationSessionID: impersonationSessionID,
		expiresAt:              time.Now().Add(ttl),
	}
	if len(forwardAuthCache) > 4096 {
		now := time.Now()
		for k, v := range forwardAuthCache {
			if now.After(v.expiresAt) {
				delete(forwardAuthCache, k)
			}
		}
	}
}

func clearForwardAuthCacheForTest() {
	forwardAuthCacheMu.Lock()
	defer forwardAuthCacheMu.Unlock()
	forwardAuthCache = map[string]forwardAuthCacheEntry{}
}
