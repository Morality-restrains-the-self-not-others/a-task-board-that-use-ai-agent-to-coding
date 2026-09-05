package main

import (
	"testing"

	"taskAiProvider/domain"
)

// OPT-20260820-027：公开目录过滤仍声明 sunset 契约版本的镜像。
func TestFilterSunsetSkillCatalogItemsDropsSunset(t *testing.T) {
	entries := []domain.SkillVersionEntry{
		{Version: "1", Status: domain.SkillVersionStatusCurrent},
		{Version: "0", Status: domain.SkillVersionStatusSunset},
	}
	items := []map[string]any{
		{"id": int64(1), "name": "current-image", "saas_inbound_skill_version": "1"},
		{"id": int64(2), "name": "sunset-image", "saas_inbound_skill_version": "0"},
		{"id": int64(3), "name": "deprecated-image", "saas_inbound_skill_version": "v2"},
	}
	entries = append(entries, domain.SkillVersionEntry{Version: "2", Status: domain.SkillVersionStatusDeprecated})

	got := filterSunsetSkillCatalogItems(items, entries)
	if len(got) != 2 {
		t.Fatalf("expected 2 kept, got %d: %#v", len(got), got)
	}
	if got[0]["name"] != "current-image" || got[1]["name"] != "deprecated-image" {
		t.Fatalf("kept wrong items: %#v", got)
	}
}

func TestFilterSunsetSkillCatalogItemsEmptyEntriesKeepsAll(t *testing.T) {
	items := []map[string]any{
		{"id": int64(1), "name": "x", "saas_inbound_skill_version": "1"},
	}
	got := filterSunsetSkillCatalogItems(items, nil)
	if len(got) != 1 {
		t.Fatalf("empty catalog must keep all, got %#v", got)
	}
}
