package main

import (
	"context"
	"strings"
	"testing"

	"taskCloudService/domain"
)

func TestPersistStepFullArchiveMergesJobs(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{Backend: "local", PathRule: domain.DefaultStepFullPathRule}

	id := domain.StepFullIDs{WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a", JobID: "J1", LayerID: "L1"}
	if _, err := persistStepFullArchive(context.Background(), id, "t1", "completed", []any{
		map[string]any{"step_number": 1, "delivery_summary": "think"},
	}); err != nil {
		t.Fatal(err)
	}
	id.JobID = "J2"
	if _, err := persistStepFullArchive(context.Background(), id, "t1", "completed", []any{
		map[string]any{"step_number": 1, "delivery_summary": "bash"},
	}); err != nil {
		t.Fatal(err)
	}
	row, found, err := latestStepFullObjectRowForComment("ws1", "task1", "cmt-a", "L1")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	raw, src, err := loadStepFullBundleBytes(context.Background(), row.ObjectKey, row.PayloadJSON)
	if err != nil || src != "saas_cos" {
		t.Fatalf("src=%q err=%v", src, err)
	}
	bundle, err := domain.ParseStepFullBundle(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Jobs) != 2 {
		t.Fatalf("jobs=%d", len(bundle.Jobs))
	}
	if got := domain.StepsForJob(bundle, "J1"); len(got) != 1 {
		t.Fatalf("J1 steps=%v", got)
	}
}

func TestPersistStepFullArchiveSeparatesLayers(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{Backend: "local", PathRule: domain.DefaultStepFullPathRule}

	id := domain.StepFullIDs{WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a", JobID: "J1", LayerID: "L1"}
	if _, err := persistStepFullArchive(context.Background(), id, "t1", "completed", []any{
		map[string]any{"step_number": 1, "delivery_summary": "layer1-job"},
	}); err != nil {
		t.Fatal(err)
	}
	// 另一层的 job 归档到独立对象 key，回退合并不得把 L1 的 jobs 带进 L2 的 bundle。
	id.LayerID = "L2"
	id.JobID = "J2"
	if _, err := persistStepFullArchive(context.Background(), id, "t1", "completed", []any{
		map[string]any{"step_number": 1, "delivery_summary": "layer2-job"},
	}); err != nil {
		t.Fatal(err)
	}
	row, found, err := getStepFullObjectRow("ws1", "task1", "cmt-a", "J2")
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if row.LayerID != "L2" {
		t.Fatalf("row layer=%q want L2", row.LayerID)
	}
	if !strings.Contains(row.ObjectKey, "/layer_L2/") {
		t.Fatalf("object_key=%q want layer_L2 segment", row.ObjectKey)
	}
	raw, _, err := loadStepFullBundleBytes(context.Background(), row.ObjectKey, row.PayloadJSON)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := domain.ParseStepFullBundle(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Jobs) != 1 {
		t.Fatalf("L2 bundle jobs=%d want 1 (L1 jobs must not leak into L2)", len(bundle.Jobs))
	}
	if got := domain.StepsForJob(bundle, "J2"); len(got) != 1 {
		t.Fatalf("J2 steps=%v", got)
	}
	if got := domain.StepsForJob(bundle, "J1"); got != nil {
		t.Fatalf("J1 must not leak into L2 bundle, got %v", got)
	}
}

func TestPersistStepFullArchiveUpsertSameJob(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	id := domain.StepFullIDs{WorkspaceID: "ws1", TaskID: "task1", CommentID: "cmt-a", JobID: "J1"}
	if _, err := persistStepFullArchive(context.Background(), id, "t1", "running", []any{}); err != nil {
		t.Fatal(err)
	}
	if _, err := persistStepFullArchive(context.Background(), id, "t1", "completed", []any{
		map[string]any{"step_number": 1},
	}); err != nil {
		t.Fatal(err)
	}
	n := 0
	if err := db.QueryRow(`SELECT COUNT(*) FROM cloud_job_step_full_object WHERE workspace_id=? AND task_id=? AND comment_id=?`,
		"ws1", "task1", "cmt-a").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("rows=%d want 1", n)
	}
}
