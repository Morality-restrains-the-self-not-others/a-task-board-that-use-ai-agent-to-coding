package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskAiProvider/infrastructure"
)

func TestVendorContainerImagesListHasImageGroupAndStatusDisplay(t *testing.T) {
	app := testApp(t)
	vendorID := infrastructure.NextID()
	groupID := infrastructure.NextID()
	email := "list_fields_" + infrastructure.IDStr(vendorID) + "@example.com"
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		vendorID, email, "x", "ListCo", "Contact", 1, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup
		(id, name, description, created_at, updated_at, vendor_id) VALUES (?,?,?,?,?,?)`,
		groupID, "g", "", now, now, vendorID)
	if err != nil {
		t.Fatal(err)
	}
	imgID, err := app.DB.CreateContainerImage(vendorID, groupID, "1.2.3", "docker.io/library/nginx:1.2.3", []any{"amd64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id=?`, imgID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, groupID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id=?`, vendorID)
	})

	tok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(vendorID), "vendor", 3600)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/vendor/container-images/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one image")
	}
	var found map[string]any
	for _, it := range items {
		if it["id"] == infrastructure.IDStr(imgID) {
			found = it
			break
		}
	}
	if found == nil {
		t.Fatalf("image %d not in list", imgID)
	}
	if found["image_group"] != infrastructure.IDStr(groupID) {
		t.Fatalf("image_group=%v want %s", found["image_group"], infrastructure.IDStr(groupID))
	}
	if found["status_display"] != "草稿" {
		t.Fatalf("status_display=%v", found["status_display"])
	}
	if found["saas_inbound_skill_version"] != "1" {
		t.Fatalf("saas_inbound_skill_version=%v", found["saas_inbound_skill_version"])
	}
	avail, ok := found["is_ai_provider_available"].(bool)
	if !ok {
		t.Fatalf("is_ai_provider_available missing or not bool: %v", found["is_ai_provider_available"])
	}
	if avail {
		t.Fatal("expected is_ai_provider_available=false when no region runtime env")
	}
	reason, _ := found["unavailable_reason"].(string)
	if reason == "" || !strings.Contains(reason, "未设置任何区域运行环境") {
		t.Fatalf("unavailable_reason=%q want contain 未设置任何区域运行环境", reason)
	}
}

// OPT-20260717-024: vendor CSI contract — userdata_template contains name/version.
func TestVendorCloudServerImageUserdataTemplateFields(t *testing.T) {
	app := testApp(t)
	vendorID := infrastructure.NextID()
	tplID := infrastructure.NextID()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	email := "csi_tpl_" + infrastructure.IDStr(vendorID) + "@example.com"

	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`, vendorID, email, "x", "CSITplCo", "Contact", 1, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_userdatatemplate
		(id, name, version, variables, container_variables, content, auto_verify_script, is_active, created_at, updated_at, os_type)
		VALUES (?,?,?,?,?,?,?,1,?,?,?)`,
		tplID, "init-script", "1.0", "{}", "{}", "echo ok", "", now, now, "linux")
	if err != nil {
		t.Fatal(err)
	}
	csiID, err := app.DB.CreateCloudServerImage(vendorID, map[string]any{
		"platform_type": "aws", "image_name": "test-image", "image_id": "ami-123",
		"region": "us-east-1", "os_type": "linux", "os_version": "22.04",
		"architecture": "amd64", "image_type": "public",
		"userdata_template_id": infrastructure.IDStr(tplID),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcloudserverimage WHERE id=?`, csiID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_userdatatemplate WHERE id=?`, tplID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id=?`, vendorID)
	})

	tok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(vendorID), "vendor", 3600)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/vendor/cloud-server-images/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one cloud server image")
	}
	var found map[string]any
	for _, it := range items {
		if it["id"] == infrastructure.IDStr(csiID) {
			found = it
			break
		}
	}
	if found == nil {
		t.Fatalf("CSI %d not in list", csiID)
	}
	tpl, ok := found["userdata_template"]
	if !ok {
		t.Fatal("userdata_template field missing from CSI response")
	}
	tplObj, ok := tpl.(map[string]any)
	if !ok {
		t.Fatalf("userdata_template is not an object: %T %v", tpl, tpl)
	}
	if tplObj["name"] != "init-script" {
		t.Fatalf("userdata_template.name=%v want init-script", tplObj["name"])
	}
	if tplObj["version"] != "1.0" {
		t.Fatalf("userdata_template.version=%v want 1.0", tplObj["version"])
	}
	if idStr, ok := tplObj["id"].(string); !ok || idStr == "" {
		t.Fatalf("userdata_template.id missing or empty: %v", tplObj["id"])
	}
}

// TestVendorDevelopmentCatalogHTTPContract asserts the HTTP layer returns all expected
// public-catalog fields (name / target_architectures / vendor.company_name / status_display).
// This guards against the handler regressing to a stub after store-level refactors.
func TestVendorDevelopmentCatalogHTTPContract(t *testing.T) {
	app := testApp(t)
	vendorID := infrastructure.NextID()
	groupID := infrastructure.NextID()
	saasUserID := infrastructure.NextID()
	email := "devcat_" + infrastructure.IDStr(vendorID) + "@example.com"
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")

	// Seed vendor
	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at, saas_user_id)
		VALUES (?,?,?,?,?,1,?,?,?)`, vendorID, email, "x", "DevCatCo", "Contact", now, now, saasUserID)
	if err != nil {
		t.Fatal(err)
	}
	// Seed group
	_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup
		(id, name, description, created_at, updated_at, vendor_id) VALUES (?,?,?,?,?,?)`,
		groupID, "dev-group", "dev desc", now, now, vendorID)
	if err != nil {
		t.Fatal(err)
	}
	// Seed draft image
	imgID, err := app.DB.CreateContainerImage(vendorID, groupID, "1.0", "docker.io/dev/img:1.0", []any{"arm64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	// Seed userdata template + CSI + association so runtime_environments is non-empty
	tplID := infrastructure.NextID()
	_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_userdatatemplate
		(id, name, version, os_type, variables, container_variables, content, auto_verify_script, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,1,?,?)`,
		tplID, "boot", "1", "linux", "[]", "[]", "#", "", now, now)
	if err != nil {
		t.Fatal(err)
	}
	csiID := infrastructure.NextID()
	_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_vendorcloudserverimage
		(id, vendor_id, platform_type, image_name, image_id, region, os_type, os_version, architecture, image_type,
		 is_active, default_instance_type_id, default_instance_type_label, base_cpu_cores, base_memory_gib,
		 userdata_template_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,1,?,?,?,?,?,?,?)`,
		csiID, vendorID, "aliyun", "dev-img", "m-dev", "cn-hangzhou", "linux", "24.04", "arm64", "system",
		"ecs.c6.large", "ecs.c6.large (2vCPU 4GiB)", 2, 4, tplID, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_containercloudserverassociation
		(id, platform_type, region, cloud_server_image_id, container_image_id)
		VALUES (?,?,?,?,?)`, infrastructure.NextID(), "aliyun", "cn-hangzhou", csiID, imgID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_containercloudserverassociation WHERE container_image_id=?`, imgID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcloudserverimage WHERE id=?`, csiID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_userdatatemplate WHERE id=?`, tplID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id=?`, imgID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, groupID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id=?`, vendorID)
	})

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/api/public/vendor-development-catalog/?saas_user_id="+infrastructure.IDStr(saasUserID), nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least one development image")
	}
	item := items[0]
	// Assert required public-catalog fields exist
	if item["name"] == nil || item["name"] == "" {
		t.Fatalf("name missing: %v", item["name"])
	}
	if item["target_architectures"] == nil {
		t.Fatal("target_architectures missing")
	}
	arch, _ := item["target_architectures"].([]any)
	if len(arch) == 0 {
		t.Fatal("target_architectures empty")
	}
	if item["status_display"] == nil || item["status_display"] == "" {
		t.Fatalf("status_display missing: %v", item["status_display"])
	}
	vendor, ok := item["vendor"].(map[string]any)
	if !ok {
		t.Fatalf("vendor missing or not object: %v", item["vendor"])
	}
	if vendor["company_name"] == nil || vendor["company_name"] == "" {
		t.Fatalf("vendor.company_name missing: %v", vendor["company_name"])
	}
	// Verify is_development marker
	if item["is_development"] != true {
		t.Fatalf("is_development=%v want true", item["is_development"])
	}
	// Runtime environments should be non-nil (at least an empty array)
	if item["runtime_environments"] == nil {
		t.Fatal("runtime_environments missing (expected at least empty array)")
	}
	envs, _ := item["runtime_environments"].([]any)
	if len(envs) == 0 {
		t.Fatal("runtime_environments empty — expected at least one env")
	}
	env := envs[0].(map[string]any)
	if env["platform_type_display"] == nil || env["platform_type_display"] == "" {
		t.Fatal("runtime_environments[0].platform_type_display missing")
	}
}
