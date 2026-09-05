package main

import "strings"

// extractAutoRunStepsFn 可替换以便单测隔离真实 registry 拉取（OPT-20260820-035）。
var extractAutoRunStepsFn = extractAutoRunStepsFromImage

func updateInstalledImageAutoRunSteps(tenantID, id, md, digest, status string) error {
	_, err := db.Exec(`UPDATE cloud_tenant_installed_images
		SET auto_run_steps_md=?, auto_run_steps_extract_status=?, auto_run_steps_digest=?
		WHERE tenant_id=? AND id=?`,
		md, status, digest, tenantID, id)
	return err
}

// ensureInstalledImageAutoRunSteps 安装时若 catalog 快照在抽取完成前被拷贝（auto_run_steps_md 为空），
// 从 image_url 再抽一次回填 tenant_installed_images，避免已安装列表/创建任务一直看不到自动运行说明。
// 抽取失败不阻断安装：仅记录 extract_status（best-effort），维持安装成功。
func ensureInstalledImageAutoRunSteps(img *TenantInstalledImage, tenantID string) error {
	if img == nil {
		return nil
	}
	if strings.TrimSpace(img.AutoRunStepsMd) != "" {
		return nil
	}
	if strings.TrimSpace(img.ImageURL) == "" {
		return nil
	}
	return applyInstalledImageAutoRun(img, tenantID, extractAutoRunStepsFn(img.ImageURL, defaultAutoRunStepsPath))
}

// applyInstalledImageAutoRun persists one auto_run_steps_* extraction outcome
// onto the installed image and its DB row. Shared by the single-file and
// combined install-time re-extract paths.
func applyInstalledImageAutoRun(img *TenantInstalledImage, tenantID string, result autoRunStepsExtractResult) error {
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = "failed"
	}
	if result.Status == "ok" && strings.TrimSpace(result.Markdown) != "" {
		img.AutoRunStepsMd = strings.TrimSpace(result.Markdown)
		img.AutoRunStepsDigest = strings.TrimSpace(result.Digest)
		img.AutoRunStepsExtractStatus = "ok"
	} else {
		img.AutoRunStepsExtractStatus = status
		if detail := strings.TrimSpace(result.Detail); detail != "" {
			img.AutoRunStepsExtractStatus = status + ":" + detail
		}
	}
	return updateInstalledImageAutoRunSteps(tenantID, img.ID, img.AutoRunStepsMd, img.AutoRunStepsDigest, img.AutoRunStepsExtractStatus)
}
