package main

import (
	"encoding/json"
	"os"
	"testing"
)

// sharedArchFixture is loaded from shareLib/archfixture/testdata — the single
// source of truth also consumed by taskCloudService (Go) and taskFE (JS) tests,
// so the three extractCPUArchitecturesFromText / inferInstanceArchitecture
// implementations cannot silently drift (OPT-20260821-007).
type sharedArchFixture struct {
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

func loadSharedArchFixture(t *testing.T) sharedArchFixture {
	t.Helper()
	data, err := os.ReadFile("../../shareLib/archfixture/testdata/instance_architecture_cases.json")
	if err != nil {
		t.Fatalf("read shared arch fixture: %v", err)
	}
	var fx sharedArchFixture
	if err := json.Unmarshal(data, &fx); err != nil {
		t.Fatalf("unmarshal shared arch fixture: %v", err)
	}
	return fx
}

func TestExtractCPUArchitecturesFromTextSharedFixture(t *testing.T) {
	fx := loadSharedArchFixture(t)
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
	fx := loadSharedArchFixture(t)
	for _, tc := range fx.InferInstanceArchitectureCases {
		if got := inferInstanceArchitecture(tc.InstanceType); got != tc.Want {
			t.Fatalf("inferInstanceArchitecture(%q) = %q, want %q", tc.InstanceType, got, tc.Want)
		}
	}
}

func TestKnownCPUArchitecturesDropsUnknown(t *testing.T) {
	got := knownCPUArchitectures([]string{"x86_64", "unknown", "amd64", "riscv64"})
	if len(got) != 1 || got[0] != "x86_64" {
		t.Fatalf("got %v, want [x86_64]", got)
	}
}

func TestExtractCPUArchitecturesFromText(t *testing.T) {
	got := extractCPUArchitecturesFromText("private_x86_64-latest", "trae-agent", "")
	if len(got) != 1 || got[0] != "x86_64" {
		t.Fatalf("private_x86_64-latest: got %v", got)
	}
	got = extractCPUArchitecturesFromText("linux/arm64", "demo", "registry.example/app:aarch64")
	if len(got) != 1 || got[0] != "arm64" {
		t.Fatalf("arm tokens: got %v", got)
	}
	if got := extractCPUArchitecturesFromText("latest", "trae-agent"); len(got) != 0 {
		t.Fatalf("no hint should be empty, got %v", got)
	}
}

func TestFormatImageRequiredAndInstanceSupportedArch(t *testing.T) {
	if got := formatImageRequiredArch(nil, nil); got != "未声明" {
		t.Fatalf("empty: %q", got)
	}
	if got := formatImageRequiredArch(nil, []string{"unknown"}); got != "unknown（无法识别为 x86_64/arm64）" {
		t.Fatalf("raw unknown: %q", got)
	}
	if got := formatImageRequiredArch([]string{"arm64"}, []string{"aarch64"}); got != "arm64" {
		t.Fatalf("canonical: %q", got)
	}
	got := formatInstanceSupportedArch("ecs.c6.large")
	if got != "x86_64（实例规格 ecs.c6.large）" {
		t.Fatalf("instance: %q", got)
	}
}
