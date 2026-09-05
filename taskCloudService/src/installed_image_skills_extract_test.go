package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnsureInstalledImageSkillsReExtracts(t *testing.T) {
	setupCloudTestDB(t)
	prevFn := extractImageSkillsFn
	extractImageSkillsFn = func(imageURL string) imageSkillsExtractResult {
		if imageURL != "registry.example/img:v1" {
			t.Fatalf("unexpected imageURL=%q", imageURL)
		}
		list := imageSkillList{
			Version: 1, DefaultSkill: "general-coding",
			Skills: []imageSkill{{Name: "general-coding", IsDefault: true}, {Name: "k8s-debug"}},
		}
		raw, _ := json.Marshal(list)
		return imageSkillsExtractResult{List: list, JSON: string(raw), Status: "ok", Digest: "sha256:sk"}
	}
	t.Cleanup(func() { extractImageSkillsFn = prevFn })

	img := &TenantInstalledImage{
		ID: "img-sk-reext", TenantID: "t1", ExternalImageID: "ext-sk-reext",
		Name: "img", ImageURL: "registry.example/img:v1",
	}
	if err := createInstalledImage(*img); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := ensureInstalledImageSkills(img, "t1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if img.ImageSkillsExtractStatus != "ok" || !strings.Contains(img.ImageSkillsJSON, "k8s-debug") {
		t.Fatalf("img after re-extract: %+v", img)
	}
	got, err := getInstalledImage("t1", "img-sk-reext")
	if err != nil || got == nil {
		t.Fatalf("get: %v got=%v", err, got)
	}
	if got.ImageSkillsExtractStatus != "ok" || !strings.Contains(got.ImageSkillsJSON, "general-coding") {
		t.Fatalf("db row: %+v", got)
	}
	// D1=B：落库的 image_skills_json 技能须带按 seed=external_id 派生的稳定 id
	stored := decodeImageSkillsJSON(got.ImageSkillsJSON)
	if len(stored.Skills) != 2 || stored.Skills[0].ID == "" || stored.Skills[1].ID == "" {
		t.Fatalf("stored skills must carry derived ids: %+v", stored.Skills)
	}
	if stored.Skills[0].ID != deriveImageSkillID("general-coding", "ext-sk-reext") ||
		stored.Skills[1].ID != deriveImageSkillID("k8s-debug", "ext-sk-reext") {
		t.Fatalf("ids mismatch: %+v", stored.Skills)
	}
	out := installedImageToJSON(*got)
	skills, _ := out["image_skills"].(imageSkillList)
	if skills.DefaultSkill != "general-coding" || len(skills.Skills) != 2 {
		t.Fatalf("json skills=%#v", out["image_skills"])
	}
	if skills.Skills[0].ID != stored.Skills[0].ID {
		t.Fatalf("installedImageToJSON must pass through ids: %+v", skills.Skills)
	}
}

func TestEnsureInstalledImageSkillsSkipsWhenPresent(t *testing.T) {
	setupCloudTestDB(t)
	called := false
	prevFn := extractImageSkillsFn
	extractImageSkillsFn = func(imageURL string) imageSkillsExtractResult {
		called = true
		return imageSkillsExtractResult{Status: "ok"}
	}
	t.Cleanup(func() { extractImageSkillsFn = prevFn })

	list := imageSkillList{Version: 1, DefaultSkill: "a", Skills: []imageSkill{{Name: "a", IsDefault: true}}}
	raw, _ := json.Marshal(list)
	img := &TenantInstalledImage{
		ID: "img-sk-has", TenantID: "t1", ExternalImageID: "ext-sk-has",
		Name: "img", ImageURL: "registry.example/img:v1", ImageSkillsJSON: string(raw),
	}
	if err := ensureInstalledImageSkills(img, "t1"); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if called {
		t.Fatal("must not re-extract when skills already present")
	}
}

func TestInstalledImageInstallCopiesAndReExtractsSkills(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	store.putMember("t1", "admin-1", "mem-1", true)
	_ = startSaasInternalMock(t, store)

	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id": "ext-sk-1", "name": "img", "image_url": "registry.example/img:v1",
				"target_architectures": []string{"x86_64"},
			},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	prevFn := extractAutoRunAndSkillsFn
	extractAutoRunAndSkillsFn = func(imageURL string) autoRunAndSkillsExtractResult {
		list := imageSkillList{
			Version: 1, DefaultSkill: "general-coding",
			Skills: []imageSkill{{Name: "general-coding", IsDefault: true}},
		}
		raw, _ := json.Marshal(list)
		return autoRunAndSkillsExtractResult{
			AutoRun: autoRunStepsExtractResult{Status: "not_found"},
			Skills:  imageSkillsExtractResult{List: list, JSON: string(raw), Status: "ok", Digest: "sha256:sk"},
		}
	}
	t.Cleanup(func() { extractAutoRunAndSkillsFn = prevFn })

	body := `{"external_image_id":"ext-sk-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "admin-1")
	rec := httptest.NewRecorder()
	handleInstalledImageCollection(rec, req, "t1")

	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	skills, _ := out["image_skills"].(map[string]interface{})
	if skills == nil || skills["default_skill"] != "general-coding" {
		t.Fatalf("image_skills=%v", out["image_skills"])
	}
	// D1=B：安装→抽取回填后的响应技能列表须带 id（seed=external_id "ext-sk-1"）
	skillsArr, _ := skills["skills"].([]interface{})
	if len(skillsArr) != 1 {
		t.Fatalf("skills array=%v", skills["skills"])
	}
	first, _ := skillsArr[0].(map[string]interface{})
	if first["id"] != deriveImageSkillID("general-coding", "ext-sk-1") {
		t.Fatalf("skill id missing/mismatch: %v", first)
	}
}
