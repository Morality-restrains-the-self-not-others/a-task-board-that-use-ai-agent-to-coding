package infrastructure_test

import (
	"strings"
	"testing"

	"taskAiProvider/infrastructure"
)

func TestParseImageSkillsFirstIsDefault(t *testing.T) {
	raw := `
version: 1
skills:
  - name: general-coding
    description: default workflow
  - name: k8s-debug
    description: kubernetes
`
	list, err := infrastructure.ParseImageSkillsYAML(raw)
	if err != nil {
		t.Fatal(err)
	}
	if list.DefaultSkill != "general-coding" {
		t.Fatalf("default=%q", list.DefaultSkill)
	}
	if len(list.Skills) != 2 || !list.Skills[0].IsDefault || list.Skills[1].IsDefault {
		t.Fatalf("skills=%+v", list.Skills)
	}
	if !list.Has("k8s-debug") || list.Has("missing") {
		t.Fatal("Has mismatch")
	}
}

func TestParseImageSkillsRejectsInvalidName(t *testing.T) {
	_, err := infrastructure.ParseImageSkillsYAML("version: 1\nskills:\n  - name: Bad_Name\n")
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseImageSkillsRejectsDuplicate(t *testing.T) {
	_, err := infrastructure.ParseImageSkillsYAML("version: 1\nskills:\n  - name: a\n  - name: a\n")
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("err=%v", err)
	}
}

func TestParseImageSkillsEmptyListOK(t *testing.T) {
	list, err := infrastructure.ParseImageSkillsYAML("version: 1\nskills: []\n")
	if err != nil {
		t.Fatal(err)
	}
	if list.DefaultSkill != "" || len(list.Skills) != 0 {
		t.Fatalf("%+v", list)
	}
}
