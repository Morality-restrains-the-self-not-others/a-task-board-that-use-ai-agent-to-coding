package infrastructure

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"taskAiProvider/domain"
)

type VendorRow = domain.Vendor
type StaffRow = domain.Staff

const vendorSelectCols = `id, saas_user_id, COALESCE(email,''), COALESCE(company_name,''), COALESCE(contact_name,''),
	COALESCE(id_card_file_key,''), COALESCE(business_license_file_key,''), COALESCE(contact_phone,''),
	is_active, COALESCE(review_note,''), reviewed_at, reviewed_by`

func (d *DB) GetVendorByID(id int64) (*VendorRow, error) {
	row := d.SQL.QueryRow(`SELECT `+vendorSelectCols+` FROM ai_provider_vendor WHERE id=?`, id)
	return scanVendor(row)
}

func (d *DB) GetVendorByEmail(email string) (*VendorRow, error) {
	row := d.SQL.QueryRow(`SELECT `+vendorSelectCols+` FROM ai_provider_vendor WHERE lower(email)=lower(?)`, email)
	return scanVendor(row)
}

func (d *DB) GetVendorBySaasUserID(uid int64) (*VendorRow, error) {
	row := d.SQL.QueryRow(`SELECT `+vendorSelectCols+` FROM ai_provider_vendor WHERE saas_user_id=?`, uid)
	return scanVendor(row)
}

func scanVendor(row *sql.Row) (*VendorRow, error) {
	var v VendorRow
	var saas sql.NullInt64
	var active int
	var reviewedAt sql.NullTime
	var reviewedBy sql.NullInt64
	if err := row.Scan(&v.ID, &saas, &v.Email, &v.CompanyName, &v.ContactName,
		&v.IDCardFileKey, &v.BusinessLicenseFileKey, &v.ContactPhone,
		&active, &v.ReviewNote, &reviewedAt, &reviewedBy); err != nil {
		return nil, err
	}
	if saas.Valid {
		v.SaasUserID = &saas.Int64
	}
	v.IsActive = active != 0
	if reviewedAt.Valid {
		v.ReviewedAt = &reviewedAt.Time
	}
	if reviewedBy.Valid {
		v.ReviewedBy = &reviewedBy.Int64
	}
	return &v, nil
}

// VendorApplicationInput 提交/重提申请的材料字段。
type VendorApplicationInput struct {
	CompanyName            string
	ContactName            string
	IDCardFileKey          string
	BusinessLicenseFileKey string
	ContactPhone           string
}

// IsVendorApplicationReviewEnabled 读取全局审核开关；读失败时默认开启（保守）。
func (d *DB) IsVendorApplicationReviewEnabled() bool {
	var enabled int
	err := d.SQL.QueryRow(`SELECT COALESCE(vendor_application_review_enabled, 1) FROM ai_provider_marketplace_settings WHERE id=1`).Scan(&enabled)
	if err != nil {
		return true
	}
	return enabled != 0
}

// GetMarketplaceSettings 返回镜像市场全局设置。
func (d *DB) GetMarketplaceSettings() (reviewEnabled bool, err error) {
	var enabled int
	err = d.SQL.QueryRow(`SELECT COALESCE(vendor_application_review_enabled, 1) FROM ai_provider_marketplace_settings WHERE id=1`).Scan(&enabled)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return true, err
	}
	return enabled != 0, nil
}

// SetVendorApplicationReviewEnabled 持久化审核开关。
func (d *DB) SetVendorApplicationReviewEnabled(enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := d.SQL.Exec(`INSERT INTO ai_provider_marketplace_settings (id, vendor_application_review_enabled)
		VALUES (1, ?) ON DUPLICATE KEY UPDATE vendor_application_review_enabled=VALUES(vendor_application_review_enabled)`, v)
	return err
}

// activateVendorIfInactive 审核关闭时将待审/驳回厂商直接激活。
func (d *DB) activateVendorIfInactive(id int64, now string) error {
	_, err := d.SQL.Exec(`UPDATE ai_provider_vendor SET is_active=1, review_note=NULL, reviewed_at=NULL, reviewed_by=NULL, updated_at=? WHERE id=? AND is_active=0`, now, id)
	return err
}

// UpsertVendorFromBridge 匹配/绑定厂商档案：
// 审核开启时（默认）不再自动建号——申请为唯一建档路径；
// 审核关闭时允许自动建档/激活，租户可直达厂商门户 SSO。
func (d *DB) UpsertVendorFromBridge(saasID int64, email string) (*VendorRow, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	reviewEnabled := d.IsVendorApplicationReviewEnabled()
	existing, err := d.GetVendorBySaasUserID(saasID)
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	if err == nil && existing != nil {
		if strings.ToLower(strings.TrimSpace(existing.Email)) != email {
			_, err = d.SQL.Exec(`UPDATE ai_provider_vendor SET email=?, updated_at=? WHERE id=?`, email, now, existing.ID)
			if err != nil {
				return nil, err
			}
		}
		if !reviewEnabled && !existing.IsActive {
			if err := d.activateVendorIfInactive(existing.ID, now); err != nil {
				return nil, err
			}
		}
		return d.GetVendorByID(existing.ID)
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	existing, err = d.GetVendorByEmail(email)
	if err == nil && existing != nil {
		if existing.SaasUserID != nil && *existing.SaasUserID != saasID {
			return nil, fmt.Errorf("该邮箱已绑定其他主站用户")
		}
		if existing.SaasUserID == nil {
			_ = d.BindVendorSaasUser(existing.ID, saasID)
		}
		if !reviewEnabled && !existing.IsActive {
			if err := d.activateVendorIfInactive(existing.ID, now); err != nil {
				return nil, err
			}
		}
		return d.GetVendorByID(existing.ID)
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if !reviewEnabled {
		id := NextID()
		_, err = d.SQL.Exec(`INSERT INTO ai_provider_vendor (id, saas_user_id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at) VALUES (?,?,?,?,?,?,1,?,?)`,
			id, saasID, email, RandomPasswordHash(), "", "", now, now)
		if err != nil {
			return nil, err
		}
		return d.GetVendorByID(id)
	}
	return nil, fmt.Errorf("未找到厂商档案，请先在厂商门户申请认证")
}

// ApplyVendorApplication 提交/重提厂商申请（OPT-20260806-065 审核流 + 证照/联系方式）：
// 无记录 → INSERT（is_active=0 待审核）；已驳回（review_note 非空）→ UPDATE 覆盖申请单。
// 返回 (vendor, alreadyExists, err)：alreadyExists=true 表示 pending/qualified 冲突（409）。
func (d *DB) ApplyVendorApplication(saasID int64, email string, in VendorApplicationInput) (*VendorRow, bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	existing, err := d.GetVendorBySaasUserID(saasID)
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	if err == nil && existing != nil {
		if existing.IsActive || existing.ReviewNote == "" {
			return existing, true, nil // qualified 或 pending 中，冲突
		}
		// rejected：覆盖申请单重新申请
		_, err = d.SQL.Exec(`UPDATE ai_provider_vendor SET email=?, company_name=?, contact_name=?, id_card_file_key=?, business_license_file_key=?, contact_phone=?, review_note=NULL, reviewed_at=NULL, reviewed_by=NULL, updated_at=? WHERE id=?`,
			email, in.CompanyName, in.ContactName, in.IDCardFileKey, in.BusinessLicenseFileKey, in.ContactPhone, now, existing.ID)
		if err != nil {
			return nil, false, err
		}
		v, err := d.GetVendorByID(existing.ID)
		return v, false, err
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}
	// 无 saas_user_id 记录：邮箱可能已建档（运营预建档）→ 绑定并转为申请；否则 INSERT
	existing, err = d.GetVendorByEmail(email)
	if err == nil && existing != nil {
		if existing.SaasUserID != nil && *existing.SaasUserID != saasID {
			return nil, false, fmt.Errorf("该邮箱已绑定其他主站用户")
		}
		if existing.SaasUserID == nil {
			_ = d.BindVendorSaasUser(existing.ID, saasID)
		}
		if existing.IsActive || existing.ReviewNote == "" {
			return existing, true, nil
		}
		_, err = d.SQL.Exec(`UPDATE ai_provider_vendor SET company_name=?, contact_name=?, id_card_file_key=?, business_license_file_key=?, contact_phone=?, review_note=NULL, reviewed_at=NULL, reviewed_by=NULL, updated_at=? WHERE id=?`,
			in.CompanyName, in.ContactName, in.IDCardFileKey, in.BusinessLicenseFileKey, in.ContactPhone, now, existing.ID)
		if err != nil {
			return nil, false, err
		}
		v, err := d.GetVendorByID(existing.ID)
		return v, false, err
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}
	// 新建申请单（is_active=0 待审核）
	id := NextID()
	_, err = d.SQL.Exec(`INSERT INTO ai_provider_vendor (id, saas_user_id, email, password_hash, company_name, contact_name, id_card_file_key, business_license_file_key, contact_phone, is_active, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,0,?,?)`,
		id, saasID, email, RandomPasswordHash(), in.CompanyName, in.ContactName, in.IDCardFileKey, in.BusinessLicenseFileKey, in.ContactPhone, now, now)
	if err != nil {
		return nil, false, err
	}
	v, err := d.GetVendorByID(id)
	return v, false, err
}

// ReviewNowFunc 是 ReviewVendor 审核时间戳的时钟源，默认真实 UTC 时钟。
// 测试可覆盖为固定值以确定性复现「同秒内同值重复审核」幂等场景（OPT-20260807-014），
// 覆盖后须恢复原值，避免影响其他测试。
var ReviewNowFunc = func() string { return time.Now().UTC().Format("2006-01-02 15:04:05.000000") }

// ReviewVendor 运营审核（approve/reject），记录审核人与时间。重复审核允许覆盖（运营二次修正）。
func (d *DB) ReviewVendor(vendorID int64, action, note string, staffID int64) (*VendorRow, error) {
	active := 0
	if action == "approve" {
		active = 1
	}
	now := ReviewNowFunc()
	if _, err := d.SQL.Exec(`UPDATE ai_provider_vendor SET is_active=?, review_note=?, reviewed_at=?, reviewed_by=?, updated_at=? WHERE id=?`,
		active, note, now, staffID, now, vendorID); err != nil {
		return nil, err
	}
	// OPT-20260807-014: 同值重复审核（同秒内时间戳被 DATETIME 秒精度截断后不变）
	// 时 RowsAffected=0，原实现据此误判 ErrNoRows → handler 404「厂商不存在」。
	// 改为 UPDATE 后按 id 复查：存在即返回（幂等覆盖），不存在 GetVendorByID
	// 自然返回 sql.ErrNoRows，保持 404 契约。
	return d.GetVendorByID(vendorID)
}

func (d *DB) BindVendorSaasUser(vendorID, saasID int64) error {
	_, err := d.SQL.Exec(`UPDATE ai_provider_vendor SET saas_user_id=?, updated_at=? WHERE id=?`, saasID, time.Now().UTC().Format("2006-01-02 15:04:05.000000"), vendorID)
	return err
}

func (d *DB) GetStaffByID(id int64) (*StaffRow, error) {
	row := d.SQL.QueryRow(`SELECT id, saas_superadmin_id, COALESCE(username,''), COALESCE(display_name,''), is_active FROM ai_provider_platformstaff WHERE id=?`, id)
	return scanStaff(row)
}

func (d *DB) GetStaffBySaasSuperadminID(saasID int64) (*StaffRow, error) {
	row := d.SQL.QueryRow(`SELECT id, saas_superadmin_id, COALESCE(username,''), COALESCE(display_name,''), is_active FROM ai_provider_platformstaff WHERE saas_superadmin_id=?`, saasID)
	return scanStaff(row)
}

func (d *DB) GetStaffByUsername(username string) (*StaffRow, error) {
	row := d.SQL.QueryRow(`SELECT id, saas_superadmin_id, COALESCE(username,''), COALESCE(display_name,''), is_active FROM ai_provider_platformstaff WHERE username=?`, username)
	return scanStaff(row)
}

func scanStaff(row *sql.Row) (*StaffRow, error) {
	var s StaffRow
	var saas sql.NullInt64
	var active int
	if err := row.Scan(&s.ID, &saas, &s.Username, &s.DisplayName, &active); err != nil {
		return nil, err
	}
	if saas.Valid {
		s.SaasSuperadminID = &saas.Int64
	}
	s.IsActive = active != 0
	return &s, nil
}

func (d *DB) UsernameTaken(username string, excludeSaasID int64) (bool, error) {
	var n int
	err := d.SQL.QueryRow(`SELECT COUNT(1) FROM ai_provider_platformstaff WHERE username=? AND (saas_superadmin_id IS NULL OR saas_superadmin_id!=?)`, username, excludeSaasID).Scan(&n)
	return n > 0, err
}

func (d *DB) UpsertStaffFromBridge(saasID int64, username, display string) (*StaffRow, error) {
	existing, err := d.GetStaffBySaasSuperadminID(saasID)
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	if err == nil && existing != nil {
		_, err = d.SQL.Exec(`UPDATE ai_provider_platformstaff SET username=?, display_name=?, updated_at=? WHERE id=?`, username, display, now, existing.ID)
		if err != nil {
			return nil, err
		}
		return d.GetStaffByID(existing.ID)
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	id := NextID()
	_, err = d.SQL.Exec(`INSERT INTO ai_provider_platformstaff (id, username, password_hash, display_name, is_active, created_at, updated_at, saas_superadmin_id) VALUES (?,?,?,?,1,?,?,?)`,
		id, username, RandomPasswordHash(), display, now, now, saasID)
	if err != nil {
		return nil, err
	}
	return d.GetStaffByID(id)
}

func PickStaffUsername(d *DB, saasID int64, preferred string) (string, error) {
	base := preferred
	if base == "" {
		base = fmt.Sprintf("%d", saasID)
	}
	if len(base) > 64 {
		base = base[:64]
	}
	for i := 0; i < 30; i++ {
		cand := base
		if i > 0 {
			suffix := fmt.Sprintf("_%d", i)
			cut := 64 - len(suffix)
			if cut < 1 {
				cut = 1
			}
			cand = base[:min(len(base), cut)] + suffix
		}
		taken, err := d.UsernameTaken(cand, saasID)
		if err != nil {
			return "", err
		}
		if !taken {
			return cand, nil
		}
	}
	return fmt.Sprintf("saas_%d", saasID)[:64], nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ExchangeBridge(cfg *Config, db *DB, payload map[string]any) (access string, role string, err error) {
	typ := ClaimString(payload, "typ")
	switch typ {
	case "staff_bridge":
		saasID, saasOK := claimInt64(payload["sub"])
		usernameHint := strings.TrimSpace(ClaimString(payload, "username"))
		display := strings.TrimSpace(ClaimString(payload, "name"))

		// When sub is non-numeric (e.g. "bootstrap-admin" from Django legacy user IDs),
		// fall back to username-based lookup instead of saas_superadmin_id matching.
		if !saasOK {
			subStr := strings.TrimSpace(fmt.Sprint(payload["sub"]))
			if usernameHint == "" {
				usernameHint = subStr
			}
			if display == "" {
				display = usernameHint
			}
			if len(display) > 100 {
				display = display[:100]
			}
			// Try to find existing staff by username
			staff, err := db.GetStaffByUsername(usernameHint)
			if err != nil && err != sql.ErrNoRows {
				return "", "", fmt.Errorf("bridge staff lookup failed: %w", err)
			}
			if staff == nil {
				// Create new staff WITHOUT saas_superadmin_id binding (saasID is non-numeric).
				// We bypass UpsertStaffFromBridge to avoid collision at saasID=0 with
				// other non-numeric users.
				uname, err := PickStaffUsername(db, 0, usernameHint)
				if err != nil {
					return "", "", err
				}
				id := NextID()
				now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
				_, err = db.SQL.Exec(`INSERT INTO ai_provider_platformstaff (id, username, password_hash, display_name, is_active, created_at, updated_at) VALUES (?,?,?,?,1,?,?)`,
					id, uname, RandomPasswordHash(), display, now, now)
				if err != nil {
					return "", "", fmt.Errorf("bridge staff creation failed: %w", err)
				}
				staff, err = db.GetStaffByID(id)
				if err != nil {
					return "", "", err
				}
			}
			tok, err := IssueToken(cfg.SecretKey, fmt.Sprintf("%d", staff.ID), "staff", cfg.JWTTTLSeconds)
			return tok, "staff", err
		}

		// Normal path: sub is a numeric SaaS user ID
		if usernameHint == "" {
			usernameHint = fmt.Sprintf("%d", saasID)
		}
		if display == "" {
			display = usernameHint
		}
		if len(display) > 100 {
			display = display[:100]
		}
		uname, err := PickStaffUsername(db, saasID, usernameHint)
		if err != nil {
			return "", "", err
		}
		staff, err := db.UpsertStaffFromBridge(saasID, uname, display)
		if err != nil {
			return "", "", err
		}
		tok, err := IssueToken(cfg.SecretKey, fmt.Sprintf("%d", staff.ID), "staff", cfg.JWTTTLSeconds)
		return tok, "staff", err
	case "vendor_bridge":
		email := strings.ToLower(strings.TrimSpace(ClaimString(payload, "email")))
		if email == "" {
			return "", "", fmt.Errorf("bridge 缺少 email")
		}
		// OPT-20260806-065 纵深防御：厂商门户需绑定邮箱账号后方可申请/使用。
		// taskAuth 已拦截无邮箱登录方式的用户（合成邮箱 sso-<id>@sso.invalid，
		// RFC 2606 保留域），此处兜底拒绝旧版 taskAuth 签发或手工构造的合成邮箱，
		// 防止绕过入口闸门自动建号。
		if strings.HasSuffix(email, "@sso.invalid") {
			// OPT-20260806-066: 提示无邮箱用户前往个人资料页绑定邮箱
			return "", "", fmt.Errorf("厂商门户需先绑定邮箱账号，请前往个人资料页绑定邮箱")
		}
		uid, uidOK := claimInt64(payload["sub"])
		if !uidOK {
			// Non-numeric sub: fall back to email-based vendor lookup (saasID=0)
			uid = 0
		}
		vendor, err := db.UpsertVendorFromBridge(uid, email)
		if err != nil {
			return "", "", err
		}
		if !vendor.IsActive {
			// OPT-20260806-065 审核流：is_active=0 区分待审核/已驳回
			if strings.TrimSpace(vendor.ReviewNote) != "" {
				return "", "", fmt.Errorf("厂商申请已被驳回：%s", vendor.ReviewNote)
			}
			return "", "", fmt.Errorf("厂商申请审核中，请等待运营审核通过")
		}
		tok, err := IssueToken(cfg.SecretKey, fmt.Sprintf("%d", vendor.ID), "vendor", cfg.JWTTTLSeconds)
		return tok, "vendor", err
	default:
		return "", "", fmt.Errorf("不支持的 bridge 类型")
	}
}
