package taskcommentimagementioned

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskEvents/domain"
)

type mockTaskClient struct {
	linked []map[string]interface{}
	err    error
}

func (m *mockTaskClient) FetchTaskLinkedProjects(ctx context.Context, taskID, tenantID, userID string) ([]map[string]interface{}, error) {
	_ = ctx
	_ = taskID
	_ = tenantID
	_ = userID
	return m.linked, m.err
}

type mockProjectClient struct {
	byID map[string]map[string]interface{}
}

func (m *mockProjectClient) FetchProject(ctx context.Context, tenantID, projectID, userID string) (map[string]interface{}, error) {
	_ = ctx
	_ = tenantID
	_ = userID
	p, ok := m.byID[projectID]
	if !ok {
		return nil, errNotFound
	}
	return p, nil
}

var errNotFound = &simpleErr{"not found"}

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }

type captureCloud struct {
	apiPath string
	body    map[string]interface{}
	userID  string
	calls   int
}

func (c *captureCloud) StartVM(ctx context.Context, tenantID, workspaceID, apiPath, userID string, body map[string]interface{}) error {
	_ = ctx
	_ = tenantID
	_ = workspaceID
	c.calls++
	c.apiPath = apiPath
	c.body = body
	c.userID = userID
	return nil
}

type captureAI struct {
	parent string
	status string
	calls  int
}

func (c *captureAI) SetRunStatusByParent(ctx context.Context, parentCommentID, runStatus string) error {
	_ = ctx
	c.calls++
	c.parent = parentCommentID
	c.status = runStatus
	return nil
}

// captureAINotFound 模拟 by-parent 回写返回 ErrAICommentNotFound（评论不存在/已清理）。
type captureAINotFound struct {
	calls int
}

func (c *captureAINotFound) SetRunStatusByParent(ctx context.Context, parentCommentID, runStatus string) error {
	_ = ctx
	_ = parentCommentID
	_ = runStatus
	c.calls++
	return ErrAICommentNotFound
}

type capturePub struct {
	n int
}

func (p *capturePub) PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	_ = ctx
	_ = eventType
	_ = data
	_ = key
	p.n++
	return nil
}

func TestDispatchStartVmAutoWithMissingNetwork(t *testing.T) {
	h := &Handler{
		Publisher: &capturePub{},
		Tasks: &mockTaskClient{linked: []map[string]interface{}{
			{"project_id": "p1"},
		}},
		Projects: &mockProjectClient{byID: map[string]map[string]interface{}{
			"p1": {
				"id": "p1",
				"server_run_template": map[string]interface{}{
					"region":            "cn-hangzhou",
					"cloud_platform_id": "plat-1",
					"authorization_id":  "auth-1",
					"zone_id":           "cn-hangzhou-b",
					"hardware_config": map[string]interface{}{
						"cpu_cores": "2", "memory_gb": "4", "storage_gb": "40",
					},
				},
			},
		}},
		Cloud:      &captureCloud{},
		AIComments: &captureAI{},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "tenant_id": "ten1", "workspace_id": "ws1",
		"installed_image_id": "img-9", "created_by_id": "u1",
		"comment_id": "cmt-1", "parent_comment_id": "cmt-1",
		"repo_identities": []map[string]interface{}{
			{"repo_url": "https://github.com/acme/demo.git", "git_identity_id": "gid-c", "github_user_id": "9"},
		},
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_COMMENT_IMAGE_MENTIONED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_COMMENT_IMAGE_MENTIONED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	cloud := h.Cloud.(*captureCloud)
	if cloud.apiPath != "start-vm-auto" {
		t.Fatalf("apiPath=%s want start-vm-auto", cloud.apiPath)
	}
	if cloud.body["container_image_id"] != "img-9" {
		t.Fatalf("body=%v", cloud.body)
	}
	if cloud.userID != "u1" {
		t.Fatalf("userID=%s", cloud.userID)
	}
	idents, _ := cloud.body["repo_identities"].([]interface{})
	if len(idents) != 1 {
		t.Fatalf("repo_identities not forwarded: %v", cloud.body["repo_identities"])
	}
	ai := h.AIComments.(*captureAI)
	if ai.calls != 1 || ai.parent != "cmt-1" || ai.status != "starting" {
		t.Fatalf("ai=%+v", ai)
	}
}

func TestDispatchStartVmWhenNetworkComplete(t *testing.T) {
	h := &Handler{
		Publisher: &capturePub{},
		Tasks: &mockTaskClient{linked: []map[string]interface{}{
			{"project_id": "p1"},
		}},
		Projects: &mockProjectClient{byID: map[string]map[string]interface{}{
			"p1": {
				"id": "p1",
				"server_run_template": map[string]interface{}{
					"region":            "cn-beijing",
					"cloud_platform_id": "plat-2",
					"authorization_id":  "auth-2",
					"vpc_id":            "vpc-1",
					"vswitch_id":        "vsw-1",
					"security_group_id": "sg-1",
					"zone_id":           "cn-beijing-a",
					"selected_instance": "ecs.g6.large",
					"hardware_config": map[string]interface{}{
						"cpu_cores": "4", "memory_gb": "8", "storage_gb": "50", "instance_type": "ecs.g6.large",
					},
				},
			},
		}},
		Cloud:      &captureCloud{},
		AIComments: &captureAI{},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t2", "company_id": "ten2", "workspace_id": "ws2",
		"installed_image_id": "img-2", "created_by_id": "u2",
		"parent_comment_id": "cmt-2",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_COMMENT_IMAGE_MENTIONED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_COMMENT_IMAGE_MENTIONED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	cloud := h.Cloud.(*captureCloud)
	if cloud.apiPath != "start-vm" {
		t.Fatalf("apiPath=%s want start-vm", cloud.apiPath)
	}
}

// by-parent 回写 404（AI 评论不存在/已清理）时降级为 warn，dispatch 仍成功，
// 不因状态回写失败而误判启服失败（OPT-20260811-056）。
func TestDispatchStillSucceedsWhenAICmtStatusByParentNotFound(t *testing.T) {
	h := &Handler{
		Publisher: &capturePub{},
		Tasks: &mockTaskClient{linked: []map[string]interface{}{
			{"project_id": "p1"},
		}},
		Projects: &mockProjectClient{byID: map[string]map[string]interface{}{
			"p1": {
				"id": "p1",
				"server_run_template": map[string]interface{}{
					"region":            "cn-beijing",
					"cloud_platform_id": "plat-2",
					"authorization_id":  "auth-2",
					"vpc_id":            "vpc-1",
					"vswitch_id":        "vsw-1",
					"security_group_id": "sg-1",
					"zone_id":           "cn-beijing-a",
					"selected_instance": "ecs.g6.large",
					"hardware_config": map[string]interface{}{
						"cpu_cores": "4", "memory_gb": "8", "storage_gb": "50", "instance_type": "ecs.g6.large",
					},
				},
			},
		}},
		Cloud:      &captureCloud{},
		AIComments: &captureAINotFound{},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-404", "company_id": "ten2", "workspace_id": "ws2",
		"installed_image_id": "img-2", "created_by_id": "u2",
		"parent_comment_id": "cmt-gone",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_COMMENT_IMAGE_MENTIONED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_COMMENT_IMAGE_MENTIONED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v, want DispatchSuccess (start-vm succeeded, status write-back downgraded)", out)
	}
	ai := h.AIComments.(*captureAINotFound)
	if ai.calls != 1 {
		t.Fatalf("ai calls = %d, want 1", ai.calls)
	}
}

func TestDispatchPermanentWhenProjectFetchFails(t *testing.T) {
	h := &Handler{
		Publisher: &capturePub{},
		Tasks: &mockTaskClient{linked: []map[string]interface{}{
			{"project_id": "p-missing"},
		}},
		Projects:   &mockProjectClient{byID: map[string]map[string]interface{}{}},
		Cloud:      &captureCloud{},
		AIComments: &captureAI{},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-fail", "tenant_id": "ten1", "workspace_id": "ws1",
		"installed_image_id": "img-9", "created_by_id": "u1",
		"parent_comment_id": "cmt-1",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_COMMENT_IMAGE_MENTIONED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_COMMENT_IMAGE_MENTIONED", Data: data},
	})
	if out != domain.DispatchPermanent {
		t.Fatalf("outcome %v", out)
	}
	if err == nil || !strings.Contains(err.Error(), "加载任务关联项目详情失败") {
		t.Fatalf("err=%v want project fetch failure message (not misleading template-missing)", err)
	}
	if h.Cloud.(*captureCloud).calls != 0 {
		t.Fatal("StartVM must not be called when project fetch fails")
	}
}

func TestDispatchHTTPIntegrationMocks(t *testing.T) {
	var cloudPath string
	var aiPatched bool
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"projects": []map[string]interface{}{{"project_id": "p1"}},
		})
	})
	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/projects/") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "p1",
				"server_run_template": map[string]interface{}{
					"region": "cn-hangzhou", "cloud_platform_id": "plat-1",
					"vpc_id": "vpc-x", "vswitch_id": "vsw-x", "security_group_id": "sg-x",
					"hardware_config": map[string]interface{}{"cpu_cores": "2", "memory_gb": "4", "storage_gb": 40},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/api/cloud/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/cloud/compute/") {
			cloudPath = r.URL.Path
			if r.Header.Get("X-User-Id") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/api/tenant/", func(w http.ResponseWriter, r *http.Request) {
		// Legacy paths must not be used by the handler after convention migration.
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404 page not found"))
	})
	mux.HandleFunc("/api/internal/task-ai-comment/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/status") {
			aiPatched = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("TASK_TASK_SERVICE_BASE_URL", srv.URL)
	t.Setenv("TASK_PROJECT_SERVICE_BASE_URL", srv.URL)
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)
	t.Setenv("TASK_AI_COMMENT_URL", srv.URL)

	h := &Handler{Publisher: &capturePub{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t3", "tenant_id": "ten3", "workspace_id": "ws3",
		"installed_image_id": "img-3", "created_by_id": "u3",
		"parent_comment_id": "cmt-3",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_COMMENT_IMAGE_MENTIONED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_COMMENT_IMAGE_MENTIONED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if !strings.Contains(cloudPath, "/start-vm/") && !strings.Contains(cloudPath, "start-vm") {
		t.Fatalf("cloudPath=%s", cloudPath)
	}
	if !aiPatched {
		t.Fatal("expected AI comment status patch")
	}
}

func TestDispatchUsesEventServerRunTemplateOverProject(t *testing.T) {
	tasks := &mockTaskClient{linked: []map[string]interface{}{{"project_id": "p1"}}}
	h := &Handler{
		Publisher: &capturePub{},
		Tasks:     tasks,
		Projects: &mockProjectClient{byID: map[string]map[string]interface{}{
			"p1": {
				"id": "p1",
				"server_run_template": map[string]interface{}{
					"region":            "cn-beijing",
					"cloud_platform_id": "plat-project",
					"vpc_id":            "vpc-1",
					"vswitch_id":        "vsw-1",
					"security_group_id": "sg-1",
					"hardware_config":   map[string]interface{}{"cpu_cores": "4", "memory_gb": "8", "storage_gb": "50"},
				},
			},
		}},
		Cloud:      &captureCloud{},
		AIComments: &captureAI{},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-ov", "tenant_id": "ten1", "workspace_id": "ws1",
		"installed_image_id": "img-9", "created_by_id": "u1",
		"parent_comment_id": "cmt-ov",
		"server_run_template": map[string]interface{}{
			"region":            "cn-hangzhou",
			"cloud_platform_id": "plat-temp",
			"vpc_id":            "vpc-temp",
			"vswitch_id":        "vsw-temp",
			"security_group_id": "sg-temp",
			"hardware_config":   map[string]interface{}{"cpu_cores": "2", "memory_gb": "4", "storage_gb": "40"},
		},
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_COMMENT_IMAGE_MENTIONED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_COMMENT_IMAGE_MENTIONED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	cloud := h.Cloud.(*captureCloud)
	if cloud.body["region_id"] != "cn-hangzhou" {
		t.Fatalf("region_id=%v want override cn-hangzhou", cloud.body["region_id"])
	}
	if cloud.body["cloud_platform_id"] != "plat-temp" {
		t.Fatalf("cloud_platform_id=%v want plat-temp", cloud.body["cloud_platform_id"])
	}
}

func TestDispatchSkipsProjectFetchWhenEventTemplateComplete(t *testing.T) {
	tasks := &countingTaskClient{}
	h := &Handler{
		Publisher:  &capturePub{},
		Tasks:      tasks,
		Projects:   &mockProjectClient{byID: map[string]map[string]interface{}{}},
		Cloud:      &captureCloud{},
		AIComments: &captureAI{},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-skip", "tenant_id": "ten1", "workspace_id": "ws1",
		"installed_image_id": "img-9", "created_by_id": "u1",
		"parent_comment_id": "cmt-skip",
		"server_run_template": map[string]interface{}{
			"region":            "cn-hangzhou",
			"cloud_platform_id": "plat-temp",
			"hardware_config":   map[string]interface{}{"cpu_cores": "2", "memory_gb": "4", "storage_gb": "40"},
		},
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_COMMENT_IMAGE_MENTIONED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_COMMENT_IMAGE_MENTIONED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if tasks.calls != 0 {
		t.Fatalf("FetchTaskLinkedProjects calls=%d want 0 when event template is complete", tasks.calls)
	}
	if h.Cloud.(*captureCloud).calls != 1 {
		t.Fatal("StartVM must still run")
	}
}

func TestDispatchSkipsStartVMWhenWaitPreviousHasPredecessors(t *testing.T) {
	cloud := &captureCloud{}
	h := &Handler{
		Publisher:  &capturePub{},
		Tasks:      &countingTaskClient{},
		Projects:   &mockProjectClient{byID: map[string]map[string]interface{}{}},
		Cloud:      cloud,
		AIComments: &captureAI{},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-serial", "tenant_id": "ten1", "workspace_id": "ws1",
		"installed_image_id": "img-9", "created_by_id": "u1",
		"parent_comment_id":           "cmt-java",
		"execution_mode":              "wait_previous",
		"has_unfinished_predecessors": true,
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_COMMENT_IMAGE_MENTIONED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_COMMENT_IMAGE_MENTIONED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if cloud.calls != 0 {
		t.Fatalf("StartVM calls=%d want 0 for serial wait_previous", cloud.calls)
	}
	if h.AIComments.(*captureAI).calls != 0 {
		t.Fatal("must not mark AI starting when start-vm is deferred")
	}
}

func TestDispatchStartsVMWhenIndependentDespitePredecessors(t *testing.T) {
	cloud := &captureCloud{}
	h := &Handler{
		Publisher: &capturePub{},
		Tasks: &mockTaskClient{linked: []map[string]interface{}{
			{"project_id": "p1"},
		}},
		Projects: &mockProjectClient{byID: map[string]map[string]interface{}{
			"p1": {
				"id": "p1",
				"server_run_template": map[string]interface{}{
					"region":            "cn-beijing",
					"cloud_platform_id": "plat-2",
					"authorization_id":  "auth-2",
					"vpc_id":            "vpc-1",
					"vswitch_id":        "vsw-1",
					"security_group_id": "sg-1",
					"zone_id":           "cn-beijing-a",
					"selected_instance": "ecs.g6.large",
					"hardware_config": map[string]interface{}{
						"cpu_cores": "4", "memory_gb": "8", "storage_gb": "50", "instance_type": "ecs.g6.large",
					},
				},
			},
		}},
		Cloud:      cloud,
		AIComments: &captureAI{},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-par", "company_id": "ten2", "workspace_id": "ws2",
		"installed_image_id": "img-2", "created_by_id": "u2",
		"parent_comment_id":           "cmt-lisp",
		"execution_mode":              "independent",
		"has_unfinished_predecessors": true,
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "TASK_COMMENT_IMAGE_MENTIONED",
		Envelope:  domain.EventEnvelope{EventType: "TASK_COMMENT_IMAGE_MENTIONED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if cloud.calls != 1 {
		t.Fatalf("StartVM calls=%d want 1 for independent", cloud.calls)
	}
}

type countingTaskClient struct {
	calls int
}

func (m *countingTaskClient) FetchTaskLinkedProjects(ctx context.Context, taskID, tenantID, userID string) ([]map[string]interface{}, error) {
	_ = ctx
	_ = taskID
	_ = tenantID
	_ = userID
	m.calls++
	return nil, errNotFound
}
