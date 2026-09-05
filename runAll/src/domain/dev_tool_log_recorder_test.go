package domain

import "testing"

func TestIsValidDevTool(t *testing.T) {
	tests := []struct {
		tool   string
		expect bool
	}{
		{ToolConfSync, true},
		{ToolConfSyncRemote, true},
		{ToolDbClear, true},
		{ToolDbInit, true},
		{ToolObservabilityClear, true},
		{"unknown", false},
		{"", false},
		{"conf-sync-extra", false},
	}

	for _, tt := range tests {
		got := IsValidDevTool(tt.tool)
		if got != tt.expect {
			t.Errorf("IsValidDevTool(%q) = %v, want %v", tt.tool, got, tt.expect)
		}
	}
}

func TestValidateDevTool(t *testing.T) {
	if err := ValidateDevTool(ToolDbClear); err != nil {
		t.Errorf("ValidateDevTool(%q) unexpected error: %v", ToolDbClear, err)
	}
	if err := ValidateDevTool("bad-tool"); err == nil {
		t.Error("ValidateDevTool(\"bad-tool\") expected error, got nil")
	}
}

func TestAllDevTools_HasFiveEntries(t *testing.T) {
	if len(AllDevTools) != 5 {
		t.Errorf("AllDevTools length = %d, want 5", len(AllDevTools))
	}
}

func TestAllDevTools_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, tool := range AllDevTools {
		if seen[tool] {
			t.Errorf("duplicate tool name in AllDevTools: %q", tool)
		}
		seen[tool] = true
	}
}
