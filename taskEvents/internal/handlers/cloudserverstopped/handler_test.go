package cloudserverstopped

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"taskEvents/domain"
	"taskEvents/internal/cloud/aliyun"
	"taskEvents/internal/repository/saas"
	"taskEvents/internal/saastest"
)

type mockStopper struct{}

func (mockStopper) StopVM(accessKey, secretKey string, in aliyun.StopVMInput) (aliyun.StopVMResult, error) {
	return aliyun.StopVMResult{InstanceID: in.InstanceID, RegionID: in.RegionID, RequestIDs: []string{"req-stop"}}, nil
}

type countingStopper struct{ calls int }

func (s *countingStopper) StopVM(accessKey, secretKey string, in aliyun.StopVMInput) (aliyun.StopVMResult, error) {
	s.calls++
	return aliyun.StopVMResult{InstanceID: in.InstanceID, RegionID: in.RegionID, RequestIDs: []string{"req-stop"}}, nil
}

type recordingPub struct {
	mu     sync.Mutex
	events []publishedEvent
}

type publishedEvent struct {
	EventType string
	Data      map[string]interface{}
	Key       string
}

func (p *recordingPub) PublishEvent(ctx context.Context, eventType string, data map[string]interface{}, key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	copied := map[string]interface{}{}
	for k, v := range data {
		copied[k] = v
	}
	p.events = append(p.events, publishedEvent{EventType: eventType, Data: copied, Key: key})
	return nil
}

func (p *recordingPub) snapshot() []publishedEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]publishedEvent, len(p.events))
	copy(out, p.events)
	return out
}

func TestDispatchStopMockInstanceSkipsCloudAPI(t *testing.T) {
	var cleared bool
	var cloudDeleteCalled bool
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-platform-authorizations/lookup"):
			cloudDeleteCalled = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"platform_type": "aliyun", "secret_id": "ak", "secret_key": "sk",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/internal/cloud-server-config/clear-after-stop/":
			cleared = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	repo := saas.New("")
	pub := &recordingPub{}
	h := &Handler{Repo: repo, Publisher: pub, Stopper: mockStopper{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-mock-stop", "company_id": 100, "workspace_id": "ws",
		"comment_id":       "cmt_live",
		"authorization_id": "cpa_1", "cloud_platform_type": "aliyun",
		"instance_id": "mock-e2e-1", "region_id": "cn-hongkong",
		"stop_request_id": "sr-mock-1",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STOPPED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STOPPED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if !cleared {
		t.Fatal("expected clear-after-stop for mock instance")
	}
	if cloudDeleteCalled {
		t.Fatal("mock instance must not look up cloud auth / call DeleteInstance")
	}
	evs := pub.snapshot()
	foundProcessing, foundSuccess := false, false
	for _, ev := range evs {
		if sd, ok := ev.Data["status_data"].(map[string]interface{}); ok {
			msg := fmt.Sprint(sd["message"])
			switch {
			case strings.Contains(msg, "正在停止 Mock 实例"):
				foundProcessing = true
				if !strings.Contains(msg, "触发：") {
					t.Fatalf("processing SSE must name trigger: %q", msg)
				}
			case strings.Contains(msg, "Mock 实例已停止"):
				foundSuccess = true
			case strings.Contains(msg, "正在调用") && strings.Contains(msg, "停止服务器"):
				t.Fatalf("mock stop must not use cloud-API SSE copy: %+v", sd)
			case msg == "停止虚拟机成功":
				t.Fatalf("mock stop must not use cloud-API SSE copy: %+v", sd)
			}
		}
	}
	if !foundProcessing || !foundSuccess {
		t.Fatalf("expected Mock SSE copy, processing=%v success=%v events=%+v", foundProcessing, foundSuccess, evs)
	}
	foundLive := false
	for _, ev := range evs {
		sd, ok := ev.Data["status_data"].(map[string]interface{})
		if !ok {
			continue
		}
		if strings.Contains(fmt.Sprint(sd["message"]), "Mock 实例已停止") {
			if fmt.Sprint(sd["comment_id"]) != "cmt_live" {
				t.Fatalf("success SSE comment_id=%v want cmt_live", sd["comment_id"])
			}
			if fmt.Sprint(sd["runtime_status"]) != "Stopped" {
				t.Fatalf("success SSE runtime_status=%v want Stopped", sd["runtime_status"])
			}
			foundLive = true
		}
		if strings.Contains(fmt.Sprint(sd["message"]), "正在停止 Mock 实例") && fmt.Sprint(sd["runtime_status"]) != "Stopping" {
			t.Fatalf("processing SSE runtime_status=%v want Stopping", sd["runtime_status"])
		}
	}
	if !foundLive {
		t.Fatalf("expected success SSE with comment_id + runtime_status Stopped, events=%+v", evs)
	}
}

// TestDispatchStopIdempotencySkip（OPT-20260818-015）：owner DB 已按 stop_request_id
// 落库（cloud_stop_request 唯一键）时，重放事件直接跳过，不得再次 DeleteInstance / ClearAfterStop。
func TestDispatchStopIdempotencySkip(t *testing.T) {
	var cleared bool
	stopper := &countingStopper{}
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/internal/cloud/stops/processed":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"processed": true})
		case r.Method == http.MethodPost && r.URL.Path == "/api/internal/cloud-server-config/clear-after-stop/":
			cleared = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	h := &Handler{Repo: saas.New(""), Publisher: &recordingPub{}, Stopper: stopper}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-replay", "company_id": 100, "workspace_id": "ws",
		"authorization_id": "cpa_1", "cloud_platform_type": "aliyun",
		"instance_id": "i-replay", "region_id": "cn-hongkong",
		"stop_request_id": "sr-already-processed",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STOPPED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STOPPED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v want success", out)
	}
	if stopper.calls != 0 {
		t.Fatalf("replay must not call DeleteInstance, calls=%d", stopper.calls)
	}
	if cleared {
		t.Fatal("replay must not clear-after-stop again")
	}
}

// TestDispatchStopIdempotencyCheckFailOpen（OPT-20260818-015）：processed 检查失败时
// fail-open 继续处理，真实停止绝不能被幂等误跳。
func TestDispatchStopIdempotencyCheckFailOpen(t *testing.T) {
	var cleared bool
	stopper := &countingStopper{}
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/internal/cloud/stops/processed":
			w.WriteHeader(http.StatusInternalServerError)
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-platform-authorizations/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"platform_type": "aliyun", "secret_id": "ak", "secret_key": "sk",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/internal/cloud-server-config/clear-after-stop/":
			cleared = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	h := &Handler{Repo: saas.New(""), Publisher: &recordingPub{}, Stopper: stopper}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-checkfail", "company_id": 100, "workspace_id": "ws",
		"authorization_id": "cpa_1", "cloud_platform_type": "aliyun",
		"instance_id": "i-checkfail", "region_id": "cn-hongkong",
		"stop_request_id": "sr-checkfail",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STOPPED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STOPPED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v want success", out)
	}
	if stopper.calls != 1 {
		t.Fatalf("check failure must not skip DeleteInstance, calls=%d", stopper.calls)
	}
	if !cleared {
		t.Fatal("expected clear-after-stop")
	}
}

func TestDispatchStopSuccess(t *testing.T) {
	var cleared bool
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-platform-authorizations/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"platform_type": "aliyun", "secret_id": "ak", "secret_key": "sk",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/internal/cloud-server-config/clear-after-stop/":
			cleared = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	repo := saas.New("")
	pub := &recordingPub{}
	h := &Handler{Repo: repo, Publisher: pub, Stopper: mockStopper{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-stop", "company_id": 100, "workspace_id": "ws",
		"authorization_id": "cpa_-3077015452416368750", "cloud_platform_type": "aliyun",
		"instance_id": "i-stop", "region_id": "cn-hongkong",
		"stop_reason":       "instruction_idle",
		"stop_reason_label": "容器指令空闲超时回收",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STOPPED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STOPPED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("outcome %v", out)
	}
	if !cleared {
		t.Fatal("expected clear-after-stop")
	}
	foundTrigger := false
	for _, ev := range pub.snapshot() {
		sd, ok := ev.Data["status_data"].(map[string]interface{})
		if !ok {
			continue
		}
		msg := fmt.Sprint(sd["message"])
		if strings.Contains(msg, "正在调用aliyunAPI停止服务器") && strings.Contains(msg, "触发：容器指令空闲超时回收") {
			foundTrigger = true
		}
	}
	if !foundTrigger {
		t.Fatalf("expected annotated aliyun stop SSE, events=%+v", pub.snapshot())
	}
}

// TestDispatchStopMissingAuthorizationID 回归（OPT-20260809-026）：authorization_id 缺失时
// 报"缺少 authorization_id"并永久失败；平台 SSOT 的 ID 为 cpa_<snowflake> 字符串格式，
// 历史代码用 Int64Field 解析字符串 ID 失败也会误报此错误。
func TestDispatchStopMockPlatformSkipsCloudAPI(t *testing.T) {
	var cleared bool
	var cloudLookupCalled bool
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-platform-authorizations/lookup"):
			cloudLookupCalled = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"platform_type": "aliyun", "secret_id": "ak", "secret_key": "sk",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/internal/cloud-server-config/clear-after-stop/":
			cleared = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	stopper := &countingStopper{}
	h := &Handler{Repo: saas.New(""), Publisher: &recordingPub{}, Stopper: stopper}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-mock-plat", "company_id": 100, "workspace_id": "ws",
		"cloud_platform_type": "mock",
		"instance_id":         "mock-e2e-plat", "region_id": "cn-hongkong",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STOPPED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STOPPED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("mock platform must not DLT, outcome %v", out)
	}
	if !cleared {
		t.Fatal("expected clear-after-stop for mock platform")
	}
	if cloudLookupCalled || stopper.calls != 0 {
		t.Fatal("mock platform must not look up cloud auth or DeleteInstance")
	}
}

func TestDispatchStopMockLabelOnAliyunInstanceUsesCloudAPI(t *testing.T) {
	var cleared bool
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/internal/cloud-platform-authorizations/lookup"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"platform_type": "aliyun", "secret_id": "ak", "secret_key": "sk",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/internal/cloud-server-config/clear-after-stop/":
			cleared = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	stopper := &countingStopper{}
	h := &Handler{Repo: saas.New(""), Publisher: &recordingPub{}, Stopper: stopper}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-stale-mock", "company_id": 100, "workspace_id": "ws",
		"authorization_id": "cpa_1", "cloud_platform_type": "mock",
		"instance_id": "i-stale-mock", "region_id": "cn-hongkong",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STOPPED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STOPPED", Data: data},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out != domain.DispatchSuccess {
		t.Fatalf("stale mock label on ECS must succeed, outcome %v", out)
	}
	if stopper.calls != 1 {
		t.Fatalf("expected Aliyun DeleteInstance, calls=%d", stopper.calls)
	}
	if !cleared {
		t.Fatal("expected clear-after-stop")
	}
}

func TestDispatchStopUnsupportedVendorPlatformPermanent(t *testing.T) {
	h := &Handler{Repo: saas.New(""), Publisher: &recordingPub{}, Stopper: mockStopper{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-tencent", "company_id": 100, "workspace_id": "ws",
		"cloud_platform_type": "tencent",
		"instance_id":         "ins-x", "region_id": "ap-guangzhou",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STOPPED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STOPPED", Data: data},
	})
	if out != domain.DispatchPermanent {
		t.Fatalf("expected permanent, got %v", out)
	}
	if err == nil || !strings.Contains(err.Error(), "unsupported platform tencent") {
		t.Fatalf("expected unsupported platform tencent, got %v", err)
	}
}

func TestDispatchStopMissingAuthorizationID(t *testing.T) {
	cloudSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer cloudSrv.Close()
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", cloudSrv.URL)

	intent := saastest.NewIntentMux()
	intentSrv := intent.Server()
	defer intentSrv.Close()

	repo := saas.New("")
	pub := &recordingPub{}
	h := &Handler{Repo: repo, Publisher: pub, Stopper: mockStopper{}}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t-noid", "company_id": 100, "workspace_id": "ws",
		"cloud_platform_type": "aliyun",
		"instance_id":         "i-noid", "region_id": "cn-hongkong",
	})
	out, err := h.Dispatch(context.Background(), domain.DomainCommand{
		EventType: "CLOUD_SERVER_STOPPED",
		Envelope:  domain.EventEnvelope{EventType: "CLOUD_SERVER_STOPPED", Data: data},
	})
	if out != domain.DispatchPermanent {
		t.Fatalf("expected permanent outcome, got %v", out)
	}
	if err == nil || err.Error() != "missing authorization_id" {
		t.Fatalf("expected missing authorization_id error, got %v", err)
	}
	evs := pub.snapshot()
	foundErr := false
	for _, ev := range evs {
		if sd, ok := ev.Data["status_data"].(map[string]interface{}); ok {
			if fmt.Sprint(sd["status"]) == "error" && fmt.Sprint(sd["message"]) == "缺少 authorization_id" {
				foundErr = true
			}
		}
	}
	if !foundErr {
		t.Fatalf("expected error SSE copy, events=%+v", evs)
	}
}
