package infrastructure

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func buildGzipTarWithFile(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Name: name,
		Mode: 0o644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFindAutoRunStepsInLayerBlob(t *testing.T) {
	blob := buildGzipTarWithFile(t, "app/autoRunStep.md", "# hello steps\n")
	md, found, err := findAutoRunStepsInLayerBlob(blob, "/app/autoRunStep.md")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected file found")
	}
	if !strings.Contains(md, "hello steps") {
		t.Fatalf("unexpected markdown: %q", md)
	}
}

func TestFindAutoRunStepsInLayerBlobNotFound(t *testing.T) {
	blob := buildGzipTarWithFile(t, "app/other.md", "x")
	_, found, err := findAutoRunStepsInLayerBlob(blob, "/app/autoRunStep.md")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("expected not found")
	}
}

func TestExtractAutoRunStepsFromImageOK(t *testing.T) {
	prev := registryScheme
	registryScheme = "http"
	t.Cleanup(func() { registryScheme = prev })

	layer := buildGzipTarWithFile(t, "app/autoRunStep.md", "# auto run from image\n")
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
	got := extractAutoRunStepsFromImageWithClient(srv.Client(), host+"/hello:latest")
	if got.Status != "ok" {
		t.Fatalf("status=%s detail=%s", got.Status, got.Detail)
	}
	if !strings.Contains(got.Markdown, "auto run from image") {
		t.Fatalf("markdown=%q", got.Markdown)
	}
}

func TestExtractAutoRunStepsFromImageNotFound(t *testing.T) {
	prev := registryScheme
	registryScheme = "http"
	t.Cleanup(func() { registryScheme = prev })

	layer := buildGzipTarWithFile(t, "app/readme.md", "nope")
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
	got := extractAutoRunStepsFromImageWithClient(srv.Client(), host+"/hello:latest")
	if got.Status != "not_found" {
		t.Fatalf("status=%s detail=%s", got.Status, got.Detail)
	}
}
