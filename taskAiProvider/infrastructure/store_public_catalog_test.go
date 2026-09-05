package infrastructure

import (
	"strings"
	"testing"
	"time"

	dbload "dbload"
)

func openPublicCatalogTestDB(t *testing.T) *DB {
	t.Helper()
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	dsn, cleanup, err := dbload.OpenTestMySQL("ai-provider", root)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}
	t.Cleanup(cleanup)
	db, err := OpenDB(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	stmts := []string{
		`CREATE TABLE ai_provider_vendor (
			id BIGINT PRIMARY KEY, saas_user_id INTEGER, email TEXT, password_hash TEXT,
			company_name TEXT, contact_name TEXT,
			id_card_file_key VARCHAR(512) NULL, business_license_file_key VARCHAR(512) NULL,
			contact_phone VARCHAR(32) NULL,
			is_active INTEGER, review_note TEXT,
			reviewed_at DATETIME, reviewed_by INTEGER, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE ai_provider_containerimagegroup (
			id BIGINT PRIMARY KEY, name TEXT, description TEXT, icon_file_key VARCHAR(512) NOT NULL DEFAULT '', vendor_id INTEGER, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE ai_provider_vendorcontainerimage (
			id BIGINT PRIMARY KEY, version TEXT, image_url TEXT, target_architectures TEXT, size INTEGER,
			status TEXT, is_active INTEGER DEFAULT 0, review_note TEXT, reviewed_at TEXT, created_at TEXT, updated_at DATETIME,
			image_group_id INTEGER, reviewer_id INTEGER, vendor_id INTEGER,
			auto_run_steps_md TEXT, auto_run_steps_extract_status TEXT, auto_run_steps_digest TEXT,
			image_skills_json TEXT, image_skills_extract_status TEXT, image_skills_digest TEXT,
			saas_inbound_skill_version VARCHAR(32) NOT NULL DEFAULT '1'
		)`,
		`CREATE TABLE ai_provider_userdatatemplate (
			id BIGINT PRIMARY KEY, name TEXT, version TEXT, os_type TEXT,
			variables TEXT, container_variables TEXT, content TEXT, auto_verify_script TEXT, is_active INTEGER,
			created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE ai_provider_vendorcloudserverimage (
			id BIGINT PRIMARY KEY, vendor_id INTEGER, platform_type TEXT, image_name TEXT, image_id TEXT,
			region TEXT, os_type TEXT, os_version TEXT, architecture TEXT, image_type TEXT,
			image_size_gb INTEGER, is_active INTEGER, default_instance_type_id TEXT,
			default_instance_type_label TEXT, base_cpu_cores INTEGER, base_memory_gib INTEGER,
			userdata_template_id INTEGER, created_at TEXT, updated_at TEXT
		)`,
		`CREATE TABLE ai_provider_containercloudserverassociation (
			id BIGINT PRIMARY KEY, platform_type TEXT, region TEXT,
			cloud_server_image_id INTEGER, container_image_id INTEGER
		)`,
	}
	for _, s := range stmts {
		if _, err := db.SQL.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, saas_user_id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (10, 8501, 'v@example.com', 'x', '测试厂商', '', 1, 't', 't')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup
		(id, name, description, icon_file_key, vendor_id, created_at, updated_at)
		VALUES (20, 'trae0630', 'dev image desc', '10/image_group_icon_1.png', 10, 't', 't')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.SQL.Exec(`INSERT INTO ai_provider_vendorcontainerimage
		(id, version, image_url, target_architectures, size, status, review_note, reviewed_at, created_at, updated_at,
		 image_group_id, reviewer_id, vendor_id, auto_run_steps_md, auto_run_steps_extract_status, auto_run_steps_digest,
		 saas_inbound_skill_version)
		VALUES (30, '1344', 'registry.example/trae:1344', '["x86_64","unknown"]', NULL, 'draft', '', NULL, 't', '2026-08-20 12:00:00',
		        20, NULL, 10, '# steps', 'done', 'digest', '1')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.SQL.Exec(`INSERT INTO ai_provider_userdatatemplate
		(id, name, version, os_type, variables, container_variables, content, auto_verify_script, is_active, created_at, updated_at)
		VALUES (40, 'boot', '1', 'linux', '[]', '[]', '#', '', 1, 't', 't')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.SQL.Exec(`INSERT INTO ai_provider_vendorcloudserverimage
		(id, vendor_id, platform_type, image_name, image_id, region, os_type, os_version, architecture, image_type,
		 is_active, default_instance_type_id, default_instance_type_label, base_cpu_cores, base_memory_gib,
		 userdata_template_id, created_at, updated_at)
		VALUES (50, 10, 'aliyun', 'ubuntu-hk', 'm-hk', 'cn-hongkong', 'linux', '24.04', 'x86_64', 'system',
		        1, 'ecs.s2.large', 'ecs.s2.large (2vCPU 4GiB)', 2, 4, 40, 't', 't')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.SQL.Exec(`INSERT INTO ai_provider_containercloudserverassociation
		(id, platform_type, region, cloud_server_image_id, container_image_id)
		VALUES (60, 'aliyun', 'cn-hongkong', 50, 30)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestVendorDevelopmentCatalogReturnsFullPublicPayload(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	items, err := db.VendorDevelopmentCatalog(8501)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d want 1", len(items))
	}
	item := items[0]
	if item["name"] != "trae0630" {
		t.Fatalf("name=%v", item["name"])
	}
	if item["description"] != "dev image desc" {
		t.Fatalf("description=%v", item["description"])
	}
	iconURL, _ := item["icon_url"].(string)
	if iconURL == "" || !strings.Contains(iconURL, "/api/ai-provider/public-image-groups/20/icon") {
		t.Fatalf("icon_url=%v", item["icon_url"])
	}
	ig, _ := item["image_group"].(map[string]any)
	if ig["icon_url"] != iconURL {
		t.Fatalf("image_group.icon_url=%v want %s", ig["icon_url"], iconURL)
	}
	if item["version"] != "1344" {
		t.Fatalf("version=%v", item["version"])
	}
	if item["saas_inbound_skill_version"] != "1" {
		t.Fatalf("saas_inbound_skill_version=%v", item["saas_inbound_skill_version"])
	}
	if item["image_skills"] == nil {
		t.Fatal("image_skills missing")
	}
	if item["status_display"] != "草稿" {
		t.Fatalf("status_display=%v", item["status_display"])
	}
	if item["is_development"] != true {
		t.Fatalf("is_development=%v", item["is_development"])
	}
	arch, ok := item["target_architectures"].([]any)
	if !ok || len(arch) != 2 || arch[0] != "x86_64" {
		t.Fatalf("target_architectures=%v", item["target_architectures"])
	}
	vendor, ok := item["vendor"].(map[string]any)
	if !ok || vendor["company_name"] != "测试厂商" || vendor["id"] != "10" {
		t.Fatalf("vendor=%v", item["vendor"])
	}
	if item["id"] != "30" {
		t.Fatalf("id=%v want string 30", item["id"])
	}
	updatedAt, _ := item["updated_at"].(string)
	if updatedAt == "" || !strings.Contains(updatedAt, "2026-08-20") {
		t.Fatalf("updated_at=%v want date 2026-08-20", item["updated_at"])
	}
	envs, ok := item["runtime_environments"].([]map[string]any)
	if !ok || len(envs) != 1 {
		t.Fatalf("runtime_environments=%T %#v", item["runtime_environments"], item["runtime_environments"])
	}
	if envs[0]["platform_type_display"] != "阿里云" {
		t.Fatalf("platform_type_display=%v", envs[0]["platform_type_display"])
	}
	if envs[0]["hardware_summary"] == "" {
		t.Fatal("hardware_summary empty")
	}
}

func TestUnsubmittedImageByIDsReturnsFullPayload(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	item, err := db.UnsubmittedImageByIDs(10, 30)
	if err != nil {
		t.Fatal(err)
	}
	if item["name"] != "trae0630" {
		t.Fatalf("name=%v", item["name"])
	}
	vendor := item["vendor"].(map[string]any)
	if vendor["company_name"] != "测试厂商" {
		t.Fatalf("vendor=%v", vendor)
	}
}

func TestListApprovedCatalogFlattensName(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	_, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status='approved' WHERE id=30`)
	if err != nil {
		t.Fatal(err)
	}
	items, err := db.ListApprovedCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0]["name"] != "trae0630" {
		t.Fatalf("name=%v", items[0]["name"])
	}
	updatedAt, _ := items[0]["updated_at"].(string)
	if updatedAt == "" || !strings.Contains(updatedAt, "2026-08-20") {
		t.Fatalf("updated_at=%v want date 2026-08-20", items[0]["updated_at"])
	}
	if items[0]["saas_inbound_skill_version"] != "1" {
		t.Fatalf("saas_inbound_skill_version=%v", items[0]["saas_inbound_skill_version"])
	}
	if items[0]["vendor"].(map[string]any)["company_name"] != "测试厂商" {
		t.Fatalf("vendor=%v", items[0]["vendor"])
	}
}

// 同组多 approved 时目录只返回激活版本；无激活版本时兜底返回最新 approved。
func TestListApprovedCatalogPrefersActiveVersion(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	// 新组：两个 approved 版本，仅 v2 激活
	gid := int64(22) // 测试库 image_group_id 为 INTEGER，NextID snowflake 会溢出
	if _, err := db.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup (id, name, description, vendor_id, created_at, updated_at) VALUES (?,?,?,10,?,?)`,
		gid, "multi-approved", "multi approved catalog test", now, now); err != nil {
		t.Fatal(err)
	}
	v1, err := db.CreateContainerImage(10, gid, "v1", "registry.example/m:v1", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	v2, err := db.CreateContainerImage(10, gid, "v2", "registry.example/m:v2", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{v1, v2} {
		if _, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=? WHERE id=?`, "approved", id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET is_active=1 WHERE id=?`, v2); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id IN (?,?)`, v1, v2)
		_, _ = db.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, gid)
	})
	items, err := db.ListApprovedCatalog()
	if err != nil {
		t.Fatal(err)
	}
	var matched bool
	for _, it := range items {
		if it["name"] != "multi-approved" {
			continue
		}
		matched = true
		if it["version"] != "v2" {
			t.Fatalf("catalog version=%v want v2 (active wins over non-active approved)", it["version"])
		}
		if it["is_active"] != true {
			t.Fatalf("is_active=%v want true", it["is_active"])
		}
	}
	if !matched {
		t.Fatal("multi-approved group missing from catalog")
	}

	// 撤销激活后：目录兜底返回（仅剩的）approved 版本 v1
	if _, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET is_active=0 WHERE id=?`, v2); err != nil {
		t.Fatal(err)
	}
	items, err = db.ListApprovedCatalog()
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it["name"] != "multi-approved" {
			continue
		}
		if it["version"] != "v1" {
			t.Fatalf("catalog version=%v want v1 (fallback to latest approved)", it["version"])
		}
		if it["is_active"] != false {
			t.Fatalf("is_active=%v want false", it["is_active"])
		}
	}
}

// 关联表 region 与 CSI.region 漂移时，运行环境须以 CSI 为准（否则 start-vm 按模板地域过滤会「未找到匹配地域」）
func TestPublicRuntimeEnvironmentsPrefersCloudServerImageRegion(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	_, err := db.SQL.Exec(`UPDATE ai_provider_vendorcloudserverimage SET region='cn-qingdao' WHERE id=50`)
	if err != nil {
		t.Fatal(err)
	}
	// 故意保留关联表旧地域 cn-hongkong
	_, err = db.SQL.Exec(`UPDATE ai_provider_containercloudserverassociation SET region='cn-hongkong' WHERE id=60`)
	if err != nil {
		t.Fatal(err)
	}
	envs, err := db.publicRuntimeEnvironments(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(envs) != 1 {
		t.Fatalf("len=%d want 1", len(envs))
	}
	if envs[0]["region"] != "cn-qingdao" {
		t.Fatalf("region=%v want cn-qingdao (CSI wins over association)", envs[0]["region"])
	}
}
