package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRunAllServiceNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runall.yaml")
	content := `
version: "1"
groups:
  - name: platform
    services:
      - name: saas-backend
        command: "true"
        health_check:
          url: "http://127.0.0.1:1"
      - name: git-oauth
        command: "true"
        health_check:
          url: "http://127.0.0.1:2"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	names, err := LoadRunAllServiceNames(path)
	if err != nil {
		t.Fatal(err)
	}
	if !names["saas-backend"] || !names["git-oauth"] {
		t.Fatalf("names = %v", names)
	}
	if !names["runall"] {
		t.Fatalf("expected synthetic runall provider, names = %v", names)
	}
}
