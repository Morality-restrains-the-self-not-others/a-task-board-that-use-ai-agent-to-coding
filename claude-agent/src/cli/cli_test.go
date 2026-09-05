package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFirst(t *testing.T) {
	tests := []struct {
		name     string
		vals     []string
		expected string
	}{
		{"first non-empty", []string{"", "b", "c"}, "b"},
		{"all empty", []string{"", "", ""}, ""},
		{"first set", []string{"a", "b", "c"}, "a"},
		{"only one", []string{"x"}, "x"},
		{"empty list", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := first(tt.vals...)
			if got != tt.expected {
				t.Errorf("first(%v) = %q, want %q", tt.vals, got, tt.expected)
			}
		})
	}
}

func TestResolveConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .yaml config
	yamlPath := filepath.Join(tmpDir, "claude_config.yaml")
	if err := os.WriteFile(yamlPath, []byte("agents: {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test exact path
	found, err := resolveConfigFile(yamlPath)
	if err != nil {
		t.Fatalf("resolveConfigFile(%q) error: %v", yamlPath, err)
	}
	if found != yamlPath {
		t.Errorf("found = %q, want %q", found, yamlPath)
	}

	// Test missing file
	_, err = resolveConfigFile(filepath.Join(tmpDir, "nonexistent.yaml"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestDefaultConfigFile(t *testing.T) {
	// Without env var, should return default
	result := defaultConfigFile()
	if result != "claude_config.yaml" {
		t.Errorf("defaultConfigFile() = %q, want claude_config.yaml", result)
	}

	// With env var
	os.Setenv("CLAUDE_CONFIG_FILE", "/custom/path.yaml")
	defer os.Unsetenv("CLAUDE_CONFIG_FILE")

	result = defaultConfigFile()
	if result != "/custom/path.yaml" {
		t.Errorf("defaultConfigFile() = %q, want /custom/path.yaml", result)
	}
}

func TestResolveConfigFileWithExtension(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .yml config (no .yaml)
	ymlPath := filepath.Join(tmpDir, "claude_config.yml")
	if err := os.WriteFile(ymlPath, []byte("agents: {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Try resolving without extension
	basePath := filepath.Join(tmpDir, "claude_config")
	found, err := resolveConfigFile(basePath)
	if err != nil {
		t.Fatalf("resolveConfigFile(%q) error: %v", basePath, err)
	}
	if found != basePath+".yml" {
		t.Errorf("found = %q, want %q", found, basePath+".yml")
	}
}
