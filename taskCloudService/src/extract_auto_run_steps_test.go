package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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

func TestTarEntryMatchesWanted(t *testing.T) {
	if !tarEntryMatchesWanted("./app/autoRunStep.md", "/app/autoRunStep.md") {
		t.Fatal("expected match")
	}
	if tarEntryMatchesWanted("app/other.md", "/app/autoRunStep.md") {
		t.Fatal("expected no match")
	}
}

func TestNormalizeInImagePath(t *testing.T) {
	if got := normalizeInImagePath(""); got != defaultAutoRunStepsPath {
		t.Fatalf("got %s", got)
	}
	if got := normalizeInImagePath("app/autoRunStep.md"); got != "/app/autoRunStep.md" {
		t.Fatalf("got %s", got)
	}
}
