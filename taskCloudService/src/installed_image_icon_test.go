package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCatalogIconURL(t *testing.T) {
	t.Parallel()
	if got := catalogIconURL(map[string]interface{}{"icon_url": "/a"}); got != "/a" {
		t.Fatalf("top-level icon_url = %q", got)
	}
	nested := map[string]interface{}{
		"image_group": map[string]interface{}{"icon_url": "/nested"},
	}
	if got := catalogIconURL(nested); got != "/nested" {
		t.Fatalf("nested icon_url = %q", got)
	}
	if got := catalogIconURL(map[string]interface{}{}); got != "" {
		t.Fatalf("empty payload = %q", got)
	}
}

func TestInstalledImageJSONIncludesIconURL(t *testing.T) {
	t.Parallel()
	img := TenantInstalledImage{
		ID:          "img-1",
		TenantID:    "t1",
		Name:        "trae-agent",
		Version:     "x86_64-latest",
		IconURL:     "/api/ai-provider/public-image-groups/20/icon?h=abc",
		InstalledAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(installedImageToJSON(img))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, `"icon_url":"/api/ai-provider/public-image-groups/20/icon?h=abc"`) {
		t.Fatalf("expected icon_url in JSON, got %s", body)
	}

	zero := TenantInstalledImage{ID: "img-2", TenantID: "t1", Name: "no-icon"}
	rawZero, _ := json.Marshal(installedImageToJSON(zero))
	if !strings.Contains(string(rawZero), `"icon_url":""`) {
		t.Fatalf("expected empty icon_url when snapshot missing, got %s", rawZero)
	}
}

func TestInstalledImageInstallSnapshotsIconURL(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	store.putMember("t1", "admin-1", "mem-1", true)
	_ = startSaasInternalMock(t, store)

	const extID = "859670982529273856"
	const iconURL = "/api/ai-provider/public-image-groups/20/icon?h=abc"
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/public/catalog/" {
			t.Fatalf("unexpected ai path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":        extID,
				"name":      "trae-agent",
				"version":   "x86_64-latest",
				"image_url": "registry.example/trae-agent",
				"icon_url":  iconURL,
				"image_group": map[string]interface{}{
					"id":       "20",
					"icon_url": iconURL,
				},
				"vendor": map[string]interface{}{
					"id":           "vendor-1",
					"company_name": "Demo Vendor",
				},
			},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	body := `{"external_image_id":"` + extID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "admin-1")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleInstalledImageCollection(rec, req, "t1")
	if rec.Code != 201 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if created["icon_url"] != iconURL {
		t.Fatalf("install response icon_url=%v want %s", created["icon_url"], iconURL)
	}

	var stored string
	err := db.QueryRow(`SELECT icon_url FROM cloud_tenant_installed_images WHERE external_image_id=?`, extID).Scan(&stored)
	if err != nil {
		t.Fatalf("read back icon_url: %v", err)
	}
	if stored != iconURL {
		t.Fatalf("stored icon_url=%q want %s", stored, iconURL)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/installed-images/", nil)
	listRec := httptest.NewRecorder()
	handleInstalledImageCollection(listRec, listReq, "t1")
	if listRec.Code != 200 {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}
	var listed []map[string]interface{}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("list unmarshal: %v", err)
	}
	if len(listed) != 1 || listed[0]["icon_url"] != iconURL {
		t.Fatalf("list icon_url=%v", listed)
	}
}
