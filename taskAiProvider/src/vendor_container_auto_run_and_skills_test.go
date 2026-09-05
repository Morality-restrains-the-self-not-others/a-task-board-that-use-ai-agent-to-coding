package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

type recordingCombinedExtractor struct {
	mu     sync.Mutex
	calls  int
	result infrastructure.AutoRunAndSkillsExtractResult
}

func (r *recordingCombinedExtractor) Extract(imageURL string) infrastructure.AutoRunAndSkillsExtractResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return r.result
}

func (r *recordingCombinedExtractor) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// OPT-20260821-018 regression: creating a vendor container image must extract
// autoRunStep.md and imageSkills.yaml with a single combined walk (one Extract
// call), persisting both fields and emitting both events.
func TestVendorCreateContainerImageExtractsAutoRunAndSkillsInOneWalk(t *testing.T) {
	app := testApp(t)
	_, groupID, tok := seedVendorImageGroup(t, app)
	skillsList := domain.ImageSkillList{
		Version:      1,
		DefaultSkill: "general-coding",
		Skills:       []domain.ImageSkill{{Name: "general-coding", Description: "d", IsDefault: true}},
	}
	skillsJSON, _ := json.Marshal(skillsList)
	ext := &recordingCombinedExtractor{result: infrastructure.AutoRunAndSkillsExtractResult{
		AutoRun: infrastructure.AutoRunStepsExtractResult{Markdown: "# single walk\n", Status: "ok", Digest: "sha256:one"},
		Skills:  infrastructure.ImageSkillsExtractResult{List: skillsList, JSON: string(skillsJSON), Status: "ok", Digest: "sha256:one"},
	}}
	app.AutoRunAndSkillsExtractor = ext
	spy := &infrastructure.SpyEventBus{}
	app.Events = spy
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	payload := `{"image_group_id":"` + infrastructure.IDStr(groupID) + `","version":"1.0","image_url":"example.com/app","target_architectures":["amd64"],"saas_inbound_skill_version":"1"}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/container-images/", bytes.NewReader([]byte(payload)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 201 {
		t.Fatalf("create status %d body %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id, err := parseIDFlexible(created["id"])
	if err != nil {
		t.Fatal(err)
	}
	if ext.callCount() != 1 {
		t.Fatalf("expected exactly 1 combined walk, got %d", ext.callCount())
	}
	img, err := app.DB.GetContainerImage(id)
	if err != nil {
		t.Fatal(err)
	}
	if img.AutoRunStepsExtractStatus != "ok" || !strings.Contains(img.AutoRunStepsMD, "single walk") {
		t.Fatalf("auto md=%q status=%q", img.AutoRunStepsMD, img.AutoRunStepsExtractStatus)
	}
	if img.ImageSkillsExtractStatus != "ok" || !strings.Contains(img.ImageSkillsJSON, "general-coding") {
		t.Fatalf("skills json=%q status=%q", img.ImageSkillsJSON, img.ImageSkillsExtractStatus)
	}
	var sawAuto, sawSkills bool
	for _, n := range spy.Names {
		if n == domain.EventContainerImageAutoRunStepsExtracted {
			sawAuto = true
		}
		if n == domain.EventContainerImageSkillsExtracted {
			sawSkills = true
		}
	}
	if !sawAuto || !sawSkills {
		t.Fatalf("expected both extract events, got %v", spy.Names)
	}
}

// Catalog backfill with both files missing must also use the combined walk.
func TestPublicCatalogBackfillUsesCombinedWalkWhenBothMissing(t *testing.T) {
	app := testApp(t)
	vendorID, groupID, _ := seedVendorImageGroup(t, app)
	imgID, err := app.DB.CreateContainerImage(vendorID, groupID, "2.0", "example.com/catalog", []any{"amd64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.DB.UpdateContainerImageFields(imgID, map[string]any{"status": domain.StatusApproved}); err != nil {
		t.Fatal(err)
	}
	ext := &recordingCombinedExtractor{result: infrastructure.AutoRunAndSkillsExtractResult{
		AutoRun: infrastructure.AutoRunStepsExtractResult{Markdown: "# catalog\n", Status: "ok", Digest: "sha256:cat"},
		Skills:  infrastructure.ImageSkillsExtractResult{Status: "not_found", Digest: "sha256:cat"},
	}}
	app.AutoRunAndSkillsExtractor = ext
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/public/catalog/", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("catalog status %d body %s", rr.Code, rr.Body.String())
	}
	if ext.callCount() != 1 {
		t.Fatalf("expected 1 combined walk for catalog backfill, got %d", ext.callCount())
	}
	img, err := app.DB.GetContainerImage(imgID)
	if err != nil {
		t.Fatal(err)
	}
	if img.AutoRunStepsExtractStatus != "ok" || !strings.Contains(img.AutoRunStepsMD, "catalog") {
		t.Fatalf("auto md=%q status=%q", img.AutoRunStepsMD, img.AutoRunStepsExtractStatus)
	}
	if img.ImageSkillsExtractStatus != "not_found" {
		t.Fatalf("skills status=%q (want not_found)", img.ImageSkillsExtractStatus)
	}
}
