package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestInternalReconcileWorkspaceMachineRuntimesDescribes(t *testing.T) {
	setupCloudTestDB(t)

	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-int','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-int", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-int", CommentID: "cmt-int",
		Platform: "aliyun", InstanceID: "i-gone-int", Region: "cn-hongkong",
		AuthorizationID: "auth-int", LastRuntimeStatus: "Running",
	}); err != nil {
		t.Fatal(err)
	}

	var describeCalls atomic.Int32
	oldDesc := describeInstanceForRuntime
	oldByName := describeInstancesByName
	t.Cleanup(func() {
		describeInstanceForRuntime = oldDesc
		describeInstancesByName = oldByName
	})
	describeInstanceForRuntime = func(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, bool, string, error) {
		describeCalls.Add(1)
		return nil, false, "req", nil
	}
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		return nil, "req", nil
	}

	req := httptest.NewRequest(http.MethodPost,
		"/api/internal/cloud/compute/reconcile-workspace-machine-runtimes/", nil)
	rec := httptest.NewRecorder()
	handleInternalReconcileWorkspaceMachineRuntimes(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "success" {
		t.Fatalf("body=%v", body)
	}
	if body["workspaces"] != float64(1) {
		t.Fatalf("workspaces=%v want 1", body["workspaces"])
	}
	if describeCalls.Load() < 1 {
		t.Fatalf("internal reconcile must DescribeInstances, calls=%d", describeCalls.Load())
	}

	cfg, err := loadCloudServerConfigForComment("t1", "ws1", "task-int", "cmt-int")
	if err != nil || cfg == nil {
		t.Fatalf("load: %v", err)
	}
	if strings.TrimSpace(cfg.InstanceID) != "" {
		t.Fatalf("expected released instance cleared, got %q", cfg.InstanceID)
	}
}

func TestInternalReconcileWorkspaceMachineRuntimesRejectsNonPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/cloud/compute/reconcile-workspace-machine-runtimes/", nil)
	rec := httptest.NewRecorder()
	handleInternalReconcileWorkspaceMachineRuntimes(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 405", rec.Code)
	}
}
