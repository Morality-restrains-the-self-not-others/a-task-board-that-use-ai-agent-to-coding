package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnsureInstalledImageAutoRunStepsReExtracts(t *testing.T) {
	setupCloudTestDB(t)
	prevFn := extractAutoRunStepsFn
	extractAutoRunStepsFn = func(imageURL, filePath string) autoRunStepsExtractResult {
		if imageURL != "registry.example/img:v1" {
			t.Fatalf("unexpected imageURL=%q", imageURL)
		}
		return autoRunStepsExtractResult{Status: "ok", Markdown: "# 自动运行\nstep 1", Digest: "sha256:abc"}
	}
	t.Cleanup(func() { extractAutoRunStepsFn = prevFn })

	img := &TenantInstalledImage{
		ID: "img-reext", TenantID: "t1", ExternalImageID: "ext-reext",
		Name: "img", ImageURL: "registry.example/img:v1",
	}
	if err := createInstalledImage(*img); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := ensureInstalledImageAutoRunSteps(img, "t1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if img.AutoRunStepsMd != "# 自动运行\nstep 1" || img.AutoRunStepsExtractStatus != "ok" || img.AutoRunStepsDigest != "sha256:abc" {
		t.Fatalf("img after re-extract: %+v", img)
	}
	got, err := getInstalledImage("t1", "img-reext")
	if err != nil || got == nil {
		t.Fatalf("get: %v got=%v", err, got)
	}
	if got.AutoRunStepsMd != "# 自动运行\nstep 1" || got.AutoRunStepsExtractStatus != "ok" {
		t.Fatalf("db row: %+v", got)
	}
}

func TestEnsureInstalledImageAutoRunStepsSkipsWhenPresent(t *testing.T) {
	setupCloudTestDB(t)
	called := false
	prevFn := extractAutoRunStepsFn
	extractAutoRunStepsFn = func(imageURL, filePath string) autoRunStepsExtractResult {
		called = true
		return autoRunStepsExtractResult{Status: "ok", Markdown: "x"}
	}
	t.Cleanup(func() { extractAutoRunStepsFn = prevFn })

	img := &TenantInstalledImage{
		ID: "img-has", TenantID: "t1", ExternalImageID: "ext-has",
		Name: "img", ImageURL: "registry.example/img:v1", AutoRunStepsMd: "已有说明",
	}
	if err := ensureInstalledImageAutoRunSteps(img, "t1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if called {
		t.Fatal("must not re-extract when auto_run_steps_md already present")
	}
	// 无 image_url 也不抽取
	img2 := &TenantInstalledImage{ID: "img-nourl", TenantID: "t1", ExternalImageID: "ext-nourl", Name: "nourl"}
	if err := ensureInstalledImageAutoRunSteps(img2, "t1"); err != nil {
		t.Fatalf("ensure2: %v", err)
	}
	if called {
		t.Fatal("must not re-extract when image_url empty")
	}
}

func TestEnsureInstalledImageAutoRunStepsFailureDoesNotBlock(t *testing.T) {
	setupCloudTestDB(t)
	prevFn := extractAutoRunStepsFn
	extractAutoRunStepsFn = func(imageURL, filePath string) autoRunStepsExtractResult {
		return autoRunStepsExtractResult{Status: "auth_failed", Detail: "no token", Digest: "sha256:xyz"}
	}
	t.Cleanup(func() { extractAutoRunStepsFn = prevFn })

	img := &TenantInstalledImage{
		ID: "img-fail", TenantID: "t1", ExternalImageID: "ext-fail",
		Name: "img", ImageURL: "registry.example/private:v1",
	}
	if err := createInstalledImage(*img); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := ensureInstalledImageAutoRunSteps(img, "t1"); err != nil {
		t.Fatalf("ensure should not error on extract failure: %v", err)
	}
	if !strings.Contains(img.AutoRunStepsExtractStatus, "auth_failed") {
		t.Fatalf("status=%q want auth_failed", img.AutoRunStepsExtractStatus)
	}
}

func TestInstalledImageInstallReExtractsEmptyAutoRunSteps(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	store.putMember("t1", "admin-1", "mem-1", true)
	_ = startSaasInternalMock(t, store)

	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id": "ext-1", "name": "img", "image_url": "registry.example/img:v1",
				"target_architectures": []string{"x86_64"},
			},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	prevFn := extractAutoRunAndSkillsFn
	extractAutoRunAndSkillsFn = func(imageURL string) autoRunAndSkillsExtractResult {
		return autoRunAndSkillsExtractResult{
			AutoRun: autoRunStepsExtractResult{Status: "ok", Markdown: "# 自动运行说明", Digest: "sha256:step"},
			Skills:  imageSkillsExtractResult{Status: "not_found"},
		}
	}
	t.Cleanup(func() { extractAutoRunAndSkillsFn = prevFn })

	body := `{"external_image_id":"ext-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "admin-1")
	rec := httptest.NewRecorder()
	handleInstalledImageCollection(rec, req, "t1")

	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["auto_run_steps_md"] != "# 自动运行说明" {
		t.Fatalf("auto_run_steps_md=%v want re-extracted", out["auto_run_steps_md"])
	}
	if out["auto_run_steps_extract_status"] != "ok" {
		t.Fatalf("extract_status=%v want ok", out["auto_run_steps_extract_status"])
	}
}
