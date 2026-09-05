package main

import (
	"database/sql"
	"net/http"
	"strings"
)

// --- Workspace Settings（/api/workspace/* 遗留路径）---
// 前端 WorkspaceSettings.vue 使用 /api/workspace/info/ + /api/workspace/settings/
// （Django 时代路径，OPT-049 迁移后网关无路由 → 落入 502 孤儿桶）。
// 此处复用 project_workspace_entries 表提供基本信息读写；通知/安全/高级设置
// 等 action 无对应后端存储（半成品区块），返回 501 明确提示而非 502。

func handleWorkspaceInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := requireAuthUser(w, r); !ok {
		return
	}
	wid := strings.TrimSpace(r.URL.Query().Get("workspace_id"))
	if wid == "" {
		writeError(w, r, http.StatusBadRequest, "workspace_id required")
		return
	}
	row := db.QueryRow(`SELECT id,name,COALESCE(description,'') FROM project_workspace_entries WHERE id=?`, wid)
	var id, name, desc string
	if err := row.Scan(&id, &name, &desc); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, r, http.StatusNotFound, "workspace not found")
			return
		}
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"workspace": map[string]string{
			"name":        name,
			"description": desc,
		},
	})
}

func handleWorkspaceSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := requireAuthUser(w, r); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid form")
		return
	}
	action := r.FormValue("action")
	wid := strings.TrimSpace(r.FormValue("workspace_id"))
	if wid == "" {
		writeError(w, r, http.StatusBadRequest, "workspace_id required")
		return
	}
	switch action {
	case "update_basic_info":
		name := strings.TrimSpace(r.FormValue("name"))
		desc := strings.TrimSpace(r.FormValue("description"))
		if name == "" && desc == "" {
			writeError(w, r, http.StatusBadRequest, "name or description required")
			return
		}
		if name != "" {
			if _, err := db.Exec("UPDATE project_workspace_entries SET name=? WHERE id=?", name, wid); err != nil {
				writeError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
		}
		if desc != "" {
			if _, err := db.Exec("UPDATE project_workspace_entries SET description=? WHERE id=?", desc, wid); err != nil {
				writeError(w, r, http.StatusInternalServerError, err.Error())
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"status": "success"})
	case "update_notification_settings", "update_security_settings", "update_advanced_settings":
		// 前端半成品区块：无对应后端存储，明确提示而非网关 502。
		writeError(w, r, http.StatusNotImplemented, "该设置暂未提供（无后端存储）")
	default:
		writeError(w, r, http.StatusBadRequest, "未知操作类型: "+action)
	}
}
