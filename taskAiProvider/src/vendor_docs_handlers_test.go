package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

func vendorDocApplicantHeaders() map[string]string {
	return map[string]string{
		"X-User-Id":    "4242",
		"X-User-Email": "vendor-docs@example.com",
	}
}

func doJSON(t *testing.T, mux *http.ServeMux, method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestVendorDocUploadURLLocalPutComplete(t *testing.T) {
	app := testApp(t)
	spy := &infrastructure.SpyEventBus{}
	app.Events = spy
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	hdr := vendorDocApplicantHeaders()

	rec := doJSON(t, mux, http.MethodPost, "/api/ai-provider/vendor-application/upload-url/", map[string]any{
		"kind": "id_card", "filename": "id.png", "content_type": "image/png", "size": 4,
	}, hdr)
	if rec.Code != 200 {
		t.Fatalf("upload-url %d %s", rec.Code, rec.Body.String())
	}
	var issued map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &issued)
	key, _ := issued["file_key"].(string)
	up, _ := issued["upload_url"].(string)
	if key == "" || !strings.Contains(up, "local-put") {
		t.Fatalf("issued=%v", issued)
	}

	putReq := httptest.NewRequest(http.MethodPut, up, bytes.NewReader([]byte("png!")))
	putReq.Header.Set("Content-Type", "image/png")
	for k, v := range hdr {
		putReq.Header.Set(k, v)
	}
	putRec := httptest.NewRecorder()
	mux.ServeHTTP(putRec, putReq)
	if putRec.Code != 200 {
		t.Fatalf("local-put %d %s", putRec.Code, putRec.Body.String())
	}

	rec = doJSON(t, mux, http.MethodPost, "/api/ai-provider/vendor-application/upload-complete/", map[string]any{
		"kind": "id_card", "file_key": key,
	}, hdr)
	if rec.Code != 200 {
		t.Fatalf("complete %d %s", rec.Code, rec.Body.String())
	}
	found := false
	for _, n := range spy.Names {
		if n == domain.EventVendorDocumentUploaded {
			found = true
		}
	}
	if !found {
		t.Fatalf("events=%v", spy.Names)
	}
}

func TestVendorDocUploadGoneWhenCOS(t *testing.T) {
	app := testApp(t)
	fake := infrastructure.NewFakeCOS()
	app.Cfg.VendorDocsBackend = domain.VendorDocBackendCOS
	app.Docs = infrastructure.NewVendorDocStore(app.Cfg, app.Path, fake)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodPost, "/api/ai-provider/vendor-application/upload/", bytes.NewReader(nil))
	for k, v := range vendorDocApplicantHeaders() {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusGone {
		t.Fatalf("want 410 got %d %s", rec.Code, rec.Body.String())
	}
}

func TestVendorDocCOSPostPresign(t *testing.T) {
	app := testApp(t)
	fake := infrastructure.NewFakeCOS()
	app.Cfg.VendorDocsBackend = domain.VendorDocBackendCOS
	app.Docs = infrastructure.NewVendorDocStore(app.Cfg, app.Path, fake)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	hdr := vendorDocApplicantHeaders()

	rec := doJSON(t, mux, http.MethodPost, "/api/ai-provider/vendor-application/upload-url/", map[string]any{
		"kind": "id_card", "filename": "id.png", "content_type": "image/png", "size": 4,
	}, hdr)
	if rec.Code != 200 {
		t.Fatalf("upload-url %d %s", rec.Code, rec.Body.String())
	}
	var issued map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &issued)
	key, _ := issued["file_key"].(string)
	up, _ := issued["upload_url"].(string)
	method, _ := issued["method"].(string)
	fields, _ := issued["form_fields"].(map[string]any)
	if key == "" || method != "POST" || strings.Contains(up, "local-put") || !strings.Contains(up, "fake-cos.example") {
		t.Fatalf("issued=%v", issued)
	}
	if fields["key"] != key || fields["x-cos-server-side-encryption"] != "AES256" {
		t.Fatalf("form_fields=%v", fields)
	}
	if err := app.Docs.Put(context.Background(), 4242, key, "image/png", bytes.NewReader([]byte("png!")), 4); err != nil {
		t.Fatalf("simulate COS POST: %v", err)
	}
	if err := app.Docs.Head(context.Background(), 4242, key); err != nil {
		t.Fatalf("cos head after POST Object: %v", err)
	}
}

func TestVendorDocApplyHeadCOSThenLocalFallback(t *testing.T) {
	app := testApp(t)
	fake := infrastructure.NewFakeCOS()
	app.Cfg.VendorDocsBackend = domain.VendorDocBackendCOS
	app.Docs = infrastructure.NewVendorDocStore(app.Cfg, app.Path, fake)
	idKey, _, err := infrastructure.SaveVendorDocument(app.Cfg.VendorDocsDir, 4242, infrastructure.VendorDocKindIDCard, "id.jpg", bytes.NewReader([]byte("idimg")), 5)
	if err != nil {
		t.Fatal(err)
	}
	licKey, _, err := infrastructure.SaveVendorDocument(app.Cfg.VendorDocsDir, 4242, infrastructure.VendorDocKindBusinessLicense, "lic.pdf", bytes.NewReader([]byte("%PDF-1")), 6)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rec := doJSON(t, mux, http.MethodPost, "/api/ai-provider/vendor-application/", map[string]string{
		"company_name": "COS Fallback Co", "contact_name": "张三",
		"id_card_file_key": idKey, "business_license_file_key": licKey,
		"contact_phone": "+8613800138000",
	}, vendorDocApplicantHeaders())
	if rec.Code != 200 {
		t.Fatalf("apply fallback %d %s", rec.Code, rec.Body.String())
	}

	staffID := infrastructure.NextID()
	now := "2026-08-14 00:00:00.000000"
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_platformstaff
		(id, username, password_hash, display_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,1,?,?)`, staffID, "doc_staff", "x", "Doc Staff", now, now); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_platformstaff WHERE id=?`, staffID) })
	tok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(staffID), "staff", 3600)
	if err != nil {
		t.Fatal(err)
	}
	var applied map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &applied)
	vendor, _ := applied["vendor"].(map[string]any)
	vid, _ := vendor["id"].(string)
	if vid == "" {
		t.Fatalf("no vendor id in %v", applied)
	}
	docReq := httptest.NewRequest(http.MethodGet, "/api/admin/vendors/"+vid+"/documents/id_card/", nil)
	docReq.Header.Set("Authorization", "Bearer "+tok)
	docRec := httptest.NewRecorder()
	mux.ServeHTTP(docRec, docReq)
	if docRec.Code != 200 {
		t.Fatalf("staff get doc %d %s", docRec.Code, docRec.Body.String())
	}
	if !bytes.Contains(docRec.Body.Bytes(), []byte("idimg")) {
		t.Fatalf("expected local fallback body, got %q", docRec.Body.String())
	}
}

func TestAdminVendorDocsStoragePatch(t *testing.T) {
	app := testApp(t)
	orig := filepath.Join(app.Cfg.RepoRoot, "conf", "ai", "ai-provider", "vendor-docs-path.yaml")
	backup, err := os.ReadFile(orig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.WriteFile(orig, backup, 0o644) })
	spy := &infrastructure.SpyEventBus{}
	app.Events = spy
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rec := doJSON(t, mux, http.MethodPatch, "/api/ai-provider/admin-vendor-docs-storage/", map[string]string{
		"keyPrefix": "vendor-docs",
		"pathRule":  "{keyPrefix}/{yyyy}/{userId}/{kind}_{id}{ext}",
	}, map[string]string{"X-User-Id": "1", "X-User-Roles": "super_admin"})
	if rec.Code != 200 {
		t.Fatalf("patch %d %s", rec.Code, rec.Body.String())
	}
	_, rule := app.Path.Get()
	if !strings.Contains(rule, "{yyyy}") {
		t.Fatalf("hot reload missed: %s", rule)
	}
	found := false
	for _, n := range spy.Names {
		if n == domain.EventVendorDocPathRuleUpdated {
			found = true
		}
	}
	if !found {
		t.Fatalf("events=%v", spy.Names)
	}
}

func TestVendorDocUploadURLRejectsBadMeta(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rec := doJSON(t, mux, http.MethodPost, "/api/ai-provider/vendor-application/upload-url/", map[string]any{
		"kind": "id_card", "filename": "x.exe", "content_type": "application/octet-stream", "size": 1,
	}, vendorDocApplicantHeaders())
	if rec.Code != 400 {
		t.Fatalf("want 400 got %d", rec.Code)
	}
}
