package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func resetCloneProgressForwarded() {
	cloneProgressForwardedMu.Lock()
	defer cloneProgressForwardedMu.Unlock()
	cloneProgressForwarded = map[string]int{}
}

func TestShouldDropCloneProgress100After9(t *testing.T) {
	resetCloneProgressForwarded()
	recordCloneProgressForwarded("cmt1", "https://gitlab.daydaymoney.com/a/b.git", 100)

	// 同仓 100 后再来 9：普通进度必须忽略
	if !shouldDropCloneProgress("cmt1", "https://gitlab.daydaymoney.com/a/b.git", 9, "Receiving objects: 9%") {
		t.Fatal("9 after 100 should be dropped")
	}
	// 更高进度不忽略
	if shouldDropCloneProgress("cmt1", "https://gitlab.daydaymoney.com/a/b.git", 100, "Receiving objects: 100%") {
		t.Fatal("100 after 9 should be forwarded")
	}
	// 失败语义不忽略（可覆盖高水位展示失败）
	if shouldDropCloneProgress("cmt1", "https://gitlab.daydaymoney.com/a/b.git", 9, "克隆失败: fatal: unable to access") {
		t.Fatal("failure message must bypass monotonic guard")
	}
	// 重试/重新克隆语义不忽略
	if shouldDropCloneProgress("cmt1", "https://gitlab.daydaymoney.com/a/b.git", 9, "准备第 1/3 次重试") {
		t.Fatal("retry message must bypass monotonic guard")
	}
	if shouldDropCloneProgress("cmt1", "https://gitlab.daydaymoney.com/a/b.git", 9, "【重新克隆】开始") {
		t.Fatal("reclone message must bypass monotonic guard")
	}
}

func TestShouldDropCloneProgressIsolatedByCommentAndRepo(t *testing.T) {
	resetCloneProgressForwarded()
	recordCloneProgressForwarded("cmt1", "https://gitlab.daydaymoney.com/a/b.git", 100)

	// 不同评论不受影响
	if shouldDropCloneProgress("cmt2", "https://gitlab.daydaymoney.com/a/b.git", 9, "Receiving objects: 9%") {
		t.Fatal("different comment must not be affected")
	}
	// 不同仓库不受影响
	if shouldDropCloneProgress("cmt1", "https://gitlab.daydaymoney.com/c/d.git", 9, "Receiving objects: 9%") {
		t.Fatal("different repo must not be affected")
	}
	// 全局（无 repo_url）与具体仓互不干扰
	if shouldDropCloneProgress("cmt1", "", 9, "Receiving objects: 9%") {
		t.Fatal("global key must not be affected by repo-specific watermark")
	}
}

func TestHandleGitCloneProgressDropsStaleLowerProgress(t *testing.T) {
	resetCloneProgressForwarded()
	var published []map[string]any
	prevPublish := publishCloneProgressSSE
	publishCloneProgressSSE = func(_ context.Context, _, _ string, statusData map[string]any) error {
		published = append(published, statusData)
		return nil
	}
	t.Cleanup(func() { publishCloneProgressSSE = prevPublish })

	cfgRow := &CloudServerConfig{TaskID: "task1", CommentID: "cmt1"}
	repo := "https://gitlab.daydaymoney.com/a/b.git"

	// 首轮 100% → 转发
	rec := httptest.NewRecorder()
	handleGitCloneProgress(rec, context.Background(), cfgRow, map[string]any{
		"progress": 100, "message": "Receiving objects: 100%", "repo_url": repo,
	}, "t1", "w1", "task1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(published) != 1 {
		t.Fatalf("first 100 should be published, got %d", len(published))
	}
	if strings.Contains(rec.Body.String(), `"dropped":true`) {
		t.Fatalf("first 100 must not be dropped, body=%s", rec.Body.String())
	}

	// 同仓迟到 9% → 忽略（SSE 不再新增）
	rec = httptest.NewRecorder()
	handleGitCloneProgress(rec, context.Background(), cfgRow, map[string]any{
		"progress": 9, "message": "Receiving objects: 9%", "repo_url": repo,
	}, "t1", "w1", "task1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(published) != 1 {
		t.Fatalf("stale 9 after 100 must not be published, got %d", len(published))
	}
	if !strings.Contains(rec.Body.String(), `"dropped":true`) {
		t.Fatalf("stale 9 should ack with dropped=true, body=%s", rec.Body.String())
	}
}

func TestHandleGitCloneProgressFailureBypassesGuard(t *testing.T) {
	resetCloneProgressForwarded()
	var published []map[string]any
	prevPublish := publishCloneProgressSSE
	publishCloneProgressSSE = func(_ context.Context, _, _ string, statusData map[string]any) error {
		published = append(published, statusData)
		return nil
	}
	t.Cleanup(func() { publishCloneProgressSSE = prevPublish })

	cfgRow := &CloudServerConfig{TaskID: "task1", CommentID: "cmt1"}
	repo := "https://gitlab.daydaymoney.com/a/b.git"

	handleGitCloneProgress(httptest.NewRecorder(), context.Background(), cfgRow, map[string]any{
		"progress": 100, "message": "Receiving objects: 100%", "repo_url": repo,
	}, "t1", "w1", "task1")

	// 失败语义即使进度更低也转发
	rec := httptest.NewRecorder()
	handleGitCloneProgress(rec, context.Background(), cfgRow, map[string]any{
		"progress": 9, "message": "克隆失败: fatal: unable to access", "repo_url": repo,
	}, "t1", "w1", "task1")
	if len(published) != 2 {
		t.Fatalf("failure message must be published despite lower progress, got %d", len(published))
	}
	if strings.Contains(rec.Body.String(), `"dropped":true`) {
		t.Fatalf("failure message must not be dropped, body=%s", rec.Body.String())
	}
}
