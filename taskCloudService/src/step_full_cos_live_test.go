package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"taskCloudService/domain"
)

func TestLiveStepFullCOSPutGetDelete(t *testing.T) {
	if os.Getenv("STEP_FULL_COS_LIVE") != "1" {
		t.Skip("set STEP_FULL_COS_LIVE=1 to Put/Get/Delete against configured Tencent COS")
	}
	root, err := findMonorepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	loadStepFullCOSConf(root)
	if !strings.EqualFold(strings.TrimSpace(stepFullCOSCfg.Backend), "cos") {
		t.Fatalf("backend=%q want cos", stepFullCOSCfg.Backend)
	}
	store, err := newRealStepFullCOS(
		stepFullCOSCfg.Bucket,
		stepFullCOSCfg.Region,
		stepFullCOSCfg.SecretID,
		stepFullCOSCfg.SecretKey,
	)
	if err != nil {
		t.Fatal(err)
	}
	id := domain.StepFullIDs{
		WorkspaceID: "probe-ws",
		TaskID:      "probe-task",
		CommentID:   "probe-cmt",
		JobID:       "probe-job",
		LayerID:     "probe-layer",
	}
	key, err := domain.RenderStepFullObjectKey(stepFullCOSCfg.PathRule, id)
	if err != nil {
		t.Fatal(err)
	}
	wantKey := "workspace_probe-ws/task_probe-task/comment_probe-cmt/layer_probe-layer/step_full.json"
	if key != wantKey {
		t.Fatalf("object key=%q want %q", key, wantKey)
	}
	bundle := domain.MergeStepFullJob(domain.EmptyStepFullBundle(id), id, "completed", []any{
		map[string]any{"step_number": 1, "delivery_summary": "cos live probe", "llm": "kept"},
	})
	raw, err := domain.MarshalStepFullBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if os.Getenv("STEP_FULL_COS_KEEP") != "1" {
		t.Cleanup(func() {
			_ = store.Delete(context.Background(), key)
		})
	}
	if _, err := store.Put(ctx, key, raw); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, found, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("Get: object not found after Put")
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("Get body mismatch len=%d want=%d", len(got), len(raw))
	}
}
