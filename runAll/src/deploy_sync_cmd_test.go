package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunDeploySyncUsesFileFetcherAndSkipsCompile(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(t.TempDir(), "taskAuth")
	if err := os.WriteFile(src, []byte("NEW"), 0o755); err != nil {
		t.Fatal(err)
	}
	relPath := filepath.Join(root, "releases.yaml")
	body := "artifacts:\n  taskAuth:\n    sha: abc1234\n    package: file://" + src + "\n"
	if err := os.WriteFile(relPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RunDeploySync(root, relPath, FileArtifactFetcher{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "bin", "taskAuth"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "NEW" {
		t.Fatalf("got %q", got)
	}
}
