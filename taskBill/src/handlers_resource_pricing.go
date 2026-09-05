package main

import (
	"net/http"
	"strings"

	"tracelog"
)

// handleInternalResourcePricing GET/PUT /api/internal/taskbill/resource-pricing/
// 管理员读取/更新资源定价（直接操作 billing_unit 表）
func handleInternalResourcePricing(w http.ResponseWriter, r *http.Request) {
	if !requireInternalOrGatewayAuth(w, r) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		rp, err := getCurrentResourcePricing()
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "success",
			"pricing": resourcePricingToJSON(rp),
		})

	case http.MethodPut:
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
			return
		}

		rp, err := getCurrentResourcePricing()
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}

		// 解析新价格（元→分），只更新提供的字段
		if v, ok := body["task_post_price_yuan"]; ok {
			if f, e := parseFloat64Field(v); e == nil && f >= 0 {
				rp.TaskPostUnitPriceCents = yuanToCents(f)
			}
		}
		if v, ok := body["gitlab_disk_price_yuan"]; ok {
			if f, e := parseFloat64Field(v); e == nil && f >= 0 {
				rp.GitlabDiskUnitPriceCents = yuanToCents(f)
			}
		}
		if v, ok := body["gitlab_traffic_price_yuan"]; ok {
			if f, e := parseFloat64Field(v); e == nil && f >= 0 {
				rp.GitlabTrafficUnitPriceCents = yuanToCents(f)
			}
		}
		if v, ok := body["gitlab_traffic_requires_unit_type"]; ok {
			if s, ok := v.(string); ok {
				rp.GitlabTrafficRequiresUnitType = s
			}
		}

		if err := updateResourcePricing(rp); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}

		// 约束字段单独更新（仅当显式传入时）
		if _, ok := body["gitlab_traffic_requires_unit_type"]; ok {
			_ = updateResourcePricingConstraint("gitlab_traffic", rp.GitlabTrafficRequiresUnitType)
		}

		// 重新读取确认
		updated, _ := getCurrentResourcePricing()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "success",
			"message": "资源定价已更新，新订单将按最新价格计算",
			"pricing": resourcePricingToJSON(updated),
		})

	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

// handlePublicResourcePricing GET /api/public/resource-pricing/
// 公开端点，无需认证，供 Pricing.vue 使用
func handlePublicResourcePricing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	rp, err := getCurrentResourcePricing()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":               "success",
		"task_post_price":      rp.TaskPostUnitPriceCents,
		"gitlab_disk_price":    rp.GitlabDiskUnitPriceCents,
		"gitlab_traffic_price": rp.GitlabTrafficUnitPriceCents,
		"pricing":              resourcePricingToJSON(rp),
	})
}

// handleOrderPricingV2 GET /api/tenant/{id}/billing/order-pricing/
// 返回当前 billing_unit 实时价格（不再使用账户锁价）
func handleOrderPricingV2(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	rp, err := getCurrentResourcePricing()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pricing": resourcePricingToJSON(rp),
	})
}

// handleInternalPricingPackages GET /api/internal/taskbill/pricing-packages/
// 列出价格套餐（管理端）。billing_pricing_package 表已在 migration 018 删除，
// billing_unit 是当前唯一定价来源；此端点返回当前资源定价的兼容视图。
func handleInternalPricingPackages(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	switch r.Method {
	case http.MethodGet:
		rp, err := getCurrentResourcePricing()
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":                      "success",
			"items":                       []interface{}{},
			"current_resolved_package_id": nil,
			"pricing":                     resourcePricingToJSON(rp),
		})

	case http.MethodPost:
		// billing_pricing_package 表已在 migration 018 删除；创建套餐不再支持
		writeErrorJSONMap(w, r, http.StatusGone, map[string]string{
			"detail": "pricing_package table removed (migration 018); use resource-pricing endpoint instead",
			"refer":  "/api/internal/taskbill/resource-pricing/",
		})

	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

// ---- 内部路由 ----

func handleInternalResourcePricingRouter(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// 兼容 internal 和 system-admin 两条路由（前缀不同）
	for _, prefix := range []string{"/api/internal/taskbill/resource-pricing", "/api/system-admin/resource-pricing"} {
		if after, ok := strings.CutPrefix(path, prefix); ok {
			path = after
			break
		}
	}
	path = strings.Trim(path, "/")
	if path == "" {
		handleInternalResourcePricing(w, r)
		return
	}
	writeErrorJSON(w, http.StatusNotFound, "not found", tracelog.TraceIDFromContext(r.Context()))
}
