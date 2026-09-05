package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskTaskService/src/domain"
)

func skillsTestImage() *InstalledImageLookup {
	return &InstalledImageLookup{
		ID: "img-1", Name: "trae-agent", Version: "v1", ExternalImageID: "ext-1",
		ImageSkills: domain.ImageSkillList{Skills: []domain.ImageSkill{
			{ID: "sk_general01", Name: "general-coding"},
			{ID: "sk_k8s0001", Name: "k8s-debug"},
		}},
	}
}

// D4 契约：container_image_skill_id 必须属于镜像技能列表（fail-closed）。
func TestResolveTaskImageSkill_RequiresSkillInImage(t *testing.T) {
	stubInstalledImageLookup(t, skillsTestImage(), nil)
	_, serr := resolveTaskImageSkill("t1", "img-1", "sk_notexist", "desc")
	if serr == nil || serr.status != 400 || serr.code != "skill_not_in_image" {
		t.Fatalf("want 400 skill_not_in_image, got %+v", serr)
	}
}

// D4 契约：skill_id 提供但无 image → 400 skill_id_requires_image（不查目录）。
func TestResolveTaskImageSkill_RequiresImage(t *testing.T) {
	called := false
	prev := lookupInstalledImageFn
	lookupInstalledImageFn = func(tenantID, imageID string, _ ...string) (*InstalledImageLookup, error) {
		called = true
		return nil, errors.New("should not be called")
	}
	t.Cleanup(func() { lookupInstalledImageFn = prev })

	_, serr := resolveTaskImageSkill("t1", "", "sk_abc", "desc")
	if serr == nil || serr.status != 400 || serr.code != "skill_id_requires_image" {
		t.Fatalf("want 400 skill_id_requires_image, got %+v", serr)
	}
	if called {
		t.Fatal("lookup must not run when image_id empty")
	}
}

// D4 契约：镜像目录 404 → 400 image_not_found；其他错误 → 502 image_lookup_failed。
func TestResolveTaskImageSkill_ImageLookupErrors(t *testing.T) {
	stubInstalledImageLookup(t, nil, ErrInstalledImageNotFound)
	_, serr := resolveTaskImageSkill("t1", "img-1", "", "desc")
	if serr == nil || serr.status != 400 || serr.code != "image_not_found" {
		t.Fatalf("want 400 image_not_found, got %+v", serr)
	}

	stubInstalledImageLookup(t, nil, errors.New("boom"))
	_, serr = resolveTaskImageSkill("t1", "img-1", "", "desc")
	if serr == nil || serr.status != 502 || serr.code != "image_lookup_failed" {
		t.Fatalf("want 502 image_lookup_failed, got %+v", serr)
	}
}

// 绑定成功：快照携带 image_id/image_name + skill_id/skill_name。
func TestResolveTaskImageSkill_BindsByID(t *testing.T) {
	stubInstalledImageLookup(t, skillsTestImage(), nil)
	snap, serr := resolveTaskImageSkill("t1", "img-1", "sk_k8s0001", "desc 无 token")
	if serr != nil {
		t.Fatalf("unexpected error: %+v", serr)
	}
	if snap.ImageID != "img-1" || snap.ImageName != "trae-agent" ||
		snap.SkillID != "sk_k8s0001" || snap.SkillName != "k8s-debug" {
		t.Fatalf("snapshot unexpected: %+v", snap)
	}
}

// 老前端兼容：无 skill_id 时描述内第一个列表内 /token 按名反解。
func TestResolveTaskImageSkill_ResolvesNameFromDescription(t *testing.T) {
	stubInstalledImageLookup(t, skillsTestImage(), nil)
	snap, serr := resolveTaskImageSkill("t1", "img-1", "", "跑一下 /general-coding 默认流程")
	if serr != nil {
		t.Fatalf("unexpected error: %+v", serr)
	}
	if snap.SkillID != "sk_general01" || snap.SkillName != "general-coding" {
		t.Fatalf("name-resolved snapshot unexpected: %+v", snap)
	}
}

// 老前端兼容：描述中无列表内 token（如普通路径 /api）→ 不绑定而非报错。
func TestResolveTaskImageSkill_NoTokenNoBind(t *testing.T) {
	stubInstalledImageLookup(t, skillsTestImage(), nil)
	snap, serr := resolveTaskImageSkill("t1", "img-1", "", "调用 /api/foo 接口")
	if serr != nil {
		t.Fatalf("unexpected error: %+v", serr)
	}
	if snap == nil || snap.SkillID != "" || snap.SkillName != "" {
		t.Fatalf("should bind image without skill, got %+v", snap)
	}

	snap, serr = resolveTaskImageSkill("t1", "img-1", "", "")
	if serr != nil {
		t.Fatalf("unexpected error: %+v", serr)
	}
	if snap == nil || snap.SkillID != "" || snap.SkillName != "" {
		t.Fatalf("empty desc should bind image only, got %+v", snap)
	}
}

// 快照编解码与响应渲染。
func TestContainerImageSnapshotEncodeDecode(t *testing.T) {
	s := &containerImageSnapshot{ImageID: "img-1", ImageName: "trae-agent", SkillID: "sk_a", SkillName: "n"}
	raw, err := s.encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !strings.Contains(raw, `"skill_id":"sk_a"`) {
		t.Fatalf("encode missing skill_id: %s", raw)
	}
	dec := decodeContainerImageSnapshot(raw)
	if dec == nil || dec.SkillID != "sk_a" || dec.ImageName != "trae-agent" {
		t.Fatalf("decode roundtrip failed: %+v", dec)
	}
	if decodeContainerImageSnapshot("") != nil || decodeContainerImageSnapshot("{bad") != nil {
		t.Fatal("empty/bad snapshot should decode to nil")
	}
	noSkill := &containerImageSnapshot{ImageID: "img-1", ImageName: "trae-agent"}
	rawNoSkill, _ := noSkill.encode()
	if out := snapshotToJSON(rawNoSkill); out != nil {
		if m, ok := out.(map[string]interface{}); ok {
			if _, has := m["skill_id"]; has {
				t.Fatalf("skill_id should be omitted when unbound: %#v", m)
			}
		} else {
			t.Fatalf("snapshotToJSON type: %T", out)
		}
	}
}

// 集成：创建任务携带 container_image_id + container_image_skill_id → 持久化并回显。
func TestCreateTaskPersistsImageSkillSnapshot(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	stubInstalledImageLookup(t, skillsTestImage(), nil)

	body := `{"title":"带技能","workspace_id":"ws1","container_image_id":"img-1","container_image_skill_id":"sk_general01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	ci := out["container_image"].(map[string]interface{})
	skill := ci["skill"].(map[string]interface{})
	if skill["id"] != "sk_general01" || skill["name"] != "general-coding" {
		t.Fatalf("container_image.skill unexpected: %#v", skill)
	}
	snap := out["container_image_snapshot"].(map[string]interface{})
	if snap["image_name"] != "trae-agent" || snap["skill_id"] != "sk_general01" {
		t.Fatalf("container_image_snapshot unexpected: %#v", snap)
	}

	taskID := strField(out, "id")
	var skillID string
	var snapshotRaw sql.NullString
	if err := db.QueryRow(`SELECT image_skill_id, container_image_snapshot FROM task_tasks WHERE id=?`, taskID).Scan(&skillID, &snapshotRaw); err != nil {
		t.Fatalf("select persisted: %v", err)
	}
	if skillID != "sk_general01" || !snapshotRaw.Valid || !strings.Contains(snapshotRaw.String, `general-coding`) {
		t.Fatalf("persisted image_skill_id=%q snapshot=%q", skillID, snapshotRaw.String)
	}
}

// 集成：创建任务 skill_id 不在镜像列表 → 400 skill_not_in_image，不落库。
func TestCreateTaskRejectsSkillNotInImage(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	stubInstalledImageLookup(t, skillsTestImage(), nil)

	body := `{"title":"坏技能","workspace_id":"ws1","container_image_id":"img-1","container_image_skill_id":"sk_nope"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "skill_not_in_image") {
		t.Fatalf("expected skill_not_in_image in body: %s", rec.Body.String())
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM task_tasks`).Scan(&count)
	if count != 0 {
		t.Fatalf("task should not persist on 400, count=%d", count)
	}
}

// 集成：创建任务镜像目录 404 → 400 image_not_found。
func TestCreateTaskRejectsImageNotFound(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	stubInstalledImageLookup(t, nil, ErrInstalledImageNotFound)

	body := `{"title":"坏镜像","workspace_id":"ws1","container_image_id":"img-gone"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "image_not_found") {
		t.Fatalf("expected 400 image_not_found, got %d: %s", rec.Code, rec.Body.String())
	}
}

// 集成：更新任务换技能 → 快照刷新；body 显式空 container_image_id → 全部解绑清空。
func TestUpdateTaskRefreshesAndClearsImageSkill(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	stubInstalledImageLookup(t, skillsTestImage(), nil)

	createBody := `{"title":"绑定","workspace_id":"ws1","container_image_id":"img-1","container_image_skill_id":"sk_general01"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspace/ws1/todos/", strings.NewReader(createBody))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleTaskRoutes(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&created)
	taskID := strField(created, "id")

	// 换技能 → 快照刷新
	updBody := `{"container_image_skill_id":"sk_k8s0001"}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(updBody))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-Auth-User-Id", "u1")
	rec2 := httptest.NewRecorder()
	handleTaskRoutes(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec2.Code, rec2.Body.String())
	}
	var updated map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&updated)
	skill := updated["container_image"].(map[string]interface{})["skill"].(map[string]interface{})
	if skill["id"] != "sk_k8s0001" {
		t.Fatalf("updated skill unexpected: %#v", skill)
	}
	var skillID string
	var snapRaw sql.NullString
	db.QueryRow(`SELECT image_skill_id, container_image_snapshot FROM task_tasks WHERE id=?`, taskID).Scan(&skillID, &snapRaw)
	if skillID != "sk_k8s0001" || !strings.Contains(snapRaw.String, `sk_k8s0001`) {
		t.Fatalf("persisted after update: skill=%q snap=%q", skillID, snapRaw.String)
	}

	// 解绑：body 显式传空 container_image_id → 镜像/技能/快照全部清空
	unbindBody := `{"container_image_id":""}`
	req3 := httptest.NewRequest(http.MethodPut, "/api/tenant/t1/workspace/ws1/todos/"+taskID+"/", strings.NewReader(unbindBody))
	req3.Header.Set("X-Auth-Tenant-Id", "t1")
	req3.Header.Set("X-Auth-User-Id", "u1")
	rec3 := httptest.NewRecorder()
	handleTaskRoutes(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("unbind: %d %s", rec3.Code, rec3.Body.String())
	}
	var unbound map[string]interface{}
	json.NewDecoder(rec3.Body).Decode(&unbound)
	// taskToJSON 的 container_image 是既有占位 key（值为 nil）；快照 key 仅在有时出现。
	if v, has := unbound["container_image"]; has && v != nil {
		t.Fatalf("container_image should be nil after unbind: %#v", v)
	}
	if v, has := unbound["container_image_snapshot"]; has && v != nil {
		t.Fatalf("snapshot should be nil after unbind: %#v", v)
	}
	var skillID2 string
	var snapVal interface{}
	db.QueryRow(`SELECT image_skill_id, container_image_snapshot FROM task_tasks WHERE id=?`, taskID).Scan(&skillID2, &snapVal)
	if skillID2 != "" || snapVal != nil {
		t.Fatalf("not cleared after unbind: skill=%q snap=%v", skillID2, snapVal)
	}
}
