package taskcommentimagementioned

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchProjectUsesConventionPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if strings.Contains(r.URL.Path, "/api/tenant/") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "proj_1",
			"server_run_template": map[string]interface{}{
				"region": "cn-hongkong",
			},
		})
	}))
	defer srv.Close()

	c := &httpProjectClient{BaseURL: srv.URL, HTTPClient: srv.Client()}
	out, err := c.FetchProject(context.Background(), "ten_1", "proj_1", "user_1")
	if err != nil {
		t.Fatalf("FetchProject: %v (path=%s)", err, gotPath)
	}
	if out["id"] != "proj_1" {
		t.Fatalf("id=%v path=%s", out["id"], gotPath)
	}
	wantPrefix := "/api/projects/tenant_id/ten_1/proj_1"
	if !strings.HasPrefix(gotPath, wantPrefix) {
		t.Fatalf("path=%s want prefix %s (legacy /api/tenant/.../projects is retired)", gotPath, wantPrefix)
	}
}

// by-parent 回写状态遇 404（AI 评论不存在/已清理）返回 ErrAICommentNotFound 哨兵，
// 供调用方降级为 warn 而非视为启服失败（OPT-20260811-056）。
func TestSetRunStatusByParent_NotFoundReturnsSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not found"}`))
	}))
	defer srv.Close()

	c := &httpAICommentClient{BaseURL: srv.URL, HTTPClient: srv.Client()}
	err := c.SetRunStatusByParent(context.Background(), "cmt_missing", "starting")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !errors.Is(err, ErrAICommentNotFound) {
		t.Fatalf("err = %v, want errors.Is(ErrAICommentNotFound)", err)
	}
}

func TestStartVMUsesConventionPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if strings.Contains(r.URL.Path, "/api/tenant/") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := &httpCloudClient{BaseURL: srv.URL, HTTPClient: srv.Client()}
	err := c.StartVM(context.Background(), "ten_1", "ws_1", "start-vm-auto", "user_1", map[string]interface{}{
		"task_id": "task_1",
	})
	if err != nil {
		t.Fatalf("StartVM: %v (path=%s)", err, gotPath)
	}
	wantPrefix := "/api/cloud/compute/start-vm-auto/tenant_id/ten_1/workspace_id/ws_1"
	if !strings.HasPrefix(gotPath, wantPrefix) {
		t.Fatalf("path=%s want prefix %s (legacy /api/tenant/.../cloud/compute is retired)", gotPath, wantPrefix)
	}
}
