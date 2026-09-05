package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"tracelog"
)

func TestPatchWorkspaceAllowPersonalHTTPSetsInternalUser(t *testing.T) {
	var gotUser, gotTenant, gotMethod, gotTraceID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = r.Header.Get("X-Auth-User-Id")
		gotTenant = r.Header.Get("X-Auth-Tenant-Id")
		gotTraceID = r.Header.Get("X-Trace-Id")
		gotMethod = r.Method
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	prevURL, prevClient := cfg.ProjectServiceURL, featureParamsHTTPClient
	t.Cleanup(func() {
		cfg.ProjectServiceURL = prevURL
		featureParamsHTTPClient = prevClient
	})
	cfg.ProjectServiceURL = srv.URL
	featureParamsHTTPClient = srv.Client()

	// OPT-20260821-012: 入站 X-Trace-Id 必须透传到 taskProjectService。
	ctx := tracelog.ContextWithCorrelation(context.Background(), tracelog.Correlation{TraceID: "trace-opt-012-2"})
	if err := patchWorkspaceAllowPersonalHTTP(ctx, "t1", "ws_-2309487803472456748", true); err != nil {
		t.Fatalf("patchWorkspaceAllowPersonalHTTP: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Fatalf("method=%q want PATCH", gotMethod)
	}
	if gotTraceID != "trace-opt-012-2" {
		t.Fatalf("X-Trace-Id=%q want trace-opt-012-2 (OPT-20260821-012 propagation)", gotTraceID)
	}
	if gotTenant != "t1" {
		t.Fatalf("X-Auth-Tenant-Id=%q want t1", gotTenant)
	}
	if gotUser != "internal" {
		t.Fatalf("X-Auth-User-Id=%q want internal (OPT-20260820-015 requireTenantMember bypass)", gotUser)
	}
}
