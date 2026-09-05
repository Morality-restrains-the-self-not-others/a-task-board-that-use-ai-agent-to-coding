package infrastructure

import (
	"database/sql"
	"encoding/json"
	"strings"

	"taskAiProvider/domain"
)

// dedupeActiveVersionPerGroup 每组只保留一个版本：优先「激活」版本
// （is_active=1，公开目录生效版本）；组内无激活版本时取最新 approved 兜底
// （输入须按 updated_at DESC 排序，即最新版本优先）。
func dedupeActiveVersionPerGroup(scanned []publicCatalogRow) []publicCatalogRow {
	active := map[int64]publicCatalogRow{}
	for _, r := range scanned {
		if r.isActive {
			if _, ok := active[r.gid]; !ok {
				active[r.gid] = r
			}
		}
	}
	seen := map[int64]bool{}
	out := make([]publicCatalogRow, 0, len(scanned))
	for _, r := range scanned {
		if seen[r.gid] {
			continue
		}
		seen[r.gid] = true
		if a, ok := active[r.gid]; ok {
			out = append(out, a)
		} else {
			out = append(out, r)
		}
	}
	if out == nil {
		out = []publicCatalogRow{}
	}
	return out
}

func statusDisplayLabel(status string) string {
	if d, ok := domain.StatusDisplay[status]; ok {
		return d
	}
	return status
}

func decodeArchJSON(archJSON string) any {
	var arch any
	_ = json.Unmarshal([]byte(archJSON), &arch)
	if arch == nil {
		return []any{}
	}
	return arch
}

// buildPublicContainerImageItem builds the main-site ImageMarket card payload.
func buildPublicContainerImageItem(
	id, gid, vid int64,
	version, imageURL, archJSON, status, autoMD, autoSt, autoDig, gname, gdesc, company, iconKey string,
	size sql.NullInt64,
	runtimes []map[string]any,
) map[string]any {
	iconURL := ImageGroupIconURL(gid, iconKey)
	item := map[string]any{
		"id":                            IDStr(id),
		"name":                          gname,
		"description":                   gdesc,
		"icon_url":                      iconURL,
		"version":                       version,
		"image_url":                     imageURL,
		"target_architectures":          decodeArchJSON(archJSON),
		"status":                        status,
		"status_display":                statusDisplayLabel(status),
		"auto_run_steps_md":             autoMD,
		"auto_run_steps_extract_status": autoSt,
		"auto_run_steps_digest":         autoDig,
		"image_group":                   map[string]any{"id": IDStr(gid), "name": gname, "description": gdesc, "icon_url": iconURL},
		"vendor":                        map[string]any{"id": IDStr(vid), "company_name": company},
		"runtime_environments":          runtimes,
	}
	if size.Valid {
		item["size"] = size.Int64
	} else {
		item["size"] = nil
	}
	return item
}

type publicCatalogRow struct {
	id, gid, vid                                            int64
	version, skillVersion, imageURL, archJSON, status       string
	isActive                                                bool
	autoMD, autoSt, autoDig, gname, gdesc, iconKey, company string
	skillsJSON, skillsSt, skillsDig                         string
	size                                                    sql.NullInt64
	updatedAt                                               sql.NullTime
}

func scanPublicCatalogRows(rows *sql.Rows) ([]publicCatalogRow, error) {
	var scanned []publicCatalogRow
	for rows.Next() {
		var r publicCatalogRow
		if err := rows.Scan(&r.id, &r.version, &r.skillVersion, &r.imageURL, &r.archJSON, &r.size, &r.status, &r.isActive,
			&r.autoMD, &r.autoSt, &r.autoDig, &r.skillsJSON, &r.skillsSt, &r.skillsDig,
			&r.gid, &r.gname, &r.gdesc, &r.iconKey, &r.vid, &r.company, &r.updatedAt); err != nil {
			return nil, err
		}
		scanned = append(scanned, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return scanned, nil
}

func (d *DB) itemsFromPublicCatalogRows(scanned []publicCatalogRow, markDevelopment bool) ([]map[string]any, []int64) {
	out := make([]map[string]any, 0, len(scanned))
	ids := make([]int64, 0, len(scanned))
	for _, r := range scanned {
		ids = append(ids, r.id)
	}
	// Batch-load all runtime environments in one query (avoid N+1 per image).
	runtimesByImage := d.batchPublicRuntimeEnvironments(ids)
	for _, r := range scanned {
		runtimes := runtimesByImage[r.id]
		if runtimes == nil {
			runtimes = []map[string]any{}
		}
		item := buildPublicContainerImageItem(r.id, r.gid, r.vid, r.version, r.imageURL, r.archJSON, r.status,
			r.autoMD, r.autoSt, r.autoDig, r.gname, r.gdesc, r.company, r.iconKey, r.size, runtimes)
		item["saas_inbound_skill_version"] = r.skillVersion
		item["image_skills"] = DecodeImageSkillsJSON(r.skillsJSON)
		item["image_skills_extract_status"] = r.skillsSt
		item["image_skills_digest"] = r.skillsDig
		item["updated_at"] = formatCatalogTS(r.updatedAt)
		item["is_active"] = r.isActive
		if markDevelopment {
			item["is_development"] = true
		}
		out = append(out, item)
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, ids
}

func (d *DB) ListApprovedCatalog() ([]map[string]any, error) {
	rows, err := d.SQL.Query(`
SELECT ci.id, COALESCE(ci.version,''), COALESCE(ci.saas_inbound_skill_version,'1'), COALESCE(ci.image_url,''), COALESCE(ci.target_architectures,''), ci.size, COALESCE(ci.status,''), ci.is_active,
       COALESCE(ci.auto_run_steps_md,''), COALESCE(ci.auto_run_steps_extract_status,''), COALESCE(ci.auto_run_steps_digest,''),
       COALESCE(ci.image_skills_json,''), COALESCE(ci.image_skills_extract_status,''), COALESCE(ci.image_skills_digest,''),
       g.id, COALESCE(g.name,''), COALESCE(g.description,''), COALESCE(g.icon_file_key,''), v.id, COALESCE(v.company_name,''), ci.updated_at
FROM ai_provider_vendorcontainerimage ci
JOIN ai_provider_containerimagegroup g ON g.id = ci.image_group_id
JOIN ai_provider_vendor v ON v.id = ci.vendor_id
WHERE ci.status = 'approved'
ORDER BY ci.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	scanned, err := scanPublicCatalogRows(rows)
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	// 同一镜像组仅允许一个激活版本：查询按 updated_at DESC 排序，每组只保留
	// 最新一条（审批门禁已阻止新激活，此为存量/异常数据的读侧兜底）。
	scanned = dedupeActiveVersionPerGroup(scanned)
	out, ids := d.itemsFromPublicCatalogRows(scanned, false)
	d.applyBatchMarketplaceAvailability(out, ids)
	return out, nil
}

func (d *DB) marketplaceAvailability(containerImageID int64) (bool, string) {
	a := d.batchMarketplaceAvailability([]int64{containerImageID})[containerImageID]
	return a.available, a.reason
}

type mpAvail struct {
	available bool
	reason    string
}

func (d *DB) batchMarketplaceAvailability(ids []int64) map[int64]mpAvail {
	result := make(map[int64]mpAvail, len(ids))
	for _, id := range ids {
		result[id] = mpAvail{false, "未设置任何区域运行环境，无法在镜像市场展示"}
	}
	if len(ids) == 0 {
		return result
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := `SELECT a.container_image_id,
		COUNT(a.id) > 0 AS has_assoc,
		COUNT(csi.userdata_template_id) > 0 AS has_tpl
	FROM ai_provider_containercloudserverassociation a
	JOIN ai_provider_vendorcloudserverimage csi ON csi.id = a.cloud_server_image_id
	WHERE a.container_image_id IN (` + strings.Join(placeholders, ",") + `)
	GROUP BY a.container_image_id`
	rows, err := d.SQL.Query(query, args...)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var cid int64
		var hasAssoc, hasTpl bool
		if err := rows.Scan(&cid, &hasAssoc, &hasTpl); err != nil {
			continue
		}
		if hasTpl {
			result[cid] = mpAvail{true, ""}
		} else if hasAssoc {
			result[cid] = mpAvail{false, "区域运行环境未选择 UserData 模板，无法在镜像市场展示"}
		}
	}
	return result
}

func (d *DB) applyBatchMarketplaceAvailability(items []map[string]any, ids []int64) {
	availMap := d.batchMarketplaceAvailability(ids)
	for _, item := range items {
		id, err := parseIDAny(item["id"])
		if err != nil || id == 0 {
			item["is_ai_provider_available"] = false
			item["unavailable_reason"] = "无法判定市场可用性"
			continue
		}
		if a, ok := availMap[id]; ok {
			item["is_ai_provider_available"] = a.available
			item["unavailable_reason"] = a.reason
		} else {
			item["is_ai_provider_available"] = false
			item["unavailable_reason"] = "未设置任何区域运行环境，无法在镜像市场展示"
		}
	}
}

func (d *DB) attachMarketplaceAvailability(items []map[string]any) []map[string]any {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		id, err := parseIDAny(item["id"])
		if err != nil || id == 0 {
			item["is_ai_provider_available"] = false
			item["unavailable_reason"] = "无法判定市场可用性"
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return items
	}
	availMap := d.batchMarketplaceAvailability(ids)
	for _, item := range items {
		if item["is_ai_provider_available"] != nil {
			continue
		}
		id, _ := parseIDAny(item["id"])
		if a, ok := availMap[id]; ok {
			item["is_ai_provider_available"] = a.available
			item["unavailable_reason"] = a.reason
		}
	}
	return items
}

// VendorDevelopmentCatalog returns unpublished images for a SaaS-bound vendor
// (draft / pending_review / rejected), with fields expected by 主站 ImageMarket.
func (d *DB) VendorDevelopmentCatalog(saasUserID int64) ([]map[string]any, error) {
	v, err := d.GetVendorBySaasUserID(saasUserID)
	if err != nil {
		return []map[string]any{}, nil
	}
	rows, err := d.SQL.Query(`
SELECT ci.id, COALESCE(ci.version,''), COALESCE(ci.saas_inbound_skill_version,'1'), COALESCE(ci.image_url,''), COALESCE(ci.target_architectures,''), ci.size, COALESCE(ci.status,''), ci.is_active,
       COALESCE(ci.auto_run_steps_md,''), COALESCE(ci.auto_run_steps_extract_status,''), COALESCE(ci.auto_run_steps_digest,''),
       COALESCE(ci.image_skills_json,''), COALESCE(ci.image_skills_extract_status,''), COALESCE(ci.image_skills_digest,''),
       g.id, COALESCE(g.name,''), COALESCE(g.description,''), COALESCE(g.icon_file_key,''), v.id, COALESCE(v.company_name,''), ci.updated_at
FROM ai_provider_vendorcontainerimage ci
JOIN ai_provider_containerimagegroup g ON g.id = ci.image_group_id
JOIN ai_provider_vendor v ON v.id = ci.vendor_id
WHERE ci.vendor_id=? AND ci.status IN ('draft','pending_review','rejected')
ORDER BY ci.updated_at DESC
LIMIT 200`, v.ID)
	if err != nil {
		return nil, err
	}
	scanned, err := scanPublicCatalogRows(rows)
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	out, _ := d.itemsFromPublicCatalogRows(scanned, true)
	return out, nil
}

// UnsubmittedImage returns the latest draft for a saas-bound vendor (legacy).
func (d *DB) UnsubmittedImage(saasUserID int64) (map[string]any, error) {
	v, err := d.GetVendorBySaasUserID(saasUserID)
	if err != nil {
		return map[string]any{}, nil
	}
	row := d.SQL.QueryRow(`
SELECT ci.id, COALESCE(ci.version,''), COALESCE(ci.saas_inbound_skill_version,'1'), COALESCE(ci.image_url,''), COALESCE(ci.target_architectures,''), ci.size, COALESCE(ci.status,''),
       COALESCE(ci.auto_run_steps_md,''), COALESCE(ci.auto_run_steps_extract_status,''), COALESCE(ci.auto_run_steps_digest,''),
       COALESCE(ci.image_skills_json,''), COALESCE(ci.image_skills_extract_status,''), COALESCE(ci.image_skills_digest,''),
       g.id, COALESCE(g.name,''), COALESCE(g.description,''), COALESCE(g.icon_file_key,''), v.id, COALESCE(v.company_name,''), ci.updated_at
FROM ai_provider_vendorcontainerimage ci
JOIN ai_provider_containerimagegroup g ON g.id = ci.image_group_id
JOIN ai_provider_vendor v ON v.id = ci.vendor_id
WHERE ci.vendor_id=? AND ci.status='draft'
ORDER BY ci.updated_at DESC LIMIT 1`, v.ID)
	item, err := d.scanOnePublicContainerImage(row)
	if err != nil {
		return map[string]any{}, nil
	}
	return item, nil
}

// UnsubmittedImageByIDs returns an unpublished image by vendor + container id
// (主站开发安装路径：vendor_id + container_id).
func (d *DB) UnsubmittedImageByIDs(vendorID, containerID int64) (map[string]any, error) {
	row := d.SQL.QueryRow(`
SELECT ci.id, COALESCE(ci.version,''), COALESCE(ci.saas_inbound_skill_version,'1'), COALESCE(ci.image_url,''), COALESCE(ci.target_architectures,''), ci.size, COALESCE(ci.status,''),
       COALESCE(ci.auto_run_steps_md,''), COALESCE(ci.auto_run_steps_extract_status,''), COALESCE(ci.auto_run_steps_digest,''),
       COALESCE(ci.image_skills_json,''), COALESCE(ci.image_skills_extract_status,''), COALESCE(ci.image_skills_digest,''),
       g.id, COALESCE(g.name,''), COALESCE(g.description,''), COALESCE(g.icon_file_key,''), v.id, COALESCE(v.company_name,''), ci.updated_at
FROM ai_provider_vendorcontainerimage ci
JOIN ai_provider_containerimagegroup g ON g.id = ci.image_group_id
JOIN ai_provider_vendor v ON v.id = ci.vendor_id
WHERE ci.id=? AND ci.vendor_id=? AND ci.status IN ('draft','pending_review','rejected')
LIMIT 1`, containerID, vendorID)
	return d.scanOnePublicContainerImage(row)
}

func (d *DB) scanOnePublicContainerImage(row *sql.Row) (map[string]any, error) {
	var id, gid, vid int64
	var version, skillVer, imageURL, archJSON, status, autoMD, autoSt, autoDig, skillsJSON, skillsSt, skillsDig, gname, gdesc, iconKey, company string
	var size sql.NullInt64
	var updatedAt sql.NullTime
	if err := row.Scan(&id, &version, &skillVer, &imageURL, &archJSON, &size, &status, &autoMD, &autoSt, &autoDig, &skillsJSON, &skillsSt, &skillsDig, &gid, &gname, &gdesc, &iconKey, &vid, &company, &updatedAt); err != nil {
		return nil, err
	}
	runtimes, _ := d.publicRuntimeEnvironments(id)
	item := buildPublicContainerImageItem(id, gid, vid, version, imageURL, archJSON, status, autoMD, autoSt, autoDig, gname, gdesc, company, iconKey, size, runtimes)
	item["saas_inbound_skill_version"] = skillVer
	item["image_skills"] = DecodeImageSkillsJSON(skillsJSON)
	item["image_skills_extract_status"] = skillsSt
	item["image_skills_digest"] = skillsDig
	item["updated_at"] = formatCatalogTS(updatedAt)
	return item, nil
}
