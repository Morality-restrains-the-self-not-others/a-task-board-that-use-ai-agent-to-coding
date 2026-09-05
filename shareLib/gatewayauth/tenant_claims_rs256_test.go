package gatewayauth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func testRSAKey() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return key
}

func TestRS256SignAndVerify_RoundTrip(t *testing.T) {
	key := testRSAKey()
	claims := &TenantClaims{
		Subject:  "user-rs256",
		JTI:      randJTI(),
		Revision: 7,
		Members: []TenantMembership{
			{MemberID: "ma", TenantID: "ca", IsAdmin: true, IsActive: true},
		},
	}
	token, err := EncodeTenantClaimsJWTRS256(claims, key, 8192)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	der, err := RSAPublicKeyToPEMDER(key)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := VerifyTenantClaimsJWTRS256(token, der)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Subject != "user-rs256" {
		t.Fatalf("sub=%s", parsed.Subject)
	}
	if parsed.Revision != 7 {
		t.Fatalf("rev=%d", parsed.Revision)
	}
	if !parsed.Members[0].IsAdmin {
		t.Fatal("expected admin")
	}
}

func TestRS256Verify_WrongKey_Fails(t *testing.T) {
	key1 := testRSAKey()
	key2 := testRSAKey()
	token, _ := EncodeTenantClaimsJWTRS256(&TenantClaims{Subject: "u1", Members: []TenantMembership{}}, key1, 8192)
	der2, _ := RSAPublicKeyToPEMDER(key2)
	_, err := VerifyTenantClaimsJWTRS256(token, der2)
	if err == nil {
		t.Fatal("expected error with wrong key")
	}
}

func TestRS256Verify_Tampered_Fails(t *testing.T) {
	key := testRSAKey()
	token, _ := EncodeTenantClaimsJWTRS256(&TenantClaims{Subject: "u1", Members: []TenantMembership{}}, key, 8192)
	// Tamper the payload
	parts := []byte(token)
	parts[len(parts)-5] ^= 0xff
	der, _ := RSAPublicKeyToPEMDER(key)
	_, err := VerifyTenantClaimsJWTRS256(string(parts), der)
	if err == nil {
		t.Fatal("expected error with tampered token")
	}
}

func TestRS256Verify_Expired_Fails(t *testing.T) {
	key := testRSAKey()
	claims := &TenantClaims{
		Subject:   "u1",
		IssuedAt:  1000,
		ExpiresAt: 1001,
		Members:   []TenantMembership{},
	}
	token, _ := EncodeTenantClaimsJWTRS256(claims, key, 8192)
	der, _ := RSAPublicKeyToPEMDER(key)
	_, err := VerifyTenantClaimsJWTRS256(token, der)
	if err == nil {
		t.Fatal("expected expiry error")
	}
}

func TestRS256Verify_EmptyToken(t *testing.T) {
	key := testRSAKey()
	der, _ := RSAPublicKeyToPEMDER(key)
	_, err := VerifyTenantClaimsJWTRS256("", der)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestJWKSPublicKeyPEM(t *testing.T) {
	key := testRSAKey()
	pemBytes, err := JWKSPublicKeyPEM(key)
	if err != nil {
		t.Fatal(err)
	}
	if len(pemBytes) == 0 {
		t.Fatal("empty PEM")
	}
}

func TestJWKSPublicKeyPEM_Nil(t *testing.T) {
	_, err := JWKSPublicKeyPEM(nil)
	if err == nil {
		t.Fatal("expected error for nil key")
	}
}
