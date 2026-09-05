package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

type recordingExtractor struct {
	mu     sync.Mutex
	calls  []string
	result infrastructure.AutoRunStepsExtractResult
}

func (r *recordingExtractor) Extract(imageURL string) infrastructure.AutoRunStepsExtractResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, imageURL)
	return r.result
}

func (r *recordingExtractor) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func seedVendorImageGroup(t *testing.T, app *App) (vendorID, groupID int64, token string) {
	t.Helper()
	vendorID = infrastructure.NextID()
	groupID = infrastructure.NextID()
	email := "autorun_" + infrastructure.IDStr(vendorID) + "@example.com"
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`, vendorID, email, "x", "AutoRunCo", "C", 1, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup
		(id, name, description, created_at, updated_at, vendor_id) VALUES (?,?,?,?,?,?)`,
		groupID, "g", "group desc", now, now, vendorID); err != nil {
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
	return vendorID, groupID, tok
}

func TestVendorCreateContainerImageExtractsAutoRunSteps(t *testing.T) {
	app := testApp(t)
	_, groupID, tok := seedVendorImageGroup(t, app)
	ext := &recordingExtractor{result: infrastructure.AutoRunStepsExtractResult{
		Markdown: "# from extract\n", Status: "ok", Digest: "sha256:abc",
	}}
	app.AutoRunExtractor = ext
	spy := &infrastructure.SpyEventBus{}
	app.Events = spy
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	payload := `{"image_group_id":"` + infrastructure.IDStr(groupID) + `","version":"1.0","image_url":"example.com/app","target_architectures":["amd64"],"saas_inbound_skill_version":"1"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/container-images/", bytes.NewReader([]byte(payload)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("create status %d body %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id, err := parseIDFlexible(created["id"])
	if err != nil {
		t.Fatal(err)
	}
	img, err := app.DB.GetContainerImage(id)
	if err != nil {
		t.Fatal(err)
	}
	if img.AutoRunStepsExtractStatus != "ok" || !strings.Contains(img.AutoRunStepsMD, "from extract") {
		t.Fatalf("md=%q status=%q", img.AutoRunStepsMD, img.AutoRunStepsExtractStatus)
	}
	if ext.callCount() != 1 || ext.calls[0] != "example.com/app:1.0" {
		t.Fatalf("calls=%v", ext.calls)
	}
	found := false
	for _, n := range spy.Names {
		if n == domain.EventContainerImageAutoRunStepsExtracted {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected extract event, got %v", spy.Names)
	}
}

func TestVendorPatchImageURLReExtractsAutoRunSteps(t *testing.T) {
	app := testApp(t)
	vendorID, groupID, tok := seedVendorImageGroup(t, app)
	imgID, err := app.DB.CreateContainerImage(vendorID, groupID, "1.0", "example.com/old", []any{"amd64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	ext := &recordingExtractor{result: infrastructure.AutoRunStepsExtractResult{
		Markdown: "# patched\n", Status: "ok", Digest: "sha256:new",
	}}
	app.AutoRunExtractor = ext
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"image_url":"example.com/new"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/vendor/container-images/"+infrastructure.IDStr(imgID)+"/", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("patch status %d body %s", rr.Code, rr.Body.String())
	}
	img, err := app.DB.GetContainerImage(imgID)
	if err != nil {
		t.Fatal(err)
	}
	if img.AutoRunStepsExtractStatus != "ok" || !strings.Contains(img.AutoRunStepsMD, "patched") {
		t.Fatalf("md=%q status=%q", img.AutoRunStepsMD, img.AutoRunStepsExtractStatus)
	}
	if ext.callCount() != 1 || ext.calls[0] != "example.com/new:1.0" {
		t.Fatalf("calls=%v", ext.calls)
	}
}

func TestVendorPatchArchitecturesDoesNotExtract(t *testing.T) {
	app := testApp(t)
	vendorID, groupID, tok := seedVendorImageGroup(t, app)
	imgID, err := app.DB.CreateContainerImage(vendorID, groupID, "1.0", "example.com/app:1.0", []any{"amd64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	ext := &recordingExtractor{result: infrastructure.AutoRunStepsExtractResult{Status: "ok", Markdown: "nope"}}
	app.AutoRunExtractor = ext
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body := `{"target_architectures":["arm64"]}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/vendor/container-images/"+infrastructure.IDStr(imgID)+"/", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("patch status %d body %s", rr.Code, rr.Body.String())
	}
	if ext.callCount() != 0 {
		t.Fatalf("unexpected extract calls=%v", ext.calls)
	}
}

func TestPublicCatalogBackfillsEmptyAutoRunSteps(t *testing.T) {
	app := testApp(t)
	vendorID, groupID, _ := seedVendorImageGroup(t, app)
	imgID, err := app.DB.CreateContainerImage(vendorID, groupID, "2.0", "example.com/catalog", []any{"amd64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.DB.UpdateContainerImageFields(imgID, map[string]any{"status": domain.StatusApproved}); err != nil {
		t.Fatal(err)
	}
	ext := &recordingExtractor{result: infrastructure.AutoRunStepsExtractResult{
		Markdown: "# catalog extract\n", Status: "ok", Digest: "sha256:cat",
	}}
	app.AutoRunExtractor = ext
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/public/catalog/", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("catalog status %d body %s", rr.Code, rr.Body.String())
	}
	img, err := app.DB.GetContainerImage(imgID)
	if err != nil {
		t.Fatal(err)
	}
	if img.AutoRunStepsExtractStatus != "ok" || !strings.Contains(img.AutoRunStepsMD, "catalog extract") {
		t.Fatalf("md=%q status=%q", img.AutoRunStepsMD, img.AutoRunStepsExtractStatus)
	}
	if ext.callCount() != 1 {
		t.Fatalf("calls=%v", ext.calls)
	}
}
