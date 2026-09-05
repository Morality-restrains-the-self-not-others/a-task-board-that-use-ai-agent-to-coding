package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func seedInstalledImage(t *testing.T, id, tenantID, name, archesJSON string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,target_architectures,is_dev_mode)
		VALUES(?,?,?,?,?,?,0)`, id, tenantID, "ext-"+id, name, "registry.example/"+id, archesJSON)
	if err != nil {
		t.Fatalf("seed image %s: %v", id, err)
	}
}

func TestInternalInstalledImageLookupFallsBackToUniqueName(t *testing.T) {
	setupCloudTestDB(t)
	seedInstalledImage(t, "878236719185424384", "t1", "trae-agent", `["x86_64"]`)

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/tenant-installed-images/lookup?tenant_id=t1&id=878236807722987520&name=trae-agent", nil)
	rec := httptest.NewRecorder()
	handleInternalTenantInstalledImagesLookup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stale id + unique name should 200, got %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["id"] != "878236719185424384" {
		t.Fatalf("expected rebound live id, got %v", out["id"])
	}
	if out["name"] != "trae-agent" {
		t.Fatalf("expected name trae-agent, got %v", out["name"])
	}
}

func TestInternalInstalledImageLookupStaleIDWithoutNameIs404(t *testing.T) {
	setupCloudTestDB(t)
	seedInstalledImage(t, "878236719185424384", "t1", "trae-agent", `["x86_64"]`)

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/tenant-installed-images/lookup?tenant_id=t1&id=878236807722987520", nil)
	rec := httptest.NewRecorder()
	handleInternalTenantInstalledImagesLookup(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("stale id without name should 404, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestInternalInstalledImageLookupAmbiguousNameStays404(t *testing.T) {
	setupCloudTestDB(t)
	seedInstalledImage(t, "img-a", "t1", "trae-agent", `["x86_64"]`)
	seedInstalledImage(t, "img-b", "t1", "trae-agent", `["x86_64"]`)

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/tenant-installed-images/lookup?tenant_id=t1&id=missing&name=trae-agent", nil)
	rec := httptest.NewRecorder()
	handleInternalTenantInstalledImagesLookup(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("ambiguous name must not rebound, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestInternalInstalledImageLookupExactIDWins(t *testing.T) {
	setupCloudTestDB(t)
	seedInstalledImage(t, "img-live", "t1", "trae-agent", `["x86_64"]`)
	seedInstalledImage(t, "img-other", "t1", "other", `["arm64"]`)

	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/tenant-installed-images/lookup?tenant_id=t1&id=img-other&name=trae-agent", nil)
	rec := httptest.NewRecorder()
	handleInternalTenantInstalledImagesLookup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("exact id should 200, got %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["id"] != "img-other" {
		t.Fatalf("id match must win over name, got %v", out["id"])
	}
}
