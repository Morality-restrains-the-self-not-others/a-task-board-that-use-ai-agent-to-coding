package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReleasesRequiresArtifactPin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "releases.yaml")
	if err := os.WriteFile(path, []byte("artifacts: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadReleases(path)
	if err == nil {
		t.Fatal("expected error for empty artifacts")
	}
}

func TestLoadReleasesParsesArchivePin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "releases.yaml")
	body := "artifacts:\n  taskFE-dist:\n    sha: abc\n    package: github://o/r/taskFE-dist.tar.gz@tag\n    dest: taskFE/app/public\n    unpack: tar.gz\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rel, err := LoadReleases(path)
	if err != nil {
		t.Fatal(err)
	}
	pin := rel.Artifacts["taskFE-dist"]
	if pin.Dest != "taskFE/app/public" || pin.Unpack != "tar.gz" {
		t.Fatalf("pin %+v", pin)
	}
}

func TestLoadReleasesParsesPins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "releases.yaml")
	body := "artifacts:\n  taskAuth:\n    sha: abc1234\n    package: github://task2money/daydaymoney-deploy/taskAuth@abc1234\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rel, err := LoadReleases(path)
	if err != nil {
		t.Fatal(err)
	}
	pin, ok := rel.Artifacts["taskAuth"]
	if !ok || pin.SHA != "abc1234" {
		t.Fatalf("pin: %+v ok=%v", pin, ok)
	}
}

func TestSyncPinnedArtifactsDoesNotOverwriteLastGoodOnFetchError(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(binDir, "taskAuth")
	old := []byte("LAST-GOOD-ELF")
	if err := os.WriteFile(binPath, old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binPath+".sha", []byte("oldsha\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rel := &ReleasesFile{Artifacts: map[string]ArtifactPin{
		"taskAuth": {SHA: "newsha", Package: "github://example/taskAuth@newsha"},
	}}
	err := SyncPinnedArtifacts(root, rel, ArtifactFetcherFunc(func(pin ArtifactPin) (string, error) {
		return "", errors.New("network down")
	}))
	if err == nil {
		t.Fatal("expected fetch error")
	}
	got, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, old) {
		t.Fatalf("last-good overwritten: %q", got)
	}
}

func TestSyncPinnedArtifactsWritesOnSuccess(t *testing.T) {
	root := t.TempDir()
	payload := []byte("NEW-ELF")
	rel := &ReleasesFile{Artifacts: map[string]ArtifactPin{
		"taskAuth": {SHA: "abc1234", Package: "github://example/taskAuth@abc1234"},
	}}
	err := SyncPinnedArtifacts(root, rel, ArtifactFetcherFunc(func(pin ArtifactPin) (string, error) {
		tmp := filepath.Join(t.TempDir(), "dl")
		if err := os.WriteFile(tmp, payload, 0o755); err != nil {
			return "", err
		}
		return tmp, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "bin", "taskAuth"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("got %q", got)
	}
	sha, err := os.ReadFile(filepath.Join(root, "bin", "taskAuth.sha"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(sha)) != "abc1234" {
		t.Fatalf("sha sidecar %q", sha)
	}
}

func TestSyncPinnedArtifactsNeverInvokesGoBuild(t *testing.T) {
	src, err := os.ReadFile("deploy_pin.go")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(src, []byte("go build")) {
		t.Fatal("deploy_pin.go must not invoke go build")
	}
	if bytes.Contains(src, []byte("exec.Command")) {
		t.Fatal("deploy_pin.go must not exec compilers")
	}
	arch, err := os.ReadFile("deploy_archive.go")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(arch, []byte("exec.Command")) {
		t.Fatal("deploy_archive.go must not exec")
	}
	var f ArtifactFetcher = ArtifactFetcherFunc(func(ArtifactPin) (string, error) {
		return "", errors.New("unused")
	})
	if f == nil {
		t.Fatal("fetcher required")
	}
}

func TestSyncPinnedArtifactsSkipsWhenShaMatches(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(binDir, "taskAuth")
	if err := os.WriteFile(binPath, []byte("KEEP"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binPath+".sha", []byte("abc1234\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fetches := 0
	rel := &ReleasesFile{Artifacts: map[string]ArtifactPin{
		"taskAuth": {SHA: "abc1234", Package: "github://example/taskAuth@abc1234"},
	}}
	err := SyncPinnedArtifacts(root, rel, ArtifactFetcherFunc(func(pin ArtifactPin) (string, error) {
		fetches++
		return "", errors.New("must not fetch")
	}))
	if err != nil {
		t.Fatal(err)
	}
	if fetches != 0 {
		t.Fatalf("idempotent skip expected 0 fetches, got %d", fetches)
	}
}
