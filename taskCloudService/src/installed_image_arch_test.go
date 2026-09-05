package main

import "testing"

func TestArchitecturesFromImageDropsUnknownAndInfersVersion(t *testing.T) {
	got := architecturesFromImage(map[string]interface{}{
		"target_architectures": []interface{}{"x86_64", "unknown"},
		"version":              "latest",
	})
	if len(got) != 1 || got[0] != "x86_64" {
		t.Fatalf("declared+unknown: got %v", got)
	}
	got = architecturesFromImage(map[string]interface{}{
		"target_architectures": []interface{}{},
		"version":              "private_x86_64-latest",
		"name":                 "trae-agent",
	})
	if len(got) != 1 || got[0] != "x86_64" {
		t.Fatalf("empty declared should infer version: got %v", got)
	}
}

func TestInstalledImageToJSONFillsArchFromVersion(t *testing.T) {
	out := installedImageToJSON(TenantInstalledImage{
		ID:                  "1",
		Name:                "trae-agent",
		Version:             "private_x86_64-latest",
		TargetArchitectures: []string{},
	})
	arches, _ := out["target_architectures"].([]string)
	if len(arches) != 1 || arches[0] != "x86_64" {
		t.Fatalf("json arches=%v (%T)", out["target_architectures"], out["target_architectures"])
	}
}

func TestInstalledImageToJSONKeepsUnrecognizedWhenUninferable(t *testing.T) {
	out := installedImageToJSON(TenantInstalledImage{
		ID:                  "1",
		Name:                "trae-agent",
		Version:             "latest",
		TargetArchitectures: []string{"unknown"},
	})
	arches, _ := out["target_architectures"].([]string)
	if len(arches) != 1 || arches[0] != "unknown" {
		t.Fatalf("uninferable declared token should stay visible, got %v (%T)", out["target_architectures"], out["target_architectures"])
	}
}
