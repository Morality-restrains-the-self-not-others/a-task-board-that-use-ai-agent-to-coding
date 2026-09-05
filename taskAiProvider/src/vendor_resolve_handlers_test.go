package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskAiProvider/infrastructure"
)

func TestResolveTargetArchitecturesRejectsAliyunVPC(t *testing.T) {
	app := testApp(t)
	id := infrastructure.NextID()
	email := "resolve_vpc_" + infrastructure.IDStr(id) + "@example.com"
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		id, email, "x", "Co", "Contact", 1, now, now)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id=?`, id)
	})
	tok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(id), "vendor", 3600)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	body := []byte(`{"image_url":"registry-vpc.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64-latest"}`)
	start := time.Now()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/container-images/resolve-target-architectures/", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if time.Since(start) > 2*time.Second {
		t.Fatalf("private registry must fail-fast, took %s", time.Since(start))
	}
	if rr.Code != 400 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "无法触及") {
		t.Fatalf("missing intranet hint: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "registry.cn-qingdao.aliyuncs.com") {
		t.Fatalf("missing public mapping: %s", rr.Body.String())
	}
}

// registryResolveTestServer serves a manifest index + single manifest + one
// gzip-tar layer blob, mirroring what ExtractAutoRunAndSkillsFromImage and
// ResolveContainerImageMetadata fetch for a single image.
func registryResolveTestServer(t *testing.T, layerFiles map[string]string) *httptest.Server {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, content := range layerFiles {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	layerBlob := buf.Bytes()

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/hello/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"manifests": []map[string]any{{
				"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
				"platform":  map[string]any{"architecture": "amd64", "os": "linux"},
				"digest":    "sha256:single",
				"size":      500,
			}},
		})
	})
	mux.HandleFunc("/v2/hello/manifests/sha256:single", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"schemaVersion": 2,
			"mediaType":     "application/vnd.docker.distribution.manifest.v2+json",
			"config": map[string]any{
				"mediaType": "application/vnd.docker.container.image.v1+json",
				"size":      50,
				"digest":    "sha256:cfg",
			},
			"layers": []map[string]any{{
				"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
				"size":      100,
				"digest":    "sha256:layer0",
			}},
		})
	})
	mux.HandleFunc("/v2/hello/blobs/sha256:layer0", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(layerBlob)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	prev := infrastructure.ExportRegistrySchemeForTest()
	infrastructure.SetRegistrySchemeForTest("http")
	t.Cleanup(func() { infrastructure.SetRegistrySchemeForTest(prev) })
	return srv
}

// vendorResolveApp inserts a test vendor row on app and returns (app, token).
// The same app must serve the test routes so the vendor row is visible.
func vendorResolveApp(t *testing.T, email string) (*App, string) {
	t.Helper()
	app := testApp(t)
	id := infrastructure.NextID()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		id, email, "x", "Co", "Contact", 1, now, now)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id=?`, id)
	})
	tok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(id), "vendor", 3600)
	if err != nil {
		t.Fatal(err)
	}
	return app, tok
}

func TestResolveTargetArchitecturesReturnsSkillsAndAutoRun(t *testing.T) {
	srv := registryResolveTestServer(t, map[string]string{
		"app/autoRunStep.md":   "# 自动运行\n1. 启动服务\n",
		"app/imageSkills.yaml": "version: 1\ndefault_skill: coding\nskills:\n  - name: coding\n    description: 通用编码\n    is_default: true\n  - name: review\n    description: 代码审查\n",
	})
	app, tok := vendorResolveApp(t, "resolve_skills_ok@example.com")

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	host := strings.TrimPrefix(srv.URL, "http://")
	body := []byte(`{"image_url":"` + host + `/hello","version":""}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/container-images/resolve-target-architectures/", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	arches, _ := out["target_architectures"].([]any)
	if len(arches) != 1 || arches[0] != "x86_64" {
		t.Fatalf("target_architectures=%v", out["target_architectures"])
	}
	if out["size"] == nil {
		t.Fatalf("missing size: %v", out)
	}
	if out["skills_status"] != "ok" {
		t.Fatalf("skills_status=%v body=%s", out["skills_status"], rr.Body.String())
	}
	skills, _ := out["skills"].(map[string]any)
	list, _ := skills["skills"].([]any)
	if len(list) != 2 {
		t.Fatalf("skills=%v", out["skills"])
	}
	first, _ := list[0].(map[string]any)
	if first["name"] != "coding" || first["is_default"] != true {
		t.Fatalf("first skill=%v", first)
	}
	if out["auto_run_steps_status"] != "ok" {
		t.Fatalf("auto_run_steps_status=%v", out["auto_run_steps_status"])
	}
	if !strings.Contains(out["auto_run_steps_md"].(string), "启动服务") {
		t.Fatalf("auto_run_steps_md=%v", out["auto_run_steps_md"])
	}
}

func TestResolveTargetArchitecturesReportsMissingSkillsFiles(t *testing.T) {
	srv := registryResolveTestServer(t, map[string]string{
		"app/other.txt": "unrelated",
	})
	app, tok := vendorResolveApp(t, "resolve_skills_missing@example.com")

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	host := strings.TrimPrefix(srv.URL, "http://")
	body := []byte(`{"image_url":"` + host + `/hello"}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/container-images/resolve-target-architectures/", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["skills_status"] != "not_found" {
		t.Fatalf("skills_status=%v body=%s", out["skills_status"], rr.Body.String())
	}
	if out["skills"] != nil {
		t.Fatalf("skills must be null when not found: %v", out["skills"])
	}
	if out["auto_run_steps_status"] != "not_found" {
		t.Fatalf("auto_run_steps_status=%v", out["auto_run_steps_status"])
	}
	if out["auto_run_steps_md"] != "" {
		t.Fatalf("auto_run_steps_md must be empty: %v", out["auto_run_steps_md"])
	}
	arches, _ := out["target_architectures"].([]any)
	if len(arches) != 1 || arches[0] != "x86_64" {
		t.Fatalf("architecture must still resolve: %v", out["target_architectures"])
	}
}
