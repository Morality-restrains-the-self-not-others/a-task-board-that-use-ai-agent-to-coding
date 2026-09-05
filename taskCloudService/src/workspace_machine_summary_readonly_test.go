package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestWorkspaceMachineSummaryGETDoesNotDescribeInstances(t *testing.T) {
	setupCloudTestDB(t)

	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-ro','aliyun','access_key','sid','skey','', 't1',1)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := upsertCloudServerConfig(CloudServerConfig{
		ID: "cfg-ro", CompanyID: "t1", WorkspaceID: "ws1", TaskID: "task-ro", CommentID: "cmt-ro",
		Platform: "aliyun", InstanceID: "i-ro", Region: "cn-hongkong",
		AuthorizationID: "auth-ro", LastRuntimeStatus: "Running",
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
		return map[string]interface{}{"Status": "Running"}, true, "req", nil
	}
	describeInstancesByName = func(accessKey, secretKey, regionID, instanceName string) ([]map[string]interface{}, string, error) {
		describeCalls.Add(1)
		return nil, "req", nil
	}

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-machine-summary/", nil)
	req.Header.Set("X-User-Id", "test-user")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Workspace-Id", "ws1")
	rec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(rec, req, "cloud/compute/workspace-machine-summary/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["started_count"] != float64(1) {
		t.Fatalf("started_count=%v want 1 from last_runtime_status", body["started_count"])
	}
	if n := describeCalls.Load(); n != 0 {
		t.Fatalf("GET summary must not DescribeInstances, calls=%d", n)
	}

	indReq := httptest.NewRequest(http.MethodGet,
		"/api/tenant/t1/workspace/ws1/cloud/compute/workspace-runtime-indicators/", nil)
	indReq.Header.Set("X-User-Id", "test-user")
	indReq.Header.Set("X-Auth-Tenant-Id", "t1")
	indReq.Header.Set("X-Workspace-Id", "ws1")
	indRec := httptest.NewRecorder()
	handleCloudWorkspaceRoutes(indRec, indReq, "cloud/compute/workspace-runtime-indicators/")
	if indRec.Code != http.StatusOK {
		t.Fatalf("indicators status=%d body=%s", indRec.Code, indRec.Body.String())
	}
	if n := describeCalls.Load(); n != 0 {
		t.Fatalf("GET indicators must not DescribeInstances, calls=%d", n)
	}
}
