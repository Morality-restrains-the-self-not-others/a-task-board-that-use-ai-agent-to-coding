package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskTaskService/src/domain"
)

func startMockProjectServiceWithAtMode(t *testing.T, atModeEnabled bool) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if pathHasWorkspace(r, "ws1") && !strings.Contains(r.URL.Path, "workspace-access") {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                              "ws1",
				"company_id":                      "t1",
				"container_image_at_mode_enabled": atModeEnabled,
			})
			return
		}
		if strings.Contains(r.URL.Path, "workspace-access") {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"user_id": "u1"},
			})
			return
		}
		if strings.Contains(r.URL.Path, "/projects/tenant_id/") {
			json.NewEncoder(w).Encode(map[string]interface{}{"git_repos": []string{"https://git.example/repo.git"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	cfg.ProjectServiceURL = srv.URL
	cfg.TaskBillURL = ""
	cfg.AICommentServiceURL = ""
	prevNotify := notifyContainerAgentPendingFn
	notifyContainerAgentPendingFn = func(tenantID, workspaceID, taskID, parentCommentID, imageID, imageName, content, createdBy string, source ...string) error {
		return nil
	}
	t.Cleanup(func() { notifyContainerAgentPendingFn = prevNotify })
}

func startMockCloudInstalledImageLookup(t *testing.T, images map[string]string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/tenant-installed-images/lookup" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		tenantID := r.URL.Query().Get("tenant_id")
		id := r.URL.Query().Get("id")
		if tenantID == "" || id == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		name, ok := images[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   id,
			"name": name,
		})
	}))
	t.Cleanup(srv.Close)
	cfg.CloudServiceURL = srv.URL
	cfg.InternalSecret = "test-secret"
	prev := lookupInstalledImageFn
	lookupInstalledImageFn = lookupInstalledImage
	t.Cleanup(func() { lookupInstalledImageFn = prev })
}

func stubInstalledImageLookup(t *testing.T, img *InstalledImageLookup, err error) {
	t.Helper()
	prev := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return img, err
	}
	t.Cleanup(func() { lookupInstalledImageFn = prev })
}

func createTestTaskForComments(t *testing.T) string {
	t.Helper()
	createBody := `{"title":"Comment task","workspace_id":"ws1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create task: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID, _ := created["id"].(string)
	if taskID == "" {
		t.Fatal("expected task id")
	}
	return taskID
}

func postComment(t *testing.T, taskID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/tasks/"+taskID+"/comments/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleCommentRoutes(rec, req)
	return rec
}

func commentMentionsJSON(t *testing.T, commentID string) string {
	t.Helper()
	var mentionsJSON string
	err := db.QueryRow(`SELECT mentions_json FROM task_comments WHERE id=?`, commentID).Scan(&mentionsJSON)
	if err != nil {
		t.Fatalf("query mentions_json: %v", err)
	}
	return mentionsJSON
}

func TestCreateCommentRejectsContentEqualToTaskID(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)
	taskID := createTestTaskForComments(t)

	rec := postComment(t, taskID, `{"content":"`+taskID+`"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if got := strings.TrimSpace(fmt.Sprint(resp["error"])); got != errMsgCommentContentIsTaskID {
		t.Fatalf("error=%q want %q", got, errMsgCommentContentIsTaskID)
	}
}

func TestCreateCommentWithoutMention(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)
	taskID := createTestTaskForComments(t)

	rec := postComment(t, taskID, `{"content":"plain comment"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if _, ok := resp["mentions"]; ok {
		t.Fatalf("expected no mentions in response, got %#v", resp["mentions"])
	}
	commentID, _ := resp["id"].(string)
	if got := commentMentionsJSON(t, commentID); got != "" {
		t.Fatalf("expected empty mentions_json, got %q", got)
	}
}

func TestCreateCommentRejectsMentionWhenAtModeDisabled(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, false)
	taskID := createTestTaskForComments(t)

	body := `{"content":"@镜像","mentions":[{"type":"installed_image","id":"img-1","name":"My Image"}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "container image at mode is disabled") {
		t.Fatalf("expected at mode disabled error, got: %s", rec.Body.String())
	}
}

func TestCreateCommentRejectsMultipleMentions(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	taskID := createTestTaskForComments(t)

	body := `{"content":"@a @b","mentions":[
		{"type":"installed_image","id":"img-1","name":"A"},
		{"type":"installed_image","id":"img-2","name":"B"}
	]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "at most one mention allowed") {
		t.Fatalf("expected too many mentions error, got: %s", rec.Body.String())
	}
}

func TestCreateCommentWithSingleMentionStoresJSON(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "My Image"})
	taskID := createTestTaskForComments(t)

	body := `{"content":"@My Image run tests","mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	commentID, _ := resp["id"].(string)
	if commentID == "" {
		t.Fatal("expected comment id")
	}
	mentions, ok := resp["mentions"].([]interface{})
	if !ok || len(mentions) != 1 {
		t.Fatalf("expected mentions in response, got %#v", resp["mentions"])
	}
	m0 := mentions[0].(map[string]interface{})
	if m0["id"] != "img-42" {
		t.Fatalf("mention id=%v", m0["id"])
	}

	got := commentMentionsJSON(t, commentID)
	want := `[{"type":"installed_image","id":"img-42","name":"My Image"}]`
	if got != want {
		t.Fatalf("mentions_json=%q want %q", got, want)
	}
}

func TestCreateCommentAtModeReaderInjectable(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t) // workspace response without at_mode field → false
	startMockCloudInstalledImageLookup(t, map[string]string{"img-stub": "Stub"})
	taskID := createTestTaskForComments(t)

	prev := readWorkspaceAtModeEnabled
	readWorkspaceAtModeEnabled = func(tenantID, workspaceID string) (bool, error) {
		return true, nil
	}
	t.Cleanup(func() { readWorkspaceAtModeEnabled = prev })

	prevNotify := notifyContainerAgentPendingFn
	notifyContainerAgentPendingFn = func(tenantID, workspaceID, taskID, parentCommentID, imageID, imageName, content, createdBy string, source ...string) error {
		return nil
	}
	t.Cleanup(func() { notifyContainerAgentPendingFn = prevNotify })

	body := `{"content":"@img","mentions":[{"type":"installed_image","id":"img-stub","name":"Stub"}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("stub reader: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateCommentRejectsMentionWhenInstalledImageNotFound(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	stubInstalledImageLookup(t, nil, ErrInstalledImageNotFound)
	taskID := createTestTaskForComments(t)

	body := `{"content":"@missing","mentions":[{"type":"installed_image","id":"img-missing","name":"Missing"}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	bodyStr := rec.Body.String()
	if !strings.Contains(bodyStr, "installed image not found") && !strings.Contains(bodyStr, "未找到已安装镜像") {
		t.Fatalf("expected not found error, got: %s", bodyStr)
	}
}

func TestCreateCommentEnrichesMentionNameFromCloud(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-99": "Cloud Name"})
	taskID := createTestTaskForComments(t)

	body := `{"content":"@x","mentions":[{"type":"installed_image","id":"img-99","name":""}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	mentions, _ := resp["mentions"].([]interface{})
	m0 := mentions[0].(map[string]interface{})
	if m0["name"] != "Cloud Name" {
		t.Fatalf("expected enriched name, got %#v", m0)
	}
}

func TestLookupInstalledImageNotFoundViaHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
	t.Cleanup(srv.Close)
	cfg.CloudServiceURL = srv.URL
	cfg.InternalSecret = "s"
	_, err := lookupInstalledImage("t1", "missing")
	if !errors.Is(err, ErrInstalledImageNotFound) {
		t.Fatalf("expected ErrInstalledImageNotFound, got %v", err)
	}
}

func TestLookupInstalledImageSuccessViaHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Secret") != "sec" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "img-1", "name": "N1"})
	}))
	t.Cleanup(srv.Close)
	cfg.CloudServiceURL = srv.URL
	cfg.InternalSecret = "sec"
	img, err := lookupInstalledImage("t1", "img-1")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if img.ID != "img-1" || img.Name != "N1" {
		t.Fatalf("got %+v", img)
	}
}

// TestLookupInstalledImageReboundByUniqueName（OPT-20260827-035）：
// 传入唯一镜像名时，id= 命中失败后 Cloud 端按 name= 回退现网 ID；
// 客户端必须把 name 放进 query，返回的 ID 应是现网 ID 而非陈旧 ID。
func TestLookupInstalledImageReboundByUniqueName(t *testing.T) {
	var gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotName = r.URL.Query().Get("name")
		// id=stale-id 命中失败，但 name=trae-agent 唯一 → 回退现网 ID。
		if r.URL.Query().Get("id") == "stale-id" && gotName == "trae-agent" {
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "img-current", "name": "trae-agent", "version": "v3"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
	}))
	t.Cleanup(srv.Close)
	cfg.CloudServiceURL = srv.URL
	cfg.InternalSecret = "s"

	// 有唯一名 + 陈旧 id → 回退现网 ID。
	img, err := lookupInstalledImage("t1", "stale-id", "trae-agent")
	if err != nil {
		t.Fatalf("lookup with name: %v", err)
	}
	if gotName != "trae-agent" {
		t.Fatalf("name query = %q, want trae-agent", gotName)
	}
	if img.ID != "img-current" {
		t.Fatalf("rebound id = %q, want img-current", img.ID)
	}

	// 无名（仅 id）→ Cloud 404 → ErrInstalledImageNotFound。
	if _, err := lookupInstalledImage("t1", "stale-id"); !errors.Is(err, ErrInstalledImageNotFound) {
		t.Fatalf("no-name stale id: expected ErrInstalledImageNotFound, got %v", err)
	}
}

func TestCreateCommentFillsDefaultSkill(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	stubInstalledImageLookup(t, &InstalledImageLookup{
		ID: "img-42", Name: "My Image",
		ImageSkills: domain.ImageSkillList{
			Version: 1, DefaultSkill: "general-coding",
			Skills: []domain.ImageSkill{{Name: "general-coding", IsDefault: true}, {Name: "k8s-debug"}},
		},
	}, nil)
	taskID := createTestTaskForComments(t)
	body := `{"content":"@My Image run tests","mentions":[{"type":"installed_image","id":"img-42","name":"My Image"}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	got := commentMentionsJSON(t, recToCommentID(t, rec))
	if !strings.Contains(got, `"skill":"general-coding"`) {
		t.Fatalf("mentions_json=%q want default skill", got)
	}
}

func TestCreateCommentUnknownSkillRejected(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	stubInstalledImageLookup(t, &InstalledImageLookup{
		ID: "img-42", Name: "My Image",
		ImageSkills: domain.ImageSkillList{
			Version: 1, DefaultSkill: "general-coding",
			Skills: []domain.ImageSkill{{Name: "general-coding", IsDefault: true}},
		},
	}, nil)
	taskID := createTestTaskForComments(t)
	body := `{"content":"@My Image /nope","mentions":[{"type":"installed_image","id":"img-42","name":"My Image","skill":"nope"}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "unknown image skill") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func recToCommentID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	id, _ := resp["id"].(string)
	if id == "" {
		t.Fatal("expected comment id")
	}
	return id
}

func TestCreateCommentWithMentionNotifiesAgentBeforeEvent(t *testing.T) {
	setupTestDB(t)
	startMockProjectServiceWithAtMode(t, true)
	startMockCloudInstalledImageLookup(t, map[string]string{"img-42": "trae-agent"})
	taskID := createTestTaskForComments(t)

	var seq []string
	prevNotify := notifyContainerAgentPendingFn
	notifyContainerAgentPendingFn = func(tenantID, workspaceID, gotTaskID, parentCommentID, imageID, imageName, content, createdBy string, source ...string) error {
		seq = append(seq, "notify:"+parentCommentID)
		return nil
	}
	t.Cleanup(func() { notifyContainerAgentPendingFn = prevNotify })

	prevPub := publishDomainEventFn
	publishDomainEventFn = func(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
		seq = append(seq, "publish:"+eventType)
		return nil
	}
	t.Cleanup(func() { publishDomainEventFn = prevPub })

	body := `{"content":"@trae-agent 删除 hello","mentions":[{"type":"installed_image","id":"img-42","name":"trae-agent"}]}`
	rec := postComment(t, taskID, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(seq) < 2 || !strings.HasPrefix(seq[0], "notify:") || seq[1] != "publish:TASK_COMMENT_IMAGE_MENTIONED" {
		t.Fatalf("seq=%v want notify then TASK_COMMENT_IMAGE_MENTIONED", seq)
	}
}
