package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"confload"
)

// ----- key management ---------------------------------------------------------

type oidcSigningKey struct {
	Private *rsa.PrivateKey
	KeyID   string
}

var signingKey *oidcSigningKey

// initOidcSigningKey loads or generates the OIDC signing key.
// If pemPath is empty, uses the default signing key path.
// If the PEM file exists → load it; otherwise generate a new key and persist.
func initOidcSigningKey(pemPath string) error {
	if signingKey != nil {
		return nil
	}
	if pemPath == "" {
		pemPath = defaultSigningKeyPath()
	}
	return loadOrGenerateKey(pemPath)
}

func defaultSigningKeyPath() string {
	rel := filepath.Join("conf-local", "auth", "task-auth", "oidc_signing_key.pem")
	root, err := confload.FindConfigRoot()
	if err != nil {
		return rel
	}
	return filepath.Join(root, rel)
}

func maybeMigrateLegacySigningKey(dest string) bool {
	root, err := confload.FindConfigRoot()
	if err != nil {
		return false
	}
	candidates := []string{
		filepath.Join(root, "db", "task-auth", "oidc_signing_key.pem"),
		filepath.Join(root, "taskAuth", "oidc_signing_key.pem"),
	}
	for _, src := range candidates {
		if src == dest {
			continue
		}
		raw, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			log.Printf("oidc signing key: migrate mkdir %s: %v", dest, err)
			return false
		}
		if err := os.WriteFile(dest, raw, 0o600); err != nil {
			log.Printf("oidc signing key: migrate write %s: %v", dest, err)
			return false
		}
		log.Printf("oidc signing key: migrated legacy key into conf-local (one-shot)")
		return true
	}
	return false
}

func loadOrGenerateKey(path string) error {
	if err := loadPEMKey(path); err == nil {
		return nil
	}
	if maybeMigrateLegacySigningKey(path) {
		if err := loadPEMKey(path); err == nil {
			return nil
		}
	}
	if os.Getenv("DEPLOY_MODE") == "1" {
		return fmt.Errorf("oidc signing key missing at %s (DEPLOY_MODE=1: will not mint)", path)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("oidc signing key: generate: %w", err)
	}
	if err := writePEMKey(path, key); err != nil {
		return fmt.Errorf("oidc signing key: write: %w", err)
	}
	kid := keyIDFromPublic(&key.PublicKey)
	signingKey = &oidcSigningKey{Private: key, KeyID: kid}
	return nil
}

func writePEMKey(path string, key *rsa.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.Encode(f, block)
}

func loadPEMKey(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("oidc signing key: %w", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return fmt.Errorf("oidc signing key: no PEM block in %s", path)
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		key2, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return fmt.Errorf("oidc signing key: parse: pkcs1=%v pkcs8=%v", err, err2)
		}
		rsaKey, ok := key2.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("oidc signing key: not an RSA private key")
		}
		key = rsaKey
	}
	kid := keyIDFromPublic(&key.PublicKey)
	signingKey = &oidcSigningKey{Private: key, KeyID: kid}
	return nil
}

func keyIDFromPublic(pub *rsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "k1"
	}
	h := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(h[:12])
}

func jwksJSON() (map[string]interface{}, error) {
	if signingKey == nil {
		return nil, fmt.Errorf("oidc: signing key not initialized")
	}
	pub := &signingKey.Private.PublicKey
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, err
	}
	n := pub.N.Bytes()
	e := pub.E
	eBytes := make([]byte, 4)
	eBytes[0] = byte(e >> 24)
	eBytes[1] = byte(e >> 16)
	eBytes[2] = byte(e >> 8)
	eBytes[3] = byte(e)
	for len(eBytes) > 1 && eBytes[0] == 0 {
		eBytes = eBytes[1:]
	}

	return map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"kid": signingKey.KeyID,
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(n),
				"e":   base64.RawURLEncoding.EncodeToString(eBytes),
				"x5c": []string{base64.StdEncoding.EncodeToString(der)},
			},
		},
	}, nil
}

// ----- JWT -------------------------------------------------------------------

func base64urlEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func signRS256(headerJSON, payloadJSON []byte) (string, error) {
	if signingKey == nil {
		return "", fmt.Errorf("oidc: signing key not initialized")
	}
	headerB64 := base64urlEncode(headerJSON)
	payloadB64 := base64urlEncode(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	hashed := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, signingKey.Private, crypto.SHA256, hashed[:])
	if err != nil {
		return "", fmt.Errorf("oidc: sign: %w", err)
	}
	return signingInput + "." + base64urlEncode(sig), nil
}

func makeJWT(claims map[string]interface{}) (string, error) {
	if signingKey == nil {
		return "", fmt.Errorf("oidc: signing key not initialized")
	}
	header := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
		"kid": signingKey.KeyID,
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	return signRS256(headerJSON, payloadJSON)
}

func makeIDToken(userID, email, username, issuer, clientID, nonce string, scopes []string) (string, error) {
	now := jwtNow()
	exp := now + int64(cfg.OidcIDTokenTTL)
	claims := map[string]interface{}{
		"iss":       issuer,
		"sub":       userID,
		"aud":       clientID,
		"exp":       exp,
		"iat":       now,
		"auth_time": now,
	}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	for _, s := range scopes {
		switch s {
		case "email":
			claims["email"] = email
			claims["email_verified"] = true
		case "profile":
			claims["name"] = username
			claims["preferred_username"] = username
		}
	}
	return makeJWT(claims)
}

func makeAccessToken(userID, issuer, clientID string, scopes []string) (string, error) {
	now := jwtNow()
	exp := now + int64(cfg.OidcAccessTokenTTL)
	scopeStr := ""
	if len(scopes) > 0 {
		scopeStr = strings.Join(scopes, " ")
	}
	claims := map[string]interface{}{
		"iss":   issuer,
		"sub":   userID,
		"aud":   clientID,
		"exp":   exp,
		"iat":   now,
		"scope": scopeStr,
		"jti":   mustRandHex(16),
	}
	return makeJWT(claims)
}

func jwtNow() int64 {
	return cfg.nowUnix()
}

func mustRandHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("rand: %v", err))
	}
	return fmt.Sprintf("%x", buf)
}

// parseJWTUnvalidated decodes a JWT and returns the claims WITHOUT verifying the
// signature.  This is safe for access_token lookups where we only need the
// subject and trust our own issuer (the self-issued token will be verified
// against the known public key before use).
func parseJWTUnvalidated(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("jwt decode: %w", err)
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("jwt unmarshal: %w", err)
	}
	return claims, nil
}

func verifyRS256Signature(token string) (map[string]interface{}, error) {
	if signingKey == nil {
		return nil, fmt.Errorf("oidc: signing key not initialized")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt")
	}
	signingInput := parts[0] + "." + parts[1]
	hashed := sha256.Sum256([]byte(signingInput))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("jwt sig decode: %w", err)
	}
	if err := rsa.VerifyPKCS1v15(&signingKey.Private.PublicKey, crypto.SHA256, hashed[:], sig); err != nil {
		return nil, fmt.Errorf("jwt sig verify: %w", err)
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// ----- client secret helpers -------------------------------------------------

// hashClientSecret produces a salted SHA-256 hash in format "salt$hash".
// salt is 16 random bytes (32 hex chars); hash is sha256(salt+secret).
func hashClientSecret(secret string) string {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		panic(fmt.Sprintf("rand: %v", err))
	}
	salt := fmt.Sprintf("%x", saltBytes)
	return salt + "$" + sha256Hash(salt+secret)
}

// verifyClientSecret checks a plaintext secret against a stored salted hash.
func verifyClientSecretHash(stored, secret string) bool {
	parts := strings.SplitN(stored, "$", 2)
	if len(parts) != 2 {
		return false
	}
	salt := parts[0]
	return stored == salt+"$"+sha256Hash(salt+secret)
}

func verifyClientSecret(clientID, secret string) (bool, *oidcClientRow, error) {
	row, err := loadOidcClient(clientID)
	if err != nil || row == nil {
		return false, nil, err
	}
	valid := verifyClientSecretHash(row.ClientSecretHash, secret)
	return valid, row, nil
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}
