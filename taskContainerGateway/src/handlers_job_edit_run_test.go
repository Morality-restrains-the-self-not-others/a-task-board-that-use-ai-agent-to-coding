package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tracelog"
)

func TestHandleContainerJobEditRun_DeletesDescendantsCreatesAndStartsStream(t *testing.T) {
	var deleted []string
	var createBody map[string]any
	var sawPublish atomic.Bool

	prevPublish := jobStreamPublishFunc
	jobStreamPublishFunc = func(ctx context.Context, sc scope, jobID, traceID string, fields map[string]any) {
		sawPublish.Store(true)
	}
	t.Cleanup(func() { jobStreamPublishFunc = prevPublish })

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(path, "/jobs"):
			if r.Header.Get("X-Access-Token") != "tok" {
				http.Error(w, "token", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"jobs": []any{
					map[string]any{"id": "J1", "parent_job_id": "P1", "repo_layer_id": nil, "command_kind": "trae"},
					map[string]any{"id": "C1", "parent_job_id": "J1", "repo_layer_id": nil, "command_kind": "trae"},
					map[string]any{"id": "GC1", "parent_job_id": "C1", "repo_layer_id": nil, "command_kind": "trae"},
				},
			})
		case r.Method == http.MethodDelete && strings.Contains(path, "/jobs/"):
			deleted = append(deleted, path)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/jobs"):
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &createBody)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "JNEW", "status": "pending"})
		case r.Method == http.MethodGet && strings.Contains(path, "/jobs/JNEW"):
			// job-stream poll may hit events/job; return empty terminal-ish payload
			if strings.HasSuffix(path, "/events") {
				_ = json.NewEncoder(w).Encode(map[string]any{"events": []any{}, "next_offset": 0})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "JNEW", "status": "succeeded", "output": ""})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	authURL, cloudURL := startAuthCloudHotPathMocks(t, upstream.URL, "tok")

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloudURL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-job-edit-run/"
	body := `{"job_id":"J1","command":"echo hi","command_kind":"trae","agent_auto_iteration_count":222,"parent_comment_id":"cmt-parent","installed_image_id":"img-9"}`
	mux := http.NewServeMux()
	mountRoutes(mux)
	handler := tracelog.Middleware(mux)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token test-token-abc12345")
	req.Header.Set(tracelog.Header, "trace-editrun1234567890")
	req.Header.Set(tracelog.ParentSpanHeader, "1122334455667788")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("ok=%v", resp["ok"])
	}
	if resp["deleted_descendants"] != float64(2) {
		t.Fatalf("deleted_descendants=%v", resp["deleted_descendants"])
	}
	job, _ := resp["job"].(map[string]any)
	if job["id"] != "JNEW" {
		t.Fatalf("job=%v", job)
	}
	if len(deleted) != 3 {
		t.Fatalf("deleted paths=%v", deleted)
	}
	// leaf before parent: GC1 → C1 → J1
	if !strings.HasSuffix(deleted[0], "/jobs/GC1") ||
		!strings.HasSuffix(deleted[1], "/jobs/C1") ||
		!strings.HasSuffix(deleted[2], "/jobs/J1") {
		t.Fatalf("delete order unexpected: %v", deleted)
	}
	if createBody["parent_job_id"] != "P1" {
		t.Fatalf("create body=%v", createBody)
	}
	if createBody["edit_run_delivery"] != true {
		t.Fatalf("edit_run_delivery=%v create body=%v", createBody["edit_run_delivery"], createBody)
	}
	if createBody["mounted_parent_comment_id"] != "cmt-parent" {
		t.Fatalf("mounted_parent_comment_id=%v", createBody["mounted_parent_comment_id"])
	}
	if createBody["edit_run_installed_image_id"] != "img-9" {
		t.Fatalf("edit_run_installed_image_id=%v", createBody["edit_run_installed_image_id"])
	}
	if createBody["auto_run_commit_message"] != "echo hi" {
		t.Fatalf("auto_run_commit_message=%v", createBody["auto_run_commit_message"])
	}
	env, _ := createBody["env"].(map[string]any)
	if env["TASK_AGENT_MAX_STEPS"] != "222" {
		t.Fatalf("env=%v", env)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sawPublish.Load() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !sawPublish.Load() {
		t.Fatal("expected job-stream publish after create")
	}
}

func TestHandleContainerJobEditRun_RejectsMissingJob(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"jobs": []any{}})
	}))
	defer upstream.Close()
	authURL, cloudURL := startAuthCloudHotPathMocks(t, upstream.URL, "tok")
	cfg = serviceConfig{
		TaskAuthURL: authURL, CloudServiceURL: cloudURL,
		InternalSecret: "secret", ForwardReadSec: 5, ForwardConnectSec: 1,
	}

	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/tenant/1/workspace/2/task/3/cloud/compute/container-job-edit-run/",
		strings.NewReader(`{"job_id":"missing","command":"x","command_kind":"shell"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token t")
	req.Header.Set(tracelog.Header, "trace-editrun-missing01")
	req.Header.Set(tracelog.ParentSpanHeader, "1122334455667788")
	rec := httptest.NewRecorder()
	tracelog.Middleware(mux).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleContainerJobEditRun_UpstreamJobs404(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"detail":"Not Found"}`)
	}))
	defer upstream.Close()
	authURL, cloudURL := startAuthCloudHotPathMocks(t, upstream.URL, "tok")
	cfg = serviceConfig{
		TaskAuthURL: authURL, CloudServiceURL: cloudURL,
		InternalSecret: "secret", ForwardReadSec: 5, ForwardConnectSec: 1,
	}

	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/tenant/1/workspace/2/task/3/cloud/compute/container-job-edit-run/",
		strings.NewReader(`{"job_id":"J1","command":"echo","command_kind":"trae"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token t")
	req.Header.Set(tracelog.Header, "trace-editrun-jobs40401")
	req.Header.Set(tracelog.ParentSpanHeader, "1122334455667788")
	rec := httptest.NewRecorder()
	tracelog.Middleware(mux).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if payload["http_status"] != float64(404) {
		t.Fatalf("payload=%v", payload)
	}
	up, _ := payload["upstream"].(map[string]any)
	if up["status"] != float64(404) {
		t.Fatalf("upstream=%v", up)
	}
}
