package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCredentialTokenInitURLRequiresComment(t *testing.T) {
	prev := cfg
	t.Cleanup(func() { cfg = prev })
	cfg.CredentialServiceURL = "http://cred:8015"

	got := credentialTokenInitURL(scope{TenantID: "t", WorkspaceID: "w", TaskID: "x", CommentID: "cmt_1"})
	want := "http://cred:8015/v1/token/init/tenant/t/workspace/w/task/x/comment/cmt_1"
	if got != want {
		t.Fatalf("url=%q want %q", got, want)
	}
	if credentialTokenInitURL(scope{TenantID: "t", WorkspaceID: "w", TaskID: "x"}) != "" {
		t.Fatal("empty comment must not build token-init URL")
	}
	if credentialTokenInitURL(scope{TenantID: "t", WorkspaceID: "w", TaskID: "x", CommentID: "-"}) != "" {
		t.Fatal("dash comment must not build token-init URL")
	}
}

func TestCredentialTokenInitRejectsEmptyComment(t *testing.T) {
	_, code, raw := credentialTokenInit(nil, scope{TenantID: "t", WorkspaceID: "w", TaskID: "x"})
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", code, raw)
	}
	var body map[string]string
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body["error_code"] != "COMMENT_ID_REQUIRED" {
		t.Fatalf("body=%v", body)
	}
}

func TestCredentialRepoCloneCredentialsRejectsEmptyComment(t *testing.T) {
	code, raw := credentialRepoCloneCredentials(nil, scope{TenantID: "t", WorkspaceID: "w", TaskID: "x"}, "tok")
	if code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", code, raw)
	}
	var body map[string]string
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body["error_code"] != "COMMENT_ID_REQUIRED" {
		t.Fatalf("body=%v", body)
	}
}
