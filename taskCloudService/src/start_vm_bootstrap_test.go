package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mockCredentialTokenServer(t *testing.T, token string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/v1/token/init/") {
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": token})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestBootstrapStartVmTokens(t *testing.T) {
	setupCloudTestDB(t)
	credSrv := mockCredentialTokenServer(t, "bootstrap-token-abc")
	prev := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = credSrv.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prev })

	token, err := bootstrapStartVmTokens(
		t.Context(),
		"c1", "w1", "task1", "cmt1", "auth1", "aliyun", "cn-hangzhou", "cn-hangzhou-h",
	)
	if err != nil {
		t.Fatal(err)
	}
	if token != "bootstrap-token-abc" {
		t.Fatalf("token=%q", token)
	}
	cfgRow, err := loadCloudServerConfig("c1", "w1", "task1")
	if err != nil {
		t.Fatal(err)
	}
	if cfgRow.AuthorizationID != "auth1" || cfgRow.Region != "cn-hangzhou" {
		t.Fatalf("config=%+v", cfgRow)
	}
}

// 回归：start-vm bootstrap 必须在 RunInstances 前写出评论级 CSC。
// 否则 persist instance_id 0 行 → runtime-status「未找到服务器配置记录」。
func TestBootstrapStartVmTokensCreatesCommentCSC(t *testing.T) {
	setupCloudTestDB(t)
	credSrv := mockCredentialTokenServer(t, "bootstrap-token-cmt")
	prev := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = credSrv.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prev })

	if _, err := bootstrapStartVmTokens(
		t.Context(),
		"c1", "w1", "task-cmt-csc", "cmt_boot_csc", "auth1", "aliyun", "cn-hangzhou", "cn-hangzhou-h",
	); err != nil {
		t.Fatal(err)
	}
	cmt, err := loadCloudServerConfigForComment("c1", "w1", "task-cmt-csc", "cmt_boot_csc")
	if err != nil || cmt == nil {
		t.Fatalf("comment CSC missing after bootstrap: err=%v", err)
	}
	if cmt.CommentID != "cmt_boot_csc" {
		t.Fatalf("comment_id=%q", cmt.CommentID)
	}
	if cmt.Platform != "aliyun" || cmt.Region != "cn-hangzhou" || cmt.AuthorizationID != "auth1" {
		t.Fatalf("comment CSC meta platform=%s region=%s auth=%s", cmt.Platform, cmt.Region, cmt.AuthorizationID)
	}
}

func TestBootstrapPreservesExistingInstanceFields(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`
		INSERT INTO cloud_server_configs
			(id, company_id, workspace_id, task_id, platform, instance_id, public_ip, region, zone_id, authorization_id)
		VALUES ('cfg-old', 'c1', 'w1', 'task1', 'aliyun', 'i-existing', '1.2.3.4', 'cn-old', 'cn-old-a', 'auth-old')
	`)
	if err != nil {
		t.Fatal(err)
	}
	credSrv := mockCredentialTokenServer(t, "tok")
	prev := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = credSrv.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prev })

	_, err = bootstrapStartVmTokens(t.Context(), "c1", "w1", "task1", "cmt1", "auth-new", "aliyun", "cn-new", "cn-new-b")
	if err != nil {
		t.Fatal(err)
	}
	cfgRow, err := loadCloudServerConfig("c1", "w1", "task1")
	if err != nil {
		t.Fatal(err)
	}
	if cfgRow.InstanceID != "i-existing" || cfgRow.PublicIP != "1.2.3.4" {
		t.Fatalf("instance fields overwritten: %+v", cfgRow)
	}
	if cfgRow.AuthorizationID != "auth-new" || cfgRow.Region != "cn-new" {
		t.Fatalf("metadata not updated: %+v", cfgRow)
	}
}

func TestBootstrapStartVmTokensIncludesCommentInPath(t *testing.T) {
	setupCloudTestDB(t)
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/v1/token/init/") {
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "tok-cmt"})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	prev := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = srv.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prev })

	token, err := bootstrapStartVmTokens(t.Context(), "c1", "w1", "task1", "cmt_abc", "auth1", "aliyun", "cn-hangzhou", "cn-hangzhou-h")
	if err != nil {
		t.Fatal(err)
	}
	if token != "tok-cmt" {
		t.Fatalf("token=%q", token)
	}
	if !strings.Contains(gotPath, "/comment/cmt_abc") {
		t.Fatalf("init path missing comment: %s", gotPath)
	}
}
