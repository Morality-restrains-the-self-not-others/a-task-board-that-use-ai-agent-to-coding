package containermigrateawaitready

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestContainerStarterPostsRelayToTraeStartPath 锁定 HTTPContainerStarter 出站路径约定：
// taskContainerGateway 的 parseRelayToTraePath 只解析
//
//	/api/tenant/{t}/workspace/{w}/task/{task}/cloud/compute/relay-to-trae/{action}
//
// 形态（见 taskContainerGateway/src/relay_handlers.go）。任何把该路径误判为 legacy /api/tenant
// 云路径而改写成 /api/cloud/compute/... 的“迁移”，都会让网关 404、事件永久投 DLT。
// 回归范围：OPT-20260811-055。
func TestContainerStarterPostsRelayToTraeStartPath(t *testing.T) {
	t.Parallel()

	var (
		gotPath   string
		gotHeader http.Header
		gotBody   []byte
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHeader = r.Header.Clone()
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	s := &HTTPContainerStarter{
		BaseURL:        srv.URL,
		InternalSecret: "internal-secret",
		GatewaySecret:  "gateway-secret",
		Client:         &http.Client{Timeout: 5 * time.Second},
	}
	err := s.Start(context.Background(), "t1", "w1", "task1", "img-1", "repo/img:latest")
	if err != nil {
		t.Fatalf("Start err=%v", err)
	}

	wantPath := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/relay-to-trae/start/"
	if gotPath != wantPath {
		t.Fatalf("path=%q want=%q", gotPath, wantPath)
	}
	if gotHeader.Get("X-Internal-Secret") != "internal-secret" {
		t.Errorf("X-Internal-Secret=%q", gotHeader.Get("X-Internal-Secret"))
	}
	if gotHeader.Get("X-TaskContainerGateway-Internal-Secret") != "gateway-secret" {
		t.Errorf("X-TaskContainerGateway-Internal-Secret=%q", gotHeader.Get("X-TaskContainerGateway-Internal-Secret"))
	}
	if gotHeader.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type=%q", gotHeader.Get("Content-Type"))
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal body err=%v body=%s", err, gotBody)
	}
	for key, want := range map[string]string{
		"tenant_id":           "t1",
		"workspace_id":        "w1",
		"task_id":             "task1",
		"installed_image_id":  "img-1",
		"container_image_id":  "img-1",
		"image":               "repo/img:latest",
		"container_image_url": "repo/img:latest",
	} {
		if got, _ := payload[key].(string); got != want {
			t.Errorf("payload[%s]=%q want %q", key, got, want)
		}
	}
}

// TestContainerStarterReturnsErrorOnGatewayNon2xx 保证网关 4xx/5xx 时 Start 透出可诊断错误，
// 供 DLT 侧区分「网关拒绝」与「网络不可达」。
func TestContainerStarterReturnsErrorOnGatewayNon2xx(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"detail":"not found"}`))
	}))
	defer srv.Close()

	s := &HTTPContainerStarter{BaseURL: srv.URL, Client: srv.Client()}
	err := s.Start(context.Background(), "t1", "w1", "task1", "", "")
	if err == nil {
		t.Fatal("expected error on gateway 404")
	}
	if !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err=%v want 404 + body detail", err)
	}
}

// TestContainerStarterNilClientFallback 覆盖 BaseURL 缺省与 Client 为 nil 的兜底分支。
func TestContainerStarterNilClientFallback(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := &HTTPContainerStarter{BaseURL: srv.URL}
	if err := s.Start(context.Background(), "t1", "w1", "task1", "", ""); err != nil {
		t.Fatalf("Start err=%v", err)
	}
	if want := "/api/tenant/t1/workspace/w1/task/task1/cloud/compute/relay-to-trae/start/"; gotPath != want {
		t.Fatalf("path=%q want=%q", gotPath, want)
	}
}
