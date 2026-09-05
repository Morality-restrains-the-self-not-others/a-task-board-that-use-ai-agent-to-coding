package infrastructure

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

func (d *DB) ListCloudServerImages(vendorID int64) ([]map[string]any, error) {
	rows, err := d.SQL.Query(`SELECT csi.id, csi.vendor_id, COALESCE(csi.platform_type,''), COALESCE(csi.image_name,''), COALESCE(csi.image_id,''), COALESCE(csi.region,''), COALESCE(csi.os_type,''), COALESCE(csi.os_version,''), COALESCE(csi.architecture,''), COALESCE(csi.image_type,''),
		csi.image_size_gb, csi.is_active, csi.default_instance_type_id, csi.default_instance_type_label, csi.base_cpu_cores, csi.base_memory_gib, csi.userdata_template_id,
		t.name, t.version
		FROM ai_provider_vendorcloudserverimage csi
		LEFT JOIN ai_provider_userdatatemplate t ON t.id = csi.userdata_template_id
		WHERE csi.vendor_id=? ORDER BY csi.updated_at DESC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCSI(rows)
}

func (d *DB) ListCloudServerImagesAdmin() ([]map[string]any, error) {
	rows, err := d.SQL.Query(`SELECT csi.id, csi.vendor_id, COALESCE(csi.platform_type,''), COALESCE(csi.image_name,''), COALESCE(csi.image_id,''), COALESCE(csi.region,''), COALESCE(csi.os_type,''), COALESCE(csi.os_version,''), COALESCE(csi.architecture,''), COALESCE(csi.image_type,''),
		csi.image_size_gb, csi.is_active, csi.default_instance_type_id, csi.default_instance_type_label, csi.base_cpu_cores, csi.base_memory_gib, csi.userdata_template_id,
		t.name, t.version
		FROM ai_provider_vendorcloudserverimage csi
		LEFT JOIN ai_provider_userdatatemplate t ON t.id = csi.userdata_template_id
		ORDER BY csi.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCSI(rows)
}

func scanCSI(rows *sql.Rows) ([]map[string]any, error) {
	var out []map[string]any
	for rows.Next() {
		var id, vid int64
		var platform, name, imageID, region, osType, osVer, arch, imageType string
		var defID, defLabel sql.NullString
		var sizeGB, cpu, mem sql.NullInt64
		var active int
		var tpl sql.NullInt64
		var tplName, tplVer sql.NullString
		if err := rows.Scan(&id, &vid, &platform, &name, &imageID, &region, &osType, &osVer, &arch, &imageType, &sizeGB, &active, &defID, &defLabel, &cpu, &mem, &tpl, &tplName, &tplVer); err != nil {
			return nil, err
		}
		m := map[string]any{
			"id": IDStr(id), "vendor": IDStr(vid), "platform_type": platform, "platform_type_display": platformTypeDisplay(platform),
			"image_name": name, "image_id": imageID,
			"region": region, "os_type": osType, "os_version": osVer, "architecture": arch, "image_type": imageType,
			"is_active":                   active != 0,
			"default_instance_type_id":    nullStrOrEmpty(defID),
			"default_instance_type_label": nullStrOrEmpty(defLabel),
		}
		if sizeGB.Valid {
			m["image_size_gb"] = sizeGB.Int64
		} else {
			m["image_size_gb"] = nil
		}
		if cpu.Valid {
			m["base_cpu_cores"] = cpu.Int64
		} else {
			m["base_cpu_cores"] = nil
		}
		if mem.Valid {
			m["base_memory_gib"] = mem.Int64
		} else {
			m["base_memory_gib"] = nil
		}
		m["userdata_template"] = userdataTemplateObj(tpl, tplName, tplVer)
		out = append(out, m)
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, rows.Err()
}

func nullStrOrEmpty(n sql.NullString) string {
	if n.Valid {
		return n.String
	}
	return ""
}

func userdataTemplateObj(tpl sql.NullInt64, name, version sql.NullString) any {
	if !tpl.Valid {
		return nil
	}
	return map[string]any{
		"id":      IDStr(tpl.Int64),
		"name":    nullStrOrEmpty(name),
		"version": nullStrOrEmpty(version),
	}
}

// OptionalUserdataTemplateID reads userdata_template_id from a JSON body.
// present=false when the key is absent; id=0 with present=true means clear.
func OptionalUserdataTemplateID(body map[string]any) (id int64, present bool) {
	if body == nil {
		return 0, false
	}
	v, ok := body["userdata_template_id"]
	if !ok {
		return 0, false
	}
	if v == nil {
		return 0, true
	}
	parsed, err := parseIDAny(v)
	if err != nil {
		return 0, true
	}
	return parsed, true
}

func optionalBodyInt64(body map[string]any, key string) sql.NullInt64 {
	v, ok := body[key]
	if !ok || v == nil {
		return sql.NullInt64{}
	}
	id, err := parseIDAny(v)
	if err != nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: id, Valid: true}
}

func (d *DB) CreateCloudServerImage(vendorID int64, body map[string]any) (int64, error) {
	id := NextID()
	now := nowTS()
	get := func(k, def string) string {
		if v, ok := body[k].(string); ok {
			return v
		}
		return def
	}
	active := 1
	if v, ok := body["is_active"].(bool); ok && !v {
		active = 0
	}
	sizeGB := optionalBodyInt64(body, "image_size_gb")
	cpu := optionalBodyInt64(body, "base_cpu_cores")
	mem := optionalBodyInt64(body, "base_memory_gib")
	tplID, tplPresent := OptionalUserdataTemplateID(body)
	var tpl any
	if tplPresent && tplID != 0 {
		tpl = tplID
	} else {
		tpl = nil
	}
	var sizeAny, cpuAny, memAny any
	if sizeGB.Valid {
		sizeAny = sizeGB.Int64
	}
	if cpu.Valid {
		cpuAny = cpu.Int64
	}
	if mem.Valid {
		memAny = mem.Int64
	}
	_, err := d.SQL.Exec(`INSERT INTO ai_provider_vendorcloudserverimage
		(id, platform_type, image_name, image_id, region, os_type, os_version, architecture, image_type, image_size_gb, is_active, created_at, updated_at, vendor_id,
		 default_instance_type_id, default_instance_type_label, base_cpu_cores, base_memory_gib, userdata_template_id)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, get("platform_type", ""), get("image_name", ""), get("image_id", ""), get("region", ""), get("os_type", ""), get("os_version", ""),
		get("architecture", ""), get("image_type", ""), sizeAny, active, now, now, vendorID,
		get("default_instance_type_id", ""), get("default_instance_type_label", ""), cpuAny, memAny, tpl)
	return id, err
}

// UpdateCloudServerImage applies a vendor portal PATCH to a cloud-server image row.
func (d *DB) UpdateCloudServerImage(id int64, body map[string]any) error {
	if body == nil {
		return nil
	}
	sets := []string{}
	args := []any{}
	for _, k := range []string{
		"image_name", "os_type", "os_version", "architecture", "image_type",
		"default_instance_type_id", "default_instance_type_label",
	} {
		if v, ok := body[k].(string); ok {
			sets = append(sets, k+"=?")
			args = append(args, v)
		}
	}
	if v, ok := body["is_active"].(bool); ok {
		active := 0
		if v {
			active = 1
		}
		sets = append(sets, "is_active=?")
		args = append(args, active)
	}
	for _, k := range []string{"image_size_gb", "base_cpu_cores", "base_memory_gib"} {
		if _, ok := body[k]; !ok {
			continue
		}
		n := optionalBodyInt64(body, k)
		sets = append(sets, k+"=?")
		if n.Valid {
			args = append(args, n.Int64)
		} else {
			args = append(args, nil)
		}
	}
	if tplID, ok := OptionalUserdataTemplateID(body); ok {
		sets = append(sets, "userdata_template_id=?")
		if tplID == 0 {
			args = append(args, nil)
		} else {
			args = append(args, tplID)
		}
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "updated_at=?")
	args = append(args, nowTS(), id)
	_, err := d.SQL.Exec(`UPDATE ai_provider_vendorcloudserverimage SET `+strings.Join(sets, ",")+` WHERE id=?`, args...)
	return err
}

// SetCloudServerImageUserdataTemplate sets or clears CSI.userdata_template_id (0 clears).
func (d *DB) SetCloudServerImageUserdataTemplate(csiID, templateID int64) error {
	if csiID == 0 {
		return fmt.Errorf("invalid cloud_server_image_id")
	}
	var tpl any
	if templateID != 0 {
		tpl = templateID
	}
	_, err := d.SQL.Exec(
		`UPDATE ai_provider_vendorcloudserverimage SET userdata_template_id=?, updated_at=? WHERE id=?`,
		tpl, nowTS(), csiID,
	)
	return err
}

func (d *DB) DeleteCloudServerImage(id int64) error {
	_, _ = d.SQL.Exec(`DELETE FROM ai_provider_vendorcloudserverimageuserdata WHERE cloud_server_image_id=?`, id)
	_, _ = d.SQL.Exec(`DELETE FROM ai_provider_containercloudserverassociation WHERE cloud_server_image_id=?`, id)
	_, err := d.SQL.Exec(`DELETE FROM ai_provider_vendorcloudserverimage WHERE id=?`, id)
	return err
}

func (d *DB) GetCloudServerImageVendor(id int64) (int64, error) {
	var vid int64
	err := d.SQL.QueryRow(`SELECT vendor_id FROM ai_provider_vendorcloudserverimage WHERE id=?`, id).Scan(&vid)
	return vid, err
}

func scanUserDataTemplateRow(scanner interface {
	Scan(dest ...any) error
}) (map[string]any, error) {
	var id int64
	var name, version, osType, vars, cvars, content, script, createdAt, updatedAt string
	var active int
	if err := scanner.Scan(&id, &name, &version, &osType, &vars, &cvars, &content, &script, &active, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	var vj, cj any
	_ = json.Unmarshal([]byte(vars), &vj)
	_ = json.Unmarshal([]byte(cvars), &cj)
	return map[string]any{
		"id": IDStr(id), "name": name, "version": version, "os_type": osType,
		"variables": vj, "container_variables": cj, "content": content, "auto_verify_script": script,
		"is_active": active != 0, "created_at": createdAt, "updated_at": updatedAt,
	}, nil
}

const userDataTemplateSelectCols = `id, COALESCE(name,''), COALESCE(version,''), COALESCE(os_type,''), COALESCE(variables,''), COALESCE(container_variables,''), COALESCE(content,''), COALESCE(auto_verify_script,''), is_active, created_at, updated_at`

func (d *DB) ListUserDataTemplates(activeOnly bool) ([]map[string]any, error) {
	q := `SELECT ` + userDataTemplateSelectCols + ` FROM ai_provider_userdatatemplate`
	if activeOnly {
		q += ` WHERE is_active=1`
	}
	q += ` ORDER BY name`
	rows, err := d.SQL.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		item, err := scanUserDataTemplateRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	// OPT-20260724-005: attach CSI reference count so admin UI can warn
	// "将解除 N 个云镜像的模板关联" before deleting.
	for _, item := range out {
		id, err := parseIDAny(item["id"])
		if err != nil || id == 0 {
			item["csi_ref_count"] = 0
			continue
		}
		var n int
		if err := d.SQL.QueryRow(
			`SELECT COUNT(*) FROM ai_provider_vendorcloudserverimage WHERE userdata_template_id=?`, id,
		).Scan(&n); err != nil {
			item["csi_ref_count"] = 0
		} else {
			item["csi_ref_count"] = n
		}
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

// GetUserDataTemplate returns one template including created_at/updated_at for admin list/detail.
func (d *DB) GetUserDataTemplate(id int64) (map[string]any, error) {
	row := d.SQL.QueryRow(`SELECT `+userDataTemplateSelectCols+` FROM ai_provider_userdatatemplate WHERE id=?`, id)
	item, err := scanUserDataTemplateRow(row)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (d *DB) CreateUserDataTemplate(body map[string]any) (int64, error) {
	id := NextID()
	now := nowTS()
	get := func(k string) string {
		if v, ok := body[k].(string); ok {
			return v
		}
		return ""
	}
	vars := MustJSON(body["variables"])
	if body["variables"] == nil {
		vars = "{}"
	}
	cvars := MustJSON(body["container_variables"])
	if body["container_variables"] == nil {
		cvars = "{}"
	}
	_, err := d.SQL.Exec(`INSERT INTO ai_provider_userdatatemplate (id, name, version, variables, container_variables, content, auto_verify_script, is_active, created_at, updated_at, os_type)
		VALUES (?,?,?,?,?,?,?,1,?,?,?)`, id, get("name"), get("version"), vars, cvars, get("content"), get("auto_verify_script"), now, now, get("os_type"))
	return id, err
}

func (d *DB) UpdateUserDataTemplate(id int64, body map[string]any) error {
	sets := []string{"updated_at=?"}
	args := []any{nowTS()}
	for _, k := range []string{"name", "version", "content", "auto_verify_script", "os_type"} {
		if v, ok := body[k].(string); ok {
			sets = append(sets, k+"=?")
			args = append(args, v)
		}
	}
	if v, ok := body["is_active"].(bool); ok {
		sets = append(sets, "is_active=?")
		if v {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if v, ok := body["variables"]; ok {
		sets = append(sets, "variables=?")
		args = append(args, MustJSON(v))
	}
	if v, ok := body["container_variables"]; ok {
		sets = append(sets, "container_variables=?")
		args = append(args, MustJSON(v))
	}
	args = append(args, id)
	_, err := d.SQL.Exec(`UPDATE ai_provider_userdatatemplate SET `+strings.Join(sets, ",")+` WHERE id=?`, args...)
	return err
}

// DeleteUserDataTemplate removes a template after clearing CSI FK references.
// Without nullifying ai_provider_vendorcloudserverimage.userdata_template_id,
// SQLite FOREIGN KEY fails and a naive DELETE no-ops while handlers historically
// still returned 204 — admin UI then removed the row until refresh restored it.
func (d *DB) DeleteUserDataTemplate(id int64) error {
	tx, err := d.SQL.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(
		`UPDATE ai_provider_vendorcloudserverimage SET userdata_template_id=NULL, updated_at=? WHERE userdata_template_id=?`,
		nowTS(), id,
	); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM ai_provider_userdatatemplate WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("userdata template %d not found", id)
	}
	return tx.Commit()
}
