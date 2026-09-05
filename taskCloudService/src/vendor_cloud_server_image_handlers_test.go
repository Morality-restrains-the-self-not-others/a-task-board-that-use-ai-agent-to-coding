package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestVendorCloudRegionsSuccess(t *testing.T) {
	os.Setenv("USE_IN_MEMORY_CLOUD", "true")
	t.Setenv("DJANGO_SECRET_KEY", "test-vendor-jwt-secret")
	setupCloudTestDB(t)

	vendorID := "850256676127797248"
	token := makeTestVendorJWT(vendorID)

	createReq := httptest.NewRequest(http.MethodPost, "/api/vendor/cloud-platform-credentials/", strings.NewReader(`{
		"platform_type":"aliyun","secret_id":"LTAI_TEST","secret_key":"SK_TEST","remark":"regions test"
	}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handleVendorCloudCredentialRoutes(createRec, createReq)
	if createRec.Code != 201 {
		t.Fatalf("create credential status=%d body=%s", createRec.Code, createRec.Body.String())
	}

	regionsReq := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-server-images/regions/?platform_type=aliyun", nil)
	regionsReq.Header.Set("Authorization", "Bearer "+token)
	regionsRec := httptest.NewRecorder()
	handleVendorCloudServerImageRoutes(regionsRec, regionsReq)
	if regionsRec.Code != 200 {
		t.Fatalf("regions status=%d body=%s", regionsRec.Code, regionsRec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(regionsRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body["status"])
	}
	regions, ok := body["regions"].([]interface{})
	if !ok || len(regions) == 0 {
		t.Fatalf("expected non-empty regions: %v", body["regions"])
	}
	first, ok := regions[0].(map[string]interface{})
	if !ok || first["id"] == "" || first["name"] == "" {
		t.Fatalf("expected region id/name: %v", regions[0])
	}
}

func TestVendorCloudRegionsMissingCredential(t *testing.T) {
	t.Setenv("DJANGO_SECRET_KEY", "test-vendor-jwt-secret")
	setupCloudTestDB(t)

	token := makeTestVendorJWT("999888777666555444")
	regionsReq := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-server-images/regions/?platform_type=aliyun", nil)
	regionsReq.Header.Set("Authorization", "Bearer "+token)
	regionsRec := httptest.NewRecorder()
	handleVendorCloudServerImageRoutes(regionsRec, regionsReq)
	if regionsRec.Code != 400 {
		t.Fatalf("expected 400, got %d body=%s", regionsRec.Code, regionsRec.Body.String())
	}
	if !strings.Contains(regionsRec.Body.String(), "vendor_cloud_credential_missing") {
		t.Fatalf("expected credential missing code: %s", regionsRec.Body.String())
	}
}

func TestVendorCloudRegionsMissingPlatformType(t *testing.T) {
	t.Setenv("DJANGO_SECRET_KEY", "test-vendor-jwt-secret")
	token := makeTestVendorJWT("1")
	regionsReq := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-server-images/regions/", nil)
	regionsReq.Header.Set("Authorization", "Bearer "+token)
	regionsRec := httptest.NewRecorder()
	handleVendorCloudServerImageRoutes(regionsRec, regionsReq)
	if regionsRec.Code != 400 {
		t.Fatalf("expected 400, got %d", regionsRec.Code)
	}
	if !strings.Contains(regionsRec.Body.String(), "platform_type") {
		t.Fatalf("expected platform_type error: %s", regionsRec.Body.String())
	}
}

func TestVendorCloudImagesSuccess(t *testing.T) {
	os.Setenv("USE_IN_MEMORY_CLOUD", "true")
	t.Setenv("DJANGO_SECRET_KEY", "test-vendor-jwt-secret")
	setupCloudTestDB(t)

	vendorID := "850256676127797248"
	token := makeTestVendorJWT(vendorID)

	createReq := httptest.NewRequest(http.MethodPost, "/api/vendor/cloud-platform-credentials/", strings.NewReader(`{
		"platform_type":"aliyun","secret_id":"LTAI_IMG","secret_key":"SK_IMG","remark":"images test"
	}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handleVendorCloudCredentialRoutes(createRec, createReq)
	if createRec.Code != 201 {
		t.Fatalf("create credential status=%d body=%s", createRec.Code, createRec.Body.String())
	}

	imagesReq := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-server-images/images/?platform_type=aliyun&region_id=cn-hangzhou", nil)
	imagesReq.Header.Set("Authorization", "Bearer "+token)
	imagesRec := httptest.NewRecorder()
	handleVendorCloudServerImageRoutes(imagesRec, imagesReq)
	if imagesRec.Code != 200 {
		t.Fatalf("images status=%d body=%s", imagesRec.Code, imagesRec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(imagesRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body["status"])
	}
	images, ok := body["images"].([]interface{})
	if !ok || len(images) == 0 {
		t.Fatalf("expected non-empty images: %v", body["images"])
	}
}

func TestVendorCloudInstanceTypesSuccess(t *testing.T) {
	os.Setenv("USE_IN_MEMORY_CLOUD", "true")
	t.Setenv("DJANGO_SECRET_KEY", "test-vendor-jwt-secret")
	setupCloudTestDB(t)

	vendorID := "850256676127797248"
	token := makeTestVendorJWT(vendorID)

	createReq := httptest.NewRequest(http.MethodPost, "/api/vendor/cloud-platform-credentials/", strings.NewReader(`{
		"platform_type":"aliyun","secret_id":"LTAI_IT","secret_key":"SK_IT","remark":"instance types test"
	}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handleVendorCloudCredentialRoutes(createRec, createReq)
	if createRec.Code != 201 {
		t.Fatalf("create credential status=%d body=%s", createRec.Code, createRec.Body.String())
	}

	itReq := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-server-images/instance-types/?platform_type=aliyun&region_id=cn-hangzhou&image_id=m-ubuntu-001", nil)
	itReq.Header.Set("Authorization", "Bearer "+token)
	itRec := httptest.NewRecorder()
	handleVendorCloudServerImageRoutes(itRec, itReq)
	if itRec.Code != 200 {
		t.Fatalf("instance-types status=%d body=%s", itRec.Code, itRec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(itRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body["status"])
	}
	types, ok := body["instance_types"].([]interface{})
	if !ok || len(types) == 0 {
		t.Fatalf("expected non-empty instance_types: %v", body["instance_types"])
	}
	first, ok := types[0].(map[string]interface{})
	if !ok || first["instance_type_id"] == "" {
		t.Fatalf("expected instance_type_id: %v", types[0])
	}
}

func TestVendorCloudInstanceTypesMissingImageID(t *testing.T) {
	t.Setenv("DJANGO_SECRET_KEY", "test-vendor-jwt-secret")
	setupCloudTestDB(t)

	vendorID := "850256676127797248"
	token := makeTestVendorJWT(vendorID)
	createReq := httptest.NewRequest(http.MethodPost, "/api/vendor/cloud-platform-credentials/", strings.NewReader(`{
		"platform_type":"aliyun","secret_id":"LTAI_IT2","secret_key":"SK_IT2","remark":"it missing test"
	}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	handleVendorCloudCredentialRoutes(createRec, createReq)
	if createRec.Code != 201 {
		t.Fatalf("create credential status=%d", createRec.Code)
	}

	itReq := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-server-images/instance-types/?platform_type=aliyun&region_id=cn-hangzhou", nil)
	itReq.Header.Set("Authorization", "Bearer "+token)
	itRec := httptest.NewRecorder()
	handleVendorCloudServerImageRoutes(itRec, itReq)
	if itRec.Code != 400 {
		t.Fatalf("expected 400, got %d body=%s", itRec.Code, itRec.Body.String())
	}
	if !strings.Contains(itRec.Body.String(), "image_id") {
		t.Fatalf("expected image_id error: %s", itRec.Body.String())
	}
}

func TestFilterChinaRegions(t *testing.T) {
	regions := []map[string]string{
		{"id": "cn-hangzhou", "name": "华东1（杭州）"},
		{"id": "cn-beijing", "name": "华北2（北京）"},
		{"id": "cn-hongkong", "name": "中国香港"},
		{"id": "ap-southeast-1", "name": "新加坡"},
		{"id": "us-west-1", "name": "美国（硅谷）"},
		{"id": "eu-central-1", "name": "德国（法兰克福）"},
		{"id": "me-east-1", "name": "阿联酋（迪拜）"},
		{"id": "cn-shanghai", "name": "华东2（上海）"},
	}
	filtered := filterChinaRegions(regions)
	if len(filtered) != 4 {
		t.Fatalf("expected 4 china regions, got %d: %v", len(filtered), filtered)
	}
	for _, r := range filtered {
		id := r["id"]
		if !strings.HasPrefix(id, "cn-") {
			t.Fatalf("expected cn- prefix, got %s", id)
		}
	}
	ids := map[string]bool{}
	for _, r := range filtered {
		ids[r["id"]] = true
	}
	for _, expectedID := range []string{"cn-hangzhou", "cn-beijing", "cn-hongkong", "cn-shanghai"} {
		if !ids[expectedID] {
			t.Fatalf("expected region %s to be present, got %v", expectedID, ids)
		}
	}
}

func TestFilterChinaRegionsWithRegionIDKey(t *testing.T) {
	regions := []map[string]string{
		{"region_id": "cn-hangzhou", "region_name": "华东1（杭州）"},
		{"region_id": "ap-southeast-1", "region_name": "新加坡"},
	}
	filtered := filterChinaRegions(regions)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 china region, got %d: %v", len(filtered), filtered)
	}
	if filtered[0]["region_id"] != "cn-hangzhou" {
		t.Fatalf("expected cn-hangzhou, got %s", filtered[0]["region_id"])
	}
}

func TestFilterChinaRegionsEmpty(t *testing.T) {
	regions := []map[string]string{
		{"id": "ap-southeast-1", "name": "新加坡"},
		{"id": "us-west-1", "name": "美国（硅谷）"},
	}
	filtered := filterChinaRegions(regions)
	if len(filtered) != 0 {
		t.Fatalf("expected 0 regions, got %d: %v", len(filtered), filtered)
	}
	if filterChinaRegions(nil) != nil {
		t.Fatalf("expected nil for nil input")
	}
}
