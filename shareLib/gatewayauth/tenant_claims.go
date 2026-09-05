package gatewayauth

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HeaderAuthTenantClaims is the HTTP header carrying the signed membership JWT.
const HeaderAuthTenantClaims = "X-Auth-Tenant-Claims"

// TenantMembership represents a single company membership for a user.
type TenantMembership struct {
	MemberID string `json:"id"`
	TenantID string `json:"cid"`
	IsAdmin  bool   `json:"a"`
	IsActive bool   `json:"ac"`
}

// TenantClaims is the JWT claims payload for tenant memberships.
type TenantClaims struct {
	Subject   string             `json:"sub"`
	IssuedAt  int64              `json:"iat"`
	ExpiresAt int64              `json:"exp"`
	JTI       string             `json:"jti"`
	Revision  int64              `json:"rev"`
	Members   []TenantMembership `json:"m"`
}

// ---------- HS256 (shared-secret, server-to-server) -------------------------------

// signHS256 signs claims with HMAC-SHA256 and returns a compact JWT string.
func signHS256(secret []byte, claims map[string]any) (string, error) {
	headerB64 := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	message := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(message))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return message + "." + sig, nil
}

// verifyHS256 validates an HS256 JWT signature and returns the decoded claims.
func verifyHS256(tokenString string, secret []byte) (map[string]any, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT: expected 3 parts, got %d", len(parts))
	}
	message := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(message))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("signature mismatch")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid payload JSON: %w", err)
	}
	return claims, nil
}

// EncodeTenantClaimsJWT serialises TenantClaims, signs with HS256, returns JWT.
func EncodeTenantClaimsJWT(claims *TenantClaims, secret string, maxBytes int) (string, error) {
	if claims == nil {
		return "", fmt.Errorf("claims must not be nil")
	}
	now := time.Now().Unix()
	if claims.IssuedAt == 0 {
		claims.IssuedAt = now
	}
	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = now + 300
	}
	members := claims.Members
	if members == nil {
		members = []TenantMembership{}
	}
	raw := map[string]any{
		"iss": "taskauth",
		"sub": claims.Subject,
		"iat": claims.IssuedAt,
		"exp": claims.ExpiresAt,
		"jti": claims.JTI,
		"rev": claims.Revision,
		"m":   members,
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	if maxBytes > 0 && len(payload) > maxBytes {
		for len(payload) > maxBytes && len(members) > 0 {
			members = members[:len(members)-1]
			raw["m"] = members
			payload, _ = json.Marshal(raw)
		}
		if len(members) != len(claims.Members) {
			raw["truncated"] = true
			payload, _ = json.Marshal(raw)
		}
	}
	raw["iss"] = "taskauth"
	return signHS256([]byte(secret), raw)
}

// VerifyTenantClaimsJWT validates HS256 signature + exp and returns parsed claims.
func VerifyTenantClaimsJWT(tokenString string, secret string) (*TenantClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, fmt.Errorf("empty token")
	}
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("empty secret")
	}
	raw, err := verifyHS256(tokenString, []byte(secret))
	if err != nil {
		return nil, err
	}
	return parseClaimsFromMap(raw)
}

// ---------- RS256 (public-key, frontend-verifiable) --------------------------------

// signRS256 signs a JWT with RSASSA-PKCS1-v1_5 + SHA-256.
func signRS256(priv *rsa.PrivateKey, headerJSON, payloadJSON []byte) (string, error) {
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	message := headerB64 + "." + payloadB64
	hashed := sha256.Sum256([]byte(message))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return message + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// EncodeTenantClaimsJWTRS256 serialises + signs with RS256 (for frontend).
// TTL is 15 min (longer than the 5 min server-to-server JWT).
func EncodeTenantClaimsJWTRS256(claims *TenantClaims, priv *rsa.PrivateKey, maxBytes int) (string, error) {
	if claims == nil {
		return "", fmt.Errorf("claims must not be nil")
	}
	if priv == nil {
		return "", fmt.Errorf("private key must not be nil")
	}
	now := time.Now().Unix()
	if claims.IssuedAt == 0 {
		claims.IssuedAt = now
	}
	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = now + 900
	}
	members := claims.Members
	if members == nil {
		members = []TenantMembership{}
	}
	raw := map[string]any{
		"iss": "taskauth",
		"sub": claims.Subject,
		"iat": claims.IssuedAt,
		"exp": claims.ExpiresAt,
		"jti": claims.JTI,
		"rev": claims.Revision,
		"m":   members,
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	if maxBytes > 0 && len(payload) > maxBytes {
		for len(payload) > maxBytes && len(members) > 0 {
			members = members[:len(members)-1]
			raw["m"] = members
			payload, _ = json.Marshal(raw)
		}
		if len(members) != len(claims.Members) {
			raw["truncated"] = true
			payload, _ = json.Marshal(raw)
		}
	}
	return signRS256(priv, []byte(`{"alg":"RS256","typ":"JWT"}`), payload)
}

// VerifyTenantClaimsJWTRS256 verifies an RS256 JWT using a DER-encoded public key.
func VerifyTenantClaimsJWTRS256(tokenString string, pubDER []byte) (*TenantClaims, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, fmt.Errorf("empty token")
	}
	pub, err := x509.ParsePKIXPublicKey(pubDER)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT")
	}
	message := parts[0] + "." + parts[1]
	hashed := sha256.Sum256([]byte(message))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}
	if err := rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, hashed[:], sig); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	return parseClaimsFromMap(raw)
}

// RSAPublicKeyToPEMDER returns the DER-encoded SubjectPublicKeyInfo for pub.
func RSAPublicKeyToPEMDER(priv *rsa.PrivateKey) ([]byte, error) {
	if priv == nil {
		return nil, fmt.Errorf("private key is nil")
	}
	return x509.MarshalPKIXPublicKey(&priv.PublicKey)
}

// JWKSPublicKeyPEM returns the PEM-encoded public key for frontend distribution.
func JWKSPublicKeyPEM(priv *rsa.PrivateKey) ([]byte, error) {
	if priv == nil {
		return nil, fmt.Errorf("private key is nil")
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

// ---------- Shared parsing helpers -------------------------------------------------

func parseClaimsFromMap(raw map[string]any) (*TenantClaims, error) {
	sub, _ := raw["sub"].(string)
	if sub == "" {
		return nil, fmt.Errorf("missing sub claim")
	}
	exp, _ := toInt64(raw["exp"])
	if exp > 0 && time.Now().Unix() > exp {
		return nil, fmt.Errorf("token expired (exp=%d, now=%d)", exp, time.Now().Unix())
	}
	iat, _ := toInt64(raw["iat"])
	jti, _ := raw["jti"].(string)
	rev, _ := toInt64(raw["rev"])
	claims := &TenantClaims{
		Subject:   sub,
		IssuedAt:  iat,
		ExpiresAt: exp,
		JTI:       jti,
		Revision:  rev,
	}
	membersRaw, ok := raw["m"].([]any)
	if !ok {
		claims.Members = []TenantMembership{}
		return claims, nil
	}
	for _, v := range membersRaw {
		m, ok := v.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		cid, _ := m["cid"].(string)
		isAdmin, _ := m["a"].(bool)
		isActive, _ := m["ac"].(bool)
		claims.Members = append(claims.Members, TenantMembership{
			MemberID: id, TenantID: cid, IsAdmin: isAdmin, IsActive: isActive,
		})
	}
	return claims, nil
}

// ---------- Request parsing --------------------------------------------------------

// ParseTenantClaims extracts and verifies the HS256 membership JWT from the request.
func ParseTenantClaims(r *http.Request, gatewayInternalSecret string) (*TenantClaims, bool) {
	if UserFromGatewayHeaders(r, gatewayInternalSecret) == "" {
		return nil, false
	}
	token := strings.TrimSpace(r.Header.Get(HeaderAuthTenantClaims))
	if token == "" {
		return nil, false
	}
	claims, err := VerifyTenantClaimsJWT(token, gatewayInternalSecret)
	if err != nil {
		return nil, false
	}
	gatewayUser := strings.TrimSpace(r.Header.Get(HeaderUserID))
	if gatewayUser != "" && claims.Subject != "" && claims.Subject != gatewayUser {
		return nil, false
	}
	return claims, true
}

// IsTenantMember returns true when tenantID appears in the membership list.
func IsTenantMember(claims *TenantClaims, tenantID string) bool {
	if claims == nil {
		return false
	}
	for _, m := range claims.Members {
		if m.TenantID == tenantID {
			return true
		}
	}
	return false
}

// IsTenantAdmin returns true when the user is an admin of tenantID.
func IsTenantAdmin(claims *TenantClaims, tenantID string) bool {
	if claims == nil {
		return false
	}
	for _, m := range claims.Members {
		if m.TenantID == tenantID && m.IsAdmin {
			return true
		}
	}
	return false
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	}
	return 0, false
}
