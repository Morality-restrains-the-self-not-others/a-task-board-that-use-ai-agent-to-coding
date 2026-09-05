package infrastructure

import (
	"os/exec"
	"strings"
	"testing"
)

const testSecret = "django-insecure-gitOauth-dev-change-for-production"

func TestFernetRoundTrip(t *testing.T) {
	f := NewFernet(testSecret)
	cipher, err := f.Encrypt("refresh-token-plain")
	if err != nil {
		t.Fatal(err)
	}
	if cipher == "" {
		t.Fatal("empty cipher")
	}
	plain, err := f.Decrypt(cipher)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "refresh-token-plain" {
		t.Fatalf("got %q", plain)
	}
}

func TestFernetCompatWithPython(t *testing.T) {
	py := `
import base64, hashlib
from cryptography.fernet import Fernet
secret = "django-insecure-gitOauth-dev-change-for-production"
raw = hashlib.sha256(secret.encode("utf-8")).digest()
key = base64.urlsafe_b64encode(raw)
print(Fernet(key).encrypt(b"hello-gitoauth").decode("ascii"))
`
	cmd := exec.Command("python3", "-c", py)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "No module named") || strings.Contains(err.Error(), "executable file not found") {
			t.Skip("python cryptography not available:", string(out), err)
		}
		if err != nil {
			t.Skip("skip python compat:", err, string(out))
		}
	}
	cipher := strings.TrimSpace(string(out))
	if cipher == "" {
		t.Skip("empty python cipher")
	}
	f := NewFernet(testSecret)
	plain, err := f.Decrypt(cipher)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "hello-gitoauth" {
		t.Fatalf("compat decrypt got %q", plain)
	}
}
