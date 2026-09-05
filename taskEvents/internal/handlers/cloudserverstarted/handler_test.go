package cloudserverstarted

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/cloud/aliyun"
	"taskEvents/internal/repository/saas"
	"taskEvents/internal/saastest"
)

type mockStarter struct {
	lastInput aliyun.StartVMInput
	starts    int
}

func (m *mockStarter) StartVM(ctx context.Context, accessKey, secretKey string, in aliyun.StartVMInput) (aliyun.StartVMResult, error) {
	m.lastInput = in
	m.starts++
	return aliyun.StartVMResult{InstanceID: "i-mock", RequestID: "req-1", ClientToken: in.TaskID}, nil
}

type mockStopper struct {
	stopped []string
}

func (m *mockStopper) StopVM(accessKey, secretKey string, in aliyun.StopVMInput) (aliyun.StopVMResult, error) {
	m.stopped = append(m.stopped, in.InstanceID)
	return aliyun.StopVMResult{InstanceID: in.InstanceID, RegionID: in.RegionID}, nil
}

type noopPub struct{}

func (noopPub) PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	return nil
}

func TestDispatchSuccess(t *testing.T) {
	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	eventStatus := map[int64]string{}
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-platform-authorizations/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"platform_type": "aliyun", "secret_id": "ak", "secret_key": "sk",
			})
		case r.URL.Path == "/api/internal/cloud-server-config/import/":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "imported": 1})
		case r.URL.Path == "/api/internal/cloud-server-config/histories/":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "csh-1"})
		case r.URL.Path == "/api/internal/cloud-server-events/latest-pending-start":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "10", "status": "pending"})
		case r.URL.Path == "/api/internal/cloud-server-events/update-status":
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			eid, _ := body["event_id"].(string)
			st, _ := body["status"].(string)
			var id int64
			_, _ = fmt.Sscan(eid, &id)
			eventStatus[id] = st
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		case r.URL.Path == "/api/internal/cloud-server-events/update-data":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: noopPub{}, Starter: &mockStarter{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "company_id": 100, "authorization_id": "cpa_1",
		"cloud_platform_type": "aliyun", "cloud_server_image_id": "img-1",
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
		"security_group_id": "sg-1", "vswitch_id": "vsw-1",
		"container_image_id":         "img-installed-1",
		"container_image_url":        "registry.example.com/demo:v1",
		"userdata_access_token":      "access-tok",
		"userdata_task_api_endpoint": "https://api.daydaymoney.com",
		"workspace_id":               "ws",
		"comment_id":                 "cmt_876810842430009344",
		"userdata_content":           "#!/bin/bash\nCONTAINER_IMAGE='__TASK2APP_CONTAINER_IMAGE__'\n",
		"hardware_config":            map[string]interface{}{"instance_type": "ecs.t5-lc1m1.small"},
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STARTED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STARTED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	starter := h.Starter.(*mockStarter)
	if starter.lastInput.TaskID != "t1" {
		t.Fatalf("TaskID=%q", starter.lastInput.TaskID)
	}
	if starter.lastInput.CommentID != "cmt_876810842430009344" {
		t.Fatalf("CommentID=%q want comment-scoped start", starter.lastInput.CommentID)
	}
	if strings.Contains(starter.lastInput.UserData, "__TASK2APP_CONTAINER_IMAGE__") {
		t.Fatalf("UserData still contains container image placeholder: %q", starter.lastInput.UserData)
	}
	if !strings.Contains(starter.lastInput.UserData, "registry.example.com/demo:v1") {
		t.Fatalf("UserData missing resolved image ref: %q", starter.lastInput.UserData)
	}
	if eventStatus[10] != saas.CloudEventSuccess {
		t.Fatalf("event status=%v", eventStatus)
	}
}

// TestDispatchReplayAfterSuccessSkips（OPT-20260818-015）：CLOUD_SERVER_STARTED 重放时
// 已无 pending 启动事件，但 DB 事件状态为 success → 幂等跳过，不得再次 StartVM / 投 DLT。
func TestDispatchReplayAfterSuccessSkips(t *testing.T) {
	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	starter := &mockStarter{}
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/internal/cloud-server-events/latest-pending-start":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"detail": "pending start event not found"})
		case r.URL.Path == "/api/internal/cloud-server-events/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"event_id": "evt-100", "status": "success"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: noopPub{}, Starter: starter}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-replay", "company_id": 100, "authorization_id": "cpa_1",
		"cloud_platform_type": "aliyun", "cloud_server_image_id": "img-1",
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
		"container_image_id":  "img-installed-1",
		"container_image_url": "registry.example.com/demo:v1",
		"workspace_id":        "ws",
		"userdata_content":    "#!/bin/bash\n",
		"event_id":            "evt-100",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STARTED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STARTED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("replay after success must be a clean idempotency skip, outcome %v", out)
	}
	if starter.starts != 0 {
		t.Fatalf("replay must not StartVM again, starts=%d", starter.starts)
	}
}

// TestDispatchClaimLostConcurrentSkips（OPT-20260818-015 剩余 gap）：两个并发消费者同时
// claim 同一个 pending 启动事件时，taskCloudService 的 from_status 守卫只允许一方
// pending→processing 成功；另一方收到 409 后必须幂等跳过（不得再次 StartVM / 投 DLT）。
func TestDispatchClaimLostConcurrentSkips(t *testing.T) {
	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	starter := &mockStarter{}
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/internal/cloud-server-events/latest-pending-start":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "12", "status": "pending"})
		case r.URL.Path == "/api/internal/cloud-server-events/update-status":
			// 事件已被另一并发消费者抢先 claim（pending→processing），返回 409。
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"detail": "event status changed"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: noopPub{}, Starter: starter}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-race", "company_id": 100, "authorization_id": "cpa_1",
		"cloud_platform_type": "aliyun", "cloud_server_image_id": "img-1",
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
		"container_image_id":  "img-installed-1",
		"container_image_url": "registry.example.com/demo:v1",
		"workspace_id":        "ws",
		"userdata_content":    "#!/bin/bash\n",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STARTED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STARTED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("lost claim must be a clean idempotency skip, outcome %v", out)
	}
	if starter.starts != 0 {
		t.Fatalf("lost claim must not StartVM again, starts=%d", starter.starts)
	}
}

// TestDispatchReplayWhileProcessingSkips（OPT-20260818-015 剩余 gap）：重放到达时
// 事件已被并发消费者 claim 为 processing（latest-pending-start 不再命中），
// 消费者按 event_id 查得 processing 也应幂等跳过，避免在他人启动中二次 StartVM。
func TestDispatchReplayWhileProcessingSkips(t *testing.T) {
	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	starter := &mockStarter{}
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/internal/cloud-server-events/latest-pending-start":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"detail": "pending start event not found"})
		case r.URL.Path == "/api/internal/cloud-server-events/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"event_id": "evt-101", "status": "processing"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: noopPub{}, Starter: starter}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-processing", "company_id": 100, "authorization_id": "cpa_1",
		"cloud_platform_type": "aliyun", "cloud_server_image_id": "img-1",
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
		"container_image_id":  "img-installed-1",
		"container_image_url": "registry.example.com/demo:v1",
		"workspace_id":        "ws",
		"userdata_content":    "#!/bin/bash\n",
		"event_id":            "evt-101",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STARTED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STARTED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("replay while processing must be a clean idempotency skip, outcome %v", out)
	}
	if starter.starts != 0 {
		t.Fatalf("replay while processing must not StartVM again, starts=%d", starter.starts)
	}
}

func TestDispatchSupersedesExistingInstanceBeforeStart(t *testing.T) {
	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	var clearedInstance string
	var callOrder []string
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-platform-authorizations/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"platform_type": "aliyun", "secret_id": "ak", "secret_key": "sk",
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "csc-1", "instance_id": "i-old", "region": "cn-hongkong",
				"platform": "aliyun", "authorization_id": "1", "workspace_id": "ws",
				// Only Stopped bindings may be superseded before a new RunInstances.
				"last_runtime_status": "Stopped",
			})
		case r.URL.Path == "/api/internal/cloud-server-config/clear-after-stop/":
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			clearedInstance, _ = body["instance_id"].(string)
			callOrder = append(callOrder, "clear")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "cleared": true})
		case r.URL.Path == "/api/internal/cloud-server-config/import/":
			callOrder = append(callOrder, "import")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "imported": 1})
		case r.URL.Path == "/api/internal/cloud-server-config/histories/":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "csh-1"})
		case r.URL.Path == "/api/internal/cloud-server-events/latest-pending-start":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "11", "status": "pending"})
		case r.URL.Path == "/api/internal/cloud-server-events/update-status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		case r.URL.Path == "/api/internal/cloud-server-events/update-data":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	starter := &mockStarter{}
	stopper := &mockStopper{}
	h := &Handler{
		Repo: saas.New(""), Publisher: noopPub{},
		Starter: starter, Stopper: stopper,
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "company_id": 100, "authorization_id": "cpa_1",
		"cloud_platform_type": "aliyun", "cloud_server_image_id": "img-1",
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
		"security_group_id": "sg-1", "vswitch_id": "vsw-1",
		"container_image_id":         "img-installed-1",
		"container_image_url":        "registry.example.com/demo:v1",
		"userdata_access_token":      "access-tok",
		"userdata_task_api_endpoint": "https://api.daydaymoney.com",
		"workspace_id":               "ws",
		"runtime_source":             "cloud_vm_manual",
		"userdata_content":           "#!/bin/bash\nCONTAINER_IMAGE='__TASK2APP_CONTAINER_IMAGE__'\n",
		"hardware_config":            map[string]interface{}{"instance_type": "ecs.t5-lc1m1.small"},
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STARTED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STARTED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if len(stopper.stopped) != 1 || stopper.stopped[0] != "i-old" {
		t.Fatalf("stopped=%v want [i-old]", stopper.stopped)
	}
	if starter.starts != 1 {
		t.Fatalf("starts=%d want 1", starter.starts)
	}
	if clearedInstance != "i-old" {
		t.Fatalf("cleared instance_id=%q want i-old", clearedInstance)
	}
	if len(callOrder) < 2 || callOrder[0] != "clear" {
		t.Fatalf("callOrder=%v want clear before import", callOrder)
	}
}

func TestDispatchCommentMentionDoesNotReleaseRunningInstance(t *testing.T) {
	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-platform-authorizations/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"platform_type": "aliyun", "secret_id": "ak", "secret_key": "sk",
			})
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-server-config/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "csc-1", "instance_id": "i-first-comment", "region": "cn-hongkong",
				"platform": "aliyun", "authorization_id": "1", "workspace_id": "ws",
				"last_runtime_status": "Running",
			})
		case r.URL.Path == "/api/internal/cloud-server-events/latest-pending-start":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "13", "status": "pending"})
		case r.URL.Path == "/api/internal/cloud-server-events/update-status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		case r.URL.Path == "/api/internal/cloud-server-events/update-data":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	starter := &mockStarter{}
	stopper := &mockStopper{}
	h := &Handler{
		Repo: saas.New(""), Publisher: noopPub{},
		Starter: starter, Stopper: stopper,
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "company_id": 100, "authorization_id": "cpa_1",
		"cloud_platform_type": "aliyun", "cloud_server_image_id": "img-1",
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
		"security_group_id": "sg-1", "vswitch_id": "vsw-1",
		"container_image_id":         "img-second",
		"container_image_url":        "registry.example.com/demo:v2",
		"comment_id":                 "cmt-2",
		"runtime_source":             "cloud_vm_comment_mention",
		"userdata_access_token":      "access-tok",
		"userdata_task_api_endpoint": "https://api.daydaymoney.com",
		"workspace_id":               "ws",
		"userdata_content":           "#!/bin/bash\necho ok\n",
		"hardware_config":            map[string]interface{}{"instance_type": "ecs.t5-lc1m1.small"},
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STARTED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STARTED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if len(stopper.stopped) != 0 {
		t.Fatalf("second @镜像 must not release first instance; stopped=%v", stopper.stopped)
	}
	if starter.starts != 0 {
		t.Fatalf("starts=%d want 0 (attach existing, no RunInstances)", starter.starts)
	}
}

func TestDispatchMissingContainerImageURL(t *testing.T) {
	// 回归（rule 41）：go vet 非恒定格式字符串修复（fmt.Errorf(msg)→fmt.Errorf("%s", msg)），
	// 断言 container_image_url 缺失时错误消息原样透传。修复前本包因 vet 构建失败，
	// 任何单测均为红。
	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-platform-authorizations/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"platform_type": "aliyun", "secret_id": "ak", "secret_key": "sk",
			})
		case r.URL.Path == "/api/internal/cloud-server-events/latest-pending-start":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "10", "status": "pending"})
		case r.URL.Path == "/api/internal/cloud-server-events/update-status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	repo := saas.New("")
	h := &Handler{Repo: repo, Publisher: noopPub{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "company_id": 100, "authorization_id": "cpa_1",
		"cloud_platform_type": "aliyun", "cloud_server_image_id": "img-1",
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
		"container_image_id": "img-installed-1", "workspace_id": "ws",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STARTED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STARTED", Data: data},
	})
	if out != domain.DispatchPermanent || err == nil || err.Error() != "missing container_image_url" {
		t.Fatalf("expected permanent missing container_image_url error, got out=%v err=%v", out, err)
	}
}

func TestDispatchStartedMockPlatformSucceedsWithoutCloud(t *testing.T) {
	starter := &mockStarter{}
	h := &Handler{Starter: starter, Publisher: noopPub{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-mock", "company_id": 100, "cloud_platform_type": "mock",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STARTED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STARTED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("mock platform must not DLT, outcome %v", out)
	}
	if starter.starts != 0 {
		t.Fatalf("mock platform must not call RunInstances, starts=%d", starter.starts)
	}
}
