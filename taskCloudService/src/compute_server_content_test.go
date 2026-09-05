package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPersistCloudServerPublicIPByInstanceID(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`
		INSERT INTO cloud_server_configs
			(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, public_ip, server_url, region, zone_id, authorization_id)
		VALUES ('cfg1', 'c1', 'w1', 'task1', 'cmt1', 'aliyun', 'i-abc', '', '', 'cn-qingdao', 'cn-qingdao-b', 'auth1')
	`)
	if err != nil {
		t.Fatal(err)
	}

	persistCloudServerPublicIPByInstanceID("i-abc", "47.105.105.136", "")

	cfg, err := loadCloudServerConfigForComment("c1", "w1", "task1", "cmt1")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicIP != "47.105.105.136" {
		t.Fatalf("public_ip=%q", cfg.PublicIP)
	}
	// 公网 IP 回填不得写入推测性 server_url；否则 commentCSCHasRuntime 为真、
	// binding 误升 running（「容器已就绪」），而 :8080 仍可能 connection refused。
	if cfg.ServerURL != "" {
		t.Fatalf("server_url=%q want empty until register-reachability", cfg.ServerURL)
	}
	if commentCSCHasRuntime(cfg) {
		t.Fatal("public_ip alone must not satisfy commentCSCHasRuntime")
	}
}

func TestEnsureCloudServerConfigPublicIPBackfillsFromDescribe(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES ('auth1','aliyun','access_key','ak','sk','', 'c1', 1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		INSERT INTO cloud_server_configs
			(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, public_ip, server_url, region, zone_id, authorization_id)
		VALUES ('cfg1', 'c1', 'w1', 'task1', 'cmt1', 'aliyun', 'i-abc', '', '', 'cn-qingdao', 'cn-qingdao-b', 'auth1')
	`)
	if err != nil {
		t.Fatal(err)
	}

	prev := queryInstancePublicIPFn
	queryInstancePublicIPFn = func(accessKey, secretKey, regionID, instanceID string) (string, string) {
		if instanceID != "i-abc" {
			t.Fatalf("instanceID=%q", instanceID)
		}
		return "203.0.113.10", "http://203.0.113.10:8080"
	}
	t.Cleanup(func() { queryInstancePublicIPFn = prev })

	cfg, err := loadCloudServerConfigForComment("c1", "w1", "task1", "cmt1")
	if err != nil {
		t.Fatal(err)
	}
	got := ensureCloudServerConfigPublicIP(cfg)
	if got != "203.0.113.10" {
		t.Fatalf("ensure=%q", got)
	}
	reloaded, err := loadCloudServerConfigForComment("c1", "w1", "task1", "cmt1")
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.PublicIP != "203.0.113.10" {
		t.Fatalf("persisted public_ip=%q", reloaded.PublicIP)
	}
}

func TestHandleServerContentBackfillsPublicIP(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES ('auth1','aliyun','access_key','ak','sk','', 'c1', 1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		INSERT INTO cloud_server_configs
			(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, public_ip, region, zone_id, authorization_id)
		VALUES ('cfg1', 'c1', 'w1', 'task1', 'cmt-rt', 'aliyun', 'i-abc', '', 'cn-qingdao', 'cn-qingdao-b', 'auth1')
	`)
	if err != nil {
		t.Fatal(err)
	}

	prev := queryInstancePublicIPFn
	queryInstancePublicIPFn = func(accessKey, secretKey, regionID, instanceID string) (string, string) {
		return "198.51.100.7", "http://198.51.100.7:8080"
	}
	t.Cleanup(func() { queryInstancePublicIPFn = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/cloud/compute/server-content/?task_id=task1&comment_id=cmt-rt", nil)
	rec := httptest.NewRecorder()
	handleServerContent(rec, req, "c1", "w1", "task1")
	body := rec.Body.String()
	// 回填成功后应尝试拉取；测试网段 IP 不可达 → 502，但绝不能再是「公网 IP 不存在」400
	if rec.Code == http.StatusBadRequest && strings.Contains(body, "公网 IP 不存在") {
		t.Fatalf("still missing public ip: status=%d body=%s", rec.Code, body)
	}
	if rec.Code == http.StatusBadRequest {
		t.Fatalf("unexpected 400 after backfill: %s", body)
	}
	cfg, err := loadCloudServerConfigForComment("c1", "w1", "task1", "cmt-rt")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicIP != "198.51.100.7" {
		t.Fatalf("public_ip not backfilled: %q", cfg.PublicIP)
	}
}

// Regression（OPT-20260812-042）：容器业务端口尚未就绪（connection refused）时，
// server-content 应返回可区分的 status=starting 而非 502 硬错误。
func TestHandleServerContentConnectionRefusedIsStarting(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`
		INSERT INTO cloud_server_configs
			(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, public_ip, server_url, region, zone_id, authorization_id)
		VALUES ('cfg1', 'c1', 'w1', 'task1', 'cmt-rt', 'aliyun', 'i-abc', '198.51.100.7', '', 'cn-qingdao', 'cn-qingdao-b', 'auth1')
	`)
	if err != nil {
		t.Fatal(err)
	}
	prev := serverContentHTTPClient
	serverContentHTTPClient = &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
		}),
	}
	t.Cleanup(func() { serverContentHTTPClient = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/cloud/compute/server-content/?task_id=task1&comment_id=cmt-rt", nil)
	rec := httptest.NewRecorder()
	handleServerContent(rec, req, "c1", "w1", "task1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"starting"`) {
		t.Fatalf("body=%s want status=starting", rec.Body.String())
	}
}

func TestHandleServerContentMissingPublicIPNoInstance(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`
		INSERT INTO cloud_server_configs
			(id, company_id, workspace_id, task_id, comment_id, platform, instance_id, public_ip, region, zone_id, authorization_id)
		VALUES ('cfg1', 'c1', 'w1', 'task1', 'cmt-rt', 'aliyun', '', '', 'cn-qingdao', 'cn-qingdao-b', 'auth1')
	`)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/cloud/compute/server-content/?task_id=task1&comment_id=cmt-rt", nil)
	rec := httptest.NewRecorder()
	handleServerContent(rec, req, "c1", "w1", "task1")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "公网 IP 不存在") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestPublicIPFromDescribeAttr(t *testing.T) {
	attr := map[string]interface{}{
		"PublicIpAddress": map[string]interface{}{"IpAddress": []string{"1.2.3.4"}},
	}
	if got := publicIPFromDescribeAttr(attr); got != "1.2.3.4" {
		t.Fatalf("got=%q", got)
	}
	attr2 := map[string]interface{}{
		"EipAddress": map[string]interface{}{"IpAddress": "5.6.7.8"},
	}
	if got := publicIPFromDescribeAttr(attr2); got != "5.6.7.8" {
		t.Fatalf("eip got=%q", got)
	}
}
