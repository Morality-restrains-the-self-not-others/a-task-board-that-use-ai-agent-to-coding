package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveCloudAuthAgainstProdDB(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "850256677331562496", "862031128628060160")

	body := map[string]interface{}{
		"authorization_id":  "862031128628060160",
		"cloud_platform_id": "1",
	}
	rec, msg := resolveCloudAuthForStartVm("850256677331562496", body)
	if rec == nil {
		t.Fatalf("expected auth rec, msg=%q", msg)
	}
}

func TestResolveCloudAuthNumericSnowflakeID(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "850256677331562496", "862031128628060160")

	body := map[string]interface{}{
		"authorization_id":  float64(862031128628060160),
		"cloud_platform_id": "1",
	}
	rec, msg := resolveCloudAuthForStartVm("850256677331562496", body)
	if rec == nil {
		t.Fatalf("expected auth via int64(float64) or platform fallback, msg=%q", msg)
	}
}

func TestStartVmAutoStringAuthorizationIDProdDB(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "850256677331562496", "862031128628060160")

	prevKafka := cfg.KafkaBootstrapServers
	prevBill := cfg.TaskBillURL
	prevAI := cfg.AIProviderBaseURL
	cfg.KafkaBootstrapServers = ""
	cfg.TaskBillURL = ""
	cfg.AIProviderBaseURL = ""
	t.Cleanup(func() {
		cfg.KafkaBootstrapServers = prevKafka
		cfg.TaskBillURL = prevBill
		cfg.AIProviderBaseURL = prevAI
	})

	body := `{"task_id":"task_12590983282794675865","container_image_id":"862412768836349952","authorization_id":"862031128628060160","auto_create_security_group":true,"region_id":"cn-hongkong","cloud_platform_id":"1"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/850256677331562496/workspace/861623708318031872/cloud/compute/start-vm-auto/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "850256677331562496")
	req.Header.Set("X-Auth-User-Id", "850256676127797248")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/start-vm-auto/")

	if rec.Code == http.StatusBadRequest && strings.Contains(rec.Body.String(), "未配置云平台授权") {
		t.Fatalf("auth should resolve: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
