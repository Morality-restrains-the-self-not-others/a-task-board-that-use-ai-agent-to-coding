package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParsePackageRefGitHub(t *testing.T) {
	ref, err := ParsePackageRef("github://task2money/daydaymoney-deploy/taskAuth@abc1234")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Scheme != "github" || ref.Owner != "task2money" || ref.Repo != "daydaymoney-deploy" || ref.Asset != "taskAuth" || ref.SHA != "abc1234" {
		t.Fatalf("ref %+v", ref)
	}
}

func TestParsePackageRefFile(t *testing.T) {
	ref, err := ParsePackageRef("file:///tmp/bins/taskAuth")
	if err != nil {
		t.Fatal(err)
	}
	if ref.Scheme != "file" || ref.FilePath != "/tmp/bins/taskAuth" {
		t.Fatalf("ref %+v", ref)
	}
}

func TestFileArtifactFetcherCopiesPin(t *testing.T) {
	src := filepath.Join(t.TempDir(), "taskAuth")
	if err := os.WriteFile(src, []byte("ELF"), 0o755); err != nil {
		t.Fatal(err)
	}
	pin := ArtifactPin{SHA: "abc", Package: "file://" + src}
	tmp, err := FileArtifactFetcher{}.Fetch(pin)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ELF" {
		t.Fatalf("got %q", got)
	}
}

func TestGitHubReleaseFetcherDownloadsAsset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing bearer, got %q", r.Header.Get("Authorization"))
		}
		if strings.Contains(r.URL.Path, "token") {
			t.Errorf("token leaked in URL %s", r.URL.Path)
		}
		if r.URL.Path != "/repos/task2money/daydaymoney-deploy/releases/tags/abc1234" && !strings.HasSuffix(r.URL.Path, "/taskAuth") {
			// first metadata then asset
		}
		if strings.Contains(r.URL.Path, "releases/tags/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"assets":[{"name":"taskAuth","url":"ASSET_URL"}]}`))
			return
		}
		_, _ = w.Write([]byte("BIN"))
	}))
	t.Cleanup(srv.Close)

	f := GitHubReleaseFetcher{
		APIBase: srv.URL,
		Token:   "test-token",
		Download: func(assetAPIURL, token string) (string, error) {
			if token != "test-token" {
				t.Fatalf("token %q", token)
			}
			tmp := filepath.Join(t.TempDir(), "dl")
			return tmp, os.WriteFile(tmp, []byte("BIN"), 0o755)
		},
	}
	// Replace ASSET_URL placeholder path: implement Fetch to call Download with asset url from JSON
	path, err := f.Fetch(ArtifactPin{SHA: "abc1234", Package: "github://task2money/daydaymoney-deploy/taskAuth@abc1234"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "BIN" {
		t.Fatalf("got %q", got)
	}
}

func TestGitHubReleaseFetcherUsesReleaseTagNotContentSHA(t *testing.T) {
	const tag = "deploy-20260831"
	payload := []byte("BIN")
	sum := sha256.Sum256(payload)
	contentSHA := hex.EncodeToString(sum[:])
	var lookedUp string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "releases/tags/") {
			lookedUp = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"assets":[{"name":"taskAuth","url":"http://example.invalid/taskAuth"}]}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	f := GitHubReleaseFetcher{
		APIBase: srv.URL,
		Download: func(assetAPIURL, token string) (string, error) {
			tmp := filepath.Join(t.TempDir(), "dl")
			return tmp, os.WriteFile(tmp, payload, 0o755)
		},
	}
	pin := ArtifactPin{
		SHA:     contentSHA,
		Package: "github://task2money/daydaymoney-deploy/taskAuth@" + tag,
	}
	path, err := f.Fetch(pin)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(lookedUp, "/releases/tags/"+tag) {
		t.Fatalf("lookup path %q: want tag %s not content sha", lookedUp, tag)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "BIN" {
		t.Fatalf("got %q", got)
	}
}

func TestDirectHTTPClientTimeoutAllowsLargeReleaseAssets(t *testing.T) {
	c := directHTTPClient()
	if c.Timeout < 30*time.Minute {
		t.Fatalf("timeout %s too short for ~1GB GitHub archive assets", c.Timeout)
	}
}

func TestDirectHTTPClientDisablesHTTP2(t *testing.T) {
	c := directHTTPClient()
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport %T", c.Transport)
	}
	if tr.ForceAttemptHTTP2 {
		t.Fatal("ForceAttemptHTTP2: GitHub PROTOCOL_ERROR requires HTTP/1.1")
	}
	if tr.TLSNextProto == nil {
		t.Fatal("nil TLSNextProto enables HTTP/2 automatically")
	}
	if len(tr.TLSNextProto) != 0 {
		t.Fatalf("TLSNextProto=%v want empty map", tr.TLSNextProto)
	}
}

func TestGitHubAssetDownloadLogsProgress(t *testing.T) {
	origBytes, origEvery := githubAssetProgressEveryBytes, githubAssetProgressEvery
	githubAssetProgressEveryBytes = 8
	githubAssetProgressEvery = time.Hour
	t.Cleanup(func() {
		githubAssetProgressEveryBytes = origBytes
		githubAssetProgressEvery = origEvery
	})

	var logBuf strings.Builder
	prevOut := log.Writer()
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(prevOut) })

	body := strings.Repeat("abcdefghij", 3) // 30 bytes → several 8-byte ticks
	const token = "ghp_progress_secret_token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Errorf("missing bearer")
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	path, err := downloadGitHubAsset(srv.URL, token)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("stored %d bytes want %d", len(got), len(body))
	}
	logs := logBuf.String()
	if !strings.Contains(logs, "bytes=") {
		t.Fatalf("progress log missing bytes=: %s", logs)
	}
	if strings.Contains(logs, token) || strings.Contains(logs, "Bearer") {
		t.Fatalf("token leaked in progress logs: %s", logs)
	}
}

func TestGitHubReleaseFetcherLogsMustNotContainToken(t *testing.T) {
	src, err := os.ReadFile("deploy_fetch.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(src), "\n") {
		trim := strings.TrimSpace(line)
		if !strings.Contains(trim, "log.") {
			continue
		}
		if strings.Contains(trim, "Token") || strings.Contains(trim, "Bearer") {
			t.Fatalf("must not log token: %s", trim)
		}
	}
}
