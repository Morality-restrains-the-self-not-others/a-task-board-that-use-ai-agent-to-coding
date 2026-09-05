package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const x86RunTemplateJSON = `{"platform":"aliyun","region":"cn-hangzhou","cloud_platform_id":"p1","selected_instance":"ecs.g7.xlarge","hardware_config":{"instance_type":"ecs.g7.xlarge"}}`
const armRunTemplateJSON = `{"platform":"aliyun","region":"cn-hangzhou","cloud_platform_id":"p1","selected_instance":"ecs.g8y.xlarge","hardware_config":{"instance_type":"ecs.g8y.xlarge"}}`

func startInstalledImageLookup(t *testing.T, imageID string, arches []string, name string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/tenant-installed-images/lookup" {
			t.Fatalf("unexpected cloud path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("id") != imageID {
			t.Fatalf("unexpected id: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                   imageID,
			"name":                 name,
			"target_architectures": arches,
		})
	}))
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)
	t.Cleanup(srv.Close)
	return srv
}

func createNamedProject(t *testing.T, name string) string {
	t.Helper()
	body := `{"name":"` + name + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	pid, _ := created["id"].(string)
	if pid == "" {
		t.Fatalf("create failed: %s", rec.Body.String())
	}
	return pid
}

func TestUpdateProjectContainerImage(t *testing.T) {
	setupTestDB(t)
	startInstalledImageLookup(t, "859671040643174400", []string{"x86_64"}, "trae0630")
	pid := createNamedProject(t, "Image Project")

	body2 := `{"container_image_id":"859671040643174400","container_image":"trae0630","server_run_template":` + x86RunTemplateJSON + `}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(body2))
	req2.Header.Set("X-Resource-Id", pid)
	rec2 := httptest.NewRecorder()
	handleUpdateProject(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var updated map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&updated)
	if updated["container_image_id"] != "859671040643174400" {
		t.Errorf("expected container_image_id saved, got %v", updated["container_image_id"])
	}
	if updated["container_image"] != "trae0630" {
		t.Errorf("expected container_image name saved, got %v", updated["container_image"])
	}
}

func TestUpdateProjectResolvesContainerImageNameFromCloudLookup(t *testing.T) {
	setupTestDB(t)
	startInstalledImageLookup(t, "859671040643174400", []string{"x86_64"}, "trae0630")
	pid := createNamedProject(t, "Lookup Project")

	body2 := `{"container_image_id":"859671040643174400","server_run_template":` + x86RunTemplateJSON + `}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(body2))
	req2.Header.Set("X-Resource-Id", pid)
	rec2 := httptest.NewRecorder()
	handleUpdateProject(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var updated map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&updated)
	if updated["container_image"] != "trae0630" {
		t.Errorf("expected resolved container_image name, got %v", updated["container_image"])
	}
}

func TestUpdateProjectRejectsCrossArchImageWithoutMatchingTemplate(t *testing.T) {
	setupTestDB(t)
	pid := createNamedProject(t, "Arch Project")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		arch := "x86_64"
		name := "x86"
		if id == "img-arm" {
			arch = "arm64"
			name = "arm-img"
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": id, "name": name, "target_architectures": []string{arch},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

	seed := `{"container_image_id":"img-x86","container_image":"x86","server_run_template":` + x86RunTemplateJSON + `}`
	reqSeed := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(seed))
	reqSeed.Header.Set("X-Resource-Id", pid)
	recSeed := httptest.NewRecorder()
	handleUpdateProject(recSeed, reqSeed)
	if recSeed.Code != http.StatusOK {
		t.Fatalf("seed: %d %s", recSeed.Code, recSeed.Body.String())
	}

	onlyImage := `{"container_image_id":"img-arm"}`
	reqBad := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(onlyImage))
	reqBad.Header.Set("X-Resource-Id", pid)
	recBad := httptest.NewRecorder()
	handleUpdateProject(recBad, reqBad)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recBad.Code, recBad.Body.String())
	}
	reqGet := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	reqGet.Header.Set("X-Resource-Id", pid)
	recGet := httptest.NewRecorder()
	handleGetProject(recGet, reqGet)
	var stored map[string]interface{}
	json.NewDecoder(recGet.Body).Decode(&stored)
	if stored["container_image_id"] != "img-x86" {
		t.Fatalf("image must stay x86, got %v", stored["container_image_id"])
	}

	okBody := `{"container_image_id":"img-arm","server_run_template":` + armRunTemplateJSON + `}`
	reqOK := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(okBody))
	reqOK.Header.Set("X-Resource-Id", pid)
	recOK := httptest.NewRecorder()
	handleUpdateProject(recOK, reqOK)
	if recOK.Code != http.StatusOK {
		t.Fatalf("matching pair: %d %s", recOK.Code, recOK.Body.String())
	}
}

func TestUpdateProjectSameArchImageOnly(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": r.URL.Query().Get("id"), "name": "img", "target_architectures": []string{"x86_64"},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)
	pid := createNamedProject(t, "Same Arch")
	seed := `{"container_image_id":"img-a","server_run_template":` + x86RunTemplateJSON + `}`
	reqSeed := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(seed))
	reqSeed.Header.Set("X-Resource-Id", pid)
	recSeed := httptest.NewRecorder()
	handleUpdateProject(recSeed, reqSeed)
	if recSeed.Code != http.StatusOK {
		t.Fatalf("seed: %d %s", recSeed.Code, recSeed.Body.String())
	}
	only := `{"container_image_id":"img-b"}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(only))
	req.Header.Set("X-Resource-Id", pid)
	rec := httptest.NewRecorder()
	handleUpdateProject(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("same arch image-only: %d %s", rec.Code, rec.Body.String())
	}
}

// Regression: FE hardware panel often PATCHes image with region-only live draft (no instance).
// Must not 400 with errMsgImageTemplateIncomplete when stored template is complete + same arch.
func TestUpdateProjectIgnoresIncompleteLiveDraftWhenStoredComplete(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": r.URL.Query().Get("id"), "name": "img", "target_architectures": []string{"x86_64"},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)
	pid := createNamedProject(t, "Incomplete Live Draft")
	seed := `{"container_image_id":"img-a","server_run_template":` + x86RunTemplateJSON + `}`
	reqSeed := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(seed))
	reqSeed.Header.Set("X-Resource-Id", pid)
	recSeed := httptest.NewRecorder()
	handleUpdateProject(recSeed, reqSeed)
	if recSeed.Code != http.StatusOK {
		t.Fatalf("seed: %d %s", recSeed.Code, recSeed.Body.String())
	}

	incomplete := `{"container_image_id":"img-b","container_image":"img","server_run_template":{"platform":"aliyun","region":"cn-hangzhou","label":"半成品"}}`
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(incomplete))
	req.Header.Set("X-Resource-Id", pid)
	rec := httptest.NewRecorder()
	handleUpdateProject(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("incomplete live draft should be ignored: %d %s", rec.Code, rec.Body.String())
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	reqGet.Header.Set("X-Resource-Id", pid)
	recGet := httptest.NewRecorder()
	handleGetProject(recGet, reqGet)
	var stored map[string]interface{}
	json.NewDecoder(recGet.Body).Decode(&stored)
	if stored["container_image_id"] != "img-b" {
		t.Fatalf("image should update, got %v", stored["container_image_id"])
	}
	tmpl, _ := stored["server_run_template"].(map[string]interface{})
	if tmpl["selected_instance"] != "ecs.g7.xlarge" {
		t.Fatalf("stored complete template must be preserved, got %#v", tmpl)
	}
}

func TestUpdateProjectInfersArchFromVersionWhenDeclaredEmpty(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                   r.URL.Query().Get("id"),
			"name":                 "trae-agent",
			"version":              "private_x86_64-latest",
			"target_architectures": []string{},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)
	pid := createNamedProject(t, "Empty Arch Version Hint")
	body := `{"container_image_id":"img-private","server_run_template":` + x86RunTemplateJSON + `}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Resource-Id", pid)
	rec := httptest.NewRecorder()
	handleUpdateProject(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("version-hinted x86 should save, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateProjectUnknownArchErrorNamesBothSides(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                   r.URL.Query().Get("id"),
			"name":                 "trae-agent",
			"version":              "latest",
			"target_architectures": []string{},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)
	pid := createNamedProject(t, "Unknown Arch Message")
	body := `{"container_image_id":"img-empty","server_run_template":` + x86RunTemplateJSON + `}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Resource-Id", pid)
	rec := httptest.NewRecorder()
	handleUpdateProject(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	got := rec.Body.String()
	if !strings.Contains(got, "镜像要求: 未声明") || !strings.Contains(got, "实例系统支持: x86_64（实例规格 ecs.g7.xlarge）") {
		t.Fatalf("400 body should name both arches, got %s", got)
	}
}

func TestUpdateProjectUnrecognizedDeclaredArchErrorNamesBothSides(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                   r.URL.Query().Get("id"),
			"name":                 "trae-agent",
			"version":              "latest",
			"target_architectures": []string{"unknown"},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)
	pid := createNamedProject(t, "Unrecognized Arch Message")
	body := `{"container_image_id":"img-unknown","server_run_template":` + x86RunTemplateJSON + `}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Resource-Id", pid)
	rec := httptest.NewRecorder()
	handleUpdateProject(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	got := rec.Body.String()
	if !strings.Contains(got, "镜像要求: unknown（无法识别为 x86_64/arm64）") ||
		!strings.Contains(got, "实例系统支持: x86_64（实例规格 ecs.g7.xlarge）") {
		t.Fatalf("400 body should name unrecognized image ISA and instance ISA, got %s", got)
	}
}

func TestUpdateProjectHealsStaleInstalledImageIDByUniqueName(t *testing.T) {
	setupTestDB(t)
	const staleID = "878236807722987520"
	const liveID = "878236719185424384"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/tenant-installed-images/lookup" {
			t.Fatalf("unexpected cloud path: %s", r.URL.Path)
		}
		id := r.URL.Query().Get("id")
		name := r.URL.Query().Get("name")
		if id == liveID || (id == staleID && name == "trae-agent") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": liveID, "name": "trae-agent", "version": "x86_64-latest",
				"target_architectures": []string{"x86_64"},
			})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

	pid := createNamedProject(t, "Stale Image")
	_, err := db.Exec(
		`UPDATE project_entries SET installed_image_id=?, container_image_name=?, server_run_template=? WHERE id=?`,
		staleID, "trae-agent", x86RunTemplateJSON, pid,
	)
	if err != nil {
		t.Fatalf("seed stale image: %v", err)
	}

	body := `{"server_run_template":` + x86RunTemplateJSON + `}`
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Resource-Id", pid)
	rec := httptest.NewRecorder()
	handleUpdateProject(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unique-name rebound should save, got %d: %s", rec.Code, rec.Body.String())
	}
	reqGet := httptest.NewRequest(http.MethodGet, "/api/projects/"+pid+"/tenant_id/t1/", nil)
	reqGet.Header.Set("X-Resource-Id", pid)
	recGet := httptest.NewRecorder()
	handleGetProject(recGet, reqGet)
	var stored map[string]interface{}
	json.NewDecoder(recGet.Body).Decode(&stored)
	if stored["container_image_id"] != liveID {
		t.Fatalf("expected healed id %s, got %v body=%s", liveID, stored["container_image_id"], recGet.Body.String())
	}
}

func TestUpdateProjectMissingImageWithoutNameRejectsUninstalled(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TASK_CLOUD_SERVICE_BASE_URL", srv.URL)

	pid := createNamedProject(t, "Missing Image")
	_, err := db.Exec(
		`UPDATE project_entries SET installed_image_id=?, container_image_name=?, server_run_template=? WHERE id=?`,
		"878236807722987520", "", x86RunTemplateJSON, pid,
	)
	if err != nil {
		t.Fatalf("seed missing image: %v", err)
	}

	body := `{"server_run_template":` + x86RunTemplateJSON + `}`
	req := httptest.NewRequest(http.MethodPatch, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Resource-Id", pid)
	rec := httptest.NewRecorder()
	handleUpdateProject(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	got := rec.Body.String()
	if !strings.Contains(got, "镜像要求: 不存在或已卸载") ||
		!strings.Contains(got, "实例系统支持: x86_64（实例规格 ecs.g7.xlarge）") {
		t.Fatalf("400 should name missing image, got %s", got)
	}
}
