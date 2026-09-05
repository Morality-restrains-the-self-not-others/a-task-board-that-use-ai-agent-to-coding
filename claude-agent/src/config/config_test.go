package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandEnvVars(t *testing.T) {
	os.Setenv("TEST_VAR", "expanded")
	defer os.Unsetenv("TEST_VAR")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple env", "prefix_${TEST_VAR}_suffix", "prefix_expanded_suffix"},
		{"with default (var set)", "${TEST_VAR:fallback}", "expanded"},
		{"with default (var unset)", "${MISSING_VAR:fallback}", "fallback"},
		{"no default (var unset)", "${MISSING_VAR}", "${MISSING_VAR}"},
		{"plain string", "no vars here", "no vars here"},
		{"multiple vars", "${TEST_VAR}_${TEST_VAR}", "expanded_expanded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnvVars(tt.input)
			if got != tt.expected {
				t.Errorf("expandEnvVars(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "claude_config.yaml")

	yamlContent := `
model_providers:
  anthropic:
    api_key: "sk-test-key"
    provider: anthropic

models:
  default_model:
    model_provider: anthropic
    model: claude-sonnet-4-20250514
    max_tokens: 4096
    temperature: 0.5

agents:
  claude_agent:
    model: default_model
    max_steps: 100
    working_dir: ""
    enable_trajectory: true
    enable_docker: false
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.ModelProviders["anthropic"].Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic", cfg.ModelProviders["anthropic"].Provider)
	}

	resolved, err := cfg.Resolve(nil)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if resolved.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want claude-sonnet-4-20250514", resolved.Model)
	}
	if resolved.MaxSteps != 100 {
		t.Errorf("max_steps = %d, want 100", resolved.MaxSteps)
	}
	if resolved.Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic", resolved.Provider)
	}
}

func TestLoadConfigWithEnvVarExpansion(t *testing.T) {
	os.Setenv("MY_API_KEY", "sk-env-key")
	defer os.Unsetenv("MY_API_KEY")

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "claude_config.yaml")

	yamlContent := `
model_providers:
  anthropic:
    api_key: "${MY_API_KEY}"
    provider: anthropic

models:
  default_model:
    model_provider: anthropic
    model: claude-sonnet-4-20250514
    max_tokens: 4096

agents:
  claude_agent:
    model: default_model
    max_steps: 50
    enable_trajectory: true
    enable_docker: false
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.ModelProviders["anthropic"].APIKey != "sk-env-key" {
		t.Errorf("api_key = %q, want sk-env-key", cfg.ModelProviders["anthropic"].APIKey)
	}
}

func TestResolvePriorityCLI(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "claude_config.yaml")

	yamlContent := `
model_providers:
  anthropic:
    api_key: "sk-config-key"
    provider: anthropic

models:
  default_model:
    model_provider: anthropic
    model: claude-sonnet-4-20250514
    max_tokens: 4096

agents:
  claude_agent:
    model: default_model
    max_steps: 100
    enable_trajectory: true
    enable_docker: false
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// CLI overrides should take priority
	resolved, err := cfg.Resolve(&ResolvedConfig{
		Model:    "claude-opus-4-20250514",
		MaxSteps: 200,
		APIKey:   "sk-cli-key",
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if resolved.Model != "claude-opus-4-20250514" {
		t.Errorf("model = %q, want claude-opus-4-20250514", resolved.Model)
	}
	if resolved.MaxSteps != 200 {
		t.Errorf("max_steps = %d, want 200", resolved.MaxSteps)
	}
	if resolved.APIKey != "sk-cli-key" {
		t.Errorf("api_key = %q, want sk-cli-key", resolved.APIKey)
	}
}

func TestLoadConfigMissingAgents(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "bad_config.yaml")

	yamlContent := `
model_providers:
  anthropic:
    api_key: "sk-key"
    provider: anthropic
models:
  default_model:
    model_provider: anthropic
    model: claude-sonnet-4-20250514
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Error("expected error for missing agents section")
	}
}
