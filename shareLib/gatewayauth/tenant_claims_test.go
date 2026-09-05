package gatewayauth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testSecret = "test-gateway-secret-do-not-use-in-prod"

func randJTI() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ---------- sign / verify round-trip -------------------------------------------------

func TestSignAndVerifyHS256_RoundTrip(t *testing.T) {
	secret := []byte(testSecret)
	claims := map[string]any{"sub": "u1", "m": []any{}}
	token, err := signHS256(secret, claims)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := verifyHS256(token, secret)
	if err != nil {
		t.Fatal(err)
	}
	if parsed["sub"] != "u1" {
		t.Fatalf("sub=%v", parsed["sub"])
	}
}

func TestVerifyHS256_WrongSecret_Fails(t *testing.T) {
	token, _ := signHS256([]byte(testSecret), map[string]any{"sub": "u1"})
	_, err := verifyHS256(token, []byte("wrong-secret"))
	if err == nil {
		t.Fatal("expected error with wrong secret")
	}
}

func TestVerifyHS256_TamperedPayload_Fails(t *testing.T) {
	token, _ := signHS256([]byte(testSecret), map[string]any{"sub": "u1"})
	parts := strings.Split(token, ".")
	parts[1] = "ZmFrZQ" // "fake" in base64
	_, err := verifyHS256(strings.Join(parts, "."), []byte(testSecret))
	if err == nil {
		t.Fatal("expected error with tampered payload")
	}
}

func TestVerifyHS256_MalformedToken(t *testing.T) {
	_, err := verifyHS256("not.a.jwt", []byte(testSecret))
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------- Encode / Verify TenantClaimsJWT ------------------------------------------

func TestEncodeAndVerifyTenantClaimsJWT(t *testing.T) {
	claims := &TenantClaims{
		Subject:  "user-123",
		JTI:      randJTI(),
		Revision: 5,
		Members: []TenantMembership{
			{MemberID: "m1", TenantID: "c1", IsAdmin: true, IsActive: true},
			{MemberID: "m2", TenantID: "c2", IsAdmin: false, IsActive: true},
		},
	}
	token, err := EncodeTenantClaimsJWT(claims, testSecret, 8192)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	// Verify
	parsed, err := VerifyTenantClaimsJWT(token, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Subject != "user-123" {
		t.Fatalf("sub=%s", parsed.Subject)
	}
	if parsed.Revision != 5 {
		t.Fatalf("rev=%d", parsed.Revision)
	}
	if len(parsed.Members) != 2 {
		t.Fatalf("members=%d", len(parsed.Members))
	}
	if !parsed.Members[0].IsAdmin {
		t.Fatal("expected admin for c1")
	}
	if !parsed.Members[1].IsActive {
		t.Fatal("expected active for c2")
	}
}

func TestEncodeTenantClaimsJWT_AutofillsTimestamps(t *testing.T) {
	claims := &TenantClaims{Subject: "u1", Members: []TenantMembership{}}
	token, _ := EncodeTenantClaimsJWT(claims, testSecret, 8192)
	parsed, _ := VerifyTenantClaimsJWT(token, testSecret)
	if parsed.IssuedAt == 0 {
		t.Fatal("iat not set")
	}
	if parsed.ExpiresAt == 0 {
		t.Fatal("exp not set")
	}
	if parsed.ExpiresAt-parsed.IssuedAt != 300 {
		t.Fatalf("expected 300s TTL, got %d", parsed.ExpiresAt-parsed.IssuedAt)
	}
}

func TestVerifyTenantClaimsJWT_Expired(t *testing.T) {
	claims := &TenantClaims{
		Subject:   "u1",
		IssuedAt:  time.Now().Unix() - 600,
		ExpiresAt: time.Now().Unix() - 1,
		Members:   []TenantMembership{},
	}
	token, _ := EncodeTenantClaimsJWT(claims, testSecret, 8192)
	_, err := VerifyTenantClaimsJWT(token, testSecret)
	if err == nil {
		t.Fatal("expected expiry error")
	}
}

func TestVerifyTenantClaimsJWT_EmptyToken(t *testing.T) {
	_, err := VerifyTenantClaimsJWT("", testSecret)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyTenantClaimsJWT_EmptySecret(t *testing.T) {
	_, err := VerifyTenantClaimsJWT("x.y.z", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEncodeTenantClaimsJWT_MaxBytesTruncation(t *testing.T) {
	claims := &TenantClaims{
		Subject: "u1",
		Members: make([]TenantMembership, 100),
	}
	for i := range claims.Members {
		claims.Members[i] = TenantMembership{
			MemberID: "m" + strings.Repeat("x", 30),
			TenantID: "c" + strings.Repeat("x", 30),
		}
	}
	// Use a tiny maxBytes to force truncation
	token, err := EncodeTenantClaimsJWT(claims, testSecret, 200)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := VerifyTenantClaimsJWT(token, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Members) >= 100 {
		t.Fatalf("expected truncation, got %d members", len(parsed.Members))
	}
}

func TestEncodeTenantClaimsJWT_Nil(t *testing.T) {
	_, err := EncodeTenantClaimsJWT(nil, testSecret, 8192)
	if err == nil {
		t.Fatal("expected error for nil claims")
	}
}

// ---------- ParseTenantClaims (HTTP header) ------------------------------------------

func newGatewayRequest(token string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(HeaderGatewayVerified, "1")
	r.Header.Set(HeaderGatewaySecret, testSecret)
	r.Header.Set(HeaderUserID, "user-123")
	if token != "" {
		r.Header.Set(HeaderAuthTenantClaims, token)
	}
	return r
}

func TestParseTenantClaims_Valid(t *testing.T) {
	claims := &TenantClaims{
		Subject: "user-123",
		JTI:     randJTI(),
		Members: []TenantMembership{{MemberID: "m1", TenantID: "c1", IsAdmin: true}},
	}
	token, _ := EncodeTenantClaimsJWT(claims, testSecret, 8192)
	r := newGatewayRequest(token)
	parsed, ok := ParseTenantClaims(r, testSecret)
	if !ok {
		t.Fatal("expected ok")
	}
	if !IsTenantMember(parsed, "c1") {
		t.Fatal("expected member of c1")
	}
	if !IsTenantAdmin(parsed, "c1") {
		t.Fatal("expected admin of c1")
	}
}

func TestParseTenantClaims_NoGatewayHeaders(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(HeaderAuthTenantClaims, "x.y.z")
	_, ok := ParseTenantClaims(r, testSecret)
	if ok {
		t.Fatal("expected false without gateway headers")
	}
}

func TestParseTenantClaims_WrongSecret(t *testing.T) {
	claims := &TenantClaims{Subject: "user-123", Members: []TenantMembership{}}
	token, _ := EncodeTenantClaimsJWT(claims, testSecret, 8192)
	r := newGatewayRequest(token)
	_, ok := ParseTenantClaims(r, "wrong-secret")
	if ok {
		t.Fatal("expected false with wrong secret")
	}
}

func TestParseTenantClaims_MissingHeader(t *testing.T) {
	r := newGatewayRequest("")
	_, ok := ParseTenantClaims(r, testSecret)
	if ok {
		t.Fatal("expected false when header absent")
	}
}

func TestParseTenantClaims_WrongSubject(t *testing.T) {
	claims := &TenantClaims{
		Subject: "different-user",
		JTI:     randJTI(),
		Members: []TenantMembership{},
	}
	token, _ := EncodeTenantClaimsJWT(claims, testSecret, 8192)
	r := newGatewayRequest(token) // gateway says user-123
	_, ok := ParseTenantClaims(r, testSecret)
	if ok {
		t.Fatal("expected false for subject mismatch")
	}
}

func TestParseTenantClaims_ExpiredJWT(t *testing.T) {
	claims := &TenantClaims{
		Subject:   "user-123",
		IssuedAt:  time.Now().Unix() - 600,
		ExpiresAt: time.Now().Unix() - 1,
		Members:   []TenantMembership{},
	}
	token, _ := EncodeTenantClaimsJWT(claims, testSecret, 8192)
	r := newGatewayRequest(token)
	_, ok := ParseTenantClaims(r, testSecret)
	if ok {
		t.Fatal("expected false for expired token")
	}
}

// ---------- IsTenantMember / IsTenantAdmin helpers ------------------------------------

func TestIsTenantMember_Empty(t *testing.T) {
	if IsTenantMember(nil, "c1") {
		t.Fatal("expected false for nil claims")
	}
	empty := &TenantClaims{Members: []TenantMembership{}}
	if IsTenantMember(empty, "c1") {
		t.Fatal("expected false for empty memberships")
	}
}

func TestIsTenantAdmin_NonAdmin(t *testing.T) {
	claims := &TenantClaims{
		Members: []TenantMembership{
			{MemberID: "m1", TenantID: "c1", IsAdmin: false},
		},
	}
	if IsTenantAdmin(claims, "c1") {
		t.Fatal("expected false for non-admin")
	}
	if IsTenantMember(claims, "c1") {
		// member check should still pass
	}
}
