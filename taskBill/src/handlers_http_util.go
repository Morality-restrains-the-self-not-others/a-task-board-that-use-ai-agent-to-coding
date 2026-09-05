package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"authz"
	"gatewaycors"
	"tracelog"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeErrorJSON 写入含 trace_id 的错误 JSON 响应，便于前端 data-traceId 展示
func writeErrorJSON(w http.ResponseWriter, status int, msg string, traceID string) {
	m := map[string]string{"error": msg}
	if traceID != "" {
		m["trace_id"] = traceID
	}
	writeJSON(w, status, m)
}

// writeErrorJSONMap 与 writeErrorJSON 同语义但保留调用方附加字段（path/refer 等
// 前端流程或排障需要的数据键），同时注入 trace_id。OPT-20260807-059：复合错误体
// 统一走此入口，保证 body 兜底契约（trace_id）对全部错误分支成立。
func writeErrorJSONMap(w http.ResponseWriter, r *http.Request, status int, body map[string]string) {
	if tid := tracelog.TraceIDFromContext(r.Context()); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

func readJSONBody(r *http.Request) (map[string]interface{}, error) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var body map[string]interface{}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&body); err != nil {
		return nil, err
	}
	return body, nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", gatewaycors.AllowHeaders)
			w.Header().Set("Access-Control-Expose-Headers", gatewaycors.ExposeHeaders)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requireInternalSecret(r *http.Request) bool {
	// 无密钥配置时允许所有请求（仅限开发/测试环境）
	if cfg.InternalSecret == "" && cfg.GatewayInternalSecret == "" {
		return true
	}
	// 通道 A：直连内部服务调用（X-TaskBill-Internal-Secret）
	if cfg.InternalSecret != "" && r.Header.Get("X-TaskBill-Internal-Secret") == cfg.InternalSecret {
		return true
	}
	// 通道 B：经网关代理的请求（APISIX proxy-rewrite 注入 X-TaskGateway-Internal-Secret）
	if cfg.GatewayInternalSecret != "" && r.Header.Get("X-TaskGateway-Internal-Secret") == cfg.GatewayInternalSecret {
		return true
	}
	return false
}

// requireInternalOrGatewayAuth 双通道鉴权：
// 1. 内部服务调用（X-TaskBill-Internal-Secret）
// 2. 网关转发（X-Gateway-Auth-Verified + X-User-Roles 平台角色）
// v63: 平台角色（super_admin/employee）判定取代遗留 X-Auth-Superuser/X-Auth-Staff 头。
func requireInternalOrGatewayAuth(w http.ResponseWriter, r *http.Request) bool {
	// 通道 1：内部密钥
	if requireInternalSecret(r) {
		return true
	}
	// 通道 2：网关 Token 鉴权（管理员）
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) == "1" {
		if authz.IsPlatformStaff(r) {
			return true
		}
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
		return false
	}
	writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
	return false
}

// stripTenantIDKV removes convention segments `tenant_id/<id>/` so suffix-based
// routers (…/billing/gitlab-regions/) also match convention URLs.
// Legacy `/api/tenant/<id>/…` paths are left unchanged.
func stripTenantIDKV(path string) string {
	const marker = "/tenant_id/"
	for {
		idx := strings.Index(path, marker)
		if idx < 0 {
			return path
		}
		rest := path[idx+len(marker):]
		slash := strings.IndexByte(rest, '/')
		if slash < 0 {
			// …/tenant_id/<id> with no trailing slash → drop to parent dir + /
			return path[:idx] + "/"
		}
		path = path[:idx] + rest[slash:]
	}
}

func parseTenantID(path string) (int64, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if (p == "tenant" || p == "tenant_id") && i+1 < len(parts) {
			tid, err := strconv.ParseInt(parts[i+1], 10, 64)
			return tid, err == nil
		}
	}
	return 0, false
}

func accountJSON(acc *BillingAccount) map[string]interface{} {
	return map[string]interface{}{
		"id":                formatID(acc.ID),
		"tenant":            formatID(acc.TenantID),
		"points":            acc.Balance,
		"available_balance": acc.Balance,
		"frozen_balance":    acc.FrozenBalance,
		"created_at":        acc.CreatedAt,
		"updated_at":        acc.UpdatedAt,
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "taskBill"})
}
