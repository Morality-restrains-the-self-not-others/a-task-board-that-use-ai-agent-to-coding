package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateBootstrapAdminEmail(t *testing.T) {
	if _, err := validateBootstrapAdminEmail(""); err == nil {
		t.Fatal("empty should fail")
	}
	if _, err := validateBootstrapAdminEmail("no-at"); err == nil {
		t.Fatal("missing @ should fail")
	}
	if _, err := validateBootstrapAdminEmail("bad'x@example.com"); err == nil {
		t.Fatal("quote should fail")
	}
	got, err := validateBootstrapAdminEmail("  ops@example.com ")
	if err != nil || got != "ops@example.com" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestWriteAndReadBootstrapAdminEmailConfLocal(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf", "auth", "task-auth")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tracked := []byte("bootstrapAdmin:\n  email: author@example.com\n")
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), tracked, 0o644); err != nil {
		t.Fatal(err)
	}
	email, err := readBootstrapAdminEmail(root)
	if err != nil || email != "author@example.com" {
		t.Fatalf("tracked email=%q err=%v", email, err)
	}
	if err := writeBootstrapAdminEmailConfLocal(root, "init-ops@example.com"); err != nil {
		t.Fatalf("write: %v", err)
	}
	email, err = readBootstrapAdminEmail(root)
	if err != nil || email != "init-ops@example.com" {
		t.Fatalf("overlay email=%q err=%v", email, err)
	}
	localPath := filepath.Join(root, "conf-local", "auth", "task-auth", "config.yaml")
	raw, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "init-ops@example.com") {
		t.Fatalf("conf-local missing email: %s", raw)
	}
}

func TestHandleBootstrapAdminEmailRoundTrip(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf", "auth", "task-auth")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte("bootstrapAdmin:\n  email: author@example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MONOREPO_ROOT", root)
	t.Setenv("RUNALL_ALLOW_DEV_DB_RESET", "1")

	mux := http.NewServeMux()
	registerDevHandlers(mux, nil)

	getReq := httptest.NewRequest(http.MethodGet, "/api/dev/bootstrap-admin-email", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	var getBody map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &getBody); err != nil {
		t.Fatal(err)
	}
	if getBody["email"] != "author@example.com" {
		t.Fatalf("GET email=%v", getBody["email"])
	}

	postReq := httptest.NewRequest(http.MethodPost, "/api/dev/bootstrap-admin-email", strings.NewReader(`{"email":"fresh-admin@example.com"}`))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	mux.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("POST status=%d body=%s", postRec.Code, postRec.Body.String())
	}
	got, err := readBootstrapAdminEmail(root)
	if err != nil || got != "fresh-admin@example.com" {
		t.Fatalf("after POST email=%q err=%v", got, err)
	}
}

func TestHandleBootstrapAdminEmailRejectsEmpty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MONOREPO_ROOT", root)
	t.Setenv("RUNALL_ALLOW_DEV_DB_RESET", "1")
	mux := http.NewServeMux()
	registerDevHandlers(mux, nil)
	postReq := httptest.NewRequest(http.MethodPost, "/api/dev/bootstrap-admin-email", strings.NewReader(`{"email":""}`))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	mux.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", postRec.Code, postRec.Body.String())
	}
}
