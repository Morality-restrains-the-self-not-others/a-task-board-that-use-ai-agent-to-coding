package main

import (
	"encoding/json"
	"os"
	"testing"
)

// sharedArchFixture is loaded from shareLib/archfixture/testdata — the single
// source of truth also consumed by taskProjectService (Go) and taskFE (JS) tests,
// so the three extractCPUArchitecturesFromText implementations cannot silently
// drift (OPT-20260821-007).
type cloudSharedArchFixture struct {
	ExtractFromTextCases []struct {
		Name   string   `json:"name"`
		Inputs []string `json:"inputs"`
		Want   []string `json:"want"`
	} `json:"extract_from_text_cases"`
	InferInstanceArchitectureCases []struct {
		InstanceType string `json:"instance_type"`
		Want         string `json:"want"`
	} `json:"infer_instance_architecture_cases"`
}

func loadCloudSharedArchFixture(t *testing.T) cloudSharedArchFixture {
	t.Helper()
	data, err := os.ReadFile("../../shareLib/archfixture/testdata/instance_architecture_cases.json")
	if err != nil {
		t.Fatalf("read shared arch fixture: %v", err)
	}
	var fx cloudSharedArchFixture
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatalf("unmarshal shared arch fixture: %v", err)
	}
	return fx
}

func TestExtractCPUArchitecturesFromTextSharedFixture(t *testing.T) {
	fx := loadCloudSharedArchFixture(t)
	if len(fx.ExtractFromTextCases) == 0 {
		t.Fatal("shared arch fixture extract_from_text_cases must not be empty")
	}
	for _, tc := range fx.ExtractFromTextCases {
		got := extractCPUArchitecturesFromText(tc.Inputs...)
		if len(got) != len(tc.Want) {
			t.Fatalf("extractCPUArchitecturesFromText(%q) = %v, want %v", tc.Inputs, got, tc.Want)
		}
		for i := range got {
			if got[i] != tc.Want[i] {
				t.Fatalf("extractCPUArchitecturesFromText(%q) = %v, want %v", tc.Inputs, got, tc.Want)
			}
		}
	}
}

func TestInferInstanceArchitectureSharedFixture(t *testing.T) {
	fx := loadCloudSharedArchFixture(t)
	for _, tc := range fx.InferInstanceArchitectureCases {
		if got := inferInstanceArchitecture(tc.InstanceType); got != tc.Want {
			t.Fatalf("inferInstanceArchitecture(%q) = %q, want %q", tc.InstanceType, got, tc.Want)
		}
	}
}
