package main

import (
	"authz"
	"net/http"
	"strings"
)

func handleInternalTenantBudgetPermissions(w http.ResponseWriter, r *http.Request) {
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	switch r.Method {
	case http.MethodGet:
		companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
		if companyID == "" {
			writeErrorJSON(w, r, http.StatusBadRequest, "company_id required")
			return
		}
		if err := requireLLMBudgetEnabled(companyID); err != nil {
			writeBudgetGateError(w, err)
			return
		}
		rows, err := listTenantBudgetPermissions(budgetConn, companyID)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			items = append(items, tenantBudgetPermissionToMap(row))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	default:
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleInternalTenantBudgetPermissionsUpsert(w http.ResponseWriter, r *http.Request) {
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "bad json")
		return
	}
	companyID := strField(body, "company_id")
	if companyID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "company_id required")
		return
	}
	if err := requireLLMBudgetEnabled(companyID); err != nil {
		writeBudgetGateError(w, err)
		return
	}
	rawItems, ok := body["items"].([]any)
	if !ok {
		writeErrorJSON(w, r, http.StatusBadRequest, "items 必须是数组")
		return
	}
	items := make([]tenantBudgetPermissionRow, 0, len(rawItems))
	for _, raw := range rawItems {
		m, ok := raw.(map[string]any)
		if !ok {
			writeErrorJSON(w, r, http.StatusBadRequest, "items 元素必须是对象")
			return
		}
		subjectType := strings.TrimSpace(strField(m, "subject_type"))
		subjectID := strings.TrimSpace(strField(m, "subject_id"))
		if !validBudgetSubjectType(subjectType) || subjectID == "" {
			writeErrorJSON(w, r, http.StatusBadRequest, "subject_type 或 subject_id 无效")
			return
		}
		items = append(items, tenantBudgetPermissionRow{
			CompanyID:          companyID,
			SubjectType:        subjectType,
			SubjectID:          subjectID,
			CanRaiseTaskBudget: boolField(m, "can_raise_task_budget"),
		})
	}
	rows, err := upsertTenantBudgetPermissions(budgetConn, companyID, items)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, tenantBudgetPermissionToMap(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func handleInternalTenantBudgetPermissionsEvaluateRaise(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "bad json")
		return
	}
	companyID := strField(body, "company_id")
	userID := strField(body, "user_id")
	memberID := strField(body, "member_id")
	taskOwnerMemberID := strField(body, "task_owner_member_id")
	if companyID == "" || userID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "company_id, user_id required")
		return
	}
	isTenantAdmin := boolField(body, "is_tenant_admin")
	isWorkspaceAdmin := boolField(body, "is_workspace_admin")
	if memberID == "" {
		mid, _, resolveErr := resolveUserMember(r, companyID, userID)
		if resolveErr == nil && mid != "" {
			memberID = mid
		}
	}
	// v63: 租户管理员判定迁移至 authz 权限码（PDP 已含 is_admin 过渡 + creator 回填）
	if authz.HasPerm(r, companyID, authz.PermCloudManage) {
		isTenantAdmin = true
	}
	allowed, err := evaluateUserCanRaiseTaskBudget(
		budgetConn,
		companyID,
		userID,
		memberID,
		taskOwnerMemberID,
		isTenantAdmin,
		isWorkspaceAdmin,
	)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"allowed":   allowed,
		"member_id": memberID,
	})
}

// handleUserTenantBudgetPermissions serves browser-facing GET/PATCH
// /api/tenant/{tenant_id}/budget-permissions/ (gateway forward-auth; no internal secret).
func handleUserTenantBudgetPermissions(w http.ResponseWriter, r *http.Request) {
	tenantID := getAuthTenant(r)
	if tenantID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "缺少租户")
		return
	}
	if !ensureTenantMember(w, r, tenantID) {
		return
	}
	budgetConn := requireBudgetDB(w)
	if budgetConn == nil {
		return
	}
	if err := requireLLMBudgetEnabled(tenantID); err != nil {
		writeBudgetGateError(w, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := listTenantBudgetPermissions(budgetConn, tenantID)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		items := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			items = append(items, tenantBudgetPermissionToMap(row))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPatch, http.MethodPut, http.MethodPost:
		if !ensureTenantAdminOrCreator(w, r, tenantID) {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "无效 JSON")
			return
		}
		rawItems, ok := body["items"].([]any)
		if !ok {
			writeErrorJSON(w, r, http.StatusBadRequest, "items 必须是数组")
			return
		}
		items := make([]tenantBudgetPermissionRow, 0, len(rawItems))
		for _, raw := range rawItems {
			m, ok := raw.(map[string]any)
			if !ok {
				writeErrorJSON(w, r, http.StatusBadRequest, "items 元素必须是对象")
				return
			}
			subjectType := strings.TrimSpace(strField(m, "subject_type"))
			subjectID := strings.TrimSpace(strField(m, "subject_id"))
			if !validBudgetSubjectType(subjectType) || subjectID == "" {
				writeErrorJSON(w, r, http.StatusBadRequest, "subject_type 或 subject_id 无效")
				return
			}
			items = append(items, tenantBudgetPermissionRow{
				CompanyID:          tenantID,
				SubjectType:        subjectType,
				SubjectID:          subjectID,
				CanRaiseTaskBudget: boolField(m, "can_raise_task_budget"),
			})
		}
		rows, err := upsertTenantBudgetPermissions(budgetConn, tenantID, items)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			out = append(out, tenantBudgetPermissionToMap(row))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	default:
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func ensureTenantAdminOrCreator(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	if !ensureTenantMember(w, r, tenantID) {
		return false
	}
	// v63: 租户管理员判定迁移至 authz 权限码
	if authz.HasPerm(r, tenantID, authz.PermCloudManage) {
		return true
	}
	writeErrorJSON(w, r, http.StatusForbidden, "仅租户管理员可配置预算上调权限")
	return false
}
