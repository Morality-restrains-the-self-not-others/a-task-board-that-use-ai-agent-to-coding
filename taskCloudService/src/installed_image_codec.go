package main

import (
	"database/sql"
	"encoding/json"
	"time"
)

// installedImageSelectCols 与 scanInstalledImage 的 Scan 顺序必须一致。
const installedImageSelectCols = `id,tenant_id,external_image_id,name,description,version,image_url,icon_url,target_architectures,size,is_dev_mode,vendor_id,vendor_name,installed_by_id,installed_at,updated_at,userdata_template_id,auto_run_steps_md,auto_run_steps_extract_status,auto_run_steps_digest,image_skills_json,image_skills_extract_status,image_skills_digest,saas_inbound_skill_version`

func installedImageToJSON(img TenantInstalledImage) map[string]interface{} {
	arches := knownCPUArchitectures(img.TargetArchitectures)
	if len(arches) == 0 {
		inferred := extractCPUArchitecturesFromText(img.Version, img.Name, img.ImageURL)
		if len(inferred) > 0 {
			arches = inferred
		} else {
			arches = img.TargetArchitectures
		}
	}
	out := map[string]interface{}{
		"id":                            img.ID,
		"tenant_id":                     img.TenantID,
		"external_image_id":             img.ExternalImageID,
		"name":                          img.Name,
		"description":                   nullIfEmpty(img.Description),
		"version":                       nullIfEmpty(img.Version),
		"image_url":                     img.ImageURL,
		"icon_url":                      img.IconURL,
		"target_architectures":          arches,
		"size":                          img.Size,
		"is_dev_mode":                   img.IsDevMode,
		"vendor_id":                     nullIfEmpty(img.VendorID),
		"vendor_name":                   img.VendorName,
		"installed_by_id":               nullIfEmpty(img.InstalledByID),
		"installed_by_name":             nullIfEmpty(img.InstalledByID),
		"installed_at":                  img.InstalledAt.UTC().Format(time.RFC3339Nano),
		"updated_at":                    imageTimeJSON(img.UpdatedAt),
		"userdata_template":             nil,
		"auto_run_steps_md":             img.AutoRunStepsMd,
		"auto_run_steps_extract_status": img.AutoRunStepsExtractStatus,
		"auto_run_steps_digest":         nullIfEmpty(img.AutoRunStepsDigest),
		"image_skills":                  decodeImageSkillsJSON(img.ImageSkillsJSON),
		"image_skills_extract_status":   img.ImageSkillsExtractStatus,
		"image_skills_digest":           nullIfEmpty(img.ImageSkillsDigest),
		"saas_inbound_skill_version":    nullIfEmpty(img.SaasInboundSkillVersion),
	}
	if img.UserdataTemplateID != "" {
		out["userdata_template_id"] = img.UserdataTemplateID
	}
	return out
}

func scanInstalledImage(scanner interface {
	Scan(dest ...interface{}) error
}) (*TenantInstalledImage, error) {
	var img TenantInstalledImage
	var desc, version, vendorID, vendorName, installedBy, userdataID sql.NullString
	var autoRunMd, autoRunStatus, autoRunDigest sql.NullString
	var skillsJSON, skillsStatus, skillsDigest sql.NullString
	var saasSkillVersion, iconURL sql.NullString
	var archRaw string
	var size sql.NullInt64
	var isDev int
	var installedAt, updatedAt sql.NullString
	err := scanner.Scan(
		&img.ID, &img.TenantID, &img.ExternalImageID, &img.Name, &desc, &version, &img.ImageURL, &iconURL,
		&archRaw, &size, &isDev, &vendorID, &vendorName, &installedBy, &installedAt, &updatedAt, &userdataID,
		&autoRunMd, &autoRunStatus, &autoRunDigest,
		&skillsJSON, &skillsStatus, &skillsDigest, &saasSkillVersion,
	)
	if err != nil {
		return nil, err
	}
	img.IconURL = iconURL.String
	img.SaasInboundSkillVersion = saasSkillVersion.String
	img.Description = desc.String
	img.Version = version.String
	img.VendorID = vendorID.String
	img.VendorName = vendorName.String
	img.InstalledByID = installedBy.String
	img.UserdataTemplateID = userdataID.String
	img.AutoRunStepsMd = autoRunMd.String
	img.AutoRunStepsExtractStatus = autoRunStatus.String
	img.AutoRunStepsDigest = autoRunDigest.String
	img.ImageSkillsJSON = skillsJSON.String
	img.ImageSkillsExtractStatus = skillsStatus.String
	img.ImageSkillsDigest = skillsDigest.String
	img.IsDevMode = isDev != 0
	if size.Valid {
		v := size.Int64
		img.Size = &v
	}
	if archRaw != "" {
		_ = json.Unmarshal([]byte(archRaw), &img.TargetArchitectures)
	}
	if img.TargetArchitectures == nil {
		img.TargetArchitectures = []string{}
	}
	if installedAt.Valid && installedAt.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, installedAt.String); err == nil {
			img.InstalledAt = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", installedAt.String); err == nil {
			img.InstalledAt = t
		}
	}
	if updatedAt.Valid && updatedAt.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, updatedAt.String); err == nil {
			img.UpdatedAt = t
		} else if t, err := time.Parse("2006-01-02 15:04:05", updatedAt.String); err == nil {
			img.UpdatedAt = t
		}
	}
	return &img, nil
}

func imageTimeJSON(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// catalogIconURL 从目录/厂商镜像载荷取图标：顶层 icon_url 优先，否则 image_group.icon_url。
func catalogIconURL(imageData map[string]interface{}) string {
	if u := strField(imageData, "icon_url"); u != "" {
		return u
	}
	ig, _ := imageData["image_group"].(map[string]interface{})
	if ig == nil {
		return ""
	}
	return strField(ig, "icon_url")
}
