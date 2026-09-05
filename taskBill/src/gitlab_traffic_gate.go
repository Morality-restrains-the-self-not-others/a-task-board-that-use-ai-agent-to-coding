package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"tracelog"
)

const (
	trafficGateOK           = "OK"
	trafficGateCI           = "CI_SKIP"
	trafficGateIntranet     = "INTRANET_SKIP"
	trafficGateNotPurchased = "TRAFFIC_NOT_PURCHASED"
	trafficGateExceeded     = "TRAFFIC_QUOTA_EXCEEDED"
	trafficGateUnmapped     = "UNMAPPED_PROJECT"
)

var tenantGitlabGroupPathRe = regexp.MustCompile(`^tenant-(\d+)(?:/|$)`)

type TrafficQuotaExceededError struct {
	PrepaidGB float64
	UsedGB    float64
	Code      string
}

func (e *TrafficQuotaExceededError) Error() string {
	if e == nil {
		return "GitLab 流量配额不足"
	}
	if e.Code == trafficGateNotPurchased {
		return "未预购 GitLab 流量，公网克隆/拉取已阻断，请先购买流量"
	}
	return fmt.Sprintf("GitLab 流量配额不足，已用 %.6g GB / 预购 %.0f GB，请先购买流量", e.UsedGB, e.PrepaidGB)
}

type gitlabTrafficGateRequest struct {
	TenantID       int64
	ProjectPath    string
	Region         string
	IsIntranet     bool
	FromCI         bool
	GitlabUsername string // 认证用户的 GitLab 用户名（rails 传入）；个人命名空间仓按仓所有者归租户
}

type gitlabTrafficGateResult struct {
	Allowed          bool    `json:"allowed"`
	Code             string  `json:"code"`
	TenantID         string  `json:"tenant_id,omitempty"`
	Region           string  `json:"region,omitempty"`
	TrafficPrepaidGB int64   `json:"traffic_prepaid_gb"`
	TrafficUsedGB    float64 `json:"traffic_used_gb"`
	Message          string  `json:"message"`
}

func gitlabTrafficDownloadAllowed(prepaidGB int64, usedGB float64, fromCI, isIntranet bool) (bool, string) {
	if fromCI {
		return true, trafficGateCI
	}
	if isIntranet {
		return true, trafficGateIntranet
	}
	if prepaidGB <= 0 {
		return false, trafficGateNotPurchased
	}
	if usedGB >= float64(prepaidGB) {
		return false, trafficGateExceeded
	}
	return true, trafficGateOK
}

func parseTenantIDFromGitlabProjectPath(projectPath string) (int64, error) {
	p := strings.Trim(strings.TrimSpace(projectPath), "/")
	m := tenantGitlabGroupPathRe.FindStringSubmatch(p)
	if len(m) < 2 {
		return 0, fmt.Errorf("project_path 不含 tenant-{id} 组")
	}
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid tenant id in project_path")
	}
	return id, nil
}

func evaluateGitlabTrafficGate(req gitlabTrafficGateRequest) gitlabTrafficGateResult {
	tid := req.TenantID
	if tid <= 0 && strings.TrimSpace(req.ProjectPath) != "" {
		if parsed, err := parseTenantIDFromGitlabProjectPath(req.ProjectPath); err == nil {
			tid = parsed
		}
	}
	region := strings.TrimSpace(req.Region)
	if tid <= 0 && !req.FromCI && !req.IsIntranet {
		// 个人命名空间（example-user/somanyad）与无 tenant-{id} 前缀的组仓：
		// 按 GitLab 用户名归集到租户（凭据绑定 → 成员 → 区域资源三重过滤），
		// 使该租户的流量配额同样生效；归集失败仍走 UNMAPPED fail-open。
		// CI/同区域内网本就跳过配额，不做解析。
		if resolved := resolveTenantByUsername(req, region); resolved > 0 {
			tid = resolved
			slog.Info("gitlab_traffic_gate_username_resolved",
				"level", "info",
				"tenant_id", formatID(tid),
				"region", region,
				"project_path", strings.TrimSpace(req.ProjectPath),
			)
		}
	}
	if tid <= 0 {
		// OPT-20260823-062: 闸门判定计数（fail-open 也计数，供 Grafana 观测静默失效）
		incGateCode(trafficGateUnmapped)
		if cfg.TrafficGateUnmappedReject {
			// OPT-20260823-013: 盘点+迁组完成后可置 true，未映射仓默认拒绝并打 warn。
			slog.Warn("gitlab_traffic_gate_unmapped_reject",
				"level", "warn",
				"project_path", strings.TrimSpace(req.ProjectPath),
				"region", region,
				"code", trafficGateUnmapped,
			)
			return gitlabTrafficGateResult{
				Allowed: false,
				Code:    trafficGateUnmapped,
				Region:  region,
				Message: "project not mapped to tenant group; download blocked (reject-mode)",
			}
		}
		return gitlabTrafficGateResult{
			Allowed: true,
			Code:    trafficGateUnmapped,
			Region:  region,
			Message: "project not mapped to tenant group; download allowed (fail-open)",
		}
	}
	if region == "" {
		region = defaultGitlabRegion
	}
	res, err := getTenantGitlabResourceByRegion(tid, region)
	if err != nil {
		slog.Warn("gitlab_traffic_gate_lookup_failed",
			"level", "warn",
			"tenant_id", formatID(tid),
			"region", region,
			"error", err.Error(),
		)
		// 配额查询失败仍 fail-open —— 计数 UNMAPPED，隧道/DB 故障时该码会突增
		incGateCode(trafficGateUnmapped)
		return gitlabTrafficGateResult{
			Allowed:  true,
			Code:     trafficGateUnmapped,
			TenantID: formatID(tid),
			Region:   region,
			Message:  "quota lookup failed; download allowed (fail-open)",
		}
	}
	used := gitlabTrafficUsedGB(res, 0)
	allowed, code := gitlabTrafficDownloadAllowed(res.TrafficPrepaidGB, used, req.FromCI, req.IsIntranet)
	incGateCode(code)
	msg := "ok"
	switch code {
	case trafficGateNotPurchased:
		msg = "未预购 GitLab 流量，公网与任务节点克隆/拉取已阻断"
	case trafficGateExceeded:
		msg = "GitLab 流量配额已用尽，公网与任务节点克隆/拉取已阻断"
	case trafficGateCI:
		msg = "CI download skipped"
	case trafficGateIntranet:
		msg = "intranet download skipped"
		slog.Info("gitlab_traffic_gate_intranet_skip",
			"level", "info",
			"tenant_id", formatID(tid),
			"region", region,
			"from_ci", req.FromCI,
			"is_intranet", true,
		)
	}
	return gitlabTrafficGateResult{
		Allowed:          allowed,
		Code:             code,
		TenantID:         formatID(tid),
		Region:           region,
		TrafficPrepaidGB: res.TrafficPrepaidGB,
		TrafficUsedGB:    used,
		Message:          msg,
	}
}

func handleInternalGitlabTrafficGate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	req := gitlabTrafficGateRequest{}
	if r.Method == http.MethodPost {
		body, _ := readJSONBody(r)
		if v, err := parseIDField(body["tenant_id"]); err == nil {
			req.TenantID = v
		}
		req.ProjectPath = stringField(body, "project_path")
		req.Region = stringField(body, "region")
		if v, ok := body["is_intranet"].(bool); ok {
			req.IsIntranet = v
		} else if v, ok := body["same_region_intranet"].(bool); ok {
			req.IsIntranet = v
		}
		if v, ok := body["from_ci"].(bool); ok {
			req.FromCI = v
		}
		req.GitlabUsername = strings.TrimSpace(stringField(body, "gitlab_username"))
	} else {
		q := r.URL.Query()
		if v, err := parseIDField(q.Get("tenant_id")); err == nil {
			req.TenantID = v
		}
		req.ProjectPath = strings.TrimSpace(q.Get("project_path"))
		req.Region = strings.TrimSpace(q.Get("region"))
		req.IsIntranet = q.Get("is_intranet") == "1" || q.Get("is_intranet") == "true"
		req.FromCI = q.Get("from_ci") == "1" || q.Get("from_ci") == "true"
		req.GitlabUsername = strings.TrimSpace(q.Get("gitlab_username"))
	}
	got := evaluateGitlabTrafficGate(req)
	slog.InfoContext(r.Context(), "gitlab_traffic_gate",
		"level", "info",
		"allowed", got.Allowed,
		"code", got.Code,
		"tenant_id", got.TenantID,
		"region", got.Region,
		"project_path", strings.TrimSpace(req.ProjectPath),
		"gitlab_username", strings.TrimSpace(req.GitlabUsername),
		"from_ci", req.FromCI,
		"is_intranet", req.IsIntranet,
		"traffic_prepaid_gb", got.TrafficPrepaidGB,
		"traffic_used_gb", got.TrafficUsedGB,
	)
	writeJSON(w, http.StatusOK, got)
}
