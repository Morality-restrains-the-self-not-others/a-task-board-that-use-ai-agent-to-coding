package registryhost

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

// sharedCase is one row of testdata/registry_cases.json — the single source of
// truth shared with the JS frontend helper test
// (taskAiProvider/frontend/tests/privateRegistryHint.unit.test.js) so the Go and
// JS host rules cannot silently drift.
type sharedCase struct {
	Registry             string `json:"registry"`
	WantPrivate          bool   `json:"want_private"`
	WantPublicSuggestion string `json:"want_public_suggestion"`
}

func loadSharedCases(t *testing.T) []sharedCase {
	t.Helper()
	data, err := os.ReadFile("testdata/registry_cases.json")
	if err != nil {
		t.Fatalf("read shared registry cases: %v", err)
	}
	var table struct {
		Cases []sharedCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &table); err != nil {
		t.Fatalf("parse shared registry cases: %v", err)
	}
	if len(table.Cases) == 0 {
		t.Fatal("shared registry cases must not be empty")
	}
	return table.Cases
}

func TestIsCloudVendorPrivateRegistryHost(t *testing.T) {
	for _, tc := range loadSharedCases(t) {
		if got := IsCloudVendorPrivateRegistryHost(tc.Registry); got != tc.WantPrivate {
			t.Fatalf("IsCloudVendorPrivateRegistryHost(%q) = %v, want %v", tc.Registry, got, tc.WantPrivate)
		}
	}
}

func TestPrivateRegistryUserMessageSuggestsAliyunPublic(t *testing.T) {
	for _, tc := range loadSharedCases(t) {
		msg := PrivateRegistryUserMessage(tc.Registry)
		if !strings.Contains(msg, "无法触及") {
			t.Fatalf("missing hint for %q: %s", tc.Registry, msg)
		}
		if tc.WantPublicSuggestion == "" {
			continue
		}
		if !strings.Contains(msg, tc.WantPublicSuggestion) {
			t.Fatalf("missing public mapping for %q: %s (want %s)", tc.Registry, msg, tc.WantPublicSuggestion)
		}
	}
}

func TestRejectPrivateRegistry(t *testing.T) {
	if err := RejectPrivateRegistry("registry.cn-qingdao.aliyuncs.com"); err != nil {
		t.Fatalf("public must pass: %v", err)
	}
	err := RejectPrivateRegistry("registry-vpc.cn-qingdao.aliyuncs.com/ruandao/task2app-trae")
	if err == nil {
		t.Fatal("vpc must reject")
	}
	var pr *ErrPrivateRegistry
	if !errors.As(err, &pr) {
		t.Fatalf("want ErrPrivateRegistry, got %T", err)
	}
}

func TestAnnotateRegistryFetchErrorPreservesErrPrivateRegistry(t *testing.T) {
	orig := RejectPrivateRegistry("registry-vpc.cn-qingdao.aliyuncs.com")
	if err := AnnotateRegistryFetchError(orig, "registry-vpc.cn-qingdao.aliyuncs.com"); err != orig {
		t.Fatalf("private error must pass through unchanged: %v", err)
	}
}
