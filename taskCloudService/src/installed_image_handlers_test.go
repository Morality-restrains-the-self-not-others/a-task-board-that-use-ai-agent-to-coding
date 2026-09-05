package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInstalledImageCatalogReturnsArray(t *testing.T) {
	setupCloudTestDB(t)
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/public/catalog/" {
			t.Fatalf("unexpected ai path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "img-1", "name": "demo"},
		})
	}))
	defer aiSrv.Close()

	cfg.AIProviderBaseURL = aiSrv.URL

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/installed-images/catalog/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "user-1")
	rec := httptest.NewRecorder()
	handleInstalledImageCatalog(rec, req, "t1")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `"installed_images"`) {
		t.Fatalf("expected plain array, got wrapped payload: %s", body)
	}
	if !strings.Contains(body, `"id":"img-1"`) {
		t.Fatalf("expected catalog item: %s", body)
	}
}

func TestInstalledImageDevCatalogReturnsArray(t *testing.T) {
	setupCloudTestDB(t)
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/public/vendor-development-catalog/" {
			t.Fatalf("unexpected ai path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("saas_user_id") != "user-42" {
			t.Fatalf("unexpected saas_user_id: %s", r.URL.Query().Get("saas_user_id"))
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "dev-1", "name": "draft-img", "status": "draft"},
		})
	}))
	defer aiSrv.Close()

	cfg.AIProviderBaseURL = aiSrv.URL

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/installed-images/dev-catalog/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "user-42")
	rec := httptest.NewRecorder()
	handleInstalledImageDevCatalog(rec, req, "t1")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `"installed_images"`) {
		t.Fatalf("expected plain array, got wrapped payload: %s", body)
	}
	if !strings.Contains(body, `"id":"dev-1"`) {
		t.Fatalf("expected dev catalog item: %s", body)
	}
}

func TestInstalledImageListFromGoStore(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,target_architectures,is_dev_mode)
		VALUES('859671040643174400','t1','859670982529273856','trae0630','registry.example/img','[]',1)`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/installed-images/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "user-1")
	rec := httptest.NewRecorder()
	handleInstalledImageCollection(rec, req, "t1")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"name":"trae0630"`) {
		t.Fatalf("expected installed image: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"installed_images"`) {
		t.Fatalf("expected plain array, got wrapped payload: %s", rec.Body.String())
	}
}

func TestInstalledImageDetailGet(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,target_architectures,is_dev_mode)
		VALUES('859671040643174400','t1','859670982529273856','trae0630','registry.example/img','[]',1)`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/installed-images/859671040643174400/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "user-1")
	rec := httptest.NewRecorder()
	handleInstalledImageDetail(rec, req, "t1", "859671040643174400")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"name":"trae0630"`) {
		t.Fatalf("expected installed image detail: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"id":"859671040643174400"`) {
		t.Fatalf("expected image id in detail: %s", rec.Body.String())
	}
}

func TestInstalledImageCatalogMarksIsInstalled(t *testing.T) {
	setupCloudTestDB(t)
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "ext-9", "name": "pub"},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,target_architectures,is_dev_mode)
		VALUES('local-1','t1','ext-9','installed','registry.example/img','[]',0)`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/installed-images/catalog/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleInstalledImageCatalog(rec, req, "t1")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"is_installed":true`) {
		t.Fatalf("expected is_installed true: %s", rec.Body.String())
	}
}

func TestImportInstalledImages(t *testing.T) {
	setupCloudTestDB(t)
	count, err := importInstalledImages([]map[string]interface{}{
		{
			"id": "1001", "tenant_id": "t1", "external_image_id": "9001",
			"name": "img-a", "image_url": "registry.example/a", "target_architectures": []interface{}{"x86_64"},
		},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
	rows, err := listInstalledImages("t1")
	if err != nil || len(rows) != 1 || rows[0].Name != "img-a" {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
}

func TestImportInstalledImagesCopiesAutoRunSteps(t *testing.T) {
	setupCloudTestDB(t)
	count, err := importInstalledImages([]map[string]interface{}{
		{
			"id": "1002", "tenant_id": "t1", "external_image_id": "9002",
			"name": "img-b", "image_url": "registry.example/b", "target_architectures": []interface{}{"x86_64"},
			"auto_run_steps_md": "# from catalog", "auto_run_steps_extract_status": "ok", "auto_run_steps_digest": "sha256:x",
		},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
	rows, err := listInstalledImages("t1")
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	if rows[0].AutoRunStepsMd != "# from catalog" || rows[0].AutoRunStepsExtractStatus != "ok" {
		t.Fatalf("auto_run fields not copied: %+v", rows[0])
	}
	js := installedImageToJSON(rows[0])
	if js["auto_run_steps_md"] != "# from catalog" {
		t.Fatalf("json missing md: %#v", js)
	}
}

func TestInstalledImageDuplicateInstallRejected(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	store.putMember("t1", "admin-1", "mem-1", true)
	_ = startSaasInternalMock(t, store)
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,target_architectures,is_dev_mode)
		VALUES('img-dup','t1','ext-dup','existing','registry.example/dup','[]',0)`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "ext-dup", "name": "dup-img", "image_url": "registry.example/dup"},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	body := `{"external_image_id":"ext-dup"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "admin-1")
	rec := httptest.NewRecorder()
	handleInstalledImageCollection(rec, req, "t1")

	if rec.Code != 400 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "已安装") {
		t.Fatalf("expected duplicate message: %s", rec.Body.String())
	}
}

func TestInstalledImageUniqueConstraintOnStore(t *testing.T) {
	setupCloudTestDB(t)
	img := TenantInstalledImage{
		ID:              "u1",
		TenantID:        "t1",
		ExternalImageID: "ext-uniq",
		Name:            "first",
		ImageURL:        "registry.example/first",
	}
	if err := createInstalledImage(img); err != nil {
		t.Fatalf("first create: %v", err)
	}
	dup := TenantInstalledImage{
		ID:              "u2",
		TenantID:        "t1",
		ExternalImageID: "ext-uniq",
		Name:            "second",
		ImageURL:        "registry.example/second",
	}
	if err := createInstalledImage(dup); err == nil {
		t.Fatalf("expected unique constraint error")
	}
}

func TestResolveTargetArchitecturesHandler(t *testing.T) {
	setupCloudTestDB(t)

	registrySrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/manifests/") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"manifests": []map[string]interface{}{
					{"digest": "sha256:amd64", "size": 123, "platform": map[string]interface{}{"architecture": "amd64"}},
					{"digest": "sha256:arm64", "size": 456, "platform": map[string]interface{}{"architecture": "arm64"}},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer registrySrv.Close()

	orig := manifestHTTP
	manifestHTTP = registrySrv.Client()
	t.Cleanup(func() { manifestHTTP = orig })

	hostPort := strings.TrimPrefix(registrySrv.URL, "https://")
	body := `{"image_url":"` + hostPort + `/library/nginx:latest"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/resolve-target-architectures/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "user-1")
	rec := httptest.NewRecorder()
	handleInstalledImageResolveTargetArchitectures(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"x86_64"`) || !strings.Contains(rec.Body.String(), `"arm64"`) {
		t.Fatalf("expected architectures: %s", rec.Body.String())
	}
}

func TestResolveTargetArchitecturesRequiresImageURL(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/resolve-target-architectures/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInstalledImageResolveTargetArchitectures(rec, req)
	if rec.Code != 400 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInstalledImageDevCatalogServiceUnavailable(t *testing.T) {
	setupCloudTestDB(t)
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/installed-images/dev-catalog/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "user-1")
	req.Header.Set("X-Trace-Id", "trace-dev-503")
	rec := httptest.NewRecorder()
	handleInstalledImageDevCatalog(rec, req, "t1")

	if rec.Code != 503 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "detail") || !strings.Contains(rec.Body.String(), "trace_id") {
		t.Fatalf("expected detail+trace_id: %s", rec.Body.String())
	}
}

func TestInstalledImageInstallRequiresAdmin(t *testing.T) {
	setupCloudTestDB(t)

	body := `{"external_image_id":"ext-new"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "member-1")
	rec := httptest.NewRecorder()
	handleInstalledImageCollection(rec, req, "t1")

	if rec.Code != 403 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInstalledImageListRequiresAuth(t *testing.T) {
	setupCloudTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/installed-images/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleInstalledImages(rec, req)
	if rec.Code != 401 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestInstalledImageInstallDevelopmentImage(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	store.putMember("t1", "admin-1", "mem-1", true)
	_ = startSaasInternalMock(t, store)

	const vendorID = "859457200064331776"
	const containerID = "859670982529273856"
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/public/unsubmitted-image/" {
			t.Fatalf("unexpected ai path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("vendor_id") != vendorID || r.URL.Query().Get("container_id") != containerID {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        containerID,
			"name":      "dev-img",
			"image_url": "registry.example/dev:latest",
			"vendor": map[string]interface{}{
				"id":           vendorID,
				"company_name": "Demo Vendor",
			},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	body := `{"vendor_id":"` + vendorID + `","container_id":"` + containerID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "admin-1")
	rec := httptest.NewRecorder()
	handleInstalledImageCollection(rec, req, "t1")

	if rec.Code != 201 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"external_image_id":"`+containerID+`"`) {
		t.Fatalf("expected installed dev image: %s", rec.Body.String())
	}
}

// TestCreateInstalledImageWithZeroTime verifies that createInstalledImage
// handles zero-time InstalledAt correctly via COALESCE(NULLIF(?, ”), CURRENT_TIMESTAMP)
// and does not raise MySQL Error 1292.
func TestCreateInstalledImageWithZeroTime(t *testing.T) {
	setupCloudTestDB(t)
	img := TenantInstalledImage{
		ID:              "zero-time-test",
		TenantID:        "t1",
		ExternalImageID: "ext-zero",
		Name:            "zero-time-img",
		ImageURL:        "registry.example/zero",
		// InstalledAt left as zero value (time.Time{}), simulating the pre-fix bug.
	}
	err := createInstalledImage(img)
	if err != nil {
		t.Fatalf("createInstalledImage with zero InstalledAt should succeed after NULLIF fix: %v", err)
	}
	// Verify the record was created and installed_at is a valid timestamp.
	row := db.QueryRow(`SELECT installed_at FROM cloud_tenant_installed_images WHERE id='zero-time-test'`)
	var installedAt string
	if err := row.Scan(&installedAt); err != nil {
		t.Fatalf("read back installed_at: %v", err)
	}
	if installedAt == "" || installedAt == "0000-00-00 00:00:00" {
		t.Fatalf("installed_at should be CURRENT_TIMESTAMP, got %q", installedAt)
	}
	t.Logf("installed_at = %s", installedAt)
}

// TestImportInstalledImagesBlankDatetime verifies that importInstalledImages
// handles empty installed_at strings via COALESCE(NULLIF(?, ”), CURRENT_TIMESTAMP).
func TestImportInstalledImagesBlankDatetime(t *testing.T) {
	setupCloudTestDB(t)
	count, err := importInstalledImages([]map[string]interface{}{
		{
			"id": "blank-dt-1", "tenant_id": "t1", "external_image_id": "9001",
			"name": "blank-dt-img", "image_url": "registry.example/bd",
			"target_architectures": []interface{}{"x86_64"},
			"installed_at":         "", // empty string — pre-fix would cause Error 1292
		},
	})
	if err != nil {
		t.Fatalf("import with blank installed_at should succeed: %v", err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
	var installedAt string
	if err := db.QueryRow(`SELECT installed_at FROM cloud_tenant_installed_images WHERE id='blank-dt-1'`).Scan(&installedAt); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if installedAt == "" {
		t.Fatalf("expected CURRENT_TIMESTAMP fallback")
	}
	t.Logf("installed_at = %s", installedAt)
}

// TestInstalledImageInstallStoresSaasInboundSkillVersion 回归 OPT-20260820-026：
// 安装镜像时把厂商目录的 saas_inbound_skill_version（v1）归一化为 1 落库，
// 列表/详情 JSON 暴露 saas_inbound_skill_version，供 start-vm 注入容器 env。
func TestInstalledImageInstallStoresSaasInboundSkillVersion(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	store.putMember("t1", "admin-1", "mem-1", true)
	_ = startSaasInternalMock(t, store)

	const vendorID = "859457200064331776"
	const containerID = "859670982529273856"
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/public/unsubmitted-image/" {
			t.Fatalf("unexpected ai path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                         containerID,
			"name":                       "skill-img",
			"image_url":                  "registry.example/skill:latest",
			"saas_inbound_skill_version": "v1",
			"vendor": map[string]interface{}{
				"id":           vendorID,
				"company_name": "Demo Vendor",
			},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	body := `{"vendor_id":"` + vendorID + `","container_id":"` + containerID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "admin-1")
	rec := httptest.NewRecorder()
	handleInstalledImageCollection(rec, req, "t1")

	if rec.Code != 201 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"saas_inbound_skill_version":"1"`) {
		t.Fatalf("install response must expose normalized saas_inbound_skill_version=1: %s", rec.Body.String())
	}
	// 读回库中记录，确认归一化值已落库（记录 id 为 snowflake，按 external_image_id 反查）。
	var stored string
	if err := db.QueryRow(
		`SELECT COALESCE(saas_inbound_skill_version,'') FROM cloud_tenant_installed_images
		 WHERE tenant_id='t1' AND external_image_id=?`, containerID,
	).Scan(&stored); err != nil {
		t.Fatalf("read back saas_inbound_skill_version: %v", err)
	}
	if stored != "1" {
		t.Fatalf("stored saas_inbound_skill_version=%q want 1", stored)
	}
}

// TestInstalledImageJSONExposesSaasInboundSkillVersion 覆盖 JSON 序列化暴露。
func TestInstalledImageJSONExposesSaasInboundSkillVersion(t *testing.T) {
	img := TenantInstalledImage{
		ID:                      "img-json",
		TenantID:                "t1",
		ExternalImageID:         "ext",
		Name:                    "json-img",
		ImageURL:                "registry.example/json",
		SaasInboundSkillVersion: "2",
	}
	out := installedImageToJSON(img)
	if got, _ := out["saas_inbound_skill_version"].(string); got != "2" {
		t.Fatalf("saas_inbound_skill_version=%q want 2", got)
	}
	img2 := TenantInstalledImage{ID: "img-empty", Name: "empty"}
	if got, _ := installedImageToJSON(img2)["saas_inbound_skill_version"].(string); got != "" {
		t.Fatalf("empty saas_inbound_skill_version should be empty, got=%q", got)
	}
}
