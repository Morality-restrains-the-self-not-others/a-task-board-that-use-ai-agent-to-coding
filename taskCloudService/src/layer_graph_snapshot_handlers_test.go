package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUpsertLayerGraphSnapshotLastWriteWins(t *testing.T) {
	setupCloudTestDB(t)
	row := layerGraphSnapshotRow{
		CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a",
		GraphJSON: `{"layers":[{"layer_id":"L1"}],"jobs":[]}`,
	}
	if err := upsertLayerGraphSnapshot(row); err != nil {
		t.Fatal(err)
	}
	if err := upsertLayerGraphSnapshot(layerGraphSnapshotRow{
		CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a",
		GraphJSON: `{"layers":[{"layer_id":"L2"}],"jobs":[{"id":"J1"}]}`,
	}); err != nil {
		t.Fatal(err)
	}
	n, err := countLayerGraphSnapshots("ws1", "task1", "cmt-a")
	if err != nil || n != 1 {
		t.Fatalf("count=%d err=%v", n, err)
	}
	got, found, err := getLayerGraphSnapshot("ws1", "task1", "cmt-a")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if !strings.Contains(got.GraphJSON, `"L2"`) {
		t.Fatalf("graph=%s", got.GraphJSON)
	}
	_, foundOther, err := getLayerGraphSnapshot("ws-other", "task1", "cmt-a")
	if err != nil || foundOther {
		t.Fatalf("cross-workspace leak found=%v err=%v", foundOther, err)
	}
}

func TestUpsertLayerGraphSnapshotRejectsMissingKeys(t *testing.T) {
	setupCloudTestDB(t)
	err := upsertLayerGraphSnapshot(layerGraphSnapshotRow{
		WorkspaceID: "ws1", TaskID: "task1", GraphJSON: `{"layers":[],"jobs":[]}`,
	})
	if err == nil {
		t.Fatal("want error for missing comment")
	}
}

func TestHandleLayerGraphPushPersistsAndPublishes(t *testing.T) {
	setupCloudTestDB(t)
	var published []map[string]interface{}
	orig := publishLayerGraphSnapshotPersisted
	t.Cleanup(func() { publishLayerGraphSnapshotPersisted = orig })
	publishLayerGraphSnapshotPersisted = func(ctx context.Context, data map[string]interface{}, key string) error {
		published = append(published, data)
		return nil
	}
	cfg := &CloudServerConfig{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a"}
	rec := httptest.NewRecorder()
	handleLayerGraphPush(rec, context.Background(), cfg, map[string]any{
		"layers": []any{map[string]any{"layer_id": "L1"}},
		"jobs":   []any{},
	}, "t1", "ws1", "task1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(published) != 1 || published[0]["comment_id"] != "cmt-a" {
		t.Fatalf("published=%v", published)
	}
	n, _ := countLayerGraphSnapshots("ws1", "task1", "cmt-a")
	if n != 1 {
		t.Fatalf("rows=%d", n)
	}
}

func TestHandleLayerGraphPushSkipsIdenticalGraph(t *testing.T) {
	setupCloudTestDB(t)
	var published int
	orig := publishLayerGraphSnapshotPersisted
	t.Cleanup(func() { publishLayerGraphSnapshotPersisted = orig })
	publishLayerGraphSnapshotPersisted = func(ctx context.Context, data map[string]interface{}, key string) error {
		published++
		return nil
	}
	cfg := &CloudServerConfig{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a"}
	body := map[string]any{
		"layers": []any{map[string]any{"layer_id": "L1"}},
		"jobs":   []any{},
	}
	handleLayerGraphPush(httptest.NewRecorder(), context.Background(), cfg, body, "t1", "ws1", "task1")
	handleLayerGraphPush(httptest.NewRecorder(), context.Background(), cfg, body, "t1", "ws1", "task1")
	if published != 1 {
		t.Fatalf("published=%d want 1 (identical graph skips Kafka)", published)
	}
	n, _ := countLayerGraphSnapshots("ws1", "task1", "cmt-a")
	if n != 1 {
		t.Fatalf("rows=%d", n)
	}
}

func TestHandleLayerGraphPushRejectsNonArrays(t *testing.T) {
	setupCloudTestDB(t)
	cfg := &CloudServerConfig{CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a"}
	rec := httptest.NewRecorder()
	handleLayerGraphPush(rec, context.Background(), cfg, map[string]any{
		"layers": map[string]any{},
		"jobs":   []any{},
	}, "t1", "ws1", "task1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
	n, _ := countLayerGraphSnapshots("ws1", "task1", "cmt-a")
	if n != 0 {
		t.Fatalf("should not persist n=%d", n)
	}
}

func TestGetContainerLayerGraphHydratesFromDB(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertLayerGraphSnapshot(layerGraphSnapshotRow{
		CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a",
		GraphJSON: `{"layers":[{"layer_id":"L9"}],"jobs":[]}`,
	}); err != nil {
		t.Fatal(err)
	}
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
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["source"] != "saas_db" {
		t.Fatalf("source=%v", body["source"])
	}
	layers, _ := body["layers"].([]any)
	if len(layers) != 1 {
		t.Fatalf("layers=%v", layers)
	}
}

func TestGetContainerLayerGraphEmptyFallsBackToLiveGateway(t *testing.T) {
	setupCloudTestDB(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-TaskContainerGateway-Internal-Secret") != "tcg-test" {
			http.Error(w, `{"detail":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"layers":[{"layer_id":"L-live"}],"jobs":[]}`))
	}))
	t.Cleanup(upstream.Close)
	prevURL, prevSec := cfg.ContainerGatewayURL, cfg.ContainerGatewayInternalSecret
	cfg.ContainerGatewayURL = upstream.URL
	cfg.ContainerGatewayInternalSecret = "tcg-test"
	t.Cleanup(func() {
		cfg.ContainerGatewayURL = prevURL
		cfg.ContainerGatewayInternalSecret = prevSec
	})
	req := httptest.NewRequest(http.MethodGet,
		"/api/cloud/compute/container-layer-graph/tenant_id/t1/workspace_id/ws1/task_id/task1/comment_id/cmt-missing/",
		nil)
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, req, "compute/container-layer-graph/tenant_id/t1/workspace_id/ws1/task_id/task1/comment_id/cmt-missing/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "L-live") {
		t.Fatalf("want live layers, body=%s", rec.Body.String())
	}
}

func TestGetContainerLayerGraphRequiresScope(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/compute/container-layer-graph/", nil)
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, req, "compute/container-layer-graph/")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s want 400", rec.Code, rec.Body.String())
	}
}

func TestGetContainerLayerGraphWrongWorkspaceEmpty(t *testing.T) {
	setupCloudTestDB(t)
	if err := upsertLayerGraphSnapshot(layerGraphSnapshotRow{
		CompanyID: "t1", WorkspaceID: "ws-a", TaskID: "task1", CommentID: "cmt-a",
		GraphJSON: `{"layers":[{"layer_id":"secret"}],"jobs":[]}`,
	}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet,
		"/api/cloud/compute/container-layer-graph/tenant_id/t1/workspace_id/ws-b/task_id/task1/comment_id/cmt-a/",
		nil)
	rec := httptest.NewRecorder()
	handleCloudTaskRoutes(rec, req, "compute/container-layer-graph/")
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("IDOR leak: %s", rec.Body.String())
	}
}
