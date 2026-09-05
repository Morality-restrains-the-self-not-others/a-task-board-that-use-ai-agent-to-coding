package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

func seedVendorGroupImage(t *testing.T, app *App) (vendorID, groupID, imgID int64, tok string) {
	t.Helper()
	vendorID = infrastructure.NextID()
	groupID = infrastructure.NextID()
	email := "lifecycle_" + infrastructure.IDStr(vendorID) + "@example.com"
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		vendorID, email, "x", "LifeCo", "Contact", 1, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup
		(id, name, description, created_at, updated_at, vendor_id) VALUES (?,?,?,?,?,?)`,
		groupID, "g", "", now, now, vendorID); err != nil {
		t.Fatal(err)
	}
	var err error
	imgID, err = app.DB.CreateContainerImage(vendorID, groupID, "v1", "docker.io/library/nginx:v1", []any{"amd64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id=?`, imgID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, groupID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id=?`, vendorID)
	})
	tok, err = infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(vendorID), "vendor", 3600)
	if err != nil {
		t.Fatal(err)
	}
	return vendorID, groupID, imgID, tok
}

func vendorMux(app *App) http.Handler {
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	return mux
}

func TestVendorContainerLifecycleWithdrawApprovedAndDelete(t *testing.T) {
	app := testApp(t)
	spy := &infrastructure.SpyEventBus{}
	app.Events = spy
	_, _, imgID, tok := seedVendorGroupImage(t, app)
	if _, err := app.DB.SQL.Exec(
		`UPDATE ai_provider_vendorcontainerimage SET status=?, is_active=? WHERE id=?`,
		domain.StatusApproved, 0, imgID,
	); err != nil {
		t.Fatal(err)
	}

	mux := vendorMux(app)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/container-images/"+infrastructure.IDStr(imgID)+"/withdraw/", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("withdraw status %d body %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != domain.StatusDraft {
		t.Fatalf("status=%v want draft", body["status"])
	}
	img, err := app.DB.GetContainerImage(imgID)
	if err != nil {
		t.Fatal(err)
	}
	if img.IsActive {
		t.Fatal("withdraw approved must clear is_active")
	}
	foundUnpublished := false
	for _, n := range spy.Names {
		if n == domain.EventContainerImageUnpublished {
			foundUnpublished = true
		}
	}
	if !foundUnpublished {
		t.Fatalf("events=%v missing Unpublished", spy.Names)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/vendor/container-images/"+infrastructure.IDStr(imgID)+"/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rr, req)
	if rr.Code != 204 {
		t.Fatalf("delete status %d body %s", rr.Code, rr.Body.String())
	}
	foundDeleted := false
	for _, n := range spy.Names {
		if n == domain.EventContainerImageDeleted {
			foundDeleted = true
		}
	}
	if !foundDeleted {
		t.Fatalf("events=%v missing Deleted", spy.Names)
	}
}

func TestVendorContainerLifecycleDeleteRejectsActive(t *testing.T) {
	app := testApp(t)
	_, _, imgID, tok := seedVendorGroupImage(t, app)
	if _, err := app.DB.SQL.Exec(
		`UPDATE ai_provider_vendorcontainerimage SET status=?, is_active=? WHERE id=?`,
		domain.StatusApproved, 1, imgID,
	); err != nil {
		t.Fatal(err)
	}
	mux := vendorMux(app)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/vendor/container-images/"+infrastructure.IDStr(imgID)+"/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rr, req)
	if rr.Code != 400 {
		t.Fatalf("delete active status %d body %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "激活") {
		t.Fatalf("body=%s want 激活", rr.Body.String())
	}
}

func TestVendorContainerLifecycleDeleteApprovedInactive(t *testing.T) {
	app := testApp(t)
	spy := &infrastructure.SpyEventBus{}
	app.Events = spy
	_, _, imgID, tok := seedVendorGroupImage(t, app)
	if _, err := app.DB.SQL.Exec(
		`UPDATE ai_provider_vendorcontainerimage SET status=?, is_active=? WHERE id=?`,
		domain.StatusApproved, 0, imgID,
	); err != nil {
		t.Fatal(err)
	}
	mux := vendorMux(app)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/vendor/container-images/"+infrastructure.IDStr(imgID)+"/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rr, req)
	if rr.Code != 204 {
		t.Fatalf("delete inactive approved status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestVendorContainerLifecycleDeletePendingReview(t *testing.T) {
	app := testApp(t)
	_, _, imgID, tok := seedVendorGroupImage(t, app)
	if _, err := app.DB.SQL.Exec(
		`UPDATE ai_provider_vendorcontainerimage SET status=? WHERE id=?`,
		domain.StatusPendingReview, imgID,
	); err != nil {
		t.Fatal(err)
	}
	mux := vendorMux(app)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/vendor/container-images/"+infrastructure.IDStr(imgID)+"/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	mux.ServeHTTP(rr, req)
	if rr.Code != 400 {
		t.Fatalf("delete pending status %d body %s", rr.Code, rr.Body.String())
	}
}
