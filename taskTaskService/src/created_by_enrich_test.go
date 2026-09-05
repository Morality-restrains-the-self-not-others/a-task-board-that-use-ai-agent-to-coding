package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskTaskService/src/domain"
)

func TestListTasksIncludesResolvedCreatedBy(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)

	body := `{"title":"Hello","workspace_id":"ws1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Mock taskTenantService for batchResolveTaskOwners (OPT-20260729-024 #9).
	tenantSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/api/internal/tenant/members") {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":                "m1",
					"user_id":           "u1",
					"username":          "Alice展示名",
					"company_member_id": "m1",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(tenantSrv.Close)
	prev := cfg.TaskTenantServiceURL
	cfg.TaskTenantServiceURL = tenantSrv.URL
	t.Cleanup(func() { cfg.TaskTenantServiceURL = prev })

	req2 := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/workspace/ws1/todos/", nil)
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var list []map[string]interface{}
	if err := json.NewDecoder(rec2.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 task, got %d", len(list))
	}
	cb, ok := list[0]["created_by"].(map[string]interface{})
	if !ok {
		t.Fatalf("created_by missing or wrong type: %#v", list[0]["created_by"])
	}
	if got := strings.TrimSpace(fmt.Sprint(cb["username"])); got != "Alice展示名" {
		t.Fatalf("created_by.username=%q want Alice展示名; full=%#v", got, cb)
	}
	if got := strings.TrimSpace(fmt.Sprint(cb["id"])); got != "u1" {
		t.Fatalf("created_by.id=%q want u1", got)
	}
}

func TestCreatedByObjectFallbackWithoutResolve(t *testing.T) {
	cb := createdByObject("owner-1", nil)
	if cb["id"] != "owner-1" {
		t.Fatalf("id=%v", cb["id"])
	}
	if cb["username"] != "" {
		t.Fatalf("username=%v want empty", cb["username"])
	}
}

// OPT-20260820-045: 单任务路径水合 container_image 快照（name/version）—
// 目录命中时写入对象，未命中标记 container_image_removed，空 imageID 为 no-op。
// D4：任务绑定技能时按 image_skill_id 从目录解析 container_image.skill = {id, name}；
// container_image_snapshot（名↔ID 映射快照）始终透传。
func TestHydrateTaskContainerImage_PopulatesSnapshot(t *testing.T) {
	prev := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		if tenantID != "t1" || imageID != "img-1" {
			t.Fatalf("unexpected lookup args: %s %s", tenantID, imageID)
		}
		return &InstalledImageLookup{
			ID: "img-1", Name: "trae-agent", Version: "v1", ExternalImageID: "ext-1",
			ImageSkills: domain.ImageSkillList{Skills: []domain.ImageSkill{
				{ID: "sk_abc123", Name: "general-coding"},
				{ID: "sk_def456", Name: "k8s-debug"},
			}},
		}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prev })

	snap := `{"image_id":"img-1","image_name":"trae-agent","skill_id":"sk_abc123","skill_name":"general-coding"}`
	out := map[string]interface{}{}
	hydrateTaskContainerImage(out, "t1", "img-1", snap)
	img, ok := out["container_image"].(map[string]interface{})
	if !ok {
		t.Fatalf("container_image missing: %#v", out)
	}
	if img["name"] != "trae-agent" || img["version"] != "v1" || img["external_image_id"] != "ext-1" {
		t.Fatalf("snapshot unexpected: %#v", img)
	}
	skill, ok := img["skill"].(map[string]interface{})
	if !ok {
		t.Fatalf("container_image.skill missing: %#v", img)
	}
	if skill["id"] != "sk_abc123" || skill["name"] != "general-coding" {
		t.Fatalf("skill unexpected: %#v", skill)
	}
	gotSnap, ok := out["container_image_snapshot"].(map[string]interface{})
	if !ok || gotSnap["skill_id"] != "sk_abc123" || gotSnap["skill_name"] != "general-coding" {
		t.Fatalf("container_image_snapshot not透传: %#v", out["container_image_snapshot"])
	}
	if _, removed := out["container_image_removed"]; removed {
		t.Fatalf("should not mark removed when found")
	}
}

// D4 回退：目录技能列表已变（技能 ID 不存在）时，container_image.skill 按快照回退；
// 未绑定技能时 skill 不出现。
func TestHydrateTaskContainerImage_SkillFallsBackToSnapshot(t *testing.T) {
	prev := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return &InstalledImageLookup{
			ID: "img-1", Name: "trae-agent", Version: "v1", ExternalImageID: "ext-1",
			ImageSkills: domain.ImageSkillList{Skills: []domain.ImageSkill{
				{ID: "sk_new777", Name: "renamed-skill"},
			}},
		}, nil
	}
	t.Cleanup(func() { lookupInstalledImageFn = prev })

	out := map[string]interface{}{}
	hydrateTaskContainerImage(out, "t1", "img-1",
		`{"image_id":"img-1","image_name":"trae-agent","skill_id":"sk_abc123","skill_name":"general-coding"}`)
	img := out["container_image"].(map[string]interface{})
	if img["skill"] != nil {
		t.Fatalf("stale skill_id should not resolve: %#v", img["skill"])
	}

	out2 := map[string]interface{}{}
	hydrateTaskContainerImage(out2, "t1", "img-1", "")
	img2 := out2["container_image"].(map[string]interface{})
	if _, has := img2["skill"]; has {
		t.Fatalf("unbound skill should be absent: %#v", img2)
	}
	if _, has := out2["container_image_snapshot"]; has {
		t.Fatalf("empty snapshot should be absent: %#v", out2)
	}
}

func TestHydrateTaskContainerImage_MarksRemovedOnNotFound(t *testing.T) {
	prev := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		return nil, ErrInstalledImageNotFound
	}
	t.Cleanup(func() { lookupInstalledImageFn = prev })

	out := map[string]interface{}{}
	hydrateTaskContainerImage(out, "t1", "img-1", "")
	if _, ok := out["container_image_removed"]; !ok {
		t.Fatalf("expected container_image_removed=true, got %#v", out)
	}
	if _, ok := out["container_image"]; ok {
		t.Fatalf("container_image should stay nil when removed")
	}
}

func TestHydrateTaskContainerImage_SkipsEmptyImageID(t *testing.T) {
	out := map[string]interface{}{}
	hydrateTaskContainerImage(out, "t1", "", "")
	if len(out) != 0 {
		t.Fatalf("empty image id should be no-op, got %#v", out)
	}
}
