package main

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestMain lowers the bcrypt work factor for the test binary only. Production
// default (bcryptCost = 12) is unchanged; the cost is embedded in each hash, so
// checkPasswordHash verifies any cost identically. This keeps 300+ tests that
// hash passwords during setup from spending ~30s on bcrypt.
func TestMain(m *testing.M) {
	bcryptCost = bcrypt.MinCost
	code := m.Run()
	drainAuthTestDBPool()
	os.Exit(code)
}

// TestHashPasswordRoundtrip is the regression companion to bcryptCost becoming
// a package-level var: hashing and verification must round-trip regardless of
// the test-time work factor, and a wrong password must still be rejected.
func TestHashPasswordRoundtrip(t *testing.T) {
	const password = "correct horse battery staple"
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if !checkPasswordHash(password, hash) {
		t.Fatal("checkPasswordHash should accept the just-hashed password")
	}
	if checkPasswordHash("wrong-password", hash) {
		t.Fatal("checkPasswordHash should reject a wrong password")
	}
}
