package infrastructure

import (
	"sync"
	"time"
)

const (
	AccessCacheSkewSec    = 60
	AccessCacheDefaultTTL = 7200
)

type AccessTokenCache struct {
	mu   sync.Mutex
	data map[string]accessCacheEntry
}

type accessCacheEntry struct {
	token     string
	expiresAt time.Time
}

func NewAccessTokenCache() *AccessTokenCache {
	return &AccessTokenCache{data: map[string]accessCacheEntry{}}
}

func cacheKey(providerKey string, userID string) string {
	return providerKey + "|" + userID
}

func (c *AccessTokenCache) Get(providerKey string, userID string) string {
	key := cacheKey(providerKey, userID)
	c.mu.Lock()
	defer c.mu.Unlock()
	hit, ok := c.data[key]
	if !ok || hit.token == "" || time.Now().After(hit.expiresAt) {
		delete(c.data, key)
		return ""
	}
	return hit.token
}

func (c *AccessTokenCache) Put(providerKey string, userID string, accessToken string, expiresIn int) {
	token := stringsTrim(accessToken)
	if token == "" {
		return
	}
	ttl := expiresIn
	if ttl <= 0 {
		ttl = AccessCacheDefaultTTL
	}
	ttl = ttl - AccessCacheSkewSec
	if ttl < 30 {
		ttl = 30
	}
	key := cacheKey(providerKey, userID)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = accessCacheEntry{
		token:     token,
		expiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}
}

func (c *AccessTokenCache) Clear(providerKey string, userID string) {
	key := cacheKey(providerKey, userID)
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// TTLSeconds returns remaining cache TTL; 0 if missing/expired.
func (c *AccessTokenCache) TTLSeconds(providerKey string, userID string) int {
	key := cacheKey(providerKey, userID)
	c.mu.Lock()
	defer c.mu.Unlock()
	hit, ok := c.data[key]
	if !ok || hit.token == "" {
		return 0
	}
	sec := int(time.Until(hit.expiresAt).Seconds())
	if sec < 0 {
		delete(c.data, key)
		return 0
	}
	return sec
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		c := s[len(s)-1]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	return s
}
