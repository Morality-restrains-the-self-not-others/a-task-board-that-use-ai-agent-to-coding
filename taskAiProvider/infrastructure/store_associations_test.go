package infrastructure

import (
	"testing"

	dbload "dbload"
)

func openAssocTestDB(t *testing.T) *DB {
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
		`CREATE TABLE ai_provider_userdatatemplate (
			id BIGINT PRIMARY KEY, name TEXT, version TEXT, os_type TEXT,
			variables TEXT, container_variables TEXT, content TEXT, auto_verify_script TEXT, is_active INTEGER
		) ENGINE=InnoDB`,
		`CREATE TABLE ai_provider_vendorcloudserverimage (
			id BIGINT PRIMARY KEY, vendor_id INTEGER, platform_type TEXT, image_name TEXT, image_id TEXT,
			region TEXT, os_type TEXT, os_version TEXT, architecture TEXT, image_type TEXT,
			image_size_gb INTEGER, is_active INTEGER, default_instance_type_id TEXT,
			default_instance_type_label TEXT, base_cpu_cores INTEGER, base_memory_gib INTEGER,
			userdata_template_id INTEGER, created_at TEXT, updated_at TEXT
		) ENGINE=InnoDB`,
		`CREATE TABLE ai_provider_containercloudserverassociation (
			id BIGINT PRIMARY KEY, platform_type TEXT, region TEXT,
			cloud_server_image_id INTEGER, container_image_id INTEGER
		) ENGINE=InnoDB`,
	}
	for _, s := range stmts {
		if _, err := db.SQL.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.SQL.Exec(`INSERT INTO ai_provider_userdatatemplate
		(id, name, version, os_type, variables, container_variables, content, auto_verify_script, is_active)
		VALUES (501, 'boot-linux', '3', 'linux', '[]', '[]', '#', '', 1)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.SQL.Exec(`INSERT INTO ai_provider_vendorcloudserverimage
		(id, vendor_id, platform_type, image_name, image_id, region, os_type, os_version, architecture, image_type, is_active, userdata_template_id, created_at, updated_at)
		VALUES (1001, 1, 'aliyun', 'ubuntu-hk', 'm-hk', 'cn-hongkong', 'linux', '24.04', 'x86_64', 'system', 1, 501, 't', 't'),
		       (1002, 1, 'aliyun', 'ubuntu-hz', 'm-hz', 'cn-hangzhou', 'linux', '24.04', 'x86_64', 'system', 1, NULL, 't', 't')`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCloudServerImageIDFromAssocItemAcceptsLegacyField(t *testing.T) {
	if got := CloudServerImageIDFromAssocItem(map[string]any{"cloud_server_image_id": "1001"}); got != 1001 {
		t.Fatalf("canonical=%d", got)
	}
	if got := CloudServerImageIDFromAssocItem(map[string]any{"cloud_server_image": "1001"}); got != 1001 {
		t.Fatalf("legacy string=%d", got)
	}
	if got := CloudServerImageIDFromAssocItem(map[string]any{"cloud_server_image": map[string]any{"id": "1001"}}); got != 1001 {
		t.Fatalf("legacy nested=%d", got)
	}
}

func TestSetAssociationsRejectsBadPayloadWithoutWiping(t *testing.T) {
	db := openAssocTestDB(t)
	if err := db.UpsertAssociation(2001, "aliyun", "cn-hongkong", 1001); err != nil {
		t.Fatal(err)
	}
	// Legacy-only wrong shape that previously wiped all rows (DELETE then skip insert).
	err := db.SetAssociations(2001, []map[string]any{{
		"platform_type":      "aliyun",
		"region":             "cn-hongkong",
		"cloud_server_image": nil,
	}})
	if err == nil {
		t.Fatal("expected validation error")
	}
	items, err := db.ListAssociations(2001)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("associations wiped: %#v", items)
	}
}

func TestSetAssociationsAcceptsLegacyCloudServerImageField(t *testing.T) {
	db := openAssocTestDB(t)
	err := db.SetAssociations(2001, []map[string]any{{
		"platform_type":      "aliyun",
		"region":             "cn-hongkong",
		"cloud_server_image": "1001",
	}})
	if err != nil {
		t.Fatal(err)
	}
	items, err := db.ListAssociations(2001)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d", len(items))
	}
	csi, ok := items[0]["cloud_server_image"].(map[string]any)
	if !ok || csi["id"] != "1001" {
		t.Fatalf("enriched csi=%#v", items[0]["cloud_server_image"])
	}
	if items[0]["cloud_server_image_id"] != "1001" {
		t.Fatalf("id field=%v", items[0]["cloud_server_image_id"])
	}
}

func TestUpsertAssociationPreservesOtherRegions(t *testing.T) {
	db := openAssocTestDB(t)
	if err := db.UpsertAssociation(2001, "aliyun", "cn-hongkong", 1001); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAssociation(2001, "aliyun", "cn-hangzhou", 1002); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAssociation(2001, "aliyun", "cn-hongkong", 1001); err != nil {
		t.Fatal(err)
	}
	items, err := db.ListAssociations(2001)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 regions, got %#v", items)
	}
}

func TestUpsertAssociationClearWithZeroCSIRemovesOnlyThatRegion(t *testing.T) {
	db := openAssocTestDB(t)
	if err := db.UpsertAssociation(2001, "aliyun", "cn-hongkong", 1001); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAssociation(2001, "aliyun", "cn-hangzhou", 1002); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAssociation(2001, "aliyun", "cn-hongkong", 0); err != nil {
		t.Fatal(err)
	}
	items, err := db.ListAssociations(2001)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 region left, got %#v", items)
	}
	if items[0]["region"] != "cn-hangzhou" {
		t.Fatalf("wrong remaining region: %#v", items[0])
	}
}

func TestListAssociationsEnrichesUserdataTemplateNameVersion(t *testing.T) {
	db := openAssocTestDB(t)
	if err := db.UpsertAssociation(2001, "aliyun", "cn-hongkong", 1001); err != nil {
		t.Fatal(err)
	}
	items, err := db.ListAssociations(2001)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d", len(items))
	}
	csi, ok := items[0]["cloud_server_image"].(map[string]any)
	if !ok {
		t.Fatalf("csi=%#v", items[0]["cloud_server_image"])
	}
	tpl, ok := csi["userdata_template"].(map[string]any)
	if !ok {
		t.Fatalf("userdata_template=%#v", csi["userdata_template"])
	}
	if tpl["id"] != "501" || tpl["name"] != "boot-linux" || tpl["version"] != "3" {
		t.Fatalf("enriched template=%#v", tpl)
	}
}

func TestSetCloudServerImageUserdataTemplatePersists(t *testing.T) {
	db := openAssocTestDB(t)
	if err := db.SetCloudServerImageUserdataTemplate(1002, 501); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAssociation(2001, "aliyun", "cn-hangzhou", 1002); err != nil {
		t.Fatal(err)
	}
	items, err := db.ListAssociations(2001)
	if err != nil {
		t.Fatal(err)
	}
	csi := items[0]["cloud_server_image"].(map[string]any)
	tpl := csi["userdata_template"].(map[string]any)
	if tpl["id"] != "501" || tpl["name"] != "boot-linux" {
		t.Fatalf("after set template=%#v", tpl)
	}
	if err := db.SetCloudServerImageUserdataTemplate(1002, 0); err != nil {
		t.Fatal(err)
	}
	items, err = db.ListAssociations(2001)
	if err != nil {
		t.Fatal(err)
	}
	csi = items[0]["cloud_server_image"].(map[string]any)
	if csi["userdata_template"] != nil {
		t.Fatalf("want cleared template, got %#v", csi["userdata_template"])
	}
}

func TestCreateAndUpdateCloudServerImagePersistsUserdataTemplate(t *testing.T) {
	db := openAssocTestDB(t)
	id, err := db.CreateCloudServerImage(1, map[string]any{
		"platform_type":               "aliyun",
		"image_name":                  "new-img",
		"image_id":                    "m-new",
		"region":                      "cn-beijing",
		"os_type":                     "linux",
		"os_version":                  "22.04",
		"architecture":                "x86_64",
		"image_type":                  "system",
		"userdata_template_id":        "501",
		"default_instance_type_id":    "ecs.t6",
		"default_instance_type_label": "2c4g",
		"base_cpu_cores":              float64(2),
		"base_memory_gib":             float64(4),
	})
	if err != nil {
		t.Fatal(err)
	}
	items, err := db.ListCloudServerImages(1)
	if err != nil {
		t.Fatal(err)
	}
	var created map[string]any
	for _, it := range items {
		if it["id"] == IDStr(id) {
			created = it
			break
		}
	}
	if created == nil {
		t.Fatal("created CSI missing from list")
	}
	tpl, ok := created["userdata_template"].(map[string]any)
	if !ok || tpl["id"] != "501" || tpl["name"] != "boot-linux" {
		t.Fatalf("create list template=%#v", created["userdata_template"])
	}
	if err := db.UpdateCloudServerImage(id, map[string]any{
		"image_name":           "renamed",
		"userdata_template_id": nil,
	}); err != nil {
		t.Fatal(err)
	}
	items, err = db.ListCloudServerImages(1)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it["id"] == IDStr(id) {
			if it["image_name"] != "renamed" {
				t.Fatalf("name=%v", it["image_name"])
			}
			if it["userdata_template"] != nil {
				t.Fatalf("want cleared, got %#v", it["userdata_template"])
			}
			return
		}
	}
	t.Fatal("updated CSI missing")
}

func TestOptionalUserdataTemplateID(t *testing.T) {
	id, ok := OptionalUserdataTemplateID(map[string]any{"userdata_template_id": "501"})
	if !ok || id != 501 {
		t.Fatalf("string id=%d ok=%v", id, ok)
	}
	id, ok = OptionalUserdataTemplateID(map[string]any{"userdata_template_id": nil})
	if !ok || id != 0 {
		t.Fatalf("null clear id=%d ok=%v", id, ok)
	}
	_, ok = OptionalUserdataTemplateID(map[string]any{})
	if ok {
		t.Fatal("absent should be !ok")
	}
}
