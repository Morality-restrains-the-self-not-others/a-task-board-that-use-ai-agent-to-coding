package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractImageSkillsFromImageOK(t *testing.T) {
	prev := registryScheme
	registryScheme = "http"
	t.Cleanup(func() { registryScheme = prev })

	yaml := "version: 1\nskills:\n  - name: general-coding\n    description: d\n  - name: k8s-debug\n"
	layer := buildGzipTarWithFile(t, "app/imageSkills.yaml", yaml)
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/hello/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"layers": []map[string]any{{"digest": "sha256:layer1"}},
			"config": map[string]any{"digest": "sha256:cfg"},
		})
	})
	mux.HandleFunc("/v2/hello/blobs/sha256:layer1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(layer)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	got := extractImageSkillsFromImage(srv.Client(), host+"/hello:latest")
	if got.Status != "ok" {
		t.Fatalf("status=%s detail=%s", got.Status, got.Detail)
	}
	if got.List.DefaultSkill != "general-coding" || len(got.List.Skills) != 2 {
		t.Fatalf("list=%+v", got.List)
	}
	if !strings.Contains(got.JSON, "k8s-debug") {
		t.Fatalf("json=%s", got.JSON)
	}
}

func TestExtractImageSkillsFromImageInvalidYAMLFailed(t *testing.T) {
	prev := registryScheme
	registryScheme = "http"
	t.Cleanup(func() { registryScheme = prev })

	layer := buildGzipTarWithFile(t, "app/imageSkills.yaml", "version: 1\nskills:\n  - name: NOT_VALID\n")
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/hello/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"layers": []map[string]any{{"digest": "sha256:layer1"}},
		})
	})
	mux.HandleFunc("/v2/hello/blobs/sha256:layer1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(layer)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	got := extractImageSkillsFromImage(srv.Client(), host+"/hello:latest")
	if got.Status != "failed" {
		t.Fatalf("status=%s detail=%s", got.Status, got.Detail)
	}
}
