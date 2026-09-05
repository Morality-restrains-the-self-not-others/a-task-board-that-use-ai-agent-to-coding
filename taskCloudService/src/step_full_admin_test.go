package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"taskCloudService/domain"
)

func TestSystemAdminStepFullCOSForbiddenWithoutStaff(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/step-full-cos/", nil)
	req.Header.Set("X-User-Id", "regular")
	rec := httptest.NewRecorder()
	handleSystemAdminStepFullCOS(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403 body=%s", rec.Code, rec.Body.String())
	}
}

func TestSystemAdminStepFullCOSGetHidesSecrets(t *testing.T) {
	dir := t.TempDir()
	stepFullCOSWriteDir = dir
	t.Cleanup(func() { stepFullCOSWriteDir = "" })
	stepFullCOSCfg = StepFullCOSConfig{
		Backend: "local", PathRule: domain.DefaultStepFullPathRule,
		SecretID: "AKIDxxx", SecretKey: "super-secret",
	}
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/step-full-cos/", nil)
	req.Header.Set("X-User-Id", "admin")
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	handleSystemAdminStepFullCOS(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["secretKey"]; ok {
		t.Fatalf("secretKey leaked: %v", body)
	}
	if _, ok := body["secretId"]; ok {
		t.Fatalf("secretId leaked: %v", body)
	}
	if body["secret_configured"] != true {
		t.Fatalf("secret_configured=%v", body["secret_configured"])
	}
	if _, ok := body["startupLogsPathRule"]; !ok {
		t.Fatalf("startupLogsPathRule missing: %v", body)
	}
	if strings.Contains(rec.Body.String(), "super-secret") {
		t.Fatal("secret value leaked")
	}
}

func TestSystemAdminStepFullCOSPatchWritesFragment(t *testing.T) {
	dir := t.TempDir()
	stepFullCOSWriteDir = dir
	t.Cleanup(func() { stepFullCOSWriteDir = "" })
	stepFullCOSCfg = StepFullCOSConfig{Backend: "local", PathRule: domain.DefaultStepFullPathRule}
	req := httptest.NewRequest(http.MethodPatch, "/api/system-admin/step-full-cos/", strings.NewReader(
		`{"backend":"local","bucket":"demo-bucket","region":"ap-guangzhou","pathRule":"workspace_{workspaceId}/task_{taskId}/comment_{commentId}/layer_{layerId}/step_full.json","secretId":"ak","secretKey":"sk"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin")
	req.Header.Set("X-User-Roles", "employee")
	rec := httptest.NewRecorder()
	handleSystemAdminStepFullCOS(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "step-full-cos.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "demo-bucket") {
		t.Fatalf("public yaml=%s", raw)
	}
	if strings.Contains(string(raw), "secretKey") {
		t.Fatalf("secret in public yaml: %s", raw)
	}
	sraw, err := os.ReadFile(filepath.Join(dir, "conf-local", "step-full-cos.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sraw), "sk") {
		t.Fatalf("local yaml=%s", sraw)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if strings.Contains(rec.Body.String(), `"sk"`) {
		t.Fatal("response leaked secretKey")
	}
}

func TestSystemAdminStepFullCOSPatchRejectsDotDot(t *testing.T) {
	dir := t.TempDir()
	stepFullCOSWriteDir = dir
	t.Cleanup(func() { stepFullCOSWriteDir = "" })
	req := httptest.NewRequest(http.MethodPatch, "/api/system-admin/step-full-cos/", strings.NewReader(
		`{"pathRule":"workspace_{workspaceId}/../x"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin")
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	handleSystemAdminStepFullCOS(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSystemAdminStepFullCOSPatchRejectsStartupLogsDotDot(t *testing.T) {
	dir := t.TempDir()
	stepFullCOSWriteDir = dir
	t.Cleanup(func() { stepFullCOSWriteDir = "" })
	req := httptest.NewRequest(http.MethodPatch, "/api/system-admin/step-full-cos/", strings.NewReader(
		`{"startupLogsPathRule":"workspace_{workspaceId}/../startup_logs.json"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin")
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	handleSystemAdminStepFullCOS(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
