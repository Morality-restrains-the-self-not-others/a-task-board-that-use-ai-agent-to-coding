package infrastructure

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateIDTokenJWKS(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-kid"
	n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	eBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(eBuf, uint32(key.E))
	eBytes := eBuf
	for len(eBytes) > 1 && eBytes[0] == 0 {
		eBytes = eBytes[1:]
	}
	e := base64.RawURLEncoding.EncodeToString(eBytes)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/oidc/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256", "n": n, "e": e,
			}},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	issuer := srv.URL
	audience := "ai-provider"
	now := time.Now().Unix()
	claims := map[string]any{
		"iss": issuer, "aud": audience, "sub": "42", "email": "a@b.c",
		"iat": now, "exp": now + 600,
	}
	token, err := signRS256Test(key, kid, claims)
	if err != nil {
		t.Fatal(err)
	}
	cache := NewJWKSCache(OIDCJWKSURL(issuer))
	cache.client = srv.Client()
	got, err := ValidateIDToken(token, cache, issuer, audience)
	if err != nil {
		t.Fatal(err)
	}
	if ClaimString(got, "email") != "a@b.c" {
		t.Fatalf("claims=%v", got)
	}
	if _, err := ValidateIDToken(token, cache, issuer, "other"); err == nil {
		t.Fatal("expected aud error")
	}
}

func signRS256Test(key *rsa.PrivateKey, kid string, claims map[string]any) (string, error) {
	headerObj := map[string]any{"alg": "RS256", "typ": "JWT", "kid": kid}
	hb, _ := json.Marshal(headerObj)
	pb, _ := json.Marshal(claims)
	h := base64.RawURLEncoding.EncodeToString(hb)
	p := base64.RawURLEncoding.EncodeToString(pb)
	sum := sha256.Sum256([]byte(h + "." + p))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return h + "." + p + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}
