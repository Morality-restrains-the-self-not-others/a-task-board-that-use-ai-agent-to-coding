package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleFeatureParamsEnvPreviewRequiresAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?source=personal&personal_config_id=1", nil)
	rec := httptest.NewRecorder()
	handleFeatureParamsEnvPreview(rec, req, "t1", "w1", "task1")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleFeatureParamsEnvPreviewBlocksNonPersonal(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?source=company&personal_config_id=1", nil)
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleFeatureParamsEnvPreview(rec, req, "t1", "w1", "task1")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleFeatureParamsEnvPreviewMissingConfigID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x?source=personal", nil)
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleFeatureParamsEnvPreview(rec, req, "t1", "w1", "task1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPersonalConfigDisplayName(t *testing.T) {
	if got := personalConfigDisplayName("个人配置:日常"); got != "日常" {
		t.Fatalf("got %q", got)
	}
	if got := personalConfigDisplayName("plain"); got != "plain" {
		t.Fatalf("got %q", got)
	}
}

func TestCloudTaskRoutesDispatchesEnvPreview(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/x/cloud/compute/feature-params-env-preview/?source=company", nil)
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "w1")
	req.Header.Set("X-Task-Id", "task1")
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, req, "cloud/compute/feature-params-env-preview/")
	if rec.Code == http.StatusNotImplemented {
		t.Fatalf("must not 501: %s", rec.Body.String())
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 for non-personal, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["message"] != "仅支持预览个人配置" {
		t.Fatalf("unexpected body: %v", body)
	}
}
