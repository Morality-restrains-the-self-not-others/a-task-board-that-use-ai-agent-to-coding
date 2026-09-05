package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type TenantInstalledImage struct {
	ID              string
	TenantID        string
	ExternalImageID string
	Name            string
	Description     string
	Version         string
	ImageURL        string
	// IconURL 是目录镜像组图标 URL，安装时从 catalog icon_url / image_group.icon_url 快照。
	IconURL             string
	TargetArchitectures []string
	Size                *int64
	IsDevMode           bool
	VendorID            string
	VendorName          string
	InstalledByID       string
	InstalledAt         time.Time
	// UpdatedAt 是目录镜像的 updated_at（镜像更新时间），安装时快照入库；
	// 存量行/缺失时回填 installed_at（见 038 迁移），零值表示快照缺失。
	UpdatedAt                 time.Time
	UserdataTemplateID        string
	AutoRunStepsMd            string
	AutoRunStepsExtractStatus string
	AutoRunStepsDigest        string
	ImageSkillsJSON           string
	ImageSkillsExtractStatus  string
	ImageSkillsDigest         string
	// SaasInboundSkillVersion 是镜像声明的 SaaS inbound 契约版本（ADR-0024），
	// 安装时从厂商目录/公开镜像字段拷贝；start-vm 注入 SAAS_INBOUND_SKILL_VERSION。
	SaasInboundSkillVersion string
}

func listInstalledImages(tenantID string) ([]TenantInstalledImage, error) {
	rows, err := db.Query(`SELECT `+installedImageSelectCols+`
		FROM cloud_tenant_installed_images WHERE tenant_id=? ORDER BY installed_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstalledImageRows(rows)
}

func getInstalledImage(tenantID, id string) (*TenantInstalledImage, error) {
	row := db.QueryRow(`SELECT `+installedImageSelectCols+`
		FROM cloud_tenant_installed_images WHERE tenant_id=? AND id=?`, tenantID, id)
	img, err := scanInstalledImage(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return img, err
}

func uniqueInstalledImage(rows []TenantInstalledImage) *TenantInstalledImage {
	if len(rows) != 1 {
		return nil
	}
	img := rows[0]
	return &img
}

func getInstalledImageByExternalID(tenantID, externalID string) (*TenantInstalledImage, error) {
	externalID = strings.TrimSpace(externalID)
	if strings.TrimSpace(tenantID) == "" || externalID == "" {
		return nil, nil
	}
	rows, err := db.Query(`SELECT `+installedImageSelectCols+`
		FROM cloud_tenant_installed_images WHERE tenant_id=? AND external_image_id=?`, tenantID, externalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanInstalledImageRows(rows)
	if err != nil {
		return nil, err
	}
	return uniqueInstalledImage(list), nil
}

func getInstalledImageByUniqueName(tenantID, name string) (*TenantInstalledImage, error) {
	name = strings.TrimSpace(name)
	if strings.TrimSpace(tenantID) == "" || name == "" {
		return nil, nil
	}
	rows, err := db.Query(`SELECT `+installedImageSelectCols+`
		FROM cloud_tenant_installed_images WHERE tenant_id=? AND name=?`, tenantID, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := scanInstalledImageRows(rows)
	if err != nil {
		return nil, err
	}
	return uniqueInstalledImage(list), nil
}

func installedExternalIDSet(tenantID string) (map[string]bool, error) {
	rows, err := db.Query(`SELECT external_image_id FROM cloud_tenant_installed_images WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var extID string
		if err := rows.Scan(&extID); err != nil {
			return nil, err
		}
		if extID != "" {
			out[extID] = true
		}
	}
	return out, nil
}

func installedImageExists(tenantID, externalImageID string) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM cloud_tenant_installed_images WHERE tenant_id=? AND external_image_id=?`, tenantID, externalImageID).Scan(&count)
	return count > 0, err
}

func createInstalledImage(img TenantInstalledImage) error {
	archJSON, _ := json.Marshal(img.TargetArchitectures)
	sizeVal := sql.NullInt64{}
	if img.Size != nil {
		sizeVal = sql.NullInt64{Int64: *img.Size, Valid: true}
	}
	_, err := db.Exec(`INSERT INTO cloud_tenant_installed_images
		(id,tenant_id,external_image_id,name,description,version,image_url,icon_url,target_architectures,size,is_dev_mode,vendor_id,vendor_name,installed_by_id,installed_at,updated_at,userdata_template_id,auto_run_steps_md,auto_run_steps_extract_status,auto_run_steps_digest,image_skills_json,image_skills_extract_status,image_skills_digest,saas_inbound_skill_version)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,COALESCE(?, CURRENT_TIMESTAMP),?,?,?,?,?,?,?,?,?)`,
		img.ID, img.TenantID, img.ExternalImageID, img.Name, img.Description, img.Version, img.ImageURL, img.IconURL,
		string(archJSON), sizeVal, boolToInt(img.IsDevMode), img.VendorID, img.VendorName, img.InstalledByID,
		formatTime(img.InstalledAt), formatTime(img.UpdatedAt), img.UserdataTemplateID, img.AutoRunStepsMd, img.AutoRunStepsExtractStatus, img.AutoRunStepsDigest,
		img.ImageSkillsJSON, img.ImageSkillsExtractStatus, img.ImageSkillsDigest,
		nullIfEmpty(img.SaasInboundSkillVersion))
	return err
}

func deleteInstalledImage(tenantID, id string) (bool, error) {
	res, err := db.Exec(`DELETE FROM cloud_tenant_installed_images WHERE tenant_id=? AND id=?`, tenantID, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func patchInstalledImageUserdata(tenantID, id, templateID string) (*TenantInstalledImage, error) {
	res, err := db.Exec(`UPDATE cloud_tenant_installed_images SET userdata_template_id=? WHERE tenant_id=? AND id=?`, templateID, tenantID, id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, nil
	}
	return getInstalledImage(tenantID, id)
}

func importInstalledImages(rows []map[string]interface{}) (int, error) {
	count := 0
	for _, row := range rows {
		id := strField(row, "id")
		tenantID := strField(row, "tenant_id")
		if tenantID == "" {
			tenantID = strField(row, "company_id")
		}
		if id == "" || tenantID == "" {
			continue
		}
		arch := []string{}
		if raw, ok := row["target_architectures"]; ok && raw != nil {
			switch t := raw.(type) {
			case string:
				_ = json.Unmarshal([]byte(t), &arch)
			case []interface{}:
				for _, item := range t {
					arch = append(arch, stringifyID(item))
				}
			}
		}
		archJSON, _ := json.Marshal(arch)
		isDev := 0
		if v, ok := row["is_dev_mode"].(bool); ok && v {
			isDev = 1
		}
		var size sql.NullInt64
		if v, ok := row["size"].(float64); ok {
			size = sql.NullInt64{Int64: int64(v), Valid: true}
		}
		installedAt := normalizeDatetimeString(strField(row, "installed_at"))
		skillsJSON, skillsSt, skillsDig := skillsSnapshotFromCatalog(
			row, imageSkillIDSeed(strField(row, "external_image_id"), id))
		_, err := db.Exec(`REPLACE INTO cloud_tenant_installed_images
			(id,tenant_id,external_image_id,name,description,version,image_url,icon_url,target_architectures,size,is_dev_mode,vendor_id,vendor_name,installed_by_id,installed_at,updated_at,userdata_template_id,auto_run_steps_md,auto_run_steps_extract_status,auto_run_steps_digest,image_skills_json,image_skills_extract_status,image_skills_digest,saas_inbound_skill_version)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,COALESCE(NULLIF(?, ''), CURRENT_TIMESTAMP),?,?,?,?,?,?,?,?,?)`,
			id, tenantID, strField(row, "external_image_id"), strField(row, "name"),
			strField(row, "description"), strField(row, "version"), strField(row, "image_url"), catalogIconURL(row),
			string(archJSON), size, isDev, strField(row, "vendor_id"), strField(row, "vendor_name"),
			strField(row, "installed_by_id"), installedAt, nullableDatetimeString(strField(row, "updated_at")),
			strField(row, "userdata_template_id"),
			strField(row, "auto_run_steps_md"), strField(row, "auto_run_steps_extract_status"), strField(row, "auto_run_steps_digest"),
			skillsJSON, skillsSt, skillsDig, strField(row, "saas_inbound_skill_version"))
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func scanInstalledImageRows(rows *sql.Rows) ([]TenantInstalledImage, error) {
	out := []TenantInstalledImage{}
	for rows.Next() {
		img, err := scanInstalledImage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *img)
	}
	return out, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func formatTime(t time.Time) sql.NullString {
	if t.IsZero() {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: t.UTC().Format("2006-01-02 15:04:05"), Valid: true}
}

// nullableDatetimeString normalizes a datetime string to a NULL-able value:
// empty/unparseable input maps to NULL so DATETIME columns stay NULL instead of ”.
func nullableDatetimeString(s string) sql.NullString {
	u := normalizeDatetimeString(s)
	if u == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: u, Valid: true}
}

// parseImageUpdatedAt parses 目录/厂商镜像的 updated_at 字段；无法解析时返回零值，
// 由调用方按 installed_at 兜底（038 迁移已回填存量行）。
func parseImageUpdatedAt(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02 15:04:05.999999", s); err == nil {
		return t
	}
	return time.Time{}
}

// normalizeDatetimeString parses a datetime string in common formats
// (RFC3339Nano, MySQL DATETIME) and returns a MySQL-safe "YYYY-MM-DD HH:MM:SS"
// string. Empty/unparseable input is returned as-is so COALESCE(NULLIF(?, ”), ...)
// can fall back to CURRENT_TIMESTAMP.
func normalizeDatetimeString(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Try RFC3339Nano first (Go's default JSON serialization: "2026-07-31T14:30:40.917359408Z")
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC().Format("2006-01-02 15:04:05")
	}
	// Try MySQL DATETIME format ("2006-01-02 15:04:05")
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC().Format("2006-01-02 15:04:05")
	}
	// Try with fractional seconds ("2006-01-02 15:04:05.999999")
	if t, err := time.Parse("2006-01-02 15:04:05.999999", s); err == nil {
		return t.UTC().Format("2006-01-02 15:04:05")
	}
	// Unrecognized format — return as-is (let MySQL reject it with a clear error)
	return s
}

func catalogImageByID(externalImageID string) (map[string]interface{}, error) {
	catalog, err := fetchAIPublicCatalog()
	if err != nil {
		return nil, err
	}
	want := stringifyID(externalImageID)
	for _, img := range catalog {
		if stringifyID(img["id"]) == want {
			return img, nil
		}
	}
	return nil, fmt.Errorf("未找到该镜像")
}

func imageDataFromInstallBody(body map[string]interface{}) (map[string]interface{}, string, bool, error) {
	if ext := strField(body, "external_image_id"); ext != "" {
		img, err := catalogImageByID(ext)
		if err != nil {
			return nil, ext, false, err
		}
		return img, ext, false, nil
	}
	vendorID := strField(body, "vendor_id")
	containerID := strField(body, "container_id")
	if vendorID == "" || containerID == "" {
		return nil, "", false, fmt.Errorf("请提供 external_image_id（从目录安装），或同时提供 vendor_id 与 container_id")
	}
	img, err := fetchAIPublicUnsubmittedImage(vendorID, containerID)
	if err != nil {
		return nil, containerID, true, err
	}
	return img, containerID, true, nil
}

func vendorInfoFromImage(img map[string]interface{}) (id, name string) {
	vendor, _ := img["vendor"].(map[string]interface{})
	if vendor == nil {
		return "", ""
	}
	return stringifyID(vendor["id"]), strField(vendor, "company_name")
}

func architecturesFromImage(img map[string]interface{}) []string {
	raw := make([]string, 0)
	if v, ok := img["target_architectures"]; ok && v != nil {
		switch t := v.(type) {
		case []string:
			raw = append(raw, t...)
		case []interface{}:
			for _, item := range t {
				raw = append(raw, stringifyID(item))
			}
		case string:
			var arr []interface{}
			if err := json.Unmarshal([]byte(t), &arr); err == nil {
				for _, item := range arr {
					raw = append(raw, stringifyID(item))
				}
			} else if strings.TrimSpace(t) != "" {
				raw = append(raw, t)
			}
		}
	}
	known := knownCPUArchitectures(raw)
	if len(known) > 0 {
		return known
	}
	return extractCPUArchitecturesFromText(
		strField(img, "version"),
		strField(img, "name"),
		strField(img, "image_url"),
	)
}

func sizeFromImage(img map[string]interface{}) *int64 {
	switch v := img["size"].(type) {
	case float64:
		n := int64(v)
		return &n
	case int64:
		return &v
	default:
		return nil
	}
}
