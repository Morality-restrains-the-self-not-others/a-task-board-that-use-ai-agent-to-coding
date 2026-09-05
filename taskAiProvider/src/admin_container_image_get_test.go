package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"taskAiProvider/infrastructure"
)

// TestAdminContainerImageGetByIDReturnsEnrichedFields 断言单条 GET 与列表
// 同等富化：image_skills / runtime_environments / review_histories。
func TestAdminContainerImageGetByIDReturnsEnrichedFields(t *testing.T) {
	app := testApp(t)
	staffID := infrastructure.NextID()
	vendorID := infrastructure.NextID()
	groupID := infrastructure.NextID()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")

	for _, s := range []struct {
		q   string
		arg []any
	}{
		{`INSERT INTO ai_provider_platformstaff (id, username, password_hash, display_name, is_active, created_at, updated_at) VALUES (?,?,?,?,1,?,?)`,
			[]any{staffID, "staff_get_by_id", "x", "运营甲", now, now}},
		{`INSERT INTO ai_provider_vendor (id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at) VALUES (?,?,?,?,?,1,?,?)`,
			[]any{vendorID, "get_by_id_vendor@example.com", "x", "GetByIDCo", "Contact", now, now}},
		{`INSERT INTO ai_provider_containerimagegroup (id, name, description, vendor_id, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
			[]any{groupID, "get-by-id-group", "desc", vendorID, now, now}},
	} {
		if _, err := app.DB.SQL.Exec(s.q, s.arg...); err != nil {
			t.Fatal(err)
		}
	}
	imgID, err := app.DB.CreateContainerImage(vendorID, groupID, "v-get", "registry.example/g:v-get", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET image_skills_json=? WHERE id=?`,
		`{"version":1,"default_skill":"code","skills":[{"name":"code","is_default":true}]}`, imgID); err != nil {
		t.Fatal(err)
	}
	tplID := infrastructure.NextID()
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_userdatatemplate
		(id, name, version, os_type, variables, container_variables, content, auto_verify_script, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,1,?,?)`,
		tplID, "boot", "1", "linux", "[]", "[]", "#", "", now, now); err != nil {
		t.Fatal(err)
	}
	csiID := infrastructure.NextID()
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendorcloudserverimage
		(id, vendor_id, platform_type, image_name, image_id, region, os_type, os_version, architecture, image_type,
		 is_active, default_instance_type_id, default_instance_type_label, base_cpu_cores, base_memory_gib,
		 userdata_template_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,1,?,?,?,?,?,?,?)`,
		csiID, vendorID, "aliyun", "dev-img", "m-dev", "cn-hangzhou", "linux", "24.04", "x86_64", "system",
		"ecs.c6.large", "ecs.c6.large", 2, 4, tplID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_containercloudserverassociation
		(id, platform_type, region, cloud_server_image_id, container_image_id)
		VALUES (?,?,?,?,?)`, infrastructure.NextID(), "aliyun", "cn-hangzhou", csiID, imgID); err != nil {
		t.Fatal(err)
	}
	if err := app.DB.InsertReviewHistory(imgID, "reject", "描述不完整", staffID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_containerimagereviewhistory WHERE container_image_id=?`, imgID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_containercloudserverassociation WHERE container_image_id=?`, imgID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcloudserverimage WHERE id=?`, csiID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_userdatatemplate WHERE id=?`, tplID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id=?`, imgID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, groupID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id=?`, vendorID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_platformstaff WHERE id=?`, staffID)
	})

	staffTok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(staffID), "staff", 3600)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/container-images/"+infrastructure.IDStr(imgID)+"/", nil)
	req.Header.Set("Authorization", "Bearer "+staffTok)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s want 200", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	skills, _ := body["image_skills"].(map[string]any)
	if skills == nil {
		t.Fatalf("image_skills missing: %v", body["image_skills"])
	}
	skillRows, _ := skills["skills"].([]any)
	if len(skillRows) != 1 {
		t.Fatalf("image_skills.skills=%v want [code]", skills["skills"])
	}
	envs, _ := body["runtime_environments"].([]any)
	if len(envs) != 1 {
		t.Fatalf("runtime_environments=%v want 1 aliyun env", body["runtime_environments"])
	}
	env, _ := envs[0].(map[string]any)
	if env["platform_type"] != "aliyun" {
		t.Fatalf("runtime_environments[0]=%v want platform_type=aliyun", env)
	}
	hist, _ := body["review_histories"].([]any)
	if len(hist) != 1 {
		t.Fatalf("review_histories=%v want 1 reject", body["review_histories"])
	}
	h, _ := hist[0].(map[string]any)
	if h["action"] != "reject" || h["note"] != "描述不完整" {
		t.Fatalf("review_histories[0]=%v want reject/描述不完整", h)
	}
}

func TestAdminContainerImageGetByIDNotFound(t *testing.T) {
	app := testApp(t)
	staffID := infrastructure.NextID()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_platformstaff (id, username, password_hash, display_name, is_active, created_at, updated_at) VALUES (?,?,?,?,1,?,?)`,
		staffID, "staff_get_404", "x", "Staff", now, now); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_platformstaff WHERE id=?`, staffID)
	})
	staffTok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(staffID), "staff", 3600)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	missingID := infrastructure.NextID()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/container-images/"+infrastructure.IDStr(missingID)+"/", nil)
	req.Header.Set("Authorization", "Bearer "+staffTok)
	mux.ServeHTTP(rr, req)
	if rr.Code != 404 {
		t.Fatalf("status=%d body=%s want 404", rr.Code, rr.Body.String())
	}
}
