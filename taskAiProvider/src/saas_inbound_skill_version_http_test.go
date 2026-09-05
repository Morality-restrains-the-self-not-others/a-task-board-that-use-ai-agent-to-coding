package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

func TestSaasInboundSkillVersionsPublic(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/ai-provider/saas-inbound-skill-versions/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["current"] != "1" {
		t.Fatalf("current=%v", body["current"])
	}
	vers, _ := body["versions"].([]any)
	if len(vers) == 0 {
		t.Fatal("expected versions")
	}
}

func TestSaasInboundSkillVersionsPublicWithoutDocsTree(t *testing.T) {
	app := testAppMinimal(t)
	app.Cfg.RepoRoot = t.TempDir()
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/ai-provider/saas-inbound-skill-versions/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["current"] != "1" {
		t.Fatalf("current=%v", body["current"])
	}

	req = httptest.NewRequest(http.MethodGet, "/saas-machine-container.md?version=1", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("md status %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "接口版本") {
		t.Fatalf("missing version banner: %s", truncateForTest(rec.Body.String(), 180))
	}
}

func TestSaasMachineContainerSkillVersionQuery(t *testing.T) {
	app := testAppMinimal(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/saas-machine-container.md?version=1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("v1 status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "接口版本") {
		t.Fatalf("missing version banner: %s", truncateForTest(rec.Body.String(), 180))
	}

	req = httptest.NewRequest(http.MethodGet, "/saas-machine-container.md?version=999", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown version status %d want 404", rec.Code)
	}
}

func TestVendorCreateContainerImageRequiresSkillVersion(t *testing.T) {
	app := testApp(t)
	vendorID := infrastructure.NextID()
	groupID := infrastructure.NextID()
	email := "skillver_" + infrastructure.IDStr(vendorID) + "@example.com"
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`, vendorID, email, "x", "SkillCo", "C", 1, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup
		(id, name, description, created_at, updated_at, vendor_id) VALUES (?,?,?,?,?,?)`,
		groupID, "g", "", now, now, vendorID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE vendor_id=?`, vendorID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, groupID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id=?`, vendorID)
	})
	tok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(vendorID), "vendor", 3600)
	if err != nil {
		t.Fatal(err)
	}
	spy := &infrastructure.SpyEventBus{}
	app.Events = spy
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	post := func(payload string) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/vendor/container-images/", bytes.NewReader([]byte(payload)))
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rr, req)
		return rr
	}

	rr := post(`{"image_group_id":"` + infrastructure.IDStr(groupID) + `","version":"9.0","image_url":"docker.io/library/nginx:9.0","target_architectures":["amd64"]}`)
	if rr.Code != 400 {
		t.Fatalf("missing skill version status %d body %s", rr.Code, rr.Body.String())
	}

	rr = post(`{"image_group_id":"` + infrastructure.IDStr(groupID) + `","version":"9.0","image_url":"docker.io/library/nginx:9.0","target_architectures":["amd64"],"saas_inbound_skill_version":"999"}`)
	if rr.Code != 400 {
		t.Fatalf("unknown skill version status %d body %s", rr.Code, rr.Body.String())
	}

	rr = post(`{"image_group_id":"` + infrastructure.IDStr(groupID) + `","version":"9.0","image_url":"docker.io/library/nginx:9.0","target_architectures":["amd64"],"saas_inbound_skill_version":"1"}`)
	if rr.Code != 201 {
		t.Fatalf("create status %d body %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created["saas_inbound_skill_version"] != "1" {
		t.Fatalf("created=%v", created)
	}
	foundEvent := false
	for _, n := range spy.Names {
		if n == domain.EventContainerImageSaasInboundSkillVersionAssigned {
			foundEvent = true
			break
		}
	}
	if !foundEvent {
		t.Fatalf("expected skill version assigned event, got %v", spy.Names)
	}
}
