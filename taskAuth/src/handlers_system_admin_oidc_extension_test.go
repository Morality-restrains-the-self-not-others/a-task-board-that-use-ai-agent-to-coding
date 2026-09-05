package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260808-026: taskAuth 管理端点 GET/PUT /api/system-admin/oidc-extension/
// 设计文档 §4.3.2-4.3.3（2026-08-08-task-chrome-plugin-oidc-extension-id-admin-design.md）。

const oidcExtTestID1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const oidcExtTestID2 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func oidcExtSeedChromeExtension(t *testing.T, urisJSON string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO auth_oidc_client (id, client_id, client_secret_hash, name, redirect_uris, managed_by, created_at, updated_at)
		VALUES ('9501', 'chrome-extension', 'hash', 'chrome-extension', ?, 'bootstrap', NOW(), NOW())`, urisJSON); err != nil {
		t.Fatalf("seed chrome-extension 行: %v", err)
	}
}

// oidcExtUniqueIDs 生成 n 个互不相同的合法 32 位 a-p ID。
func oidcExtUniqueIDs(n int) []string {
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ids = append(ids, strings.Repeat("a", 30)+string(rune('a'+i/16))+string(rune('a'+i%16)))
	}
	return ids
}

func oidcExtAdminReq(t *testing.T, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rd *strings.Reader
	if body == "" {
		rd = strings.NewReader("")
	} else {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, "/api/system-admin/oidc-extension/", rd)
	req.Header.Set("X-User-Id", "bootstrap-admin") // dataMigrate 022 种子超管
	req.Header.Set("X-Trace-Id", "trace-oidc-ext-test")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handleSystemAdminOidcExtension(rec, req)
	return rec
}

func oidcExtDecode(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal %d response: %v\n%s", rec.Code, err, rec.Body.String())
	}
	return body
}

func TestSystemAdminOidcExtensionGetState(t *testing.T) {
	setupAuthTestDB(t)
	oidcExtSeedChromeExtension(t, `["chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/oauth-callback.html"]`)

	rec := oidcExtAdminReq(t, http.MethodGet, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET 应 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := oidcExtDecode(t, rec)
	if body["client_id"] != "chrome-extension" {
		t.Fatalf("client_id = %v", body["client_id"])
	}
	if body["managed_by"] != "bootstrap" {
		t.Fatalf("managed_by = %v", body["managed_by"])
	}
	ids, _ := body["extension_ids"].([]interface{})
	if len(ids) != 1 || ids[0] != oidcExtTestID1 {
		t.Fatalf("extension_ids = %v", body["extension_ids"])
	}
	raw, _ := body["raw_redirect_uris"].([]interface{})
	if len(raw) != 1 || !strings.Contains(raw[0].(string), oidcExtTestID1) {
		t.Fatalf("raw_redirect_uris = %v", body["raw_redirect_uris"])
	}
	if _, ok := body["name"]; !ok {
		t.Fatalf("name 缺失")
	}
}

func TestSystemAdminOidcExtensionGetClientNotFound404(t *testing.T) {
	setupAuthTestDB(t)
	rec := oidcExtAdminReq(t, http.MethodGet, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("无 chrome-extension 行 GET 应 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if body := oidcExtDecode(t, rec); body["trace_id"] != "trace-oidc-ext-test" {
		t.Fatalf("404 错误体应含 trace_id, got %v", body["trace_id"])
	}
}

func TestSystemAdminOidcExtensionPutTakeover(t *testing.T) {
	setupAuthTestDB(t)
	oidcExtSeedChromeExtension(t, `["chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/oauth-callback.html"]`)

	// PUT 一次即置位 admin（幂等接管，防 seed 自愈覆盖竞态）
	rec := oidcExtAdminReq(t, http.MethodPut,
		`{"extension_ids":["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT 应 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := oidcExtDecode(t, rec)
	if body["managed_by"] != "admin" {
		t.Fatalf("PUT 后 managed_by 应 admin, got %v", body["managed_by"])
	}
	ids, _ := body["extension_ids"].([]interface{})
	if len(ids) != 2 {
		t.Fatalf("extension_ids = %v", body["extension_ids"])
	}

	// 写库可见：loadOidcClient 读回 redirect_uris + managed_by='admin'
	row, err := loadOidcClient("chrome-extension")
	if err != nil || row == nil {
		t.Fatalf("loadOidcClient: err=%v row=%v", err, row)
	}
	if row.ManagedBy != "admin" {
		t.Fatalf("DB managed_by 应 admin, got %s", row.ManagedBy)
	}
	if !strings.Contains(row.RedirectURIs, oidcExtTestID2) {
		t.Fatalf("DB redirect_uris 应含新 ID: %s", row.RedirectURIs)
	}

	// 接管后 seed 不再覆盖（admin 行 INSERT-only）
	if err := ensureOidcClient("chrome-extension", "chrome-extension-dev-secret", "chrome-extension", `["chrome-extension://cmkahnnaofomeaodefegkgljniiphbhj/oauth-callback.html"]`); err != nil {
		t.Fatalf("ensureOidcClient: %v", err)
	}
	row2, _ := loadOidcClient("chrome-extension")
	if row2.ManagedBy != "admin" || !strings.Contains(row2.RedirectURIs, oidcExtTestID2) {
		t.Fatalf("admin 托管行被 seed 覆盖: managed_by=%s redirect_uris=%s", row2.ManagedBy, row2.RedirectURIs)
	}
}

func TestSystemAdminOidcExtensionPutValidation(t *testing.T) {
	setupAuthTestDB(t)
	oidcExtSeedChromeExtension(t, `["chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/oauth-callback.html"]`)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"非法字符（非 a-p）", `{"extension_ids":["zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"]}`, http.StatusBadRequest},
		{"长度不足", `{"extension_ids":["aaaaaaaaaaaa"]}`, http.StatusBadRequest},
		{"大写", `{"extension_ids":["AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"]}`, http.StatusBadRequest},
		{"空字符串", `{"extension_ids":[""]}`, http.StatusBadRequest},
		{"空列表", `{"extension_ids":[]}`, http.StatusBadRequest},
		{"超上限 21 个", `{"extension_ids":["` + strings.Join(oidcExtUniqueIDs(21), `","`) + `"]}`, http.StatusBadRequest},
		{"非法 JSON", `{not-json`, http.StatusBadRequest},
		{"缺失字段", `{}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := oidcExtAdminReq(t, http.MethodPut, tc.body)
			if rec.Code != tc.want {
				t.Fatalf("PUT %s 应 %d, got %d: %s", tc.name, tc.want, rec.Code, rec.Body.String())
			}
			if body := oidcExtDecode(t, rec); body["trace_id"] != "trace-oidc-ext-test" {
				t.Fatalf("错误体应含 trace_id, got %v", body["trace_id"])
			}
			// 校验失败不应写库
			row, err := loadOidcClient("chrome-extension")
			if err != nil || row == nil {
				t.Fatalf("loadOidcClient: %v", err)
			}
			if row.ManagedBy != "bootstrap" {
				t.Fatalf("校验失败不应置 admin, got %s", row.ManagedBy)
			}
		})
	}
}

func TestSystemAdminOidcExtensionPutDedup(t *testing.T) {
	setupAuthTestDB(t)
	oidcExtSeedChromeExtension(t, `["chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/oauth-callback.html"]`)

	// 输入含重复 → 去重后落库
	rec := oidcExtAdminReq(t, http.MethodPut,
		`{"extension_ids":["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT 应 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := oidcExtDecode(t, rec)
	ids, _ := body["extension_ids"].([]interface{})
	if len(ids) != 1 {
		t.Fatalf("去重后应 1 个 ID, got %v", body["extension_ids"])
	}
}

func TestSystemAdminOidcExtensionPutClientNotFound404(t *testing.T) {
	setupAuthTestDB(t)
	rec := oidcExtAdminReq(t, http.MethodPut, `{"extension_ids":["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("无 chrome-extension 行 PUT 应 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSystemAdminOidcExtensionRequiresSuperuser(t *testing.T) {
	setupAuthTestDB(t)
	oidcExtSeedChromeExtension(t, `["chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/oauth-callback.html"]`)

	// 普通用户（非超管）
	regularID, _, err := createUserWithEmailLogin("regular-oidc-ext@test.com", "hash")
	if err != nil {
		t.Fatalf("create regular user: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/oidc-extension/", nil)
	req.Header.Set("X-User-Id", regularID)
	rec := httptest.NewRecorder()
	handleSystemAdminOidcExtension(rec, req)
	if rec.Code != http.StatusForbidden && rec.Code != http.StatusUnauthorized {
		t.Fatalf("非超管应 401/403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSystemAdminOidcExtensionMethodNotAllowed(t *testing.T) {
	setupAuthTestDB(t)
	oidcExtSeedChromeExtension(t, `["chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/oauth-callback.html"]`)
	rec := oidcExtAdminReq(t, http.MethodDelete, "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE 应 405, got %d: %s", rec.Code, rec.Body.String())
	}
}
