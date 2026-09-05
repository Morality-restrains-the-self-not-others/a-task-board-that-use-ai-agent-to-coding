package infrastructure

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"taskAiProvider/domain"
)

const adminContainerImageSelectSQL = `SELECT ci.id, ci.vendor_id, ci.image_group_id, COALESCE(ci.version,''), COALESCE(ci.saas_inbound_skill_version,'1'), COALESCE(ci.image_url,''), COALESCE(ci.target_architectures,''), ci.size, COALESCE(ci.status,''), ci.is_active, COALESCE(ci.review_note,''),
		COALESCE(ci.auto_run_steps_md,''), COALESCE(ci.auto_run_steps_extract_status,''), COALESCE(ci.auto_run_steps_digest,''),
		COALESCE(ci.image_skills_json,''), COALESCE(ci.image_skills_extract_status,''), COALESCE(ci.image_skills_digest,''),
		COALESCE(g.name,''), COALESCE(g.description,''), COALESCE(g.icon_file_key,''), COALESCE(v.company_name,''), ci.updated_at
		FROM ai_provider_vendorcontainerimage ci
		LEFT JOIN ai_provider_containerimagegroup g ON g.id = ci.image_group_id
		LEFT JOIN ai_provider_vendor v ON v.id = ci.vendor_id`

func (d *DB) queryAdminContainerImages(where string, args []any) ([]map[string]any, error) {
	q := adminContainerImageSelectSQL + where + ` ORDER BY ci.updated_at DESC`
	rows, err := d.SQL.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanContainerImageAdminRows(rows)
	if err != nil {
		return nil, err
	}
	items = d.enrichAdminContainerImages(items)
	return d.attachGroupActiveVersions(d.attachMarketplaceAvailability(items)), nil
}

func (d *DB) ListContainerImagesAdmin(statusFilter string) ([]map[string]any, error) {
	if statusFilter != "" {
		return d.queryAdminContainerImages(` WHERE ci.status=?`, []any{statusFilter})
	}
	return d.queryAdminContainerImages("", nil)
}

func (d *DB) GetContainerImageAdmin(id int64) (map[string]any, error) {
	items, err := d.queryAdminContainerImages(` WHERE ci.id=?`, []any{id})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, sql.ErrNoRows
	}
	return items[0], nil
}

// attachGroupActiveVersions 为列表项标注同镜像组内当前「激活」的版本
// （group_active_version：{id, version}，is_active=1）。自身即激活版本时不标注。
// 前端据此展示「组内已激活」提示与激活切换入口。
func (d *DB) attachGroupActiveVersions(items []map[string]any) []map[string]any {
	groups := map[int64]bool{}
	for _, item := range items {
		gid, err := parseIDAny(item["image_group"])
		if err != nil || gid == 0 {
			continue
		}
		groups[gid] = true
	}
	if len(groups) == 0 {
		return items
	}
	placeholders := make([]string, 0, len(groups))
	args := make([]any, 0, len(groups))
	for gid := range groups {
		placeholders = append(placeholders, "?")
		args = append(args, gid)
	}
	activeByGroup := map[int64]map[string]any{}
	rows, err := d.SQL.Query(`SELECT id, image_group_id, COALESCE(version,'') FROM ai_provider_vendorcontainerimage
		WHERE status=? AND is_active=1 AND image_group_id IN (`+strings.Join(placeholders, ",")+`)`,
		append([]any{domain.StatusApproved}, args...)...)
	if err != nil {
		// 查询失败 fail-open，不阻断列表
		return items
	}
	defer rows.Close()
	for rows.Next() {
		var id, gid int64
		var version string
		if err := rows.Scan(&id, &gid, &version); err != nil {
			continue
		}
		activeByGroup[gid] = map[string]any{"id": IDStr(id), "version": version}
	}
	for _, item := range items {
		gid, err := parseIDAny(item["image_group"])
		if err != nil || gid == 0 {
			continue
		}
		active, ok := activeByGroup[gid]
		if !ok {
			continue
		}
		if myID, err := parseIDAny(item["id"]); err == nil && myID != 0 && active["id"] == IDStr(myID) {
			continue // 自身即激活版本
		}
		item["group_active_version"] = active
	}
	return items
}

// scanContainerImageAdminRows parses admin list rows (JOINed with image group &
// vendor), adding name / description / vendor_company expected by AdminPortal.
func scanContainerImageAdminRows(rows *sql.Rows) ([]map[string]any, error) {
	var out []map[string]any
	for rows.Next() {
		var id, vid, gid int64
		var version, skillVer, imageURL, archJSON, status, note, autoMD, autoSt, autoDig, skillsJSON, skillsSt, skillsDig string
		var gname, gdesc, gicon, company string
		var size sql.NullInt64
		var isActive int
		var updatedAt sql.NullTime
		if err := rows.Scan(&id, &vid, &gid, &version, &skillVer, &imageURL, &archJSON, &size, &status, &isActive, &note, &autoMD, &autoSt, &autoDig, &skillsJSON, &skillsSt, &skillsDig, &gname, &gdesc, &gicon, &company, &updatedAt); err != nil {
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
			"name":                       gname,
			"description":                gdesc,
			"icon_url":                   ImageGroupIconURL(gid, gicon),
			"vendor_company":             company,
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

// enrichAdminContainerImages attaches runtime_environments, platform_types and
// review_histories to admin list items (batch queries, no N+1).
func (d *DB) enrichAdminContainerImages(items []map[string]any) []map[string]any {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		id, err := parseIDAny(item["id"])
		if err != nil || id == 0 {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return items
	}
	runtimesByImage := d.batchPublicRuntimeEnvironments(ids)
	historiesByImage := d.batchReviewHistories(ids)
	for _, item := range items {
		id, _ := parseIDAny(item["id"])
		runtimes := runtimesByImage[id]
		if runtimes == nil {
			runtimes = []map[string]any{}
		}
		item["runtime_environments"] = runtimes
		seen := map[string]bool{}
		platformTypes := []any{}
		for _, rt := range runtimes {
			pt, _ := rt["platform_type"].(string)
			if pt != "" && !seen[pt] {
				seen[pt] = true
				platformTypes = append(platformTypes, pt)
			}
		}
		item["platform_types"] = platformTypes
		histories := historiesByImage[id]
		if histories == nil {
			histories = []map[string]any{}
		}
		item["review_histories"] = histories
	}
	return items
}

// batchReviewHistories loads review history rows for many container images in
// one query, resolving the reviewer display name from platform staff.
func (d *DB) batchReviewHistories(imageIDs []int64) map[int64][]map[string]any {
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
	query := `SELECT h.container_image_id, h.id, h.action, h.note, h.reviewed_at, h.reviewer_id,
		COALESCE(NULLIF(TRIM(s.display_name),''), s.username, '')
	FROM ai_provider_containerimagereviewhistory h
	LEFT JOIN ai_provider_platformstaff s ON s.id = h.reviewer_id
	WHERE h.container_image_id IN (` + strings.Join(placeholders, ",") + `)
	ORDER BY h.reviewed_at, h.id`
	rows, err := d.SQL.Query(query, args...)
	if err != nil {
		// 表缺失（迁移未应用）时 fail-open，不阻断镜像列表
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var cid, hid, reviewerID sql.NullInt64
		var action, note, reviewerName string
		var reviewedAt sql.NullString
		if err := rows.Scan(&cid, &hid, &action, &note, &reviewedAt, &reviewerID, &reviewerName); err != nil {
			continue
		}
		if !cid.Valid {
			continue
		}
		h := map[string]any{
			"id":            NullIDStr(hid),
			"action":        action,
			"note":          note,
			"reviewer_name": reviewerName,
		}
		if t, err := time.Parse("2006-01-02 15:04:05", strings.TrimSpace(reviewedAt.String)); err == nil {
			h["reviewed_at"] = t.UTC().Format("2006-01-02T15:04:05Z")
		} else {
			h["reviewed_at"] = strings.TrimSpace(reviewedAt.String)
		}
		result[cid.Int64] = append(result[cid.Int64], h)
	}
	return result
}
