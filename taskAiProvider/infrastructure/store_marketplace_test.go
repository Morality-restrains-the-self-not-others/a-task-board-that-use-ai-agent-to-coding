package infrastructure

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"taskAiProvider/domain"
)

func TestListContainerImagesCarriesUpdatedAt(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	items, err := db.ListContainerImages(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d want 1", len(items))
	}
	updatedAt, _ := items[0]["updated_at"].(string)
	if updatedAt == "" || !strings.Contains(updatedAt, "2026-08-20") {
		t.Fatalf("updated_at=%v want non-empty date 2026-08-20", items[0]["updated_at"])
	}
}

func TestListContainerImagesAdminCarriesUpdatedAt(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	items, err := db.ListContainerImagesAdmin("draft")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d want 1", len(items))
	}
	updatedAt, _ := items[0]["updated_at"].(string)
	if updatedAt == "" || !strings.Contains(updatedAt, "2026-08-20") {
		t.Fatalf("updated_at=%v want non-empty date 2026-08-20", items[0]["updated_at"])
	}
}

// reviewHistoryTestTables creates the review-history + staff tables (mirroring
// dataMigrate/taskAiProvider/008_review_history.sql) plus seed rows.
func reviewHistoryTestTables(t *testing.T, db *DB, seed bool) {
	t.Helper()
	for _, s := range []string{
		`CREATE TABLE ai_provider_platformstaff (
			id BIGINT PRIMARY KEY, username TEXT, password_hash TEXT, display_name TEXT,
			is_active INTEGER, created_at TEXT, updated_at TEXT, saas_superadmin_id INTEGER
		)`,
		`CREATE TABLE ai_provider_containerimagereviewhistory (
			id BIGINT PRIMARY KEY, action TEXT, note TEXT, reviewed_at DATETIME NULL, created_at DATETIME NULL,
			container_image_id INTEGER, reviewer_id INTEGER
		)`,
	} {
		if _, err := db.SQL.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	if !seed {
		return
	}
	if _, err := db.SQL.Exec(`INSERT INTO ai_provider_platformstaff (id, username, display_name, is_active, created_at, updated_at) VALUES (70, 'ops', '运营甲', 1, 't', 't')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`INSERT INTO ai_provider_containerimagereviewhistory (id, action, note, reviewed_at, created_at, container_image_id, reviewer_id) VALUES (71, 'reject', '描述不完整', '2026-08-21 03:00:00', '2026-08-21 03:00:00', 30, 70)`); err != nil {
		t.Fatal(err)
	}
}

func TestListContainerImagesAdminEnrichedFields(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	reviewHistoryTestTables(t, db, true)

	items, err := db.ListContainerImagesAdmin("draft")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d want 1", len(items))
	}
	item := items[0]
	if item["name"] != "trae0630" {
		t.Fatalf("name=%v want trae0630", item["name"])
	}
	if item["description"] != "dev image desc" {
		t.Fatalf("description=%v want dev image desc", item["description"])
	}
	if item["vendor_company"] != "测试厂商" {
		t.Fatalf("vendor_company=%v want 测试厂商", item["vendor_company"])
	}
	wantIcon := ImageGroupIconURL(20, "10/image_group_icon_1.png")
	if item["icon_url"] != wantIcon {
		t.Fatalf("icon_url=%v want %s (admin list must expose group icon)", item["icon_url"], wantIcon)
	}
	rts, _ := item["runtime_environments"].([]map[string]any)
	if len(rts) != 1 || rts[0]["platform_type"] != "aliyun" || rts[0]["region"] != "cn-hongkong" {
		t.Fatalf("runtime_environments=%v want [aliyun/cn-hongkong]", rts)
	}
	pts, _ := item["platform_types"].([]any)
	if len(pts) != 1 || pts[0] != "aliyun" {
		t.Fatalf("platform_types=%v want [aliyun]", pts)
	}
	hs, _ := item["review_histories"].([]map[string]any)
	if len(hs) != 1 {
		t.Fatalf("review_histories len=%d want 1", len(hs))
	}
	if hs[0]["action"] != "reject" || hs[0]["note"] != "描述不完整" || hs[0]["reviewer_name"] != "运营甲" {
		t.Fatalf("review_histories[0]=%v want reject/描述不完整/运营甲", hs[0])
	}
}

func TestGetContainerImageAdminEnrichedFields(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	reviewHistoryTestTables(t, db, true)

	if _, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET image_skills_json=? WHERE id=30`,
		`{"version":1,"default_skill":"code","skills":[{"name":"code","is_default":true}]}`); err != nil {
		t.Fatal(err)
	}

	item, err := db.GetContainerImageAdmin(30)
	if err != nil {
		t.Fatal(err)
	}
	skills, ok := item["image_skills"].(domain.ImageSkillList)
	if !ok || !skills.Has("code") {
		t.Fatalf("image_skills=%v want list containing code", item["image_skills"])
	}
	rts, _ := item["runtime_environments"].([]map[string]any)
	if len(rts) != 1 || rts[0]["platform_type"] != "aliyun" {
		t.Fatalf("runtime_environments=%v want [aliyun]", rts)
	}
	hs, _ := item["review_histories"].([]map[string]any)
	if len(hs) != 1 || hs[0]["action"] != "reject" {
		t.Fatalf("review_histories=%v want one reject", hs)
	}

	_, err = db.GetContainerImageAdmin(999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing id err=%v want sql.ErrNoRows", err)
	}
}

// 老库（008 迁移未应用）场景：review 表缺失时列表不报错、字段缺省为空。
func TestListContainerImagesAdminFailsOpenWithoutReviewTable(t *testing.T) {
	db := openPublicCatalogTestDB(t)

	items, err := db.ListContainerImagesAdmin("draft")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d want 1", len(items))
	}
	hs, _ := items[0]["review_histories"].([]map[string]any)
	if len(hs) != 0 {
		t.Fatalf("review_histories=%v want empty", hs)
	}
}

// InsertReviewHistory 与 008 迁移表结构契约一致（此前表缺失导致写错误被忽略）。
func TestInsertReviewHistoryPersists(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	reviewHistoryTestTables(t, db, false)

	if err := db.InsertReviewHistory(30, "reject", "驳回原因", 70); err != nil {
		t.Fatalf("InsertReviewHistory: %v", err)
	}
	hist := db.batchReviewHistories([]int64{30})
	if len(hist[30]) != 1 {
		t.Fatalf("hist len=%d want 1", len(hist[30]))
	}
	h := hist[30][0]
	if h["action"] != "reject" || h["note"] != "驳回原因" {
		t.Fatalf("hist=%v want reject/驳回原因", h)
	}
	// reviewer_id 不存在于 staff 表时 reviewer_name 为空但不报错
	if h["reviewer_name"] != "" {
		t.Fatalf("reviewer_name=%v want empty", h["reviewer_name"])
	}
}

// 同一镜像组允许多个 approved 但仅一个激活版本：GroupActiveVersion 基于
// is_active=1 返回组内当前激活版本（排除 excludeID）。
func TestGroupActiveVersionExcludesSelf(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	gid := int64(21) // 测试库 image_group_id 为 INTEGER，NextID snowflake 会溢出
	if _, err := db.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup (id, name, description, vendor_id, created_at, updated_at) VALUES (?,?,?,10,?,?)`,
		gid, "single-active", "", now, now); err != nil {
		t.Fatal(err)
	}
	imgA, err := db.CreateContainerImage(10, gid, "v1", "registry.example/g:v1", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	imgB, err := db.CreateContainerImage(10, gid, "v2", "registry.example/g:v2", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	// A、B 都 approved（允许多个），但仅 A 激活
	if _, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=?, is_active=? WHERE id=?`, domain.StatusApproved, 1, imgA); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=? WHERE id=?`, domain.StatusApproved, imgB); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id IN (?,?)`, imgA, imgB)
		_, _ = db.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, gid)
	})
	active, err := db.GroupActiveVersion(gid, imgB)
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ID != imgA || active.Version != "v1" {
		t.Fatalf("active=%+v want id=%d version=v1", active, imgA)
	}
	// 排除自身：已是激活版本的镜像查询不应命中自己
	none, err := db.GroupActiveVersion(gid, imgA)
	if err != nil {
		t.Fatal(err)
	}
	if none != nil {
		t.Fatalf("exclude self failed: %+v", none)
	}
	// 非激活的 approved 不计为激活版本
	none2, err := db.GroupActiveVersion(gid, imgA)
	if err != nil || none2 != nil {
		t.Fatalf("non-active approved must not match: %+v err=%v", none2, err)
	}
}

// 激活切换：多个 approved 时显式激活一个，自动取消同组原激活，二者均保持 approved。
func TestActivateContainerImageSwitchesWithinGroup(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	gid := int64(21) // 测试库 image_group_id 为 INTEGER，NextID snowflake 会溢出
	if _, err := db.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup (id, name, description, vendor_id, created_at, updated_at) VALUES (?,?,?,10,?,?)`,
		gid, "switch-active", "", now, now); err != nil {
		t.Fatal(err)
	}
	imgA, err := db.CreateContainerImage(10, gid, "v1", "registry.example/g:v1", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	imgB, err := db.CreateContainerImage(10, gid, "v2", "registry.example/g:v2", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{imgA, imgB} {
		if _, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=? WHERE id=?`, domain.StatusApproved, id); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id IN (?,?)`, imgA, imgB)
		_, _ = db.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, gid)
	})
	// 激活 A
	if err := db.ActivateContainerImage(imgA); err != nil {
		t.Fatal(err)
	}
	if a, _ := db.GroupActiveVersion(gid, imgB); a == nil || a.ID != imgA {
		t.Fatalf("after activate A: active=%+v want A", a)
	}
	// 切换激活到 B：A 失去激活但保持 approved
	if err := db.ActivateContainerImage(imgB); err != nil {
		t.Fatal(err)
	}
	var aStatus, bStatus string
	var aActive, bActive int
	if err := db.SQL.QueryRow(`SELECT status, is_active FROM ai_provider_vendorcontainerimage WHERE id=?`, imgA).Scan(&aStatus, &aActive); err != nil {
		t.Fatal(err)
	}
	if err := db.SQL.QueryRow(`SELECT status, is_active FROM ai_provider_vendorcontainerimage WHERE id=?`, imgB).Scan(&bStatus, &bActive); err != nil {
		t.Fatal(err)
	}
	if aStatus != domain.StatusApproved || aActive != 0 {
		t.Fatalf("A after switch: status=%s is_active=%d want approved/0", aStatus, aActive)
	}
	if bStatus != domain.StatusApproved || bActive != 1 {
		t.Fatalf("B after switch: status=%s is_active=%d want approved/1", bStatus, bActive)
	}
	// 非 approved 版本不可激活
	draftID, err := db.CreateContainerImage(10, gid, "v3", "registry.example/g:v3", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ActivateContainerImage(draftID); err == nil {
		t.Fatal("draft image must not be activatable")
	}
}

// 首版自动激活：组内无激活时激活指定镜像；已有激活时不覆盖。
func TestActivateIfNoActiveVersion(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	gid := int64(21) // 测试库 image_group_id 为 INTEGER，NextID snowflake 会溢出
	if _, err := db.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup (id, name, description, vendor_id, created_at, updated_at) VALUES (?,?,?,10,?,?)`,
		gid, "auto-active", "", now, now); err != nil {
		t.Fatal(err)
	}
	imgA, err := db.CreateContainerImage(10, gid, "v1", "registry.example/g:v1", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	imgB, err := db.CreateContainerImage(10, gid, "v2", "registry.example/g:v2", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{imgA, imgB} {
		if _, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=? WHERE id=?`, domain.StatusApproved, id); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id IN (?,?)`, imgA, imgB)
		_, _ = db.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, gid)
	})
	// 组内无激活：A 自动激活
	if err := db.ActivateIfNoActiveVersion(imgA, gid); err != nil {
		t.Fatal(err)
	}
	if a, _ := db.GroupActiveVersion(gid, imgB); a == nil || a.ID != imgA {
		t.Fatalf("first activation: active=%+v want A", a)
	}
	// 已有激活（A）：B 不得覆盖
	if err := db.ActivateIfNoActiveVersion(imgB, gid); err != nil {
		t.Fatal(err)
	}
	if a, _ := db.GroupActiveVersion(gid, imgB); a == nil || a.ID != imgA {
		t.Fatalf("second approve must not override: active=%+v want A", a)
	}
}

// 列表富化：同组已有激活版本时，非激活项携带 group_active_version 与 is_active=false，
// 激活项自身 is_active=true 且不携带 group_active_version。
func TestListContainerImagesMarksGroupActiveVersion(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	// 自建组：追加同组已上架版本并校验列表富化
	gid := int64(21) // 测试库 image_group_id 为 INTEGER，NextID snowflake 会溢出
	if _, err := db.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup (id, name, description, vendor_id, created_at, updated_at) VALUES (?,?,?,10,?,?)`,
		gid, "mark-active", "", now, now); err != nil {
		t.Fatal(err)
	}
	imgID, err := db.CreateContainerImage(10, gid, "v-active", "registry.example/g:v-active", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	draftID, err := db.CreateContainerImage(10, gid, "v-next", "registry.example/g:v-next", []any{"x86_64"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=?, is_active=? WHERE id=?`, domain.StatusApproved, 1, imgID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=? WHERE id=?`, domain.StatusApproved, draftID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id IN (?,?)`, imgID, draftID)
		_, _ = db.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, gid)
	})
	items, err := db.ListContainerImages(10)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]map[string]any{}
	for _, it := range items {
		id, _ := it["id"].(string)
		byID[id] = it
	}
	draft := byID[IDStr(draftID)]
	if draft == nil {
		t.Fatalf("draft image %d missing from list", draftID)
	}
	if draft["is_active"] != false {
		t.Fatalf("non-active item is_active=%v want false", draft["is_active"])
	}
	av, ok := draft["group_active_version"].(map[string]any)
	if !ok {
		t.Fatalf("draft item missing group_active_version: %v", draft["group_active_version"])
	}
	if av["id"] != IDStr(imgID) || av["version"] != "v-active" {
		t.Fatalf("group_active_version=%v want {id:%s version:v-active}", av, IDStr(imgID))
	}
	activeItem := byID[IDStr(imgID)]
	if activeItem == nil {
		t.Fatalf("active image %d missing from list", imgID)
	}
	if activeItem["is_active"] != true {
		t.Fatalf("active item is_active=%v want true", activeItem["is_active"])
	}
	if _, has := activeItem["group_active_version"]; has {
		t.Fatal("active image itself must not carry group_active_version")
	}
	// 管理端列表走同一富化
	adminItems, err := db.ListContainerImagesAdmin("")
	if err != nil {
		t.Fatal(err)
	}
	var found map[string]any
	for _, it := range adminItems {
		if it["id"] == IDStr(draftID) {
			found = it
			break
		}
	}
	if found == nil {
		t.Fatalf("admin list missing draft image %d", draftID)
	}
	if av, ok := found["group_active_version"].(map[string]any); !ok || av["id"] != IDStr(imgID) {
		t.Fatalf("admin group_active_version=%v want id=%s", found["group_active_version"], IDStr(imgID))
	}
}
