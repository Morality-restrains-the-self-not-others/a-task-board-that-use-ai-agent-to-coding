package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeTarGz(t *testing.T, files map[string][]byte) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "a.tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPinDestPathRejectsDotDot(t *testing.T) {
	_, err := pinDestPath(t.TempDir(), "x", ArtifactPin{Dest: "../escape"})
	if err == nil {
		t.Fatal("expected dest rejection")
	}
}

func TestPinDestPathJoinsRelative(t *testing.T) {
	root := t.TempDir()
	got, err := pinDestPath(root, "taskEvents-bin", ArtifactPin{Dest: "taskEvents/bin"})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "taskEvents", "bin")
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestSyncPinnedArtifactsRestoresRelativeSymlink(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte("<html/>")
	hdr := &tar.Header{Name: "releases/build/index.html", Mode: 0o644, Size: int64(len(body))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	link := &tar.Header{Name: "html", Mode: 0o777, Typeflag: tar.TypeSymlink, Linkname: "releases/build"}
	if err := tw.WriteHeader(link); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "fe.tar.gz")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	rel := &ReleasesFile{Artifacts: map[string]ArtifactPin{
		"taskFE-dist": {SHA: "s", Package: "file://" + archive, Dest: "taskFE/app/public", Unpack: "tar.gz"},
	}}
	if err := SyncPinnedArtifacts(root, rel, FileArtifactFetcher{}); err != nil {
		t.Fatal(err)
	}
	html := filepath.Join(root, "taskFE", "app", "public", "html")
	got, err := os.Readlink(html)
	if err != nil {
		t.Fatal(err)
	}
	if got != "releases/build" {
		t.Fatalf("link %q", got)
	}
	index := filepath.Join(root, "taskFE", "app", "public", "html", "index.html")
	b, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "<html/>" {
		t.Fatalf("index %q", b)
	}
}

func TestSyncPinnedArtifactsUnpacksTarGzToDest(t *testing.T) {
	root := t.TempDir()
	archive := writeTarGz(t, map[string][]byte{
		"company_created/1_set/worker": []byte("ELF"),
	})
	rel := &ReleasesFile{Artifacts: map[string]ArtifactPin{
		"taskEvents-bin": {
			SHA:     "abc1234",
			Package: "file://" + archive,
			Dest:    "taskEvents/bin",
			Unpack:  "tar.gz",
		},
	}}
	if err := SyncPinnedArtifacts(root, rel, FileArtifactFetcher{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "taskEvents", "bin", "company_created", "1_set", "worker"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ELF" {
		t.Fatalf("got %q", got)
	}
	sha, err := os.ReadFile(filepath.Join(root, "artifacts", "taskEvents-bin.sha"))
	if err != nil {
		t.Fatal(err)
	}
	if string(bytes.TrimSpace(sha)) != "abc1234" {
		t.Fatalf("sidecar %q", sha)
	}
}

func TestSyncPinnedArtifactsUnpackRejectsTarSlip(t *testing.T) {
	root := t.TempDir()
	archive := writeTarGz(t, map[string][]byte{
		"../escape": []byte("NO"),
	})
	rel := &ReleasesFile{Artifacts: map[string]ArtifactPin{
		"bad": {SHA: "x", Package: "file://" + archive, Dest: "taskEvents/bin", Unpack: "tar.gz"},
	}}
	if err := SyncPinnedArtifacts(root, rel, FileArtifactFetcher{}); err == nil {
		t.Fatal("expected tar slip error")
	}
	if _, err := os.Stat(filepath.Join(root, "escape")); err == nil {
		t.Fatal("tar slip wrote outside dest")
	}
}

func TestSyncPinnedArtifactsKeepsDestOnFetchError(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "taskFE", "app", "public")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(dest, "index.html")
	if err := os.WriteFile(index, []byte("KEEP"), 0o644); err != nil {
		t.Fatal(err)
	}
	rel := &ReleasesFile{Artifacts: map[string]ArtifactPin{
		"taskFE-dist": {SHA: "new", Package: "github://x/y/z@tag", Dest: "taskFE/app/public", Unpack: "tar.gz"},
	}}
	err := SyncPinnedArtifacts(root, rel, ArtifactFetcherFunc(func(ArtifactPin) (string, error) {
		return "", errors.New("network down")
	}))
	if err == nil {
		t.Fatal("expected fetch error")
	}
	got, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "KEEP" {
		t.Fatalf("last-good lost: %q", got)
	}
}
