package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

func TestParseAvailableInstancesFilters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/cloud/available-instances/?region_id=cn-hongkong&zone_id=cn-hongkong-d&Cores=2&Memory=4&IoOptimized=optimized&SystemDiskCategory=cloud_essd&DataDiskCategory=cloud_essd&spot_strategy=SpotAsPriceGo&NetworkCategory=vpc&DestinationResource=InstanceType&ResourceType=instance",
		nil)

	f := parseAvailableInstancesFilters(req, "cn-hongkong", "cn-hongkong-d")
	if f.Cores != "2" || f.Memory != "4" {
		t.Fatalf("unexpected cores/memory: %+v", f)
	}
	if f.IoOptimized != "optimized" || f.SystemDiskCategory != "cloud_essd" {
		t.Fatalf("unexpected disk/io filters: %+v", f)
	}
	if f.SpotStrategy != "SpotAsPriceGo" || f.NetworkCategory != "vpc" {
		t.Fatalf("unexpected spot/network filters: %+v", f)
	}
	if f.InstanceChargeType != "PostPaid" {
		t.Fatalf("expected default PostPaid, got %q", f.InstanceChargeType)
	}
}

func TestApplyAvailableInstancesFilters(t *testing.T) {
	req := &ecsclient.DescribeAvailableResourceRequest{
		RegionId:            dara.String("cn-hongkong"),
		ResourceType:        dara.String("instance"),
		DestinationResource: dara.String("InstanceType"),
	}
	f := availableInstancesFilters{
		Cores:              "2",
		Memory:             "4",
		IoOptimized:        "optimized",
		SystemDiskCategory: "cloud_essd",
		DataDiskCategory:   "cloud_essd",
		NetworkCategory:    "vpc",
		SpotStrategy:       "SpotAsPriceGo",
		InstanceChargeType: "PostPaid",
	}
	applyAvailableInstancesFilters(req, f)

	if req.Cores == nil || *req.Cores != 2 {
		t.Fatalf("expected Cores=2, got %+v", req.Cores)
	}
	if req.Memory == nil || *req.Memory != 4 {
		t.Fatalf("expected Memory=4, got %+v", req.Memory)
	}
	if req.IoOptimized == nil || *req.IoOptimized != "optimized" {
		t.Fatalf("expected IoOptimized=optimized, got %+v", req.IoOptimized)
	}
	if req.SpotStrategy == nil || *req.SpotStrategy != "SpotAsPriceGo" {
		t.Fatalf("expected SpotStrategy=SpotAsPriceGo, got %+v", req.SpotStrategy)
	}
}

func TestInstanceTypeSpecCache(t *testing.T) {
	resetInstanceTypeSpecCacheForTest()
	spec := instanceTypeSpec{cpuCores: 2, memoryGB: 4}
	setCachedInstanceTypeSpec("ecs.test.large", spec)

	cached, ok := getCachedInstanceTypeSpec("ecs.test.large")
	if !ok || cached.cpuCores != 2 || cached.memoryGB != 4 {
		t.Fatalf("expected cached spec, got %+v ok=%v", cached, ok)
	}
}

func TestBuildPaginatedAvailableInstancesResponse(t *testing.T) {
	ids := make([]string, 0, 35)
	for i := 0; i < 35; i++ {
		ids = append(ids, "ecs.test.large")
	}
	payload := buildPaginatedAvailableInstances(ids, 1, 10)
	pagination, ok := payload["pagination"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected pagination object: %#v", payload)
	}
	if pagination["total_instances"] != 35 || pagination["total_pages"] != 4 {
		t.Fatalf("unexpected pagination: %#v", pagination)
	}
}

func TestTenantCloudAvailableInstancesMockIncludesCpuMemory(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "850256677331562496", "862031128628060160")
	t.Setenv("USE_IN_MEMORY_CLOUD", "true")

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/850256677331562496/cloud-platform/862031128628060160/cloud/available-instances/?platform_type=aliyun&region_id=cn-hongkong&zone_id=cn-hongkong-d&Cores=2&Memory=4&IoOptimized=optimized&SystemDiskCategory=cloud_essd&DataDiskCategory=cloud_essd&spot_strategy=SpotAsPriceGo&NetworkCategory=vpc&DestinationResource=InstanceType&ResourceType=instance",
		nil)
	req.Header.Set("X-Auth-Tenant-Id", "850256677331562496")
	req.Header.Set("X-User-Id", "test-user")
	rec := httptest.NewRecorder()

	handleCloudPlatformRoutes(rec, req, "850256677331562496", "862031128628060160", "available-instances/")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `"cpu_cores":0`) {
		t.Fatalf("mock response should not contain zero cpu_cores: %s", body)
	}
	if !strings.Contains(body, `"cpu_cores":2`) || !strings.Contains(body, `"memory_gb":4`) {
		t.Fatalf("expected non-zero cpu/memory in response: %s", body)
	}
}
