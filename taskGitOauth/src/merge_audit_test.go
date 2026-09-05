package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// insertMergeAudit 按 writeMergeAudit 相同编码落一条合并审计（测试用）。
func insertMergeAudit(t *testing.T, app *App, htmlURL, taskID, userID, result string, at time.Time, withMergedAt bool) {
	t.Helper()
	detail := map[string]any{
		"html_url":   htmlURL,
		"comment_id": "cmt_1",
		"user_id":    userID,
		"result":     result,
		"provider":   "gitlab",
		"host":       "gitlab-tencent-sh-1.daydaymoney.com",
	}
	if withMergedAt {
		detail["merged_at"] = at.UTC().Format(time.RFC3339)
	}
	blob, _ := json.Marshal(detail)
	_, err := app.DB.Exec(`
INSERT INTO git_oauth_taskcredentialaudit
  (provider, task2app_task_id, task2app_workspace_id, task2app_company_id, task2app_user_id, action, detail, created_at)
VALUES (?, ?, '', ?, ?, 'merge_request_merge', ?, ?)`,
		"gitlab", taskID, int64(877397588196749312), userID, string(blob),
		at.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSelectLatestMergeAuditByURL(t *testing.T) {
	app := mergeTestApp(t)
	url := gitlabHTMLURL()
	now := time.Now().UTC()

	// failed 行不得被选中（审计回显只认成功结果）
	insertMergeAudit(t, app, url, "task_1", "u-bad", "failed", now.Add(-3*time.Hour), true)
	// 较早的 merged 行不得被选中（取最近一条）
	insertMergeAudit(t, app, url, "task_1", "u-first", "merged", now.Add(-2*time.Hour), true)
	// 最近一条 merged（含 detail.merged_at）
	wantAt := now.Add(-30 * time.Minute)
	insertMergeAudit(t, app, url, "task_1", "u-last", "merged", wantAt, true)

	row, err := app.DB.SelectLatestMergeAuditByURL(url)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("want audit row")
	}
	if row.UserID != "u-last" {
		t.Fatalf("user=%q want u-last", row.UserID)
	}
	if !row.CreatedAt.UTC().Truncate(time.Second).Equal(wantAt.UTC().Truncate(time.Second)) {
		t.Fatalf("created_at=%v want %v", row.CreatedAt, wantAt)
	}

	// 无匹配 URL → nil
	row, err = app.DB.SelectLatestMergeAuditByURL("https://github.com/other/repo/pull/1")
	if err != nil {
		t.Fatal(err)
	}
	if row != nil {
		t.Fatalf("want nil for unknown url, got %+v", row)
	}
}

func TestSelectLatestMergeAuditByURLFallsBackToCreatedAt(t *testing.T) {
	app := mergeTestApp(t)
	url := "https://gitlab-tencent-sh-1.daydaymoney.com/example-user/legacy/-/merge_requests/3"
	now := time.Now().UTC()
	// 历史行 detail 无 merged_at → 回退 created_at 列（UTC）
	insertMergeAudit(t, app, url, "task_9", "u-legacy", "noop_already_merged", now.Add(-1*time.Hour), false)

	row, err := app.DB.SelectLatestMergeAuditByURL(url)
	if err != nil {
		t.Fatal(err)
	}
	if row == nil {
		t.Fatal("want audit row")
	}
	if row.UserID != "u-legacy" {
		t.Fatalf("user=%q", row.UserID)
	}
	if !row.CreatedAt.UTC().Truncate(time.Second).Equal(now.Add(-1 * time.Hour).UTC().Truncate(time.Second)) {
		t.Fatalf("created_at=%v want created_at fallback", row.CreatedAt)
	}
}

func TestMergeRequestStatusMergedIncludesAuditBy(t *testing.T) {
	app := mergeTestApp(t)
	url := gitlabHTMLURL()
	app.ResolveDisplayNameFn = func(userID, tenantID string) string {
		if userID == "42" {
			return "张三"
		}
		return userID
	}
	insertMergeAudit(t, app, url, "task_1", "42", "merged", time.Now().UTC().Add(-5*time.Minute), true)
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, `{"state":"merged","title":"done"}`), nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-status/tenant_id/877397588196749312/",
		`{"html_urls":["`+url+`"]}`, "u1")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	results, _ := out["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results=%v", out)
	}
	row := results[0].(map[string]any)
	if row["state"] != "merged" {
		t.Fatalf("state=%v", row["state"])
	}
	if row["merged_by"] != "张三" {
		t.Fatalf("merged_by=%v want 张三", row["merged_by"])
	}
	if !strings.HasSuffix(row["merged_at"].(string), "Z") {
		t.Fatalf("merged_at=%v want RFC3339 UTC", row["merged_at"])
	}
}

func TestMergeRequestStatusMergedWithoutAuditOmitsAuditBy(t *testing.T) {
	app := mergeTestApp(t)
	url := gitlabHTMLURL()
	// 未走「一键合并」（站外合并）→ 无审计行 → 不回显 merged_by/merged_at
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, `{"state":"merged","title":"done"}`), nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-status/tenant_id/877397588196749312/",
		`{"html_urls":["`+url+`"]}`, "u1")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	row := out["results"].([]any)[0].(map[string]any)
	if _, has := row["merged_by"]; has {
		t.Fatalf("merged_by must be omitted without audit, got %v", row)
	}
	if _, has := row["merged_at"]; has {
		t.Fatalf("merged_at must be omitted without audit, got %v", row)
	}
}

func TestMergeRequestMergeResponseIncludesAuditBy(t *testing.T) {
	app := mergeTestApp(t)
	url := gitlabHTMLURL()
	app.ResolveDisplayNameFn = func(userID, tenantID string) string {
		if userID == "42" {
			return "李四"
		}
		return userID
	}
	app.GitAPIDoFn = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return jsonHTTPResponse(http.StatusOK, `{"state":"opened"}`), nil
		}
		if req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/merge") {
			return jsonHTTPResponse(http.StatusOK, `{"state":"merged"}`), nil
		}
		t.Fatalf("unexpected %s %s", req.Method, req.URL)
		return nil, nil
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := postJSON(mux, "/api/git-oauth/merge-request-merge/tenant_id/877397588196749312/",
		`{"html_url":"`+url+`","task_id":"task_1","comment_id":"cmt_1"}`, "42")
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["merged_by"] != "李四" {
		t.Fatalf("merged_by=%v want 李四", out["merged_by"])
	}
	if !strings.HasSuffix(out["merged_at"].(string), "Z") {
		t.Fatalf("merged_at=%v want RFC3339 UTC", out["merged_at"])
	}
	// 审计行 detail 应带 merged_at（点击时间自描述）
	var detail string
	if err := app.DB.QueryRow(`SELECT detail FROM git_oauth_taskcredentialaudit WHERE action='merge_request_merge'`).Scan(&detail); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, `"merged_at"`) {
		t.Fatalf("audit detail must carry merged_at, got %s", detail)
	}
}
