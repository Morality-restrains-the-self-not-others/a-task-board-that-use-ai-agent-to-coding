package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveCloudServerImageIDExplicit(t *testing.T) {
	imageID, region, msg := resolveCloudServerImageID("t1", "img-explicit", "", "aliyun", "cn-hangzhou", "")
	if msg != "" {
		t.Fatalf("unexpected err: %s", msg)
	}
	if imageID != "img-explicit" || region != "cn-hangzhou" {
		t.Fatalf("got image=%s region=%s", imageID, region)
	}
}

func TestResolveCloudServerImageIDFromMarketplace(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,version,target_architectures,installed_at)
		VALUES('img1','t1','ext1','demo','registry.example/demo','v1','[]',NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"platform_type": "aliyun", "region": "cn-shanghai", "image_id": "cloud-1"},
			{"platform_type": "tencent", "region": "ap-shanghai", "image_id": "cloud-2"},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	imageID, region, msg := resolveCloudServerImageID("t1", "", "img1", "aliyun", "cn-shanghai", "")
	if msg != "" {
		t.Fatalf("unexpected err: %s", msg)
	}
	if imageID != "cloud-1" || region != "cn-shanghai" {
		t.Fatalf("got image=%s region=%s", imageID, region)
	}
}

func TestResolveCloudServerImageIDArchitectureMismatch(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,version,target_architectures,installed_at)
		VALUES('img1','t1','ext1','demo','registry.example/demo','v1','[]',NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"platform_type": "aliyun", "region": "cn-hongkong", "image_id": "ubuntu-arm64", "architecture": "arm64"},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	_, _, msg := resolveCloudServerImageID("t1", "", "img1", "aliyun", "cn-hongkong", "x86_64")
	if msg == "" || !strings.Contains(msg, "架构不匹配") {
		t.Fatalf("expected architecture mismatch, got %q", msg)
	}
}
