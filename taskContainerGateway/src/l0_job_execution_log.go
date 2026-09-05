package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// handleContainerJobExecutionLog merges container GET /api/jobs/{id}/steps (paged)
// + optional job meta (output stripped) + layer_changes into the SaaS zTree contract:
//
//	{"job": {...}, "steps": {...}, "layer_changes": {...}}
//
// Steps-first: never block the UI on huge job.output (old images may still embed it).
// Query: job_id (required), after_step, limit (default 20, max 50), layer_id (optional hint).
func handleContainerJobExecutionLog(w http.ResponseWriter, r *http.Request, sc scope, target containerTarget) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed, use GET"})
		return
	}
	q := r.URL.Query()
	jobID, errDetail := queryRequired(q, "job_id")
	if errDetail != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": errDetail})
		return
	}
	afterStep := parseNonNegIntQuery(q.Get("after_step"), 0)
	stepLimit := parseNonNegIntQuery(q.Get("limit"), 20)
	if stepLimit < 1 {
		stepLimit = 20
	}
	if stepLimit > 50 {
		stepLimit = 50
	}
	layerIDHint := strings.TrimSpace(q.Get("layer_id"))

	ctx := r.Context()
	root := scopedAPIRoot(target.BaseURL, sc)
	jobURL := root + "/jobs/" + pathEscape(jobID)
	stepsQ := url.Values{}
	stepsQ.Set("after_step", strconv.Itoa(afterStep))
	stepsQ.Set("limit", strconv.Itoa(stepLimit))
	stepsURL := jobURL + "/steps?" + stepsQ.Encode()

	// 1) steps first
	stepsBody := map[string]any{
		"steps":           []any{},
		"note":            nil,
		"trajectory_file": nil,
		"task":            nil,
		"total_steps":     0,
		"after_step":      afterStep,
		"next_after_step": nil,
		"has_more":        false,
	}
	stepsOK := false
	stepsStatus, stepsRaw, _ := forwardToOnlineService(ctx, http.MethodGet, stepsURL, target.AccessToken, nil)
	if stepsStatus < 400 {
		var parsed map[string]any
		if err := json.Unmarshal(stepsRaw, &parsed); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"detail": "invalid steps response"})
			return
		}
		stepsBody = parsed
		stepsOK = true
	} else {
		detail := strings.TrimSpace(string(stepsRaw))
		if detail == "" {
			detail = fmt.Sprintf("HTTP %d", stepsStatus)
		}
		if len(detail) > 800 {
			detail = detail[:800]
		}
		stepsBody["note"] = fmt.Sprintf("拉取代理步骤失败 (HTTP %d): %s", stepsStatus, detail)
	}

	// 2) job meta optional；旧镜像含巨量 output 时易超时，失败则 stub（steps 仍可用）
	// meta=0：跳过 GET /jobs/:id（前端层图已有 status/command 时）
	// 注意：meta=0 时不得写 status:""，否则前端 {...jobHint, ...stub} 会把层图 running 盖成空串，
	// 代理步骤凭步骤态显示「已结束」而 zTree 仍显示 running。
	skipJobMeta := strings.TrimSpace(q.Get("meta")) == "0"
	job := map[string]any{
		"id":             jobID,
		"layer_id":       nil,
		"output_omitted": true,
		"output_chars":   0,
	}
	if layerIDHint != "" {
		job["layer_id"] = layerIDHint
	}
	if !skipJobMeta {
		jobStatus, jobBody, _ := forwardToOnlineService(ctx, http.MethodGet, jobURL, target.AccessToken, nil)
		if jobStatus < 400 {
			var parsed map[string]any
			if err := json.Unmarshal(jobBody, &parsed); err != nil {
				if !stepsOK {
					writeJSON(w, http.StatusBadGateway, map[string]string{"detail": "invalid job response"})
					return
				}
			} else {
				job = stripJobOutputForExecLog(parsed)
			}
		} else if !stepsOK {
			writeRawJSON(w, jobStatus, jobBody)
			return
		}
	}

	layerChangesBody := map[string]any{
		"changes":         []any{},
		"change_count":    0,
		"truncated":       false,
		"detail":          "任务无 layer_id，无法查询变动文件",
		"layer_id":        nil,
		"parent_layer_id": nil,
		"same":            nil,
	}
	layerID := strings.TrimSpace(fmt.Sprintf("%v", job["layer_id"]))
	if layerID == "" || layerID == "<nil>" {
		layerID = layerIDHint
	}
	if layerID != "" && layerID != "<nil>" {
		if job["layer_id"] == nil || strings.TrimSpace(fmt.Sprintf("%v", job["layer_id"])) == "" {
			job["layer_id"] = layerID
		}
		// 首屏分页：默认 limit=100，前端滚动再拉 container-layer-diff-parent-files
		lcQ := url.Values{}
		lcQ.Set("offset", "0")
		lcQ.Set("limit", "100")
		layerChangesURL := root + "/layers/" + pathEscape(layerID) + "/diff/parent/files?" + lcQ.Encode()
		lcStatus, lcRaw, _ := forwardToOnlineService(ctx, http.MethodGet, layerChangesURL, target.AccessToken, nil)
		if lcStatus < 400 {
			var lb map[string]any
			if err := json.Unmarshal(lcRaw, &lb); err == nil {
				changes, _ := lb["changes"].([]any)
				if changes == nil {
					changes = []any{}
				}
				lb["changes"] = changes
				if _, ok := lb["change_count"]; !ok {
					lb["change_count"] = len(changes)
				}
				layerChangesBody = lb
			}
		} else {
			detail := strings.TrimSpace(string(lcRaw))
			if detail == "" {
				detail = fmt.Sprintf("HTTP %d", lcStatus)
			}
			if len(detail) > 800 {
				detail = detail[:800]
			}
			layerChangesBody = map[string]any{
				"changes":         []any{},
				"change_count":    0,
				"truncated":       false,
				"detail":          fmt.Sprintf("拉取文件变动失败 (HTTP %d): %s", lcStatus, detail),
				"layer_id":        layerID,
				"parent_layer_id": nil,
				"same":            nil,
			}
		}
	}

	out := map[string]any{
		"job":           job,
		"steps":         stepsBody,
		"layer_changes": layerChangesBody,
	}
	raw, _ := json.Marshal(out)
	writeRawJSON(w, http.StatusOK, raw)
}

func parseNonNegIntQuery(raw string, def int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func stripJobOutputForExecLog(job map[string]any) map[string]any {
	out := make(map[string]any, len(job)+2)
	for k, v := range job {
		if k == "output" {
			if _, ok := job["output_chars"]; !ok {
				out["output_chars"] = len(fmt.Sprintf("%v", v))
			}
			continue
		}
		out[k] = v
	}
	out["output_omitted"] = true
	return out
}
