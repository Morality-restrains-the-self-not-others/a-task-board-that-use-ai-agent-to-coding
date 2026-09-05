package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260902-020: workspace project list / detail hydrate git_repos_status with
// real token_available for repos whose site the requesting user has project-scoped
// Git OAuth grant for — local DB only, no git-oauth / remote probe (B-086 holds).
func TestHydrateProjectGitReposStatusLocalFromGrant(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"hydrate-local","git_repos":["https://github.com/o/private.git","https://gitlab.daydaymoney.com/o/other.git"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "1001")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatalf("missing project id: %#v", created)
	}

	getDetail := func() map[string]interface{} {
		reqGet := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t1/", nil)
		reqGet.Header.Set("X-Auth-Tenant-Id", "t1")
		reqGet.Header.Set("X-Auth-User-Id", "1001")
		reqGet.Header.Set("X-Resource-Id", pid)
		recGet := httptest.NewRecorder()
		handleGetProject(recGet, reqGet)
		if recGet.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d: %s", recGet.Code, recGet.Body.String())
		}
		var detail map[string]interface{}
		_ = json.NewDecoder(recGet.Body).Decode(&detail)
		return detail
	}

	// 未授权：保持 not_applicable 占位（无 git-oauth/远端调用）
	rawStatus := getDetail()["git_repos_status"].([]interface{})
	if len(rawStatus) != 2 {
		t.Fatalf("expected 2 git_repos_status rows, got %#v", getDetail()["git_repos_status"])
	}
	first := rawStatus[0].(map[string]interface{})
	if first["token_status"] != tokenStatusNotApplicable {
		t.Fatalf("without grant token_status=%#v want not_applicable", first["token_status"])
	}

	// 给 github.com 站点加 project grant
	if err := upsertProjectGitOAuthGrant("t1", pid, "1001", "github.com", "remote-1"); err != nil {
		t.Fatalf("upsert grant: %v", err)
	}

	detail := getDetail()
	rows := detail["git_repos_status"].([]interface{})
	statusByRepo := map[string]map[string]interface{}{}
	for _, item := range rows {
		m := item.(map[string]interface{})
		statusByRepo[strings.TrimSpace(m["repo_url"].(string))] = m
	}

	githubRow := statusByRepo["https://github.com/o/private.git"]
	if githubRow == nil {
		t.Fatalf("missing github status row: %#v", statusByRepo)
	}
	if githubRow["token_status"] != tokenStatusAvailable {
		t.Fatalf("granted repo token_status=%#v want token_available", githubRow["token_status"])
	}
	if githubRow["oauth_provider"] != "github" {
		t.Fatalf("granted repo oauth_provider=%#v want github", githubRow["oauth_provider"])
	}

	// 其他站点（gitlab.daydaymoney.com）无 grant 保持 not_applicable
	otherRow := statusByRepo["https://gitlab.daydaymoney.com/o/other.git"]
	if otherRow == nil {
		t.Fatalf("missing other status row: %#v", statusByRepo)
	}
	if otherRow["token_status"] != tokenStatusNotApplicable {
		t.Fatalf("ungranted repo token_status=%#v want not_applicable", otherRow["token_status"])
	}

	// 列表接口同样 hydrate
	reqList := httptest.NewRequest(http.MethodGet, "/api/projects/tenant_id/t1/", nil)
	reqList.Header.Set("X-Auth-Tenant-Id", "t1")
	reqList.Header.Set("X-Auth-User-Id", "1001")
	recList := httptest.NewRecorder()
	handleListProjects(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", recList.Code, recList.Body.String())
	}
	var projects []map[string]interface{}
	_ = json.NewDecoder(recList.Body).Decode(&projects)
	if len(projects) == 0 {
		t.Fatalf("expected projects in list")
	}
	listedRows := projects[0]["git_repos_status"].([]interface{})
	listedByRepo := map[string]map[string]interface{}{}
	for _, item := range listedRows {
		m := item.(map[string]interface{})
		listedByRepo[strings.TrimSpace(m["repo_url"].(string))] = m
	}
	if listedByRepo["https://github.com/o/private.git"]["token_status"] != tokenStatusAvailable {
		t.Fatalf("list granted repo token_status=%#v want token_available", listedByRepo["https://github.com/o/private.git"]["token_status"])
	}
	if listedByRepo["https://gitlab.daydaymoney.com/o/other.git"]["token_status"] != tokenStatusNotApplicable {
		t.Fatalf("list ungranted repo token_status=%#v want not_applicable", listedByRepo["https://gitlab.daydaymoney.com/o/other.git"]["token_status"])
	}
}
