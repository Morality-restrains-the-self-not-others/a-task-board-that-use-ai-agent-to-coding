package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// OPT-20260817-002：授权缺失回退仍展示实例带宽（读 CSC 缓存字段）。
func TestBuildAuthMissingRuntimeFallbackShowsBandwidth(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, public_ip, server_url, region, zone_id, authorization_id, last_runtime_status)
		VALUES ('cfg-bw-1','t1','ws1','task-bw-1','cmt-bw-1','aliyun','i-bw-1','203.0.113.60','http://203.0.113.60:8080/','cn-hangzhou','cn-hangzhou-i','deleted-auth','Running')`)
	if err != nil {
		t.Fatalf("seed config: %v", err)
	}
	if err := persistInstanceBandwidth("i-bw-1", "PayByTraffic", 10); err != nil {
		t.Fatalf("persist bandwidth: %v", err)
	}
	cfg := &CloudServerConfig{
		InstanceID: "i-bw-1", PublicIP: "203.0.113.60",
		ServerURL: "http://203.0.113.60:8080/", Platform: "aliyun",
		Region: "cn-hangzhou", LastRuntimeStatus: "Running",
	}
	resp := buildAuthMissingRuntimeFallback(cfg, "i-bw-1")
	attr, ok := resp["instance_attribute"].(map[string]interface{})
	if !ok {
		t.Fatal("missing instance_attribute in fallback")
	}
	body, ok := attr["body"].(map[string]interface{})
	if !ok {
		t.Fatal("missing instance_attribute.body in fallback")
	}
	if got := body["InternetChargeType"]; got != "PayByTraffic" {
		t.Fatalf("InternetChargeType=%v want PayByTraffic", got)
	}
	if got := body["InternetMaxBandwidthOut"]; got != 10 {
		t.Fatalf("InternetMaxBandwidthOut=%v want 10", got)
	}
}

// OPT-20260817-002：live Describe 成功后把带宽字段持久化到 CSC，供后续 auth_missing 复用。
func TestServerRuntimeStatusLiveDescribePersistsBandwidth(t *testing.T) {
	setupCloudTestDB(t)
	if _, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-bw','aliyun','access_key','sid-bw','skey-bw','','t1',1)`); err != nil {
		t.Fatalf("seed auth: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cloud_server_configs
		(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, public_ip, region, zone_id, authorization_id, last_runtime_status)
		VALUES ('cfg-bw-2','t1','ws1','task-bw-2','cmt-rt','aliyun','i-bw-2','203.0.113.61','cn-hangzhou','cn-hangzhou-i','auth-bw','Running')`); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	old := describeInstanceForRuntime
	t.Cleanup(func() { describeInstanceForRuntime = old })
	describeInstanceForRuntime = func(accessKey, secretKey, regionID, instanceID string) (map[string]interface{}, bool, string, error) {
		return map[string]interface{}{
			"Status":                    "Running",
			"InstanceId":                instanceID,
			"InternetChargeType":        "PayByTraffic",
			"InternetMaxBandwidthOut":   int32(10),
			"PublicIpAddress":           map[string]interface{}{"IpAddress": []string{"203.0.113.61"}},
		}, true, "req-bw-1", nil
	}
	req := httptest.NewRequest(http.MethodGet, "/?task_id=task-bw-2&comment_id="+testLiveCommentID, nil)
	rec := httptest.NewRecorder()
	handleServerRuntimeStatus(rec, req, "t1", "ws1", "task-bw-2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	chargeType, bwOut := loadInstanceBandwidth("i-bw-2")
	if chargeType != "PayByTraffic" {
		t.Fatalf("persisted InternetChargeType=%q", chargeType)
	}
	if bwOut != 10 {
		t.Fatalf("persisted InternetMaxBandwidthOut=%d", bwOut)
	}
}
