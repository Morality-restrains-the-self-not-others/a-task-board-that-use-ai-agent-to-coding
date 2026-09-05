package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMergeInstalledImageRef(t *testing.T) {
	img := &TenantInstalledImage{ImageURL: "registry.example/demo", Version: "v1.0"}
	if got := mergeInstalledImageRef(img); got != "registry.example/demo:v1.0" {
		t.Fatalf("got=%q", got)
	}
	img2 := &TenantInstalledImage{ImageURL: "registry.example/demo:latest", Version: "v2"}
	if got := mergeInstalledImageRef(img2); got != "registry.example/demo:latest" {
		t.Fatalf("already tagged got=%q", got)
	}
}

func TestHeuristicExtractEnvFromUserdata(t *testing.T) {
	ud := `
# comment
export FOO=bar
export BAZ="qux"
docker run -e HELLO=world --env OTHER=1 image
`
	env := heuristicExtractEnvFromUserdata(ud)
	if env["FOO"] != "bar" || env["BAZ"] != "qux" {
		t.Fatalf("export env=%v", env)
	}
	if env["HELLO"] != "world" || env["OTHER"] != "1" {
		t.Fatalf("dash-e env=%v", env)
	}
}

func TestInternalImageResolve(t *testing.T) {
	setupCloudTestDB(t)
	cfg.InternalSecret = ""
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,version,target_architectures,installed_at)
		VALUES('img1','t1','ext1','demo','registry.example/demo','v1.2','[]',NOW())`)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet,
		"/api/internal/image/resolve?tenant_id=t1&installed_image_id=img1", nil)
	rec := httptest.NewRecorder()
	handleInternalImageResolve(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["image"] != "registry.example/demo:v1.2" {
		t.Fatalf("image=%v", out["image"])
	}
	if out["external_image_id"] != "ext1" {
		t.Fatalf("ext=%v", out["external_image_id"])
	}
}
