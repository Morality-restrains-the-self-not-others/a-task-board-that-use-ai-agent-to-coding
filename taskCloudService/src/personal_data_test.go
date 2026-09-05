package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func insertCloudServerForUser(t *testing.T, userID, id, taskID string, extra string) {
	t.Helper()
	_, err := db.Exec(fmt.Sprintf(`
		INSERT INTO cloud_server_configs (
			id, company_id, workspace_id, task_id, image_invoker_user_id,
			platform, instance_id, region, public_ip, server_url,
			last_runtime_status, started_via, verification_secret, client_token
		) VALUES (?, 'c1', 'ws1', ?, ?, 'aliyun', 'i-test', 'cn-hangzhou',
			'203.0.113.9', 'https://terminal.example.test', 'running', 'web', 'sec-abc', 'tok-abc')%s`, extra),
		id, taskID, userID)
	if err != nil {
		t.Fatalf("insert cloud_server_configs: %v", err)
	}
}

func TestInternalCloudPersonalDataCollectsServers(t *testing.T) {
	setupCloudTestDB(t)
	userID := "cloud-user-001"
	insertCloudServerForUser(t, userID, "csc-1", "task-1", "")
	// 他人（非本用户 invoker）的机器不得泄漏
	insertCloudServerForUser(t, "other-user", "csc-2", "task-2", "")

	req := httptest.NewRequest(http.MethodGet, "/api/internal/cloud/users/"+userID+"/personal-data/", nil)
	w := httptest.NewRecorder()
	handleInternalCloudUsersRouter(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var wrapper struct {
		Data struct {
			CloudServers []map[string]interface{} `json:"cloud_servers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(wrapper.Data.CloudServers) != 1 {
		t.Fatalf("cloud_servers=%d want 1 (other user's server must not leak)", len(wrapper.Data.CloudServers))
	}
	srv := wrapper.Data.CloudServers[0]
	if srv["id"] != "csc-1" || srv["task_id"] != "task-1" {
		t.Fatalf("unexpected server: %v", srv)
	}
	// 凭证不导出
	raw := w.Body.String()
	for _, forbidden := range []string{"verification_secret", "client_token", "business_api_endpoint", "sec-abc", "tok-abc"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("export leaks credential %q", forbidden)
		}
	}
}

func TestInternalCloudPersonalDataEmptyUser(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/internal/cloud/users/unknown-user/personal-data/", nil)
	w := httptest.NewRecorder()
	handleInternalCloudUsersRouter(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var wrapper struct {
		Data struct {
			CloudServers []map[string]interface{} `json:"cloud_servers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(wrapper.Data.CloudServers) != 0 {
		t.Fatalf("cloud_servers=%d want 0", len(wrapper.Data.CloudServers))
	}
}

func TestInternalCloudPersonalDataRequiresSecret(t *testing.T) {
	setupCloudTestDB(t)
	prev := cfg.InternalSecret
	cfg.InternalSecret = "cloud-secret"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/internal/cloud/users/u1/personal-data/", nil)
	w := httptest.NewRecorder()
	handleInternalCloudUsersRouter(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", w.Code)
	}
}
