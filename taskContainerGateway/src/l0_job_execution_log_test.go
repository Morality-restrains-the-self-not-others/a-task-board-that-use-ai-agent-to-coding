package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tracelog"
)

func TestHandleContainerJobExecutionLog_ReturnsWrappedJobStepsLayerChanges(t *testing.T) {
	var sawJob, sawSteps, sawLayerChanges bool
	var stepsQuery string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Access-Token") != "tok" {
			http.Error(w, "token", http.StatusUnauthorized)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/jobs/J1") && r.Method == http.MethodGet:
			sawJob = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "J1", "layer_id": "L1", "status": "succeeded", "output": "hello from job",
			})
		case strings.HasSuffix(r.URL.Path, "/jobs/J1/steps") && r.Method == http.MethodGet:
			sawSteps = true
			stepsQuery = r.URL.RawQuery
			_ = json.NewEncoder(w).Encode(map[string]any{
				"steps":           []any{map[string]any{"step_number": 1, "kind": "assistant"}},
				"note":            nil,
				"has_more":        false,
				"next_after_step": nil,
				"total_steps":     1,
			})
		case strings.Contains(r.URL.Path, "/layers/L1/diff/parent/files") && r.Method == http.MethodGet:
			sawLayerChanges = true
			if !strings.Contains(r.URL.RawQuery, "offset=0") || !strings.Contains(r.URL.RawQuery, "limit=100") {
				t.Fatalf("layer changes query=%q", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"layer_id":        "L1",
				"parent_layer_id": "P1",
				"same":            false,
				"truncated":       false,
				"change_count":    1,
				"has_more":        false,
				"next_offset":     1,
				"changes": []any{
					map[string]any{"path": "src/a.py", "kind": "modified"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, upstream.URL, "tok")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"company_member_id": "m1", "user_id": "42", "is_admin": false,
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": "tok",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-job-execution-log/?job_id=J1&layer_id=L1&after_step=0&limit=20"
	mux := http.NewServeMux()
	mountRoutes(mux)
	handler := tracelog.Middleware(mux)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Token test-token-abc12345")
	req.Header.Set(tracelog.Header, "trace-execlog1234567890")
	req.Header.Set(tracelog.ParentSpanHeader, "1122334455667788")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !sawJob || !sawSteps || !sawLayerChanges {
		t.Fatalf("upstream calls job=%v steps=%v layer_changes=%v", sawJob, sawSteps, sawLayerChanges)
	}
	if !strings.Contains(stepsQuery, "after_step=0") || !strings.Contains(stepsQuery, "limit=20") {
		t.Fatalf("steps query=%q", stepsQuery)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	job, ok := body["job"].(map[string]any)
	if !ok {
		t.Fatalf("expected wrapped job object, got keys=%v body=%s", keysOf(body), rec.Body.String())
	}
	if _, hasOut := job["output"]; hasOut {
		t.Fatalf("output must be stripped, job=%v", job)
	}
	if job["output_omitted"] != true || job["status"] != "succeeded" {
		t.Fatalf("job=%v", job)
	}
	steps, ok := body["steps"].(map[string]any)
	if !ok {
		t.Fatalf("expected steps object, got %T %v", body["steps"], body["steps"])
	}
	stepList, ok := steps["steps"].([]any)
	if !ok || len(stepList) != 1 {
		t.Fatalf("steps.steps=%v", steps["steps"])
	}
	lc, ok := body["layer_changes"].(map[string]any)
	if !ok {
		t.Fatalf("expected layer_changes object, got %T", body["layer_changes"])
	}
	if int(lc["change_count"].(float64)) != 1 {
		t.Fatalf("change_count=%v", lc["change_count"])
	}
}

func TestHandleContainerJobExecutionLog_Meta0OmitsEmptyStatus(t *testing.T) {
	var sawJobGET bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/jobs/Jmeta") && r.Method == http.MethodGet:
			sawJobGET = true
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "Jmeta", "status": "running"})
		case strings.HasSuffix(r.URL.Path, "/jobs/Jmeta/steps"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"steps":    []any{map[string]any{"step_number": 1, "state": "completed"}},
				"has_more": false,
			})
		case strings.Contains(r.URL.Path, "/layers/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"changes": []any{}, "change_count": 0, "truncated": false,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, upstream.URL, "tok")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"company_member_id": "m1", "user_id": "42", "is_admin": false,
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": "tok",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-job-execution-log/?job_id=Jmeta&layer_id=L1&meta=0"
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Token test-token-abc12345")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if sawJobGET {
		t.Fatal("meta=0 must not GET /jobs/:id")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	job, ok := body["job"].(map[string]any)
	if !ok {
		t.Fatalf("job missing: %s", rec.Body.String())
	}
	if _, hasStatus := job["status"]; hasStatus {
		t.Fatalf("meta=0 stub must omit status key, got job=%v", job)
	}
	if job["id"] != "Jmeta" {
		t.Fatalf("job.id=%v", job["id"])
	}
}

func TestHandleContainerJobExecutionLog_JobMetaFailStillReturnsSteps(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/jobs/J2") && r.Method == http.MethodGet:
			http.Error(w, `{"detail":"too large"}`, http.StatusGatewayTimeout)
		case strings.HasSuffix(r.URL.Path, "/jobs/J2/steps"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"steps":    []any{map[string]any{"step_number": 1}},
				"has_more": false,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	authURL, _ := startAuthCloudHotPathMocks(t, upstream.URL, "tok")
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/tenant-member"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"company_member_id": "m1", "user_id": "42", "is_admin": false,
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "cfg1"})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/container-target"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"base_url": upstream.URL, "access_token": "tok",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer cloud.Close()

	cfg = serviceConfig{
		TaskAuthURL:       authURL,
		CloudServiceURL:   cloud.URL,
		InternalSecret:    "secret",
		ForwardReadSec:    5,
		ForwardConnectSec: 1,
	}

	path := "/api/tenant/1/workspace/2/task/3/cloud/compute/container-job-execution-log/?job_id=J2&layer_id=L2"
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Token test-token-abc12345")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	raw, _ := io.ReadAll(rec.Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	job, ok := body["job"].(map[string]any)
	if !ok || job["id"] != "J2" {
		t.Fatalf("expected stub job, body=%s", string(raw))
	}
	if job["output_omitted"] != true {
		t.Fatalf("job=%v", job)
	}
	steps, ok := body["steps"].(map[string]any)
	if !ok {
		t.Fatalf("expected steps object, got %T", body["steps"])
	}
	stepList, _ := steps["steps"].([]any)
	if len(stepList) != 1 {
		t.Fatalf("steps=%v", steps)
	}
}

func TestStripJobOutputForExecLog(t *testing.T) {
	out := stripJobOutputForExecLog(map[string]any{
		"id": "J", "output": "abc", "status": "completed",
	})
	if _, ok := out["output"]; ok {
		t.Fatal("output should be removed")
	}
	if out["output_omitted"] != true {
		t.Fatalf("%v", out)
	}
	if out["output_chars"] != 3 {
		t.Fatalf("output_chars=%v", out["output_chars"])
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
