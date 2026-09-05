package agent

import (
	"testing"

	"claudeAgent/src/config"
)

// 规则 41 回归：Run() 透传 config 的 APIKey / BaseURL（src/process 侧有注入单测，
// 此处验证构造与字段接线）。BaseURL 缺失曾导致 claude CLI 打官方端点 403。
func TestNewClaudeAgentCarriesAPIKey(t *testing.T) {
	cfg := &config.ResolvedConfig{
		Model:          "claude-sonnet-5",
		APIKey:         "sk-config-key-123",
		BaseURL:        "https://api.deepseek.com/anthropic",
		MaxSteps:       5,
		PermissionMode: "skip",
		WorkingDir:     t.TempDir(),
	}
	a, err := NewClaudeAgent(cfg, nil)
	if err != nil {
		t.Fatalf("NewClaudeAgent: %v", err)
	}
	if a.cfg == nil || a.cfg.APIKey != "sk-config-key-123" {
		t.Fatalf("agent cfg APIKey = %v, want config value carried through", a.cfg)
	}
	if a.cfg.BaseURL != "https://api.deepseek.com/anthropic" {
		t.Errorf("BaseURL not carried: %q", a.cfg.BaseURL)
	}
	if a.cfg.PermissionMode != "skip" {
		t.Errorf("PermissionMode not carried: %q", a.cfg.PermissionMode)
	}
}
