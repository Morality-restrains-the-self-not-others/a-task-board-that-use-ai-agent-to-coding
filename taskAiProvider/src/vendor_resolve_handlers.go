package main

import (
	"net/http"
	"strings"

	"taskAiProvider/infrastructure"
)

// handleResolveTargetArchitectures is the collection action
// POST /api/vendor/container-images/resolve-target-architectures/
//
// Response (200): target_architectures + size（必带），以及 best-effort 的技能列表
// （imageSkills.yaml）与自动运行说明（autoRunStep.md）。提取失败不影响架构解析：
// skills_status / auto_run_steps_status 携带 ok | not_found | auth_failed | failed。
func (a *App) handleResolveTargetArchitectures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
		return
	}
	var body struct {
		ImageURL string `json:"image_url"`
		Version  string `json:"version"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, 400, map[string]any{"detail": "无效 JSON"})
		return
	}
	merged, err := infrastructure.MergeImageURLWithVersion(body.ImageURL, body.Version)
	if err != nil {
		writeJSON(w, 400, map[string]any{"detail": err.Error()})
		return
	}
	meta, err := infrastructure.ResolveContainerImageMetadata(merged)
	if err != nil {
		logWarn(r.Context(), "event=resolve_target_architectures_failed err=%s", err.Error())
		writeJSON(w, 400, map[string]any{"detail": err.Error()})
		return
	}
	resp := map[string]any{
		"target_architectures": meta.TargetArchitectures,
		"size":                 meta.Size,
	}
	// Best-effort: 与保存后异步提取同源（单趟层扫描），同步返回供厂商预览。
	extract := a.autoRunAndSkillsExtractor()
	if extract == nil {
		extract = defaultAutoRunAndSkillsExtractor{}
	}
	got := extract.Extract(merged)
	resp["skills_status"] = got.Skills.Status
	resp["skills_detail"] = got.Skills.Detail
	resp["auto_run_steps_status"] = got.AutoRun.Status
	resp["auto_run_steps_detail"] = got.AutoRun.Detail
	resp["auto_run_steps_md"] = got.AutoRun.Markdown
	if got.Skills.Status == "ok" {
		resp["skills"] = got.Skills.List
	} else {
		resp["skills"] = nil
	}
	if got.AutoRun.Status == "failed" || got.AutoRun.Status == "auth_failed" || got.Skills.Status == "failed" || got.Skills.Status == "auth_failed" {
		logWarn(r.Context(), "event=resolve_target_architectures_extract_partial image=%s auto_run_status=%s skills_status=%s", merged, got.AutoRun.Status, got.Skills.Status)
	}
	writeJSON(w, 200, resp)
}

func isResolveTargetArchitecturesPath(path string) bool {
	rest := strings.Trim(strings.TrimPrefix(path, "/api/vendor/container-images/"), "/")
	return rest == "resolve-target-architectures"
}
