package infrastructure

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"taskAiProvider/domain"
)

func nowTS() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05.000000")
}

func formatCatalogTS(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format("2006-01-02T15:04:05Z")
}

func IDStr(id int64) string { return fmt.Sprintf("%d", id) }

func ImageGroupIconURL(groupID int64, fileKey string) string {
	key := strings.TrimSpace(fileKey)
	if key == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(key))
	h := hex.EncodeToString(sum[:6])
	return fmt.Sprintf("/api/ai-provider/public-image-groups/%s/icon?h=%s", IDStr(groupID), h)
}

func imageGroupJSON(id, vendorID int64, name, desc, iconKey string) map[string]any {
	return map[string]any{
		"id":            IDStr(id),
		"name":          name,
		"description":   desc,
		"vendor":        IDStr(vendorID),
		"icon_file_key": iconKey,
		"icon_url":      ImageGroupIconURL(id, iconKey),
	}
}

func NullIDStr(n sql.NullInt64) string {
	if !n.Valid {
		return ""
	}
	return IDStr(n.Int64)
}

// GroupActiveVersion 返回镜像组内除 excludeID 外当前「激活」的版本
// （is_active=1，即公开目录生效版本）；无则返回 nil。同一镜像组允许多个
// approved，但仅一个激活版本（OPT-20260824）。
func (d *DB) GroupActiveVersion(groupID, excludeID int64) (*domain.ContainerImage, error) {
	row := d.SQL.QueryRow(`SELECT id, vendor_id, image_group_id, COALESCE(version,''), COALESCE(saas_inbound_skill_version,'1'), COALESCE(image_url,''), COALESCE(target_architectures,''), size, COALESCE(status,''), is_active, COALESCE(review_note,''),
		COALESCE(auto_run_steps_md,''), COALESCE(auto_run_steps_extract_status,''), COALESCE(auto_run_steps_digest,''),
		COALESCE(image_skills_json,''), COALESCE(image_skills_extract_status,''), COALESCE(image_skills_digest,'')
		FROM ai_provider_vendorcontainerimage WHERE image_group_id=? AND status=? AND is_active=1 AND id<>? LIMIT 1`,
		groupID, domain.StatusApproved, excludeID)
	var c domain.ContainerImage
	var size sql.NullInt64
	err := row.Scan(&c.ID, &c.VendorID, &c.ImageGroupID, &c.Version, &c.SaasInboundSkillVersion, &c.ImageURL, &c.TargetArchitecturesJSON, &size, &c.Status, &c.IsActive, &c.ReviewNote,
		&c.AutoRunStepsMD, &c.AutoRunStepsExtractStatus, &c.AutoRunStepsDigest,
		&c.ImageSkillsJSON, &c.ImageSkillsExtractStatus, &c.ImageSkillsDigest)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if size.Valid {
		c.Size = &size.Int64
	}
	return &c, nil
}

func (d *DB) GetContainerImage(id int64) (*domain.ContainerImage, error) {
	row := d.SQL.QueryRow(`SELECT id, vendor_id, image_group_id, COALESCE(version,''), COALESCE(saas_inbound_skill_version,'1'), COALESCE(image_url,''), COALESCE(target_architectures,''), size, COALESCE(status,''), is_active, COALESCE(review_note,''),
		COALESCE(auto_run_steps_md,''), COALESCE(auto_run_steps_extract_status,''), COALESCE(auto_run_steps_digest,''),
		COALESCE(image_skills_json,''), COALESCE(image_skills_extract_status,''), COALESCE(image_skills_digest,'')
		FROM ai_provider_vendorcontainerimage WHERE id=?`, id)
	var c domain.ContainerImage
	var size sql.NullInt64
	err := row.Scan(&c.ID, &c.VendorID, &c.ImageGroupID, &c.Version, &c.SaasInboundSkillVersion, &c.ImageURL, &c.TargetArchitecturesJSON, &size, &c.Status, &c.IsActive, &c.ReviewNote,
		&c.AutoRunStepsMD, &c.AutoRunStepsExtractStatus, &c.AutoRunStepsDigest,
		&c.ImageSkillsJSON, &c.ImageSkillsExtractStatus, &c.ImageSkillsDigest)
	if err != nil {
		return nil, err
	}
	if size.Valid {
		c.Size = &size.Int64
	}
	return &c, nil
}

func (d *DB) UpdateContainerImageStatus(c *domain.ContainerImage, reviewerID *int64) error {
	now := nowTS()
	var reviewedAt any
	if c.Status == domain.StatusApproved || c.Status == domain.StatusRejected {
		reviewedAt = now
	} else {
		reviewedAt = nil
	}
	isActive := 0
	if c.IsActive && c.Status == domain.StatusApproved {
		isActive = 1
	}
	_, err := d.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET status=?, is_active=?, review_note=?, reviewed_at=?, reviewer_id=?, updated_at=? WHERE id=?`,
		c.Status, isActive, c.ReviewNote, reviewedAt, reviewerID, now, c.ID)
	return err
}

// ActivateContainerImage 将镜像设为组内唯一激活版本（事务）：先取消同组其他
// 激活，再激活自身。仅 approved（已上架）版本可激活。切换激活时原激活版本
// 保留 approved 状态，仅失去激活身份（OPT-20260824）。
func (d *DB) ActivateContainerImage(id int64) error {
	tx, err := d.SQL.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var gid int64
	var status string
	err = tx.QueryRow(`SELECT image_group_id, status FROM ai_provider_vendorcontainerimage WHERE id=? FOR UPDATE`, id).Scan(&gid, &status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("镜像不存在")
	}
	if err != nil {
		return err
	}
	if status != domain.StatusApproved {
		return fmt.Errorf("仅已上架（审批通过）的版本可设为激活")
	}
	now := nowTS()
	if _, err := tx.Exec(`UPDATE ai_provider_vendorcontainerimage SET is_active=0, updated_at=? WHERE image_group_id=? AND id<>? AND is_active=1`, now, gid, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE ai_provider_vendorcontainerimage SET is_active=1, updated_at=? WHERE id=?`, now, id); err != nil {
		return err
	}
	return tx.Commit()
}

// ActivateIfNoActiveVersion 组内尚无激活版本时将指定镜像置为激活（审批通过后
// 调用：首个上架版本自动成为激活版本，后续上架版本保持非激活待显式切换）。
func (d *DB) ActivateIfNoActiveVersion(id, groupID int64) error {
	_, err := d.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET is_active=1, updated_at=?
		WHERE id=? AND image_group_id=? AND NOT EXISTS (
			SELECT 1 FROM (SELECT 1 FROM ai_provider_vendorcontainerimage a WHERE a.image_group_id=? AND a.is_active=1) t
		)`, nowTS(), id, groupID, groupID)
	return err
}

func (d *DB) InsertReviewHistory(containerID int64, action, note string, reviewerID int64) error {
	_, err := d.SQL.Exec(`INSERT INTO ai_provider_containerimagereviewhistory (id, action, note, reviewed_at, created_at, container_image_id, reviewer_id)
		VALUES (?,?,?,?,?,?,?)`, NextID(), action, note, nowTS(), nowTS(), containerID, reviewerID)
	return err
}

func (d *DB) ListVendors() ([]map[string]any, error) {
	rows, err := d.SQL.Query(`SELECT id, COALESCE(email,''), COALESCE(company_name,''), COALESCE(contact_name,''),
		COALESCE(id_card_file_key,''), COALESCE(business_license_file_key,''), COALESCE(contact_phone,''),
		is_active, saas_user_id, COALESCE(review_note,''), reviewed_at, reviewed_by FROM ai_provider_vendor ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var email, company, contact, idCard, license, phone, note string
		var active int
		var saas, reviewedBy sql.NullInt64
		var reviewedAt sql.NullTime
		if err := rows.Scan(&id, &email, &company, &contact, &idCard, &license, &phone, &active, &saas, &note, &reviewedAt, &reviewedBy); err != nil {
			return nil, err
		}
		m := map[string]any{
			"id": IDStr(id), "email": email, "company_name": company, "contact_name": contact,
			"has_id_card": idCard != "", "has_business_license": license != "",
			"contact_phone_masked": maskPhone(phone),
			"is_active":            active != 0, "review_note": note,
		}
		if saas.Valid {
			m["saas_user_id"] = IDStr(saas.Int64)
		} else {
			m["saas_user_id"] = nil
		}
		if reviewedAt.Valid {
			m["reviewed_at"] = reviewedAt.Time.Format("2006-01-02 15:04:05")
		} else {
			m["reviewed_at"] = nil
		}
		if reviewedBy.Valid {
			m["reviewed_by"] = IDStr(reviewedBy.Int64)
		} else {
			m["reviewed_by"] = nil
		}
		out = append(out, m)
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

func maskPhone(phone string) string {
	p := strings.TrimSpace(phone)
	if len(p) < 7 {
		if p == "" {
			return ""
		}
		return "****"
	}
	return p[:3] + "****" + p[len(p)-4:]
}

func (d *DB) ListImageGroups(vendorID int64) ([]map[string]any, error) {
	rows, err := d.SQL.Query(`SELECT id, COALESCE(name,''), COALESCE(description,''), COALESCE(icon_file_key,''), vendor_id FROM ai_provider_containerimagegroup WHERE vendor_id=? ORDER BY updated_at DESC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, vid int64
		var name, desc, iconKey string
		if err := rows.Scan(&id, &name, &desc, &iconKey, &vid); err != nil {
			return nil, err
		}
		out = append(out, imageGroupJSON(id, vid, name, desc, iconKey))
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

func (d *DB) CreateImageGroup(vendorID int64, name, desc, iconKey string) (int64, error) {
	id := NextID()
	now := nowTS()
	_, err := d.SQL.Exec(`INSERT INTO ai_provider_containerimagegroup (id, name, description, icon_file_key, created_at, updated_at, vendor_id) VALUES (?,?,?,?,?,?,?)`,
		id, name, desc, iconKey, now, now, vendorID)
	return id, err
}

func (d *DB) GetImageGroup(id int64) (map[string]any, error) {
	var gid, vid int64
	var name, desc, iconKey string
	err := d.SQL.QueryRow(`SELECT id, COALESCE(name,''), COALESCE(description,''), COALESCE(icon_file_key,''), vendor_id FROM ai_provider_containerimagegroup WHERE id=?`, id).Scan(&gid, &name, &desc, &iconKey, &vid)
	if err != nil {
		return nil, err
	}
	return imageGroupJSON(gid, vid, name, desc, iconKey), nil
}

func (d *DB) GetImageGroupIcon(id int64) (vendorID int64, iconKey string, err error) {
	err = d.SQL.QueryRow(`SELECT vendor_id, COALESCE(icon_file_key,'') FROM ai_provider_containerimagegroup WHERE id=?`, id).Scan(&vendorID, &iconKey)
	return vendorID, iconKey, err
}

func (d *DB) UpdateImageGroup(id int64, name, desc, iconKey string) error {
	_, err := d.SQL.Exec(`UPDATE ai_provider_containerimagegroup SET name=?, description=?, icon_file_key=?, updated_at=? WHERE id=?`, name, desc, iconKey, nowTS(), id)
	return err
}

func (d *DB) DeleteImageGroup(id int64) error {
	_, err := d.SQL.Exec(`DELETE FROM ai_provider_containerimagegroup WHERE id=?`, id)
	return err
}

func (d *DB) ListContainerImages(vendorID int64) ([]map[string]any, error) {
	rows, err := d.SQL.Query(`SELECT id, vendor_id, image_group_id, COALESCE(version,''), COALESCE(saas_inbound_skill_version,'1'), COALESCE(image_url,''), COALESCE(target_architectures,''), size, COALESCE(status,''), is_active, COALESCE(review_note,''),
		COALESCE(auto_run_steps_md,''), COALESCE(auto_run_steps_extract_status,''), COALESCE(auto_run_steps_digest,''),
		COALESCE(image_skills_json,''), COALESCE(image_skills_extract_status,''), COALESCE(image_skills_digest,''), updated_at
		FROM ai_provider_vendorcontainerimage WHERE vendor_id=? ORDER BY updated_at DESC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanContainerImageRows(rows)
	if err != nil {
		return nil, err
	}
	return d.attachGroupActiveVersions(d.attachMarketplaceAvailability(items)), nil
}

func scanContainerImageRows(rows *sql.Rows) ([]map[string]any, error) {
	var out []map[string]any
	for rows.Next() {
		var id, vid, gid int64
		var version, skillVer, imageURL, archJSON, status, note, autoMD, autoSt, autoDig, skillsJSON, skillsSt, skillsDig string
		var size sql.NullInt64
		var isActive int
		var updatedAt sql.NullTime
		if err := rows.Scan(&id, &vid, &gid, &version, &skillVer, &imageURL, &archJSON, &size, &status, &isActive, &note, &autoMD, &autoSt, &autoDig, &skillsJSON, &skillsSt, &skillsDig, &updatedAt); err != nil {
			return nil, err
		}
		var arch any
		_ = json.Unmarshal([]byte(archJSON), &arch)
		if arch == nil {
			arch = []any{}
		}
		statusLabel := status
		if d, ok := domain.StatusDisplay[status]; ok {
			statusLabel = d
		}
		m := map[string]any{
			"id": IDStr(id), "vendor": IDStr(vid), "image_group": IDStr(gid), "version": version,
			"saas_inbound_skill_version": skillVer,
			"image_url":                  imageURL, "target_architectures": arch, "status": status, "status_display": statusLabel,
			"is_active":         isActive != 0,
			"review_note":       note,
			"auto_run_steps_md": autoMD, "auto_run_steps_extract_status": autoSt, "auto_run_steps_digest": autoDig,
			"image_skills": DecodeImageSkillsJSON(skillsJSON), "image_skills_extract_status": skillsSt, "image_skills_digest": skillsDig,
			"updated_at": formatCatalogTS(updatedAt),
		}
		if size.Valid {
			m["size"] = size.Int64
		} else {
			m["size"] = nil
		}
		out = append(out, m)
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, rows.Err()
}

func (d *DB) CreateContainerImage(vendorID, groupID int64, version, imageURL string, arch []any, skillVersion string) (int64, error) {
	id := NextID()
	now := nowTS()
	archJSON, _ := json.Marshal(arch)
	if arch == nil {
		archJSON = []byte("[]")
	}
	sv := domain.NormalizeSkillVersion(skillVersion)
	if sv == "" {
		sv = "1"
	}
	_, err := d.SQL.Exec(`INSERT INTO ai_provider_vendorcontainerimage
		(id, version, saas_inbound_skill_version, image_url, target_architectures, size, status, review_note, reviewed_at, created_at, updated_at, image_group_id, reviewer_id, vendor_id, auto_run_steps_md, auto_run_steps_extract_status, auto_run_steps_digest)
		VALUES (?,?,?,?,?,NULL,'draft','',NULL,?,?,?,NULL,?,'','','')`,
		id, version, sv, imageURL, string(archJSON), now, now, groupID, vendorID)
	return id, err
}

func (d *DB) UpdateContainerImageFields(id int64, fields map[string]any) error {
	sets := []string{}
	args := []any{}
	for k, v := range fields {
		sets = append(sets, k+"=?")
		args = append(args, v)
	}
	sets = append(sets, "updated_at=?")
	args = append(args, nowTS(), id)
	_, err := d.SQL.Exec(`UPDATE ai_provider_vendorcontainerimage SET `+strings.Join(sets, ",")+` WHERE id=?`, args...)
	return err
}

func (d *DB) DeleteContainerImage(id int64) error {
	_, _ = d.SQL.Exec(`DELETE FROM ai_provider_containercloudserverassociation WHERE container_image_id=?`, id)
	_, err := d.SQL.Exec(`DELETE FROM ai_provider_vendorcontainerimage WHERE id=?`, id)
	return err
}
