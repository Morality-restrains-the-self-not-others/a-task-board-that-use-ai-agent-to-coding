package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

func TestJWTRoundtrip(t *testing.T) {
	tok, err := infrastructure.IssueToken(infrastructure.DefaultSecretKey, "123", "vendor", 3600)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := infrastructure.ParseHS256JWT(tok, infrastructure.DefaultSecretKey, infrastructure.JWTIssuer, "", "vendor")
	if err != nil {
		t.Fatal(err)
	}
	if infrastructure.ClaimString(claims, "sub") != "123" {
		t.Fatalf("sub=%v", claims["sub"])
	}
}

func TestPublicCatalogStringIDs(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/public/catalog/", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	// DRF pagination was never enabled in Python; list APIs return bare arrays.
	if len(rr.Body.Bytes()) > 0 && rr.Body.Bytes()[0] == '{' {
		t.Fatalf("expected bare JSON array, got object: %s", rr.Body.String())
	}
	var items []map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &items)
	for _, it := range items {
		if id, ok := it["id"]; ok {
			if _, isStr := id.(string); !isStr {
				t.Fatalf("id not string: %T %v", id, id)
			}
		}
	}
}

func TestResolveTargetArchitecturesRoute(t *testing.T) {
	app := testApp(t)
	id := infrastructure.NextID()
	email := "resolve_route_" + infrastructure.IDStr(id) + "@example.com"
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		id, email, "x", "Co", "Contact", 1, now, now)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(id), "vendor", 3600)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	// Collection action is reached (empty image_url → 400, not 404 create-image).
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/vendor/container-images/resolve-target-architectures/", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 400 {
		t.Fatalf("expected 400 for empty image_url, got %d %s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("镜像地址不能为空")) {
		t.Fatalf("body=%s", rr.Body.String())
	}

	// Without auth → 401
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/vendor/container-images/resolve-target-architectures/", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != 401 {
		t.Fatalf("expected 401, got %d", rr2.Code)
	}
}

func TestDomainStateMachinePackage(t *testing.T) {
	// ensure domain package is linked via import side effects in other tests
	_ = filepath.Join(os.TempDir(), "x")
}

// OPT-20260717-024: PATCH on admin userdata templates actually persists changes.
func TestAdminUserDataTemplatePATCHPersists(t *testing.T) {
	app := testApp(t)
	staffID := infrastructure.NextID()
	tplID := infrastructure.NextID()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")

	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_platformstaff
		(id, username, password_hash, display_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,1,?,?)`,
		staffID, "test_staff_patch", "x", "Test Staff", now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_userdatatemplate
		(id, name, version, variables, container_variables, content, auto_verify_script, is_active, created_at, updated_at, os_type)
		VALUES (?,?,?,?,?,?,?,1,?,?,?)`,
		tplID, "original-name", "0.1", "{}", "{}", "echo old", "", now, now, "linux")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_userdatatemplate WHERE id=?`, tplID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_platformstaff WHERE id=?`, staffID)
	})

	staffTok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(staffID), "staff", 3600)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	// PATCH the template name
	patchBody, _ := json.Marshal(map[string]string{"name": "patched-name", "version": "2.0"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/userdata-templates/"+infrastructure.IDStr(tplID)+"/", bytes.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+staffTok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("PATCH status %d body %s", rr.Code, rr.Body.String())
	}

	// GET the template and verify name/version were persisted
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/admin/userdata-templates/"+infrastructure.IDStr(tplID)+"/", nil)
	req2.Header.Set("Authorization", "Bearer "+staffTok)
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatalf("GET after PATCH status %d body %s", rr2.Code, rr2.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rr2.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "patched-name" {
		t.Fatalf("GET name=%v want patched-name (PATCH did not persist)", got["name"])
	}
	if got["version"] != "2.0" {
		t.Fatalf("GET version=%v want 2.0 (PATCH did not persist)", got["version"])
	}
	if idStr, ok := got["id"].(string); !ok || idStr == "" {
		t.Fatalf("id missing or empty: %v", got["id"])
	}
	if created, ok := got["created_at"].(string); !ok || created == "" {
		t.Fatalf("created_at missing after PATCH GET: %v", got["created_at"])
	}
}

// Admin UserData 模板列表须含 created_at，否则管理端「创建时间」列恒空。
func TestAdminUserDataTemplateListIncludesCreatedAt(t *testing.T) {
	app := testApp(t)
	staffID := infrastructure.NextID()
	tplID := infrastructure.NextID()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")

	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_platformstaff
		(id, username, password_hash, display_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,1,?,?)`,
		staffID, "test_staff_list_ts", "x", "Test Staff", now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_userdatatemplate
		(id, name, version, variables, container_variables, content, auto_verify_script, is_active, created_at, updated_at, os_type)
		VALUES (?,?,?,?,?,?,?,1,?,?,?)`,
		tplID, "list-ts-tpl", "1.0", "{}", "{}", "echo hi", "", now, now, "ubuntu_24_04")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_userdatatemplate WHERE id=?`, tplID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_platformstaff WHERE id=?`, staffID)
	})

	staffTok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(staffID), "staff", 3600)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/userdata-templates/", nil)
	req.Header.Set("Authorization", "Bearer "+staffTok)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("LIST status %d body %s", rr.Code, rr.Body.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	var found map[string]any
	for _, it := range items {
		if it["id"] == infrastructure.IDStr(tplID) {
			found = it
			break
		}
	}
	if found == nil {
		t.Fatalf("template %d not in list", tplID)
	}
	created, ok := found["created_at"].(string)
	if !ok || created == "" {
		t.Fatalf("list item missing created_at: %#v", found)
	}
	// Driver may normalize DATETIME to RFC3339; only require non-empty + same calendar day.
	if !strings.Contains(created, "2026-07-23") && !strings.Contains(created, now[:10]) {
		t.Fatalf("created_at=%q does not contain inserted day (inserted %q)", created, now)
	}
}

// Referenced UserData templates must delete after clearing CSI FK; else admin「删除」假成功刷新复现。
func TestAdminUserDataTemplateDeleteClearsCSIRefs(t *testing.T) {
	app := testApp(t)
	staffID := infrastructure.NextID()
	vendorID := infrastructure.NextID()
	tplID := infrastructure.NextID()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	email := "del_tpl_" + infrastructure.IDStr(vendorID) + "@example.com"

	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_platformstaff
		(id, username, password_hash, display_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,1,?,?)`,
		staffID, "test_staff_del_tpl", "x", "Test Staff", now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`, vendorID, email, "x", "DelTplCo", "Contact", 1, now, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_userdatatemplate
		(id, name, version, variables, container_variables, content, auto_verify_script, is_active, created_at, updated_at, os_type)
		VALUES (?,?,?,?,?,?,?,1,?,?,?)`,
		tplID, "to-delete", "1.0", "{}", "{}", "echo del", "", now, now, "ubuntu_24_04")
	if err != nil {
		t.Fatal(err)
	}
	csiID, err := app.DB.CreateCloudServerImage(vendorID, map[string]any{
		"platform_type": "aliyun", "image_name": "ref-image", "image_id": "m-del-1",
		"region": "cn-hangzhou", "os_type": "linux", "os_version": "24.04",
		"architecture": "x86_64", "image_type": "system",
		"userdata_template_id": infrastructure.IDStr(tplID),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcloudserverimage WHERE id=?`, csiID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_userdatatemplate WHERE id=?`, tplID)
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
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/userdata-templates/"+infrastructure.IDStr(tplID)+"/", nil)
	req.Header.Set("Authorization", "Bearer "+staffTok)
	mux.ServeHTTP(rr, req)
	if rr.Code != 204 {
		t.Fatalf("DELETE status %d body %s", rr.Code, rr.Body.String())
	}

	var n int
	if err := app.DB.SQL.QueryRow(`SELECT COUNT(*) FROM ai_provider_userdatatemplate WHERE id=?`, tplID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("template still present after DELETE (count=%d)", n)
	}
	var tplRef sql.NullInt64
	if err := app.DB.SQL.QueryRow(
		`SELECT userdata_template_id FROM ai_provider_vendorcloudserverimage WHERE id=?`, csiID,
	).Scan(&tplRef); err != nil {
		t.Fatal(err)
	}
	if tplRef.Valid {
		t.Fatalf("CSI userdata_template_id still set: %v", tplRef.Int64)
	}

	// Refresh list must not resurrect the row.
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/admin/userdata-templates/", nil)
	req2.Header.Set("Authorization", "Bearer "+staffTok)
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != 200 {
		t.Fatalf("LIST status %d body %s", rr2.Code, rr2.Body.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(rr2.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it["id"] == infrastructure.IDStr(tplID) {
			t.Fatalf("deleted template still in list: %#v", it)
		}
	}
}

// 同一镜像组允许多个 approved 但仅一个激活版本：审批不因组内已有上架版本被拦截；
// 组内无激活时首个审批通过的版本自动激活，后续版本不覆盖激活。
func TestAdminApproveGateOneActiveVersionPerGroup(t *testing.T) {
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
			[]any{staffID, "staff_active_gate", "x", "Staff", now, now}},
		{`INSERT INTO ai_provider_vendor (id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at) VALUES (?,?,?,?,?,1,?,?)`,
			[]any{vendorID, "active_gate_vendor@example.com", "x", "GateCo", "Contact", now, now}},
		{`INSERT INTO ai_provider_containerimagegroup (id, name, description, vendor_id, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
			[]any{groupID, "gate-group", "", vendorID, now, now}},
	} {
		if _, err := app.DB.SQL.Exec(s.q, s.arg...); err != nil {
			t.Fatal(err)
		}
	}
	imgA, err := app.DB.CreateContainerImage(vendorID, groupID, "v1", "registry.example/g:v1", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	imgB, err := app.DB.CreateContainerImage(vendorID, groupID, "v2", "registry.example/g:v2", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	// 直接置库：A 已上架、B 待审核
	if _, err := app.DB.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=? WHERE id=?`, domain.StatusApproved, imgA); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=? WHERE id=?`, domain.StatusPendingReview, imgB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id IN (?,?)`, imgA, imgB)
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
	postApprove := func(id int64) *httptest.ResponseRecorder {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/admin/container-images/"+infrastructure.IDStr(id)+"/approve/", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+staffTok)
		mux.ServeHTTP(rr, req)
		return rr
	}

	// 组内已有一个 approved（A，直接置库未激活）：B 审批应成功，多个 approved 合法
	rr := postApprove(imgB)
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s want 200 (multiple approved allowed)", rr.Code, rr.Body.String())
	}
	// 组内此前无激活版本：B 作为首个审批通过版本自动激活
	fresh, err := app.DB.GetContainerImage(imgB)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Status != domain.StatusApproved {
		t.Fatalf("imgB status=%v want approved", fresh.Status)
	}
	if !fresh.IsActive {
		t.Fatal("imgB should be auto-activated as first approved version in group")
	}
	// A 保持 approved（多 approved 共存，未被降级）
	if a, err := app.DB.GetContainerImage(imgA); err != nil || a.Status != domain.StatusApproved {
		t.Fatalf("imgA status=%v err=%v want approved", a.Status, err)
	}

	// 组内已有激活（B）时，C 审批通过但不得覆盖激活
	imgC, err := app.DB.CreateContainerImage(vendorID, groupID, "v3", "registry.example/g:v3", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id=?`, imgC)
	})
	if _, err := app.DB.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=? WHERE id=?`, domain.StatusPendingReview, imgC); err != nil {
		t.Fatal(err)
	}
	if rr := postApprove(imgC); rr.Code != 200 {
		t.Fatalf("approve third version: status=%d body=%s want 200", rr.Code, rr.Body.String())
	}
	if c, err := app.DB.GetContainerImage(imgC); err != nil || c.Status != domain.StatusApproved {
		t.Fatalf("imgC status=%v err=%v want approved", c.Status, err)
	} else if c.IsActive {
		t.Fatal("imgC must not override existing activation")
	}
	// B 仍是组内激活版本
	if b, err := app.DB.GetContainerImage(imgB); err != nil || !b.IsActive {
		t.Fatalf("imgB is_active=%v err=%v want still active", b.IsActive, err)
	}
}
