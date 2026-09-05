package main

import (
	"path/filepath"
	"testing"
)

func TestProductionConfig_StartAllPlanOmitsGitLabRegions(t *testing.T) {
	path := filepath.Join("..", "..", "conf", "runAll.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q): %v", path, err)
	}
	var gitlabGroup *Group
	for i := range cfg.Groups {
		if cfg.Groups[i].Name == "gitlab-regions" {
			gitlabGroup = &cfg.Groups[i]
			break
		}
	}
	if gitlabGroup == nil {
		t.Fatal("missing group gitlab-regions")
	}
	if !gitlabGroup.SkipStartAll {
		t.Fatal("gitlab-regions must set skip_start_all: true so start-all does not touch git-service*")
	}

	store := NewStatusStore()
	runner, err := NewRunner(cfg, store)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	for _, svc := range cfg.Flatten() {
		store.Update(svc.Name, StatusStopped, "")
	}
	plan, err := runner.cascadeOrchestration().PlanStartAll()
	if err != nil {
		t.Fatalf("PlanStartAll: %v", err)
	}
	for _, name := range plan.OrderedNames {
		if isGitLabRunAllService(name) {
			t.Errorf("start-all plan includes %q; gitlab-regions must be skip_start_all", name)
		}
	}
	groupPlan, err := runner.cascadeOrchestration().PlanStartGroup("gitlab-regions")
	if err != nil {
		t.Fatalf("PlanStartGroup: %v", err)
	}
	foundPrimary, foundSH := false, false
	for _, name := range groupPlan.OrderedNames {
		if name == "git-service" {
			foundPrimary = true
		}
		if name == "git-service-tencent-sh-1" {
			foundSH = true
		}
	}
	if !foundPrimary || !foundSH {
		t.Fatalf("PlanStartGroup(gitlab-regions) = %#v, want git-service and git-service-tencent-sh-1", groupPlan.OrderedNames)
	}
}
