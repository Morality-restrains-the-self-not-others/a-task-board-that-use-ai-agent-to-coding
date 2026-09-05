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

type recordingSkillsExtractor struct {
	mu     sync.Mutex
	calls  []string
	result infrastructure.ImageSkillsExtractResult
}

func (r *recordingSkillsExtractor) Extract(imageURL string) infrastructure.ImageSkillsExtractResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, imageURL)
	return r.result
}

func TestVendorCreateContainerImageExtractsSkills(t *testing.T) {
	app := testApp(t)
	_, groupID, tok := seedVendorImageGroup(t, app)
	list := domain.ImageSkillList{
		Version:      1,
		DefaultSkill: "general-coding",
		Skills: []domain.ImageSkill{
			{Name: "general-coding", Description: "d", IsDefault: true},
			{Name: "k8s-debug", IsDefault: false},
		},
	}
	raw, _ := json.Marshal(list)
	ext := &recordingSkillsExtractor{result: infrastructure.ImageSkillsExtractResult{
		List: list, JSON: string(raw), Status: "ok", Digest: "sha256:sk",
	}}
	app.SkillsExtractor = ext
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
	img, err := app.DB.GetContainerImage(id)
	if err != nil {
		t.Fatal(err)
	}
	if img.ImageSkillsExtractStatus != "ok" || !strings.Contains(img.ImageSkillsJSON, "k8s-debug") {
		t.Fatalf("json=%q status=%q", img.ImageSkillsJSON, img.ImageSkillsExtractStatus)
	}
	if !strings.Contains(img.ImageSkillsJSON, `"default_skill":"general-coding"`) && !strings.Contains(img.ImageSkillsJSON, `"default_skill": "general-coding"`) {
		t.Fatalf("missing default in %s", img.ImageSkillsJSON)
	}
	found := false
	for _, n := range spy.Names {
		if n == domain.EventContainerImageSkillsExtracted {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected skills event, got %v", spy.Names)
	}
}
