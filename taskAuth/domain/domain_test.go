package domain

import "testing"

func TestPasswordCredentialType(t *testing.T) {
	var c PasswordCredential = "hashed"
	if string(c) != "hashed" {
		t.Fatalf("unexpected credential")
	}
}
