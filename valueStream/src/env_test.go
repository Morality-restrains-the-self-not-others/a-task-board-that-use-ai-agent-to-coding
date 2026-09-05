package main

import (
	"os"
	"strings"
	"testing"
)

func TestMergeEnv_OverridesExistingKey(t *testing.T) {
	base := []string{"DJANGO_SETTINGS_MODULE=old", "PATH=/bin"}
	extra := map[string]string{"DJANGO_SETTINGS_MODULE": "saas_project.settings_test"}
	out := mergeEnv(base, extra)
	for _, e := range out {
		if e == "DJANGO_SETTINGS_MODULE=old" {
			t.Fatal("old value should be removed")
		}
	}
	found := false
	for _, e := range out {
		if e == "DJANGO_SETTINGS_MODULE=saas_project.settings_test" {
			found = true
		}
	}
	if !found {
		t.Fatalf("env = %v", out)
	}
	if !containsPrefix(out, "PATH=") {
		t.Fatal("PATH should remain")
	}
}

func containsPrefix(ss []string, prefix string) bool {
	for _, s := range ss {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func TestMergeEnv_EmptyExtraReturnsBase(t *testing.T) {
	base := os.Environ()
	out := mergeEnv(base, nil)
	if len(out) != len(base) {
		t.Fatalf("len %d vs %d", len(out), len(base))
	}
}
