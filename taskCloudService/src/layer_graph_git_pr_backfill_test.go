package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCollectPrHtmlUrlsFromLayers(t *testing.T) {
	layers := []any{
		map[string]any{
			"layer_id": "L1",
			"git_remote": map[string]any{
				"pr_html_url": "https://gitlab.example/a/b/-/merge_requests/27",
			},
			"children": []any{
				map[string]any{
					"layer_id": "L1c",
					"git_remote": map[string]any{
						"pr_html_url": "https://gitlab.example/a/b/-/merge_requests/27",
					},
				},
			},
		},
		map[string]any{
			"layer_id": "L2",
			"pr":       map[string]any{"html_url": "https://github.com/acme/demo/pull/3"},
		},
	}
	got := collectPrHtmlUrlsFromLayers(layers)
	if len(got) != 2 {
		t.Fatalf("urls=%v", got)
	}
	if got[0] != "https://gitlab.example/a/b/-/merge_requests/27" {
		t.Fatalf("first=%q", got[0])
	}
	if got[1] != "https://github.com/acme/demo/pull/3" {
		t.Fatalf("second=%q", got[1])
	}
}

func TestHandleLayerGraphPushCreatesGitPrReply(t *testing.T) {
	setupCloudTestDB(t)
	origPub := publishLayerGraphSnapshotPersisted
	t.Cleanup(func() { publishLayerGraphSnapshotPersisted = origPub })
	publishLayerGraphSnapshotPersisted = func(ctx context.Context, data map[string]interface{}, key string) error {
		return nil
	}

	var posts int
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/created-by") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"comment_id":"cmt-a","user_id":"u-author"}`)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		posts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"cmt_pr_layer"}`)
	}))
	defer taskSrv.Close()
	prevTask, prevSec := cfg.TaskServiceURL, cfg.InternalSecret
	cfg.TaskServiceURL = taskSrv.URL
	cfg.InternalSecret = "sec-layer"
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevTask
		cfg.InternalSecret = prevSec
	})

	htmlURL := "https://gitlab.example/a/b/-/merge_requests/27"
	cfgRow := &CloudServerConfig{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a"}
	rec := httptest.NewRecorder()
	handleLayerGraphPush(rec, nil, cfgRow, map[string]any{
		"layers": []any{map[string]any{
			"layer_id":   "L1",
			"git_remote": map[string]any{"pr_html_url": htmlURL},
		}},
		"jobs": []any{},
	}, "t1", "ws1", "task1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if posts != 1 {
		t.Fatalf("git_pr comment posts=%d", posts)
	}
}

func TestGetContainerLayerGraphBackfillsGitPrReply(t *testing.T) {
	setupCloudTestDB(t)
	htmlURL := "https://gitlab.example/a/b/-/merge_requests/28"
	raw, _ := json.Marshal(map[string]any{
		"layers": []any{map[string]any{
			"layer_id":   "L9",
			"git_remote": map[string]any{"pr_html_url": htmlURL},
		}},
		"jobs": []any{},
	})
	if err := upsertLayerGraphSnapshot(layerGraphSnapshotRow{
		CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a",
		GraphJSON: string(raw),
	}); err != nil {
		t.Fatal(err)
	}

	var posts int
	taskSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/created-by") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"user_id":"u-author"}`)
			return
		}
		posts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"cmt_pr_get"}`)
	}))
	defer taskSrv.Close()
	prevTask, prevSec := cfg.TaskServiceURL, cfg.InternalSecret
	cfg.TaskServiceURL = taskSrv.URL
	cfg.InternalSecret = "sec-get"
	t.Cleanup(func() {
		cfg.TaskServiceURL = prevTask
		cfg.InternalSecret = prevSec
	})

	req := httptest.NewRequest(http.MethodGet,
		"/api/cloud/compute/container-layer-graph/tenant_id/t1/workspace_id/ws1/task_id/task1/comment_id/cmt-a/",
		nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	req.Header.Set("X-Task-Id", "task1")
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, req, "compute/container-layer-graph/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if posts != 1 {
		t.Fatalf("git_pr comment posts=%d", posts)
	}
}
