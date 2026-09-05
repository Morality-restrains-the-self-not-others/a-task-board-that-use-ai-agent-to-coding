package infrastructure

import (
	"database/sql"
	"fmt"
	"strings"
)

var cloudPlatformDisplay = map[string]string{
	"aliyun":       "阿里云",
	"tencentcloud": "腾讯云",
	"huaweicloud":  "华为云",
	"ctyun":        "天翼云",
	"cmcc":         "移动云",
	"cucloud":      "联通云",
	"baiducloud":   "百度智能云",
	"aws":          "AWS",
}

func platformTypeDisplay(platform string) string {
	if d, ok := cloudPlatformDisplay[platform]; ok {
		return d
	}
	return platform
}

func hardwareSummary(label string, cpu, mem sql.NullInt64) string {
	label = strings.TrimSpace(label)
	parts := []string{}
	if label != "" {
		parts = append(parts, label)
	}
	if cpu.Valid && mem.Valid {
		parts = append(parts, fmt.Sprintf("%dvCPU/%dGiB", cpu.Int64, mem.Int64))
	} else if cpu.Valid {
		parts = append(parts, fmt.Sprintf("%dvCPU", cpu.Int64))
	} else if mem.Valid {
		parts = append(parts, fmt.Sprintf("%dGiB", mem.Int64))
	}
	return strings.Join(parts, " · ")
}

func runtimeEnvItem(platform, region, csiImageID, imageName, osType, osVer, arch string, defID, defLabel sql.NullString, cpu, mem, tpl sql.NullInt64, tplName, tplVer sql.NullString) map[string]any {
	hw := map[string]any{
		"default_instance_type_id":    nullStrOrEmpty(defID),
		"default_instance_type_label": nullStrOrEmpty(defLabel),
	}
	if cpu.Valid {
		hw["base_cpu_cores"] = cpu.Int64
	} else {
		hw["base_cpu_cores"] = nil
	}
	if mem.Valid {
		hw["base_memory_gib"] = mem.Int64
	} else {
		hw["base_memory_gib"] = nil
	}
	return map[string]any{
		"platform_type":         platform,
		"platform_type_display": platformTypeDisplay(platform),
		"region":                region,
		"image_name":            imageName,
		"image_id":              csiImageID,
		"architecture":          strings.TrimSpace(arch),
		"os_type":               strings.TrimSpace(osType),
		"os_version":            strings.TrimSpace(osVer),
		"hardware":              hw,
		"hardware_summary":      hardwareSummary(nullStrOrEmpty(defLabel), cpu, mem),
		"userdata_template":     userdataTemplateObj(tpl, tplName, tplVer),
	}
}

// batchPublicRuntimeEnvironments loads runtime environments for many container images in one
// IN query, avoiding N+1 per-image queries in catalog listing loops.
func (d *DB) batchPublicRuntimeEnvironments(imageIDs []int64) map[int64][]map[string]any {
	result := make(map[int64][]map[string]any, len(imageIDs))
	if len(imageIDs) == 0 {
		return result
	}
	placeholders := make([]string, len(imageIDs))
	args := make([]any, len(imageIDs))
	for i, id := range imageIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	// 地域以云服务器镜像 CSI.region 为准；关联表 region 可能与 CSI 漂移（曾导致 start-vm「未找到匹配地域」）
	query := `SELECT a.container_image_id,
		COALESCE(NULLIF(TRIM(csi.platform_type),''), a.platform_type,''),
		COALESCE(NULLIF(TRIM(csi.region),''), a.region,''),
		COALESCE(csi.image_id,''), COALESCE(csi.image_name,''),
		COALESCE(csi.os_type,''), COALESCE(csi.os_version,''), COALESCE(csi.architecture,''),
		csi.default_instance_type_id, csi.default_instance_type_label,
		csi.base_cpu_cores, csi.base_memory_gib,
		csi.userdata_template_id, t.name, t.version
	FROM ai_provider_containercloudserverassociation a
	JOIN ai_provider_vendorcloudserverimage csi ON csi.id = a.cloud_server_image_id
	LEFT JOIN ai_provider_userdatatemplate t ON t.id = csi.userdata_template_id
	WHERE a.container_image_id IN (` + strings.Join(placeholders, ",") + `)
	ORDER BY a.container_image_id, a.platform_type, a.region`
	rows, err := d.SQL.Query(query, args...)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var cid int64
		var platform, region, csiImageID, imageName, osType, osVer, arch string
		var defID, defLabel sql.NullString
		var cpu, mem, tpl sql.NullInt64
		var tplName, tplVer sql.NullString
		if err := rows.Scan(&cid, &platform, &region, &csiImageID, &imageName, &osType, &osVer, &arch,
			&defID, &defLabel, &cpu, &mem, &tpl, &tplName, &tplVer); err != nil {
			return result
		}
		result[cid] = append(result[cid], runtimeEnvItem(platform, region, csiImageID, imageName, osType, osVer, arch, defID, defLabel, cpu, mem, tpl, tplName, tplVer))
	}
	return result
}

// publicRuntimeEnvironments returns nested runtime env rows for public catalog payloads.
func (d *DB) publicRuntimeEnvironments(imageID int64) ([]map[string]any, error) {
	// 地域/平台以 CSI 为准，避免关联表 denormalized region 与镜像实际地域不一致
	rows, err := d.SQL.Query(`
SELECT COALESCE(NULLIF(TRIM(csi.platform_type),''), a.platform_type,''),
       COALESCE(NULLIF(TRIM(csi.region),''), a.region,''),
       COALESCE(csi.image_id,''), COALESCE(csi.image_name,''), COALESCE(csi.os_type,''), COALESCE(csi.os_version,''), COALESCE(csi.architecture,''),
       csi.default_instance_type_id, csi.default_instance_type_label, csi.base_cpu_cores, csi.base_memory_gib,
       csi.userdata_template_id, t.name, t.version
FROM ai_provider_containercloudserverassociation a
JOIN ai_provider_vendorcloudserverimage csi ON csi.id = a.cloud_server_image_id
LEFT JOIN ai_provider_userdatatemplate t ON t.id = csi.userdata_template_id
WHERE a.container_image_id=?
ORDER BY a.platform_type, a.region`, imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var platform, region, csiImageID, imageName, osType, osVer, arch string
		var defID, defLabel sql.NullString
		var cpu, mem, tpl sql.NullInt64
		var tplName, tplVer sql.NullString
		if err := rows.Scan(&platform, &region, &csiImageID, &imageName, &osType, &osVer, &arch,
			&defID, &defLabel, &cpu, &mem, &tpl, &tplName, &tplVer); err != nil {
			return nil, err
		}
		out = append(out, runtimeEnvItem(platform, region, csiImageID, imageName, osType, osVer, arch, defID, defLabel, cpu, mem, tpl, tplName, tplVer))
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, rows.Err()
}

func (d *DB) RuntimeEnvironments(imageID int64) ([]map[string]any, error) {
	return d.publicRuntimeEnvironments(imageID)
}
