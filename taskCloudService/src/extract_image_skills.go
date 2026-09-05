package main

import (
	"encoding/json"
	"strings"
)

type imageSkillsExtractResult struct {
	List   imageSkillList
	JSON   string
	Digest string
	Status string
	Detail string
}

func extractImageSkillsFromImage(imageURL string) imageSkillsExtractResult {
	file := extractAutoRunStepsFromImage(imageURL, defaultImageSkillsPath)
	out := imageSkillsExtractResult{Digest: file.Digest, Status: file.Status, Detail: file.Detail}
	if file.Status != "ok" {
		if file.Status == "not_found" {
			out.Detail = "imageSkills.yaml not found in image layers"
		}
		return out
	}
	list, err := parseImageSkillsYAML(file.Markdown)
	if err != nil {
		out.Status = "failed"
		out.Detail = err.Error()
		return out
	}
	raw, err := json.Marshal(list)
	if err != nil {
		out.Status = "failed"
		out.Detail = err.Error()
		return out
	}
	out.List = list
	out.JSON = string(raw)
	out.Status = "ok"
	out.Detail = ""
	return out
}

var extractImageSkillsFn = extractImageSkillsFromImage

func updateInstalledImageSkills(tenantID, id, jsonBody, digest, status string) error {
	_, err := db.Exec(`UPDATE cloud_tenant_installed_images
		SET image_skills_json=?, image_skills_extract_status=?, image_skills_digest=?
		WHERE tenant_id=? AND id=?`,
		jsonBody, status, digest, tenantID, id)
	return err
}

// ensureInstalledImageSkills 安装时若 catalog 快照在抽取完成前被拷贝（技能列表为空），
// 从 image_url 再抽一次回填。失败不阻断安装。
func ensureInstalledImageSkills(img *TenantInstalledImage, tenantID string) error {
	if img == nil {
		return nil
	}
	if len(decodeImageSkillsJSON(img.ImageSkillsJSON).Skills) > 0 {
		return nil
	}
	if strings.TrimSpace(img.ImageURL) == "" {
		return nil
	}
	return applyInstalledImageSkills(img, tenantID, extractImageSkillsFn(img.ImageURL))
}

// applyInstalledImageSkills persists one image_skills_* extraction outcome onto
// the installed image and its DB row. Shared by the single-file and combined
// install-time re-extract paths.
func applyInstalledImageSkills(img *TenantInstalledImage, tenantID string, result imageSkillsExtractResult) error {
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = "failed"
	}
	if result.Status == "ok" && strings.TrimSpace(result.JSON) != "" {
		// D1=B：持久化前为缺失 id 的技能派生稳定 ID（seed=external_id 回退 installed id），
		// 单文件与合并 OCI walk 两条抽取路径落库的 image_skills_json 均带技能 id。
		jsonBody := result.JSON
		if list := assignImageSkillIDs(result.List, imageSkillIDSeed(img.ExternalImageID, img.ID)); len(list.Skills) > 0 {
			if raw, err := json.Marshal(list); err == nil {
				jsonBody = string(raw)
			}
		}
		img.ImageSkillsJSON = jsonBody
		img.ImageSkillsDigest = strings.TrimSpace(result.Digest)
		img.ImageSkillsExtractStatus = "ok"
	} else {
		img.ImageSkillsExtractStatus = status
		if result.Status == "not_found" {
			img.ImageSkillsJSON = emptyImageSkillsJSON()
		}
		if detail := strings.TrimSpace(result.Detail); detail != "" {
			img.ImageSkillsExtractStatus = status + ":" + detail
		}
	}
	return updateInstalledImageSkills(tenantID, img.ID, img.ImageSkillsJSON, img.ImageSkillsDigest, img.ImageSkillsExtractStatus)
}

// ensureInstalledImageExtracts backfills both autoRunStep.md and imageSkills.yaml
// with a single registry walk when the install-time catalog snapshot was copied
// before extraction completed (OPT-20260821-018). Short-circuits per field the
// same way the two single-file ensures do; each missing field is applied from
// the combined result, so a snapshot missing both costs one walk instead of two.
func ensureInstalledImageExtracts(img *TenantInstalledImage, tenantID string) error {
	if img == nil {
		return nil
	}
	if strings.TrimSpace(img.ImageURL) == "" {
		return nil
	}
	needAuto := strings.TrimSpace(img.AutoRunStepsMd) == ""
	needSkills := len(decodeImageSkillsJSON(img.ImageSkillsJSON).Skills) == 0
	if !needAuto && !needSkills {
		return nil
	}
	result := extractAutoRunAndSkillsFn(img.ImageURL)
	if needAuto {
		if err := applyInstalledImageAutoRun(img, tenantID, result.AutoRun); err != nil {
			return err
		}
	}
	if needSkills {
		if err := applyInstalledImageSkills(img, tenantID, result.Skills); err != nil {
			return err
		}
	}
	return nil
}
