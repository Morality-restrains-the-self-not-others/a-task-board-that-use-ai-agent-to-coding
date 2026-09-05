package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestComposeAutoRunAtCommentContent(t *testing.T) {
	got := composeAutoRunAtCommentContent("Title", "Desc", "")
	if !strings.HasPrefix(got, autoRunAtCommentPrefix) {
		t.Fatalf("prefix missing: %q", got)
	}
	if !strings.Contains(got, "Title") || !strings.Contains(got, "Desc") {
		t.Fatalf("body missing title/desc: %q", got)
	}
	if got2 := composeAutoRunAtCommentContent("", "", ""); !strings.Contains(got2, "自动运行") {
		t.Fatalf("empty fallback: %q", got2)
	}
}

func TestComposeAutoRunAtCommentContentIncludesSkipReason(t *testing.T) {
	reason := "无法获取子 Git 仓库列表：探测失败，已跳过自动启动服务器"
	got := composeAutoRunAtCommentContent("写 hello", "用 js", reason)
	if !strings.HasPrefix(got, autoRunAtCommentPrefix) {
		t.Fatalf("prefix missing: %q", got)
	}
	if !strings.Contains(got, "写 hello") || !strings.Contains(got, "用 js") {
		t.Fatalf("body missing title/desc: %q", got)
	}
	if !strings.Contains(got, "未启动服务器：") {
		t.Fatalf("missing skip marker: %q", got)
	}
	if !strings.Contains(got, reason) {
		t.Fatalf("missing skip reason: %q", got)
	}
}

func TestEnsureAutoRunAtCommentCreatesAgentWithSource(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		 VALUES(?,?,?,?,0,'medium',1,?,?,?,?,?,?,?,1,'','','','',?,?)`,
		"task_ar_cmt", "t1", "Auto Title", "Auto Desc", "ws1", "u1", "", "", "", "", "img-1", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var agentPosts int
	var sawSource string

	prevLookup := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return &InstalledImageLookup{ID: imageID, Name: "snap-image", ExternalImageID: "ext"}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prevLookup })

	aic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "active-by-task") {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "container-agent-comments") {
			mu.Lock()
			agentPosts++
			mu.Unlock()
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			pack, _ := body["context_pack"].(map[string]interface{})
			at, _ := pack["at_mention_run"].(map[string]interface{})
			if at != nil {
				mu.Lock()
				sawSource = strings.TrimSpace(fmt.Sprintf("%v", at["source"]))
				mu.Unlock()
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "agent-1"})
			return
		}
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(aic.Close)
	prevURL := cfg.AICommentServiceURL
	cfg.AICommentServiceURL = aic.URL
	t.Cleanup(func() { cfg.AICommentServiceURL = prevURL })

	prevNotify := notifyContainerAgentPendingFn
	notifyContainerAgentPendingFn = notifyContainerAgentPending
	t.Cleanup(func() { notifyContainerAgentPendingFn = prevNotify })

	id, err := ensureAutoRunAtComment(autoRunTriggerParams{
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task_ar_cmt",
		UserID:      "u1",
		ImageID:     "img-1",
		RepoIdentities: []RepoIdentitySelection{{
			RepoURL:       "https://git.example/repo.git",
			GitIdentityID: "gid-auto",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("expected parent comment id")
	}
	var content, mentionsJSON, identitiesJSON string
	if err := db.QueryRow(`SELECT content, mentions_json, COALESCE(repo_identities_json,'') FROM task_comments WHERE id=?`, id).Scan(&content, &mentionsJSON, &identitiesJSON); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(identitiesJSON, "gid-auto") || !strings.Contains(identitiesJSON, "https://git.example/repo.git") {
		t.Fatalf("repo_identities_json=%q", identitiesJSON)
	}
	if !strings.HasPrefix(content, autoRunAtCommentPrefix) {
		t.Fatalf("content=%q", content)
	}
	if strings.Contains(content, "未启动服务器：") {
		t.Fatalf("unexpected skip marker without StartSkipReason: %q", content)
	}
	if !strings.Contains(mentionsJSON, "installed_image") || !strings.Contains(mentionsJSON, "img-1") {
		t.Fatalf("mentions_json=%q", mentionsJSON)
	}
	mu.Lock()
	posts := agentPosts
	src := sawSource
	mu.Unlock()
	if posts != 0 {
		t.Fatalf("agent posts=%d want 0 (container creates after bootstrap)", posts)
	}
	if src != "" {
		t.Fatalf("source=%q want empty when platform does not POST", src)
	}
}

func TestEnsureAutoRunAtCommentPutsAgentModelsInPack(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		 VALUES(?,?,?,?,0,'medium',1,?,?,?,?,?,?,?,1,'','','','',?,?)`,
		"task_ar_models", "t1", "Auto Title", "Auto Desc", "ws1", "u1", "", "", "", "", "img-1", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var sawModels []interface{}
	var sawPackModels []interface{}

	prevLookup := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return &InstalledImageLookup{ID: imageID, Name: "snap-image", ExternalImageID: "ext"}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prevLookup })

	aic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "active-by-task") {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "container-agent-comments") {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			pack, _ := body["context_pack"].(map[string]interface{})
			at, _ := pack["at_mention_run"].(map[string]interface{})
			mu.Lock()
			if at != nil {
				sawModels, _ = at["agent_models"].([]interface{})
			}
			sawPackModels, _ = pack["agent_models"].([]interface{})
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "agent-models"})
			return
		}
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(aic.Close)
	prevURL := cfg.AICommentServiceURL
	cfg.AICommentServiceURL = aic.URL
	t.Cleanup(func() { cfg.AICommentServiceURL = prevURL })

	id, err := ensureAutoRunAtComment(autoRunTriggerParams{
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task_ar_models",
		UserID:      "u1",
		ImageID:     "img-1",
		AgentModels: []map[string]interface{}{
			{"provider": "openai", "model": "gpt-4.1-mini"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("expected parent comment id")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sawModels) != 0 || len(sawPackModels) != 0 {
		t.Fatalf("platform must not POST agent models; at_mention_run.agent_models=%#v pack.agent_models=%#v", sawModels, sawPackModels)
	}
}

func TestEnsureAutoRunAtCommentIncludesSkipReasonInContent(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		 VALUES(?,?,?,?,0,'medium',1,?,?,?,?,?,?,?,1,'','','','',?,?)`,
		"task_ar_skip", "t1", "写 hello", "用 js", "ws1", "u1", "", "", "", "", "img-1", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}

	prevLookup := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return &InstalledImageLookup{ID: imageID, Name: "snap-image", ExternalImageID: "ext"}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prevLookup })

	aic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "active-by-task") {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "container-agent-comments") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "agent-skip"})
			return
		}
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(aic.Close)
	prevURL := cfg.AICommentServiceURL
	cfg.AICommentServiceURL = aic.URL
	t.Cleanup(func() { cfg.AICommentServiceURL = prevURL })

	prevNotify := notifyContainerAgentPendingFn
	notifyContainerAgentPendingFn = notifyContainerAgentPending
	t.Cleanup(func() { notifyContainerAgentPendingFn = prevNotify })

	skipReason := "无法获取子 Git 仓库列表：探测失败，已跳过自动启动服务器"
	id, err := ensureAutoRunAtComment(autoRunTriggerParams{
		TenantID:        "t1",
		WorkspaceID:     "ws1",
		TaskID:          "task_ar_skip",
		UserID:          "u1",
		ImageID:         "img-1",
		StartSkipReason: skipReason,
	})
	if err != nil {
		t.Fatal(err)
	}
	var content string
	if err := db.QueryRow(`SELECT content FROM task_comments WHERE id=?`, id).Scan(&content); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "未启动服务器：") || !strings.Contains(content, skipReason) {
		t.Fatalf("content missing skip reason: %q", content)
	}
}

func TestEnsureAutoRunAtCommentReusesActive(t *testing.T) {
	setupTestDB(t)
	aic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "active-by-task") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "agent-existing",
				"context_pack": map[string]interface{}{
					"at_mention_run": map[string]interface{}{
						"source":            autoRunAtCommentSource,
						"parent_comment_id": "cmt-existing",
						"agent_comment_id":  "agent-existing",
					},
				},
			})
			return
		}
		t.Fatalf("unexpected call %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(aic.Close)
	prevURL := cfg.AICommentServiceURL
	cfg.AICommentServiceURL = aic.URL
	t.Cleanup(func() { cfg.AICommentServiceURL = prevURL })

	id, err := ensureAutoRunAtComment(autoRunTriggerParams{
		TenantID: "t1", TaskID: "task_x", ImageID: "img-1", UserID: "u1", WorkspaceID: "ws1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "cmt-existing" {
		t.Fatalf("id=%q want cmt-existing", id)
	}
}

func TestEnsureAutoRunAtCommentReusesLocalParentWhenNoAgent(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		 VALUES(?,?,?,?,0,'medium',1,?,?,?,?,?,?,?,1,'','','','',?,?)`,
		"task_ar_local", "t1", "T", "D", "ws1", "u1", "", "", "", "", "img-1", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_comments(id,task_id,created_by_id,content,mentions_json,repo_identities_json,created_at) VALUES(?,?,?,?,?,?,?)`,
		"cmt-local-ar", "task_ar_local", "u1", autoRunAtCommentPrefix+"\nhello", "[]", "[]", now,
	); err != nil {
		t.Fatal(err)
	}
	aic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "active-by-task") {
			http.NotFound(w, r)
			return
		}
		t.Fatalf("unexpected call %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(aic.Close)
	prevURL := cfg.AICommentServiceURL
	cfg.AICommentServiceURL = aic.URL
	t.Cleanup(func() { cfg.AICommentServiceURL = prevURL })

	id, err := ensureAutoRunAtComment(autoRunTriggerParams{
		TenantID: "t1", TaskID: "task_ar_local", ImageID: "img-1", UserID: "u1", WorkspaceID: "ws1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "cmt-local-ar" {
		t.Fatalf("id=%q want cmt-local-ar", id)
	}
}

func TestEnsureAutoRunAtComment_SeedsFromProjectL2(t *testing.T) {
	setupTestDB(t)
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		 VALUES(?,?,?,?,0,'medium',1,?,?,?,?,?,?,?,1,'','','','',?,?)`,
		"task_ar_seed", "t1", "Auto Title", "Auto Desc", "ws1", "u1", "", "", "", "", "img-1", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(
		`INSERT INTO task_projects(id,task_id,project_id,base_branch,target_branch,repo_address) VALUES(?,?,?,?,?,?)`,
		"tp_seed", "task_ar_seed", "proj_granted", "main", "feat", "https://github.com/acme/demo.git",
	); err != nil {
		t.Fatal(err)
	}

	prevLookup := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return &InstalledImageLookup{ID: imageID, Name: "snap-image", ExternalImageID: "ext"}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prevLookup })

	prevGrant := lookupProjectGitOAuthGrantFn
	lookupProjectGitOAuthGrantFn = func(projectID, userID, gitsite string) (string, bool) {
		if projectID == "proj_granted" && userID == "u1" && gitsite == "github.com" {
			return "gh-seed-99", true
		}
		return "", false
	}
	t.Cleanup(func() { lookupProjectGitOAuthGrantFn = prevGrant })

	aic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "active-by-task") {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "container-agent-comments") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "agent-seed"})
			return
		}
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(aic.Close)
	prevURL := cfg.AICommentServiceURL
	cfg.AICommentServiceURL = aic.URL
	t.Cleanup(func() { cfg.AICommentServiceURL = prevURL })

	id, err := ensureAutoRunAtComment(autoRunTriggerParams{
		TenantID:    "t1",
		WorkspaceID: "ws1",
		TaskID:      "task_ar_seed",
		UserID:      "u1",
		ImageID:     "img-1",
		RepoIdentities: []RepoIdentitySelection{{
			RepoURL:       "https://github.com/acme/demo.git",
			GitIdentityID: "gid-auto",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var identitiesJSON string
	if err := db.QueryRow(`SELECT COALESCE(repo_identities_json,'') FROM task_comments WHERE id=?`, id).Scan(&identitiesJSON); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(identitiesJSON, `"oauth_gitsite":"github.com"`) {
		t.Fatalf("missing oauth_gitsite: %s", identitiesJSON)
	}
	if !strings.Contains(identitiesJSON, `"oauth_remote_user_id":"gh-seed-99"`) {
		t.Fatalf("missing remote: %s", identitiesJSON)
	}
}
