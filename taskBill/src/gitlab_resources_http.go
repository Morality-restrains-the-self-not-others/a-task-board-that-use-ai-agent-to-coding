package main

import (
	"log/slog"
	"net/http"
	"strings"

	"tracelog"
)

func handleGitlabResources(w http.ResponseWriter, r *http.Request) {
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		regionSlug := resolveRegionSlug(r)
		// 无 region：返回 resources[]（Navbar 代码仓库下拉）；有 region：单区详情。
		if regionSlug == "" {
			list, err := listTenantGitlabResourceSummaries(tid, requestIsTester(r))
			if err != nil {
				slog.ErrorContext(ctx, "gitlab_resources_list_failed", "level", "error", "error", err.Error(), "tenant_id", formatID(tid))
				writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
				return
			}
			slog.InfoContext(ctx, "gitlab_resources_list_ok", "level", "info", "tenant_id", formatID(tid), "count", len(list))
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"tenant_id": formatID(tid),
				"resources": list,
			})
			return
		}
		if meta, mErr := getGitlabRegionBySlug(regionSlug); mErr == nil && meta != nil {
			if !CanUseGitlabRegion(meta.AccessMode, requestIsTester(r)) {
				slog.WarnContext(ctx, "gitlab_region_dev_mode_forbidden",
					"level", "warn",
					"tenant_id", formatID(tid),
					"region", regionSlug,
					"access_mode", meta.AccessMode,
				)
				writeErrorJSON(w, http.StatusForbidden, errGitlabRegionDevModeForbidden.Error(), tracelog.TraceIDFromContext(r.Context()))
				return
			}
		}
		view, err := regionResourceView(tid, regionSlug)
		if err != nil {
			slog.ErrorContext(ctx, "gitlab_resources_get_failed", "level", "error", "error", err.Error(), "tenant_id", formatID(tid), "region", regionSlug)
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, view)
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

func handleInternalReportGitlabDiskUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, err := parseIDField(body["tenant_id"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	bytes, err := parseNonNegInt64(body["disk_used_bytes"], "disk_used_bytes")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	regionSlug := strings.TrimSpace(stringField(body, "region"))
	if err := reportGitlabDiskUsage(tid, bytes, regionSlug); err != nil {
		slog.ErrorContext(r.Context(), "gitlab_disk_usage_report_failed",
			"level", "error", "error", err.Error(), "tenant_id", formatID(tid))
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	view, err := gitlabResourceView(tid)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"tenant_id":       formatID(tid),
			"disk_used_bytes": bytes,
			"disk_used_gb":    diskUsedGBFromBytes(bytes),
		})
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func handleGitlabResourcesPurchase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireTenantAdmin(w, r, formatID(tid)) {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	regionSlug := strings.TrimSpace(stringField(body, "region"))
	if regionSlug == "" {
		writeErrorJSON(w, http.StatusBadRequest, "region required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	diskGB, err := parseNonNegInt64(body["disk_gb"], "disk_gb")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	diskMonths, err := parseDiskMonths(body, diskGB)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	trafficGB, err := parseNonNegInt64(body["traffic_prepaid_gb"], "traffic_prepaid_gb")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	ctx := r.Context()
	slog.InfoContext(ctx, "gitlab_resource_purchase_use_resource_order",
		"level", "info",
		"tenant_id", formatID(tid),
		"region", regionSlug,
		"disk_gb", diskGB,
		"disk_months", diskMonths,
		"traffic_prepaid_gb", trafficGB,
	)
	writeJSON(w, http.StatusConflict, map[string]interface{}{
		"error": "GitLab 资源须通过资源订单支付购买，支付成功后直接发放配额",
		"code":  "USE_RESOURCE_ORDER",
	})
}
