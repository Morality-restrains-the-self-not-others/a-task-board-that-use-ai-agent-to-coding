package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"taskCloudService/domain"
)

// seedLegacyCCBLogRow 直接写分片表，绕过双写（模拟双写上线前的存量行）。
func seedLegacyCCBLogRow(t *testing.T, ws, company, task, comment, binding, stage, message string) {
	t.Helper()
	table, err := ccbLogTable(ws)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if _, err := db.Exec(
		`INSERT INTO `+table+`(id, workspace_id, company_id, task_id, comment_id, binding_id, stage, message, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		"legacy-"+stage+"-"+binding, ws, company, task, comment, binding, stage, message, now,
	); err != nil {
		t.Fatal(err)
	}
}

func TestBackfillCCBStartupLogsArchivesMissingComments(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{
		Backend: "local", PathRule: domain.DefaultStepFullPathRule,
		StartupLogsPathRule: domain.DefaultStartupLogPathRule,
	}
	seedLegacyCCBLogRow(t, "wsBF", "t1", "taskBF", "cBF", "bBF", "starting", "正在启动容器实例")
	seedLegacyCCBLogRow(t, "wsBF", "t1", "taskBF", "cBF", "bBF", "ready", "实例已就绪")

	res, err := backfillCCBStartupLogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Archived != 1 {
		t.Fatalf("archived=%d want 1 (res=%+v)", res.Archived, res)
	}
	if res.Errors != 0 {
		t.Fatalf("errors=%d want 0: %v", res.Errors, res.ErrorMessages)
	}
	row, found, err := getStartupLogObjectRow("wsBF", "taskBF", "cBF")
	if err != nil || !found {
		t.Fatalf("pointer found=%v err=%v", found, err)
	}
	raw, found, err := stepFullObjects.Get(context.Background(), row.ObjectKey)
	if err != nil || !found {
		t.Fatalf("cos object found=%v err=%v", found, err)
	}
	bundle, err := domain.ParseStartupLogBundle(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Logs) != 2 {
		t.Fatalf("bundle logs=%d want 2", len(bundle.Logs))
	}
	if bundle.CommentID != "cBF" || bundle.WorkspaceID != "wsBF" || bundle.TaskID != "taskBF" {
		t.Fatalf("bundle identity=%s/%s/%s", bundle.WorkspaceID, bundle.TaskID, bundle.CommentID)
	}
	n, err := countCCBLogShardRows("wsBF", "t1", "taskBF")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("backfill must evict shard after Put, got %d rows", n)
	}
}

func TestBackfillCCBStartupLogsSkipsExistingPointer(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{
		Backend: "local", PathRule: domain.DefaultStepFullPathRule,
		StartupLogsPathRule: domain.DefaultStartupLogPathRule,
	}
	// 已归档评论（指针存在）不应再次被回填。
	if _, err := insertCommentContainerBinding("t1", "taskBF2", "cBF2", ccbExecutionIndependent, "", "wsBF2"); err != nil {
		t.Fatal(err)
	}
	if _, found, err := getStartupLogObjectRow("wsBF2", "taskBF2", "cBF2"); err != nil || !found {
		t.Fatalf("expected pointer after hot-path insert found=%v err=%v", found, err)
	}
	// 另有一条真正缺指针的存量行，确保扫描仍能看到（LEFT JOIN 只跳过指针命中者）。
	seedLegacyCCBLogRow(t, "wsBF2", "t1", "taskBF2", "cBF-missing", "bBF2", "starting", "正在启动容器实例")

	res, err := backfillCCBStartupLogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.CommentsScanned != 1 {
		t.Fatalf("comments_scanned=%d want 1 (only missing pointer)", res.CommentsScanned)
	}
	if res.Archived != 1 {
		t.Fatalf("archived=%d want 1", res.Archived)
	}
	if _, found, err := getStartupLogObjectRow("wsBF2", "taskBF2", "cBF-missing"); err != nil || !found {
		t.Fatalf("missing comment pointer not created found=%v err=%v", found, err)
	}
}

func TestBackfillCCBStartupLogsIdempotent(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{
		Backend: "local", PathRule: domain.DefaultStepFullPathRule,
		StartupLogsPathRule: domain.DefaultStartupLogPathRule,
	}
	seedLegacyCCBLogRow(t, "wsBF3", "t1", "taskBF3", "cBF3", "bBF3", "starting", "正在启动容器实例")

	first, err := backfillCCBStartupLogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Archived != 1 {
		t.Fatalf("first archived=%d want 1", first.Archived)
	}
	second, err := backfillCCBStartupLogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if second.CommentsScanned != 0 || second.Archived != 0 {
		t.Fatalf("second run must be a no-op, res=%+v", second)
	}
}

func TestBackfillCCBStartupLogsFallsBackToLocalPayload(t *testing.T) {
	setupCloudTestDB(t)
	stepFullObjects = failingStepFullStore{}
	t.Cleanup(resetStepFullObjectsForTest)
	stepFullCOSCfg = StepFullCOSConfig{Backend: "cos", StartupLogsPathRule: domain.DefaultStartupLogPathRule}
	seedLegacyCCBLogRow(t, "wsBF4", "t1", "taskBF4", "cBF4", "bBF4", "starting", "正在启动容器实例")

	res, err := backfillCCBStartupLogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Archived != 1 {
		t.Fatalf("archived=%d want 1 (COS 失败也应落本地指针不丢数据)", res.Archived)
	}
	row, found, err := getStartupLogObjectRow("wsBF4", "taskBF4", "cBF4")
	if err != nil || !found {
		t.Fatalf("pointer found=%v err=%v", found, err)
	}
	if row.Source != "local" {
		t.Fatalf("source=%q want local", row.Source)
	}
	if !strings.Contains(row.PayloadJSON, "正在启动容器实例") {
		t.Fatalf("payload missing message: %q", row.PayloadJSON)
	}
	n, err := countCCBLogShardRows("wsBF4", "t1", "taskBF4")
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("must keep MySQL shard when COS put fails during backfill")
	}
}

func TestBackfillCCBStartupLogsEvictsLegacyDualWriteShards(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{
		Backend: "local", PathRule: domain.DefaultStepFullPathRule,
		StartupLogsPathRule: domain.DefaultStartupLogPathRule,
	}
	seedLegacyCCBLogRow(t, "wsEv", "t1", "taskEv", "cEv", "bEv", "starting", "正在启动容器实例")
	logs, err := listCCBLogsForComment("wsEv", "t1", "taskEv", "cEv")
	if err != nil || len(logs) == 0 {
		t.Fatalf("seed logs=%d err=%v", len(logs), err)
	}
	id := domain.StartupLogIDs{WorkspaceID: "wsEv", TaskID: "taskEv", CommentID: "cEv"}
	bundle := domain.EmptyStartupLogBundle(id)
	for _, l := range logs {
		bundle = domain.MergeStartupLogEntry(bundle, id, domain.StartupLogEntry{
			ID: l.ID, BindingID: l.BindingID, Stage: l.Stage, Message: l.Message,
			CreatedAt: formatCloudUTCJSON(l.CreatedAt), CompanyID: l.CompanyID,
			WorkspaceID: l.WorkspaceID, TaskID: l.TaskID, CommentID: l.CommentID,
		})
	}
	raw, err := domain.MarshalStartupLogBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	key, err := domain.RenderStartupLogObjectKey(domain.DefaultStartupLogPathRule, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stepFullObjects.Put(context.Background(), key, raw); err != nil {
		t.Fatal(err)
	}
	if err := upsertStartupLogObjectRow(startupLogObjectRow{
		CompanyID: "t1", WorkspaceID: "wsEv", TaskID: "taskEv", CommentID: "cEv",
		ObjectKey: key, Bytes: len(raw), Source: "local", PayloadJSON: string(raw),
	}); err != nil {
		t.Fatal(err)
	}
	before, err := countCCBLogShardRows("wsEv", "t1", "taskEv")
	if err != nil || before == 0 {
		t.Fatalf("legacy leftover shards=%d err=%v", before, err)
	}
	res, err := backfillCCBStartupLogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Archived != 0 {
		t.Fatalf("archived=%d want 0 (pointer already exists)", res.Archived)
	}
	if res.Evicted != 1 {
		t.Fatalf("evicted=%d want 1 (res=%+v)", res.Evicted, res)
	}
	after, err := countCCBLogShardRows("wsEv", "t1", "taskEv")
	if err != nil {
		t.Fatal(err)
	}
	if after != 0 {
		t.Fatalf("shard rows=%d want 0 after evict", after)
	}
	listed, err := listCommentContainerBindingLogsIn("wsEv", "t1", "taskEv")
	if err != nil || len(listed) == 0 {
		t.Fatalf("list after evict=%d err=%v", len(listed), err)
	}
}
