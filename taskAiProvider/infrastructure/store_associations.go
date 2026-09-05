package infrastructure

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// CloudServerImageIDFromAssocItem accepts both cloud_server_image_id (canonical)
// and legacy cloud_server_image (string id or nested {id}) from the vendor portal.
func CloudServerImageIDFromAssocItem(it map[string]any) int64 {
	if it == nil {
		return 0
	}
	if id, err := parseIDAny(it["cloud_server_image_id"]); err == nil && id != 0 {
		return id
	}
	switch v := it["cloud_server_image"].(type) {
	case map[string]any:
		id, _ := parseIDAny(v["id"])
		return id
	default:
		id, _ := parseIDAny(v)
		return id
	}
}

// AssocPlatformRegion reads platform_type / region from an association payload.
func AssocPlatformRegion(it map[string]any) (platform, region string) {
	if it == nil {
		return "", ""
	}
	platform, _ = it["platform_type"].(string)
	region, _ = it["region"].(string)
	return strings.TrimSpace(platform), strings.TrimSpace(region)
}

func (d *DB) SetAssociations(containerID int64, items []map[string]any) error {
	type row struct {
		csiID    int64
		platform string
		region   string
	}
	parsed := make([]row, 0, len(items))
	for _, it := range items {
		csiID := CloudServerImageIDFromAssocItem(it)
		platform, region := AssocPlatformRegion(it)
		if csiID == 0 || platform == "" || region == "" {
			continue
		}
		// 写入时与 CSI 权威地域/平台对齐，避免关联表 denormalized 字段漂移
		if syncedPlatform, syncedRegion, ok := d.cloudServerImagePlatformRegion(csiID); ok {
			if syncedPlatform != "" {
				platform = syncedPlatform
			}
			if syncedRegion != "" {
				region = syncedRegion
			}
		}
		parsed = append(parsed, row{csiID: csiID, platform: platform, region: region})
	}
	// Validate before DELETE so a bad payload cannot wipe existing associations.
	if len(items) > 0 && len(parsed) == 0 {
		return fmt.Errorf("无效的运行环境关联：需要 cloud_server_image_id（或 cloud_server_image）、platform_type、region")
	}
	tx, err := d.SQL.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM ai_provider_containercloudserverassociation WHERE container_image_id=?`, containerID); err != nil {
		return err
	}
	for _, r := range parsed {
		_, err := tx.Exec(`INSERT INTO ai_provider_containercloudserverassociation (id, platform_type, region, cloud_server_image_id, container_image_id) VALUES (?,?,?,?,?)`,
			NextID(), r.platform, r.region, r.csiID, containerID)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UpsertAssociation sets/replaces one platform+region binding without wiping other regions.
// When csiID is 0, clears that platform+region binding only (vendor portal "— 不选 —").
func (d *DB) UpsertAssociation(containerID int64, platform, region string, csiID int64) error {
	platform = strings.TrimSpace(platform)
	region = strings.TrimSpace(region)
	if containerID == 0 || platform == "" || region == "" {
		return fmt.Errorf("无效的运行环境关联：需要 platform_type、region")
	}
	if csiID != 0 {
		if syncedPlatform, syncedRegion, ok := d.cloudServerImagePlatformRegion(csiID); ok {
			if syncedPlatform != "" {
				platform = syncedPlatform
			}
			if syncedRegion != "" {
				region = syncedRegion
			}
		}
	}
	tx, err := d.SQL.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// 先清同容器下该 CSI 的旧关联（含错误地域键），再按 CSI 权威地域写入
	if csiID != 0 {
		if _, err := tx.Exec(
			`DELETE FROM ai_provider_containercloudserverassociation WHERE container_image_id=? AND cloud_server_image_id=?`,
			containerID, csiID,
		); err != nil {
			return err
		}
	} else if _, err := tx.Exec(
		`DELETE FROM ai_provider_containercloudserverassociation WHERE container_image_id=? AND platform_type=? AND region=?`,
		containerID, platform, region,
	); err != nil {
		return err
	}
	if csiID == 0 {
		return tx.Commit()
	}
	_, err = tx.Exec(
		`INSERT INTO ai_provider_containercloudserverassociation (id, platform_type, region, cloud_server_image_id, container_image_id) VALUES (?,?,?,?,?)`,
		NextID(), platform, region, csiID, containerID,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// cloudServerImagePlatformRegion 读取 CSI 权威 platform/region（关联表 denormalized 字段的 SSOT）。
func (d *DB) cloudServerImagePlatformRegion(csiID int64) (platform, region string, ok bool) {
	if csiID == 0 || d == nil || d.SQL == nil {
		return "", "", false
	}
	var p, r sql.NullString
	err := d.SQL.QueryRow(
		`SELECT COALESCE(platform_type,''), COALESCE(region,'') FROM ai_provider_vendorcloudserverimage WHERE id=?`,
		csiID,
	).Scan(&p, &r)
	if err != nil {
		return "", "", false
	}
	return strings.TrimSpace(p.String), strings.TrimSpace(r.String), true
}

func (d *DB) ListAssociations(containerID int64) ([]map[string]any, error) {
	rows, err := d.SQL.Query(`
SELECT a.id, COALESCE(a.platform_type,''), COALESCE(a.region,''), a.cloud_server_image_id, a.container_image_id,
       COALESCE(csi.image_id,''), COALESCE(csi.image_name,''), COALESCE(csi.os_type,''), COALESCE(csi.os_version,''), COALESCE(csi.architecture,''), csi.is_active, csi.userdata_template_id,
       t.name, t.version
FROM ai_provider_containercloudserverassociation a
JOIN ai_provider_vendorcloudserverimage csi ON csi.id = a.cloud_server_image_id
LEFT JOIN ai_provider_userdatatemplate t ON t.id = csi.userdata_template_id
WHERE a.container_image_id=?
ORDER BY a.platform_type, a.region`, containerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, csi, cid int64
		var platform, region, imageID, imageName, osType, osVer, arch string
		var active int
		var tpl sql.NullInt64
		var tplName, tplVer sql.NullString
		if err := rows.Scan(&id, &platform, &region, &csi, &cid, &imageID, &imageName, &osType, &osVer, &arch, &active, &tpl, &tplName, &tplVer); err != nil {
			return nil, err
		}
		csiObj := map[string]any{
			"id": IDStr(csi), "image_id": imageID, "image_name": imageName,
			"os_type": osType, "os_version": osVer, "architecture": arch,
			"is_active": active != 0, "platform_type": platform, "region": region,
			"userdata_template": userdataTemplateObj(tpl, tplName, tplVer),
		}
		out = append(out, map[string]any{
			"id":                    IDStr(id),
			"platform_type":         platform,
			"region":                region,
			"cloud_server_image_id": IDStr(csi),
			"cloud_server_image":    csiObj,
			"container_image":       IDStr(cid),
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

func parseIDAny(v any) (int64, error) {
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case string:
		var id int64
		_, err := fmt.Sscan(t, &id)
		return id, err
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case json.Number:
		return t.Int64()
	default:
		return 0, fmt.Errorf("bad id")
	}
}
