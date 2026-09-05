package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeArchitecture(t *testing.T) {
	cases := map[string]string{
		"amd64":   "x86_64",
		"x86_64":  "x86_64",
		"aarch64": "arm64",
		"arm64":   "arm64",
		"":        "",
	}
	for in, want := range cases {
		if got := NormalizeArchitecture(in); got != want {
			t.Fatalf("%q -> %q want %q", in, got, want)
		}
	}
}

func TestMergeImageURLWithVersion(t *testing.T) {
	got, err := MergeImageURLWithVersion("example.com/app", "1.2.3")
	if err != nil || got != "example.com/app:1.2.3" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	got, err = MergeImageURLWithVersion("example.com/app:v1", "2.0")
	if err != nil || got != "example.com/app:v1" {
		t.Fatalf("tag preserved got=%q", got)
	}
}

func TestResolveContainerImageMetadataManifestList(t *testing.T) {
	prev := registryScheme
	registryScheme = "http"
	t.Cleanup(func() { registryScheme = prev })

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/hello/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "manifest.list") || strings.Contains(accept, "image.index") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"manifests": []map[string]any{
					{"digest": "sha256:amd", "size": 10, "platform": map[string]any{"architecture": "amd64"}},
					{"digest": "sha256:arm", "size": 11, "platform": map[string]any{"architecture": "arm64"}},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"layers": []map[string]any{{"size": 100}, {"size": 50}},
		})
	})
	mux.HandleFunc("/v2/hello/manifests/sha256:amd", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"layers": []map[string]any{{"size": 100}, {"size": 50}},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	meta, err := resolveContainerImageMetadata(srv.Client(), host+"/hello:latest")
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.TargetArchitectures) != 2 {
		t.Fatalf("arch=%v", meta.TargetArchitectures)
	}
	if meta.Size == nil || *meta.Size != 150 {
		t.Fatalf("size=%v", meta.Size)
	}
}

// Regression: shell HTTPS_PROXY=socks5h://127.0.0.1:1234 (dead) must not break registry GETs.
func TestDefaultRegistryClientIgnoresEnvProxy(t *testing.T) {
	prev := registryScheme
	registryScheme = "http"
	t.Cleanup(func() { registryScheme = prev })

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/hello/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"manifests": []map[string]any{
				{"digest": "sha256:amd", "size": 10, "platform": map[string]any{"architecture": "amd64"}},
			},
		})
	})
	mux.HandleFunc("/v2/hello/manifests/sha256:amd", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"layers": []map[string]any{{"size": 42}},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	t.Setenv("http_proxy", "http://127.0.0.1:1")
	t.Setenv("https_proxy", "http://127.0.0.1:1")
	t.Setenv("ALL_PROXY", "socks5h://127.0.0.1:1")
	t.Setenv("all_proxy", "socks5h://127.0.0.1:1")
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	host := strings.TrimPrefix(srv.URL, "http://")
	meta, err := ResolveContainerImageMetadata(host + "/hello:latest")
	if err != nil {
		t.Fatalf("default client must ignore dead env proxy: %v", err)
	}
	if len(meta.TargetArchitectures) != 1 || meta.TargetArchitectures[0] != "x86_64" {
		t.Fatalf("arch=%v", meta.TargetArchitectures)
	}
}
