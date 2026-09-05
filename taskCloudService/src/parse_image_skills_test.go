package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseImageSkillsFirstIsDefault(t *testing.T) {
	raw := `
version: 1
skills:
  - name: general-coding
    description: default workflow
  - name: k8s-debug
`
	list, err := parseImageSkillsYAML(raw)
	if err != nil {
		t.Fatal(err)
	}
	if list.DefaultSkill != "general-coding" || list.DefaultName() != "general-coding" {
		t.Fatalf("default=%q", list.DefaultSkill)
	}
	if len(list.Skills) != 2 || !list.Skills[0].IsDefault || list.Skills[1].IsDefault {
		t.Fatalf("skills=%+v", list.Skills)
	}
}

func TestParseImageSkillsRejectsInvalidName(t *testing.T) {
	_, err := parseImageSkillsYAML("version: 1\nskills:\n  - name: Bad_Name\n")
	if err == nil || !strings.Contains(err.Error(), "name") {
		t.Fatalf("err=%v", err)
	}
}

func TestDecodeImageSkillsJSONEmpty(t *testing.T) {
	list := decodeImageSkillsJSON("")
	if list.DefaultName() != "" || len(list.Skills) != 0 {
		t.Fatalf("%+v", list)
	}
	list = decodeImageSkillsJSON("{not json")
	if len(list.Skills) != 0 {
		t.Fatalf("bad json should be empty, got %+v", list)
	}
}

func TestDeriveImageSkillID(t *testing.T) {
	id := deriveImageSkillID("general-coding", "ext-img-1")
	if !strings.HasPrefix(id, "sk_") || len(id) != len("sk_")+12 {
		t.Fatalf("id=%q", id)
	}
	// 确定性：同 seed 同技能重复派生一致
	if id != deriveImageSkillID("general-coding", "ext-img-1") {
		t.Fatalf("id not deterministic: %q", id)
	}
	// 不同 seed（不同镜像）派生不同 id
	if id == deriveImageSkillID("general-coding", "ext-img-2") {
		t.Fatalf("id must differ across seeds")
	}
	// 不同技能派生不同 id
	if id == deriveImageSkillID("k8s-debug", "ext-img-1") {
		t.Fatalf("id must differ across names")
	}
}

func TestAssignImageSkillIDs(t *testing.T) {
	list := imageSkillList{Version: 1, DefaultSkill: "a",
		Skills: []imageSkill{{Name: "a", IsDefault: true}, {Name: "b"}, {Name: "c", ID: "sk_vendor-provided"}}}
	out := assignImageSkillIDs(list, "ext-img-1")
	if len(out.Skills) != 3 {
		t.Fatalf("skills=%+v", out.Skills)
	}
	if out.Skills[0].ID == "" || out.Skills[1].ID == "" {
		t.Fatalf("missing ids: %+v", out.Skills)
	}
	// 已带 id 的技能保持不变
	if out.Skills[2].ID != "sk_vendor-provided" {
		t.Fatalf("vendor id overwritten: %+v", out.Skills[2])
	}
	// 幂等：重复调用不改变结果
	again := assignImageSkillIDs(out, "ext-img-1")
	for i := range out.Skills {
		if again.Skills[i].ID != out.Skills[i].ID {
			t.Fatalf("not idempotent: %+v vs %+v", again.Skills, out.Skills)
		}
	}
	// 空 seed / 空技能列表原样返回
	if got := assignImageSkillIDs(list, ""); got.Skills[0].ID != "" {
		t.Fatalf("empty seed must not assign ids: %+v", got.Skills)
	}
	empty := assignImageSkillIDs(imageSkillList{Version: 1, Skills: []imageSkill{}}, "ext-img-1")
	if len(empty.Skills) != 0 {
		t.Fatalf("empty list changed: %+v", empty)
	}
}

func TestSkillsSnapshotFromCatalogInjectsIDs(t *testing.T) {
	raw := `{"version":1,"default_skill":"a","skills":[{"name":"a","is_default":true},{"name":"b"}]}`
	jsonStr, _, _ := skillsSnapshotFromCatalog(map[string]interface{}{
		"image_skills": raw, "external_image_id": "ext-img-9",
	}, "ext-img-9")
	var list imageSkillList
	if err := json.Unmarshal([]byte(jsonStr), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Skills) != 2 || list.Skills[0].ID == "" || list.Skills[1].ID == "" {
		t.Fatalf("skills=%+v", list.Skills)
	}
	if list.Skills[0].ID != deriveImageSkillID("a", "ext-img-9") {
		t.Fatalf("id mismatch: %q", list.Skills[0].ID)
	}
	// seed 为空时不注入（保持原样）
	jsonStr2, _, _ := skillsSnapshotFromCatalog(map[string]interface{}{"image_skills": raw}, "")
	if strings.Contains(jsonStr2, `"id"`) {
		t.Fatalf("empty seed must not inject ids: %s", jsonStr2)
	}
}
