package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWorkspaceMachineSummaryRequiresAuth(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/cloud/compute/workspace-machine-summary/", nil)
	rec := httptest.NewRecorder()
	handleWorkspaceMachineSummary(rec, req, "t1", "ws1")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401 body=%s", rec.Code, rec.Body.String())
	}
}

func TestWorkspaceRuntimeIndicatorsRequiresAuth(t *testing.T) {
	setupCloudTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/cloud/compute/workspace-runtime-indicators/", nil)
	rec := httptest.NewRecorder()
	handleWorkspaceRuntimeIndicators(rec, req, "t1", "ws1")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401 body=%s", rec.Code, rec.Body.String())
	}
}
