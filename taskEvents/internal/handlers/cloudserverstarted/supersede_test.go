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

type errStopper struct {
	err error
}

func (e errStopper) StopVM(accessKey, secretKey string, in aliyun.StopVMInput) (aliyun.StopVMResult, error) {
	return aliyun.StopVMResult{}, e.err
}

func TestIsNonStoppableInstanceStatusError(t *testing.T) {
	raw := `停止虚拟机失败: SDKError: StatusCode: 403 Code: IncorrectInstanceStatus.Initializing Message: The specified instance status does not support this operation.`
	if !isNonStoppableInstanceStatusError(raw) {
		t.Fatal("expected Initializing to be non-stoppable")
	}
	if isNonStoppableInstanceStatusError("InvalidInstanceId.NotFound") {
		t.Fatal("NotFound should not be treated as non-stoppable transitional")
	}
}

func TestShouldPreserveBoundInstance(t *testing.T) {
	if shouldPreserveBoundInstance("cloud_vm_comment_mention", "Stopped") {
		t.Fatal("@镜像 may supersede an explicitly Stopped node")
	}
	if !shouldPreserveBoundInstance("cloud_vm_comment_mention", "") {
		t.Fatal("@镜像 must preserve empty/unknown status (mid-boot)")
	}
	if !shouldPreserveBoundInstance("cloud_vm_comment_mention", "Running") {
		t.Fatal("@镜像 must preserve Running")
	}
	if !shouldPreserveBoundInstance("cloud_vm_manual", "Running") {
		t.Fatal("Running must be preserved")
	}
	if !shouldPreserveBoundInstance("cloud_vm_manual", "Starting") {
		t.Fatal("Starting must be preserved")
	}
	if shouldPreserveBoundInstance("cloud_vm_manual", "Stopped") {
		t.Fatal("Stopped manual restart may supersede")
	}
	if shouldPreserveBoundInstance("cloud_vm_manual", "") {
		t.Fatal("manual with empty status may supersede (legacy restart)")
	}
}

func TestDispatchReusesInflightWhenSupersedeInitializing(t *testing.T) {
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
				"id": "csc-1", "instance_id": "i-init", "region": "cn-hongkong",
				"platform": "aliyun", "authorization_id": "1", "workspace_id": "ws",
				// Empty status: rely on StopVM IncorrectInstanceStatus.Initializing path.
				"last_runtime_status": "",
			})
		case r.URL.Path == "/api/internal/cloud-server-events/latest-pending-start":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "12", "status": "pending"})
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
	h := &Handler{
		Repo: saas.New(""), Publisher: noopPub{},
		Starter: starter,
		Stopper: errStopper{err: fmt.Errorf("停止虚拟机失败: SDKError: Code: IncorrectInstanceStatus.Initializing")},
	}
	data, _ := json.Marshal(map[string]interface{}{
		"task_id": "t1", "company_id": 100, "authorization_id": 1,
		"cloud_platform_type": "aliyun", "cloud_server_image_id": "img-1",
		"region_id": "cn-hongkong", "zone_id": "cn-hongkong-b",
		"security_group_id": "sg-1", "vswitch_id": "vsw-1",
		"container_image_id": "img-installed-2",
		"container_image_url": "registry.example.com/demo:v2",
		"comment_id": "cmt-2",
		"userdata_access_token": "access-tok",
		"userdata_task_api_endpoint": "https://api.daydaymoney.com",
		"workspace_id": "ws",
		"userdata_content": "#!/bin/bash\necho ok\n",
		"hardware_config": map[string]interface{}{"instance_type": "ecs.t5-lc1m1.small"},
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
	if starter.starts != 0 {
		t.Fatalf("starts=%d want 0 (reuse inflight, no RunInstances)", starter.starts)
	}
}
