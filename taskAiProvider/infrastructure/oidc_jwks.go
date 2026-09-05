package infrastructure

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

// JWKSCache caches the OIDC RP JWKS document (aligned with Python OidcRpClient).
type JWKSCache struct {
	mu       sync.Mutex
	endpoint string
	keys     map[string]*rsa.PublicKey
	fetched  time.Time
	ttl      time.Duration
	client   *http.Client
}

func NewJWKSCache(jwksURL string) *JWKSCache {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = nil // do not inherit shell HTTP(S)_PROXY (dev-only accel)
	return &JWKSCache{
		endpoint: strings.TrimSpace(jwksURL),
		ttl:      10 * time.Minute,
		client:   &http.Client{Timeout: 10 * time.Second, Transport: tr},
	}
}

func (c *JWKSCache) publicKey(kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.keys == nil || time.Since(c.fetched) > c.ttl {
		if err := c.refreshLocked(); err != nil {
			return nil, err
		}
	}
	key, ok := c.keys[kid]
	if !ok {
		// one refresh on miss (key rotation)
		if err := c.refreshLocked(); err != nil {
			return nil, err
		}
		key, ok = c.keys[kid]
	}
	if !ok {
		return nil, fmt.Errorf("id_token 签名密钥不匹配")
	}
	return key, nil
}

func (c *JWKSCache) refreshLocked() error {
	if c.endpoint == "" {
		return fmt.Errorf("JWKS endpoint empty")
	}
	req, err := http.NewRequest(http.MethodGet, c.endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("身份认证服务暂时不可用，请稍后重试")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("身份认证服务暂时不可用，请稍后重试")
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("身份认证服务暂时不可用，请稍后重试")
	}
	var doc struct {
		Keys []map[string]any `json:"keys"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("JWKS 响应无效")
	}
	next := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, jwk := range doc.Keys {
		kid, _ := jwk["kid"].(string)
		if kid == "" {
			continue
		}
		kty, _ := jwk["kty"].(string)
		if kty != "RSA" {
			continue
		}
		pub, err := rsaPublicFromJWK(jwk)
		if err != nil {
			continue
		}
		next[kid] = pub
	}
	c.keys = next
	c.fetched = time.Now()
	return nil
}

func rsaPublicFromJWK(jwk map[string]any) (*rsa.PublicKey, error) {
	nStr, _ := jwk["n"].(string)
	eStr, _ := jwk["e"].(string)
	if nStr == "" || eStr == "" {
		return nil, fmt.Errorf("incomplete jwk")
	}
	nb, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nb)
	var eInt int
	for _, b := range eb {
		eInt = eInt<<8 + int(b)
	}
	if eInt == 0 {
		return nil, fmt.Errorf("bad jwk e")
	}
	return &rsa.PublicKey{N: n, E: eInt}, nil
}

// ValidateIDToken verifies RS256 signature via JWKS and require iss/aud/exp.
func ValidateIDToken(raw string, cache *JWKSCache, issuer, audience string) (map[string]any, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("id_token 格式无效")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("id_token 格式无效")
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("id_token 格式无效")
	}
	if header.Alg != "RS256" {
		return nil, fmt.Errorf("id_token 签名算法不支持")
	}
	if header.Kid == "" {
		return nil, fmt.Errorf("id_token 缺少 kid")
	}
	pub, err := cache.publicKey(header.Kid)
	if err != nil {
		return nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("id_token 签名验证失败")
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
		return nil, fmt.Errorf("id_token 签名验证失败")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("id_token 格式无效")
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("id_token 格式无效")
	}
	now := time.Now().Unix()
	if exp, ok := claimInt64(claims["exp"]); !ok || now >= exp {
		return nil, fmt.Errorf("id_token 已过期")
	}
	if iss, _ := claims["iss"].(string); strings.TrimRight(iss, "/") != strings.TrimRight(issuer, "/") {
		return nil, fmt.Errorf("id_token issuer/audience 不匹配")
	}
	if !audienceOK(claims["aud"], audience) {
		return nil, fmt.Errorf("id_token issuer/audience 不匹配")
	}
	return claims, nil
}

// OIDCJWKSURL builds the default JWKS endpoint from issuer (Python parity).
func OIDCJWKSURL(issuer string) string {
	return strings.TrimRight(strings.TrimSpace(issuer), "/") + "/api/oidc/jwks"
}
