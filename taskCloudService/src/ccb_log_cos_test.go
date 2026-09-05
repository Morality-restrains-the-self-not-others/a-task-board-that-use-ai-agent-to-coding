package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskCloudService/domain"
)

type failingStepFullStore struct{}

func (failingStepFullStore) Put(context.Context, string, []byte) (string, error) {
	return "", fmt.Errorf("cos down")
}

func (failingStepFullStore) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}

func TestCCBLogArchivesToObjectStore(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{
		Backend: "local", PathRule: domain.DefaultStepFullPathRule,
		StartupLogsPathRule: domain.DefaultStartupLogPathRule,
	}
	if _, err := insertCommentContainerBinding("t1", "taskCosA", "cA", ccbExecutionIndependent, "", "wsCosA"); err != nil {
		t.Fatal(err)
	}
	key, err := domain.RenderStartupLogObjectKey(domain.DefaultStartupLogPathRule, domain.StartupLogIDs{
		WorkspaceID: "wsCosA", TaskID: "taskCosA", CommentID: "cA",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, found, err := stepFullObjects.Get(context.Background(), key)
	if err != nil || !found || len(raw) == 0 {
		t.Fatalf("found=%v err=%v len=%d", found, err, len(raw))
	}
	bundle, err := domain.ParseStartupLogBundle(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Logs) == 0 {
		t.Fatal("expected archived startup logs")
	}
	n, err := countCCBLogShardRows("wsCosA", "t1", "taskCosA")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("shard rows=%d want 0 after COS put (logs must live in object store)", n)
	}
	listed, err := listCommentContainerBindingLogsIn("wsCosA", "t1", "taskCosA")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) == 0 {
		t.Fatal("list must read archived logs from object store after shard eviction")
	}
}

func TestCCBLogArchiveIdempotentByLogID(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{Backend: "local", StartupLogsPathRule: domain.DefaultStartupLogPathRule}
	entry := CommentContainerBindingLog{
		ID: "log-dup-1", WorkspaceID: "wsDup", CompanyID: "t1", TaskID: "taskDup",
		CommentID: "cDup", BindingID: "b1", Stage: "starting", Message: "正在启动容器实例",
		CreatedAt: time.Date(2026, 8, 27, 2, 0, 0, 0, time.UTC),
	}
	if err := persistCommentStartupLog(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	if err := persistCommentStartupLog(context.Background(), entry); err != nil {
		t.Fatal(err)
	}
	key, err := domain.RenderStartupLogObjectKey(domain.DefaultStartupLogPathRule, domain.StartupLogIDs{
		WorkspaceID: "wsDup", TaskID: "taskDup", CommentID: "cDup",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, found, err := stepFullObjects.Get(context.Background(), key)
	if err != nil || !found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	bundle, err := domain.ParseStartupLogBundle(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Logs) != 1 {
		t.Fatalf("logs=%d want 1", len(bundle.Logs))
	}
}

func TestCCBLogInsertSucceedsWhenCOSPutFails(t *testing.T) {
	setupCloudTestDB(t)
	stepFullObjects = failingStepFullStore{}
	t.Cleanup(resetStepFullObjectsForTest)
	stepFullCOSCfg = StepFullCOSConfig{Backend: "cos", StartupLogsPathRule: domain.DefaultStartupLogPathRule}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	if err := insertCCBLogRow("wsFail", "t1", "taskFail", "cFail", "bFail", "starting", "正在启动容器实例", now); err != nil {
		t.Fatalf("insert must succeed when COS put fails: %v", err)
	}
	rows, err := listCommentContainerBindingLogsIn("wsFail", "t1", "taskFail")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected leftover shard row when COS put fails")
	}
	n, err := countCCBLogShardRows("wsFail", "t1", "taskFail")
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("must keep MySQL shard row when COS put fails")
	}
}

func TestCCBLogListHydratesFromObjectStoreAfterShardDelete(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{
		Backend: "local", PathRule: domain.DefaultStepFullPathRule,
		StartupLogsPathRule: domain.DefaultStartupLogPathRule,
	}
	if _, err := insertCommentContainerBinding("t1", "taskHyd", "cHyd", ccbExecutionIndependent, "", "wsHyd"); err != nil {
		t.Fatal(err)
	}
	before, err := listCommentContainerBindingLogsIn("wsHyd", "t1", "taskHyd")
	if err != nil || len(before) == 0 {
		t.Fatalf("before list=%d err=%v", len(before), err)
	}
	n, err := countCCBLogShardRows("wsHyd", "t1", "taskHyd")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("hot path must evict shard after Put, got %d", n)
	}
	after, err := listCommentContainerBindingLogsIn("wsHyd", "t1", "taskHyd")
	if err != nil {
		t.Fatal(err)
	}
	if len(after) == 0 {
		t.Fatal("expected hydrate from COS/pointer after shard delete")
	}
}

func TestCommentContainerBindingsListAPIHydratesFromObjectStore(t *testing.T) {
	setupCloudTestDB(t)
	resetStepFullObjectsForTest()
	stepFullCOSCfg = StepFullCOSConfig{
		Backend:             "local",
		PathRule:            domain.DefaultStepFullPathRule,
		StartupLogsPathRule: domain.DefaultStartupLogPathRule,
	}
	if _, err := insertCommentContainerBinding("t1", "taskAPIHyd", "cAPIHyd", ccbExecutionIndependent, "", "wsAPIHyd"); err != nil {
		t.Fatal(err)
	}
	listLogs := func() []interface{} {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/task/taskAPIHyd/comment-container-bindings/", nil)
		rec := httptest.NewRecorder()
		handleCommentContainerBindingsRoutes(rec, req, "t1", "taskAPIHyd", "wsAPIHyd", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("list status=%d body=%s", rec.Code, rec.Body.String())
		}
		var body map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		bindings, _ := body["bindings"].([]interface{})
		if len(bindings) != 1 {
			t.Fatalf("bindings=%v", body["bindings"])
		}
		item := bindings[0].(map[string]interface{})
		logs, _ := item["logs"].([]interface{})
		return logs
	}
	if n := len(listLogs()); n == 0 {
		t.Fatal("expected logs from object store")
	}
	n, err := countCCBLogShardRows("wsAPIHyd", "t1", "taskAPIHyd")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("list API must not keep MySQL shard after Put, got %d", n)
	}
	after := listLogs()
	if len(after) == 0 {
		t.Fatal("list API must still return COS-hydrated logs for the work-panel 启动日志")
	}
	first := after[0].(map[string]interface{})
	if strings.TrimSpace(fmt.Sprint(first["message"])) == "" {
		t.Fatalf("hydrated log missing message: %v", first)
	}
}
