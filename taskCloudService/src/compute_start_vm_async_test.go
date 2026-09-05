package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHandleStartVmNative_AckBeforeRunInstances(t *testing.T) {
	setupCloudTestDB(t)
	startVmExecuteAsync = true
	t.Cleanup(func() { startVmExecuteAsync = false })

	if _, err := db.Exec(`INSERT INTO cloud_platform_authorizations
		(id,platform_type,authorization_type,secret_id,secret_key,remark,company_id,active)
		VALUES('auth-async','aliyun','access_key','sid','skey','', 't1',1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,version,target_architectures,installed_at)
		VALUES('img-async','t1','ext1','demo','registry.example/demo','v1','[]',NOW())`); err != nil {
		t.Fatal(err)
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	var runCalls atomic.Int32
	prev := executeStartVmNativeFn
	executeStartVmNativeFn = func(accessKey, secretKey, regionID string, eventData map[string]interface{}) (map[string]interface{}, error) {
		runCalls.Add(1)
		close(entered)
		<-release
		return map[string]interface{}{"instance_id": "i-async", "status": "success"}, nil
	}
	t.Cleanup(func() { executeStartVmNativeFn = prev })

	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "runtime-userdata") {
			_ = json.NewEncoder(w).Encode(map[string]string{"content": "#!/bin/bash\necho hi"})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"platform_type": "aliyun", "region": "cn-qingdao", "image_id": "cloud-img-async"},
		})
	}))
	t.Cleanup(aiSrv.Close)
	prevAI := cfg.AIProviderBaseURL
	prevCred := cfg.CredentialServiceURL
	prevKafka := cfg.KafkaBootstrapServers
	prevBill := cfg.TaskBillURL
	cfg.AIProviderBaseURL = aiSrv.URL
	credSrv := mockCredentialTokenServer(t, "test-access-token")
	cfg.CredentialServiceURL = credSrv.URL
	cfg.KafkaBootstrapServers = ""
	cfg.TaskBillURL = ""
	t.Cleanup(func() {
		cfg.AIProviderBaseURL = prevAI
		cfg.CredentialServiceURL = prevCred
		cfg.KafkaBootstrapServers = prevKafka
		cfg.TaskBillURL = prevBill
	})

	body := `{
		"task_id":"task-async","comment_id":"cmt-async","container_image_id":"img-async",
		"region_id":"cn-qingdao","vpc_id":"vpc-1","security_group_id":"sg-1","vswitch_id":"vsw-1"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/cloud/compute/start-vm/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "u-async")
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handleStartVmNative(rec, req, "t1", "ws1")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("HTTP handler blocked on RunInstances")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "启动虚拟机请求已提交") {
		t.Fatalf("want ack message, body=%s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "vm_info") {
		t.Fatalf("ack must not wait for vm_info: %s", rec.Body.String())
	}

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("RunInstances was never scheduled")
	}
	if runCalls.Load() != 1 {
		t.Fatalf("RunInstances calls=%d", runCalls.Load())
	}
	close(release)
}

func TestFetchAvailableInstances_CachesByRegionZone(t *testing.T) {
	resetAvailableInstancesCache()
	t.Cleanup(resetAvailableInstancesCache)

	var hits atomic.Int32
	prev := aliyunAvailableInstancesFn
	aliyunAvailableInstancesFn = func(accessKey, secretKey string, f availableInstancesFilters) (availableInstancesResult, error) {
		hits.Add(1)
		return availableInstancesResult{Payload: []string{"ecs." + f.RegionID}, RequestID: "req-1"}, nil
	}
	t.Cleanup(func() { aliyunAvailableInstancesFn = prev })

	f := availableInstancesFilters{RegionID: "cn-qingdao", ZoneID: "cn-qingdao-b", Page: 1, PageSize: 10}
	p1, _, err := fetchAvailableInstances("ak", "sk", f)
	if err != nil {
		t.Fatal(err)
	}
	p2, _, err := fetchAvailableInstances("ak", "sk", f)
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("cache miss: hits=%d want 1", hits.Load())
	}
	if fmtSprint(p1) != fmtSprint(p2) {
		t.Fatalf("payload mismatch %v vs %v", p1, p2)
	}

	f2 := f
	f2.RegionID = "cn-hangzhou"
	_, _, err = fetchAvailableInstances("ak", "sk", f2)
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 {
		t.Fatalf("different region must miss cache, hits=%d", hits.Load())
	}
}

func fmtSprint(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
