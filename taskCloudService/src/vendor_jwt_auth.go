package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const vendorJWTIssuer = "saas-ai-provider"

func loadVendorJWTSecret() string {
	if s := strings.TrimSpace(os.Getenv("MARKETPLACE_VENDOR_JWT_SECRET")); s != "" {
		return s
	}
	if s := strings.TrimSpace(os.Getenv("AI_PROVIDER_JWT_SECRET")); s != "" {
		return s
	}
	if s := strings.TrimSpace(os.Getenv("DJANGO_SECRET_KEY")); s != "" {
		return s
	}
	return "dev-saas-ai-provider-change-me-in-production"
}

func verifyVendorJWT(token string) (vendorID string, err error) {
	secret := loadVendorJWTSecret()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid token format")
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return "", errors.New("invalid signature")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return "", err
	}
	if iss, _ := claims["iss"].(string); iss != vendorJWTIssuer {
		return "", fmt.Errorf("invalid issuer")
	}
	if typ, _ := claims["typ"].(string); typ != "vendor" {
		return "", errors.New("not a vendor token")
	}
	exp, ok := claims["exp"].(float64)
	if !ok || int64(exp) < time.Now().Unix() {
		return "", errors.New("token expired")
	}
	sub, _ := claims["sub"].(string)
	sub = strings.TrimSpace(sub)
	if sub == "" {
		return "", errors.New("missing subject")
	}
	return sub, nil
}

func vendorAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeErrorJSON(w, r, http.StatusUnauthorized, "需要厂商 Bearer 令牌")
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		vendorID, err := verifyVendorJWT(raw)
		if err != nil {
			writeErrorJSON(w, r, http.StatusUnauthorized, "无效或过期的令牌")
			return
		}
		r.Header.Set("X-Vendor-Id", vendorID)
		next(w, r)
	}
}

func getVendorID(r *http.Request) string {
	return r.Header.Get("X-Vendor-Id")
}

func maskCredential(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}

// credentialFieldForUpdate 若值为 API 返回的掩码形式（含 *），则不更新该字段。
func credentialFieldForUpdate(raw string) string {
	if strings.Contains(raw, "*") {
		return ""
	}
	return raw
}
