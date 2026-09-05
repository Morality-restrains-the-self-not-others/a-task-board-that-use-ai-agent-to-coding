package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// OPT-20260821-018 regression: install-time re-extract must fill both
// autoRunStep.md and imageSkills.yaml from a single combined walk.
func TestEnsureInstalledImageExtractsSingleWalkFillsBoth(t *testing.T) {
	setupCloudTestDB(t)
	calls := 0
	prevFn := extractAutoRunAndSkillsFn
	extractAutoRunAndSkillsFn = func(imageURL string) autoRunAndSkillsExtractResult {
		calls++
		list := imageSkillList{Version: 1, DefaultSkill: "a", Skills: []imageSkill{{Name: "a", IsDefault: true}}}
		raw, _ := json.Marshal(list)
		return autoRunAndSkillsExtractResult{
			AutoRun: autoRunStepsExtractResult{Status: "ok", Markdown: "# 合并抽取\n", Digest: "sha256:one"},
			Skills:  imageSkillsExtractResult{List: list, JSON: string(raw), Status: "ok", Digest: "sha256:one"},
		}
	}
	t.Cleanup(func() { extractAutoRunAndSkillsFn = prevFn })

	img := &TenantInstalledImage{
		ID: "img-both", TenantID: "t1", ExternalImageID: "ext-both",
		Name: "img", ImageURL: "registry.example/img:v1",
	}
	if err := createInstalledImage(*img); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := ensureInstalledImageExtracts(img, "t1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 combined walk, got %d", calls)
	}
	if img.AutoRunStepsMd != "# 合并抽取" || img.AutoRunStepsExtractStatus != "ok" {
		t.Fatalf("auto after: md=%q status=%q", img.AutoRunStepsMd, img.AutoRunStepsExtractStatus)
	}
	if img.ImageSkillsExtractStatus != "ok" || !strings.Contains(img.ImageSkillsJSON, `"name":"a"`) && !strings.Contains(img.ImageSkillsJSON, `"name": "a"`) {
		t.Fatalf("skills after: status=%s json=%s", img.ImageSkillsExtractStatus, img.ImageSkillsJSON)
	}
	got, err := getInstalledImage("t1", "img-both")
	if err != nil || got == nil {
		t.Fatalf("get: %v got=%v", err, got)
	}
	if got.AutoRunStepsExtractStatus != "ok" || got.ImageSkillsExtractStatus != "ok" {
		t.Fatalf("db row: %+v", got)
	}
}

func TestEnsureInstalledImageExtractsSkipsWhenBothPresent(t *testing.T) {
	setupCloudTestDB(t)
	called := false
	prevFn := extractAutoRunAndSkillsFn
	extractAutoRunAndSkillsFn = func(imageURL string) autoRunAndSkillsExtractResult {
		called = true
		return autoRunAndSkillsExtractResult{}
	}
	t.Cleanup(func() { extractAutoRunAndSkillsFn = prevFn })

	list := imageSkillList{Version: 1, DefaultSkill: "a", Skills: []imageSkill{{Name: "a", IsDefault: true}}}
	raw, _ := json.Marshal(list)
	img := &TenantInstalledImage{
		ID: "img-full", TenantID: "t1", ExternalImageID: "ext-full",
		Name: "img", ImageURL: "registry.example/img:v1",
		AutoRunStepsMd: "已有说明", ImageSkillsJSON: string(raw),
	}
	if err := ensureInstalledImageExtracts(img, "t1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if called {
		t.Fatal("must not walk when both fields already present")
	}
}

func TestEnsureInstalledImageExtractsOnlyAutoMissing(t *testing.T) {
	setupCloudTestDB(t)
	calls := 0
	prevFn := extractAutoRunAndSkillsFn
	extractAutoRunAndSkillsFn = func(imageURL string) autoRunAndSkillsExtractResult {
		calls++
		return autoRunAndSkillsExtractResult{
			AutoRun: autoRunStepsExtractResult{Status: "ok", Markdown: "# 只缺 auto\n", Digest: "sha256:a"},
			Skills:  imageSkillsExtractResult{Status: "not_found"},
		}
	}
	t.Cleanup(func() { extractAutoRunAndSkillsFn = prevFn })

	list := imageSkillList{Version: 1, DefaultSkill: "a", Skills: []imageSkill{{Name: "a", IsDefault: true}}}
	raw, _ := json.Marshal(list)
	img := &TenantInstalledImage{
		ID: "img-skill-only", TenantID: "t1", ExternalImageID: "ext-skill-only",
		Name: "img", ImageURL: "registry.example/img:v1", ImageSkillsJSON: string(raw),
	}
	if err := createInstalledImage(*img); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := ensureInstalledImageExtracts(img, "t1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 walk, got %d", calls)
	}
	if img.AutoRunStepsExtractStatus != "ok" {
		t.Fatalf("auto status=%s want ok", img.AutoRunStepsExtractStatus)
	}
	// Skills were already present: the walk must not overwrite their JSON.
	if !strings.Contains(img.ImageSkillsJSON, `"name":"a"`) && !strings.Contains(img.ImageSkillsJSON, `"name": "a"`) {
		t.Fatalf("skills json overwritten: %s", img.ImageSkillsJSON)
	}
}
