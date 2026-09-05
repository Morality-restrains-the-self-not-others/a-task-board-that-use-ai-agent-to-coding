package infrastructure

import (
	"database/sql"
	"testing"
)

// TestReviewVendorIdempotentSameValue OPT-20260807-014 回归：
// 同值重复审核不得误报 ErrNoRows（handler 层映射 404「厂商不存在」）。
//
// 根因：原实现以 RowsAffected()==0 判定厂商不存在。MySQL 无 CLIENT_FOUND_ROWS 时
// UPDATE 返回「实际变更行数」而非「命中行数」；reviewed_at/updated_at 为 DATETIME
// （秒精度），Go 侧写入微秒串被截断到秒，同一秒内以相同 action/note 重复审核 →
// 行值整体不变 → RowsAffected=0 → 误判 ErrNoRows，破坏运营二次修正的幂等覆盖语义。
// 修复：UPDATE 后按 id 复查存在性（GetVendorByID），存在即返回。
// 固定 ReviewNowFunc 时钟保证两次写入相同时间戳，确定性复现 RowsAffected=0 路径。
func TestReviewVendorIdempotentSameValue(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	const vendorID = 71
	const staffID = 77701
	if _, err := db.SQL.Exec(`INSERT INTO ai_provider_vendor
		(id, saas_user_id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
		VALUES (?, 90001, 'idem@example.com', 'x', '幂等测试厂商', '', 0, '2026-08-08 00:00:00', '2026-08-08 00:00:00')`, vendorID); err != nil {
		t.Fatalf("seed vendor: %v", err)
	}
	t.Cleanup(func() { _, _ = db.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id=?`, vendorID) })

	orig := ReviewNowFunc
	ReviewNowFunc = func() string { return "2026-08-08 00:00:00.000000" }
	t.Cleanup(func() { ReviewNowFunc = orig })

	if _, err := db.ReviewVendor(vendorID, "approve", "资料齐全", staffID); err != nil {
		t.Fatalf("first review: %v", err)
	}
	v, err := db.ReviewVendor(vendorID, "approve", "资料齐全", staffID)
	if err != nil {
		t.Fatalf("same-value repeat review should succeed (idempotent overwrite), got %v", err)
	}
	if v == nil || !v.IsActive || v.ReviewNote != "资料齐全" {
		t.Fatalf("expected active reviewed vendor, got %+v", v)
	}
}

// TestReviewVendorMissingStillErrNoRows OPT-20260807-014 契约保持：
// 真不存在的厂商仍须 ErrNoRows（handler → 404）。
func TestReviewVendorMissingStillErrNoRows(t *testing.T) {
	db := openPublicCatalogTestDB(t)
	_, err := db.ReviewVendor(999999, "approve", "x", 1)
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows for missing vendor, got %v", err)
	}
}
