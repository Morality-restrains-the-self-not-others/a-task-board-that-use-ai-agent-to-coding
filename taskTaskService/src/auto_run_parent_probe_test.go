package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestProbeParentGitReposAccessUsesTokenOnly documents OPT-20260818-047:
// auto_clone_nested_repos=false must check OAuth token presence, not remote
// GitLab REST (which took 24s on gitlab-tencent-sh-1 while the client was 15s).
func TestProbeParentGitReposAccessUsesTokenOnly(t *testing.T) {
	var gotProbe *bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !strings.Contains(r.URL.Path, "/validate-git-repos") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]interface{}
		_ = json.Unmarshal(raw, &body)
		if v, ok := body["probe_access"].(bool); ok {
			gotProbe = &v
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{"url": "https://gitlab.example/g/repo", "is_accessible": true, "message": ""},
			},
		})
	}))
	t.Cleanup(srv.Close)
	prev := cfg.ProjectServiceURL
	cfg.ProjectServiceURL = srv.URL
	t.Cleanup(func() { cfg.ProjectServiceURL = prev })

	if reason := probeParentGitReposAccess("42", "t1", "https://gitlab.example/g/repo"); reason != "" {
		t.Fatalf("token-only accessible parent should proceed, got %q", reason)
	}
	if gotProbe == nil {
		t.Fatal("validate-git-repos body missing probe_access")
	}
	if *gotProbe {
		t.Fatal("auto_clone=false parent probe must use probe_access=false (token only); remote GitLab REST timed out the 15s client")
	}
}

func TestProjectGitProbeTimeoutCoversObservedRemoteLatency(t *testing.T) {
	// Incident task_878541740905099264: validate-git-repos returned 200 in 24177ms
	// after git-oauth 200 in 3590ms; projectHTTP 15s aborted first.
	if projectGitProbeHTTP.Timeout < 40*time.Second {
		t.Fatalf("projectGitProbeHTTP.Timeout=%v want >=40s so token+remote probe cannot starve auto_run", projectGitProbeHTTP.Timeout)
	}
}

func TestParentGitProbeFailureReasonTimeout(t *testing.T) {
	got := parentGitProbeFailureReason(errors.New(`Post "http://127.0.0.1:8016/api/projects/tenant_id/t/validate-git-repos": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`), 0)
	if !strings.Contains(got, "超时") {
		t.Fatalf("timeout skip reason=%q want 超时 (not generic 探测失败)", got)
	}
	gotHTTP := parentGitProbeFailureReason(nil, http.StatusForbidden)
	if !strings.Contains(gotHTTP, "403") {
		t.Fatalf("http skip reason=%q want status 403", gotHTTP)
	}
}
