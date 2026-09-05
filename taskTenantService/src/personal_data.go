package main

// 个人信息导出内部端点（PIPL「导出权」）：taskAuth 聚合导出时调用。
// 契约：GET /api/internal/tenant/users/{user_id}/personal-data/ → {"data": {...}}
// 数据边界：该用户在全部租户的成员关系（复用 listMembersByUser，与成员列表页同源）。

import (
	"net/http"
)

// handleInternalTenantPersonalData 处理内部 personal-data 请求（注册于 handleInternalTenantUsersRouter）。
func handleInternalTenantPersonalData(w http.ResponseWriter, r *http.Request, userID string) {
	members, err := listMembersByUser(userID)
	if err != nil {
		logError("personal_data members list failed user_id="+userID+" err="+err.Error(), traceIDForError(r))
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	logInfo("personal_data memberships collected user_id="+userID+" count="+itoa(len(members)), traceIDForError(r))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{"memberships": members},
	})
}
