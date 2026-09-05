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

func buildGzipTarWithFiles(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, content := range files {
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
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// combinedExtractTestServer serves a manifest with the given per-layer blob
// payloads; each payload is a gzip tar that may contain both wanted files.
func combinedExtractTestServer(t *testing.T, layers []map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/hello/manifests/latest", func(w http.ResponseWriter, r *http.Request) {
		var digests []map[string]any
		for i, _ := range layers {
			digests = append(digests, map[string]any{"digest": "sha256:layer" + string(rune('0'+i))})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"layers": digests,
			"config": map[string]any{"digest": "sha256:cfg"},
		})
	})
	for i, files := range layers {
		blob := buildGzipTarWithFiles(t, files)
		idx := i
		mux.HandleFunc("/v2/hello/blobs/sha256:layer"+string(rune('0'+idx)), func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(blob)
		})
	}
	return httptest.NewServer(mux)
}

func httpRegistryHost(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	prev := registryScheme
	registryScheme = "http"
	t.Cleanup(func() { registryScheme = prev })
	return strings.TrimPrefix(srv.URL, "http://")
}

func TestExtractAutoRunAndSkillsFromImageBothInSameLayer(t *testing.T) {
	srv := combinedExtractTestServer(t, []map[string]string{{
		"app/autoRunStep.md":  "# steps\n",
		"app/imageSkills.yaml": "version: 1\nskills:\n  - name: general-coding\n",
	}})
	defer srv.Close()

	got := extractAutoRunAndSkillsFromImage(srv.Client(), httpRegistryHost(t, srv)+"/hello:latest")
	if got.AutoRun.Status != "ok" || !strings.Contains(got.AutoRun.Markdown, "# steps") {
		t.Fatalf("auto status=%s detail=%s md=%q", got.AutoRun.Status, got.AutoRun.Detail, got.AutoRun.Markdown)
	}
	if got.Skills.Status != "ok" || len(got.Skills.List.Skills) != 1 {
		t.Fatalf("skills status=%s detail=%s json=%s", got.Skills.Status, got.Skills.Detail, got.Skills.JSON)
	}
	if got.Skills.Digest != got.AutoRun.Digest {
		t.Fatalf("digest mismatch auto=%s skills=%s", got.AutoRun.Digest, got.Skills.Digest)
	}
}

func TestExtractAutoRunAndSkillsFromImageSeparateLayers(t *testing.T) {
	srv := combinedExtractTestServer(t, []map[string]string{
		{"app/imageSkills.yaml": "version: 1\nskills:\n  - name: k8s-debug\n"}, // newest layer
		{"app/autoRunStep.md": "# older steps\n"},
	})
	defer srv.Close()

	got := extractAutoRunAndSkillsFromImage(srv.Client(), httpRegistryHost(t, srv)+"/hello:latest")
	if got.AutoRun.Status != "ok" || !strings.Contains(got.AutoRun.Markdown, "older steps") {
		t.Fatalf("auto status=%s detail=%s md=%q", got.AutoRun.Status, got.AutoRun.Detail, got.AutoRun.Markdown)
	}
	if got.Skills.Status != "ok" || len(got.Skills.List.Skills) != 1 || got.Skills.List.Skills[0].Name != "k8s-debug" {
		t.Fatalf("skills status=%s detail=%s json=%s", got.Skills.Status, got.Skills.Detail, got.Skills.JSON)
	}
}

func TestExtractAutoRunAndSkillsFromImageOnlyAutoRunPresent(t *testing.T) {
	srv := combinedExtractTestServer(t, []map[string]string{{
		"app/autoRunStep.md": "# only auto\n",
	}})
	defer srv.Close()

	got := extractAutoRunAndSkillsFromImage(srv.Client(), httpRegistryHost(t, srv)+"/hello:latest")
	if got.AutoRun.Status != "ok" || !strings.Contains(got.AutoRun.Markdown, "only auto") {
		t.Fatalf("auto status=%s detail=%s md=%q", got.AutoRun.Status, got.AutoRun.Detail, got.AutoRun.Markdown)
	}
	if got.Skills.Status != "not_found" {
		t.Fatalf("skills status=%s detail=%s (want not_found)", got.Skills.Status, got.Skills.Detail)
	}
}

func TestExtractAutoRunAndSkillsFromImageNonePresent(t *testing.T) {
	srv := combinedExtractTestServer(t, []map[string]string{{
		"app/readme.md": "nope",
	}})
	defer srv.Close()

	got := extractAutoRunAndSkillsFromImage(srv.Client(), httpRegistryHost(t, srv)+"/hello:latest")
	if got.AutoRun.Status != "not_found" || got.Skills.Status != "not_found" {
		t.Fatalf("auto=%s skills=%s (want both not_found)", got.AutoRun.Status, got.Skills.Status)
	}
}
