package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseImageUpdatedAt(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // expected UTC RFC3339 output, empty = zero time
	}{
		{"rfc3339nano", "2026-08-20T10:11:12.345678901Z", "2026-08-20T10:11:12.345678901Z"},
		{"rfc3339", "2026-08-20T10:11:12Z", "2026-08-20T10:11:12Z"},
		{"mysql", "2026-08-20 10:11:12", "2026-08-20T10:11:12Z"},
		{"mysql-fraction", "2026-08-20 10:11:12.5", "2026-08-20T10:11:12.5Z"},
		{"empty", "", ""},
		{"garbage", "not-a-time", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseImageUpdatedAt(c.in)
			if c.want == "" {
				if !got.IsZero() {
					t.Fatalf("expected zero time, got %v", got)
				}
				return
			}
			if got.UTC().Format(time.RFC3339Nano) != c.want {
				t.Fatalf("parseImageUpdatedAt(%q) = %v, want %s", c.in, got, c.want)
			}
		})
	}
}

func TestImageTimeJSON(t *testing.T) {
	if v := imageTimeJSON(time.Time{}); v != nil {
		t.Fatalf("zero time should map to null, got %v", v)
	}
	ts := time.Date(2026, 8, 20, 10, 11, 12, 0, time.UTC)
	if v := imageTimeJSON(ts); v != "2026-08-20T10:11:12Z" {
		t.Fatalf("set time should map to RFC3339Nano, got %v", v)
	}
}

func TestInstalledImageJSONIncludesUpdatedAt(t *testing.T) {
	img := TenantInstalledImage{
		ID:          "img-1",
		TenantID:    "t1",
		Name:        "trae-agent",
		Version:     "x86_64-latest",
		Description: "agent runtime",
		InstalledAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 8, 20, 10, 11, 12, 0, time.UTC),
	}
	raw, err := json.Marshal(installedImageToJSON(img))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, `"updated_at":"2026-08-20T10:11:12Z"`) {
		t.Fatalf("expected updated_at in JSON, got %s", body)
	}

	zero := TenantInstalledImage{ID: "img-2", TenantID: "t1", Name: "no-update"}
	rawZero, _ := json.Marshal(installedImageToJSON(zero))
	if !strings.Contains(string(rawZero), `"updated_at":null`) {
		t.Fatalf("expected updated_at null when snapshot missing, got %s", rawZero)
	}
}

// TestInstalledImageListReturnsUpdatedAt verifies the collection GET surfaces the
// updated_at column (both populated and NULL rows) after the 038 migration.
func TestInstalledImageListReturnsUpdatedAt(t *testing.T) {
	setupCloudTestDB(t)
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,image_url,target_architectures,is_dev_mode,installed_at,updated_at)
		VALUES('img-updated','t1','ext-updated','trae-agent','registry.example/img','[]',0,'2026-08-01 00:00:00','2026-08-20 10:11:12'),
		       ('img-null','t1','ext-null','python-env','registry.example/py','[]',0,'2026-08-02 00:00:00',NULL)`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/installed-images/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "user-1")
	rec := httptest.NewRecorder()
	handleInstalledImageCollection(rec, req, "t1")

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	// DATETIME 字面量按服务器本地时区解释（既有约定），断言须时区无关：
	// 字面量 '2026-08-20 10:11:12' → 本地解释为 10:11:12，JSON 输出该瞬时。
	want := time.Date(2026, 8, 20, 10, 11, 12, 0, time.Local)
	var gotUpdated []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &gotUpdated); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, item := range gotUpdated {
		name := strField(item, "name")
		rawTime, has := item["updated_at"]
		if name == "python-env" {
			if !has || rawTime != nil {
				t.Fatalf("expected updated_at null for python-env, got %v", rawTime)
			}
			continue
		}
		if name != "trae-agent" {
			t.Fatalf("unexpected item %v", item)
		}
		parsed, err := time.Parse(time.RFC3339Nano, rawTime.(string))
		if err != nil {
			t.Fatalf("parse updated_at: %v", err)
		}
		if !parsed.Equal(want) {
			t.Fatalf("updated_at = %v, want %v", parsed, want)
		}
	}
}

// TestInstalledImageInstallSnapshotsUpdatedAt verifies the install path snapshots the
// catalog item's updated_at into the stored record.
func TestInstalledImageInstallSnapshotsUpdatedAt(t *testing.T) {
	setupCloudTestDB(t)
	store := newSaasHTTPStore()
	store.putMember("t1", "admin-1", "mem-1", true)
	_ = startSaasInternalMock(t, store)

	const extID = "859670982529273856"
	aiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/public/catalog/" {
			t.Fatalf("unexpected ai path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":          extID,
				"name":        "trae-agent-skill",
				"version":     "x86_64-latest",
				"description": "skill runtime",
				"image_url":   "registry.example/trae-agent-skill",
				"updated_at":  "2026-08-20T10:11:12Z",
				"vendor": map[string]interface{}{
					"id":           "vendor-1",
					"company_name": "Demo Vendor",
				},
			},
		})
	}))
	defer aiSrv.Close()
	cfg.AIProviderBaseURL = aiSrv.URL

	body := `{"external_image_id":"` + extID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/installed-images/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "admin-1")
	rec := httptest.NewRecorder()
	handleInstalledImageCollection(rec, req, "t1")

	if rec.Code != 201 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	// 安装响应直接来自内存对象：快照瞬时 = 目录原值 10:11:12Z。
	wantInstant := time.Date(2026, 8, 20, 10, 11, 12, 0, time.UTC)

	var created map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal install response: %v", err)
	}
	rawCreated, ok := created["updated_at"].(string)
	if !ok {
		t.Fatalf("expected snapshotted updated_at in install response: %s", rec.Body.String())
	}
	parsedCreated, err := time.Parse(time.RFC3339Nano, rawCreated)
	if err != nil {
		t.Fatalf("parse install response updated_at: %v", err)
	}
	if !parsedCreated.Equal(wantInstant) {
		t.Fatalf("install response updated_at = %v, want %v", parsedCreated, wantInstant)
	}

	// DB 回读：formatTime 写 UTC 字面量 '2026-08-20 10:11:12'，按服务器本地时区解释
	// （既有约定，测试机 +08：字面量视为 10:11:12+08；UTC 服务器则一致）。
	wantLocal := time.Date(2026, 8, 20, 10, 11, 12, 0, time.Local)
	var stored sql.NullTime
	err = db.QueryRow(`SELECT updated_at FROM cloud_tenant_installed_images WHERE external_image_id=?`, extID).Scan(&stored)
	if err != nil {
		t.Fatalf("read back updated_at: %v", err)
	}
	if !stored.Valid || !stored.Time.Equal(wantLocal) {
		t.Fatalf("expected stored updated_at %v, got %+v", wantLocal, stored)
	}
}
