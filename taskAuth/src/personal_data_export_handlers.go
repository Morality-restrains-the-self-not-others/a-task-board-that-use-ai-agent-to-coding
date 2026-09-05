package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// 个人信息导出 HTTP 端点（/api/accounts/users/me/personal-data-export/*）。
// 与 account-deletion 同族：仅本人可访问（requireAuthenticatedUser），
// 导出为只读操作不做密码二次验证（与注销的破坏性语义区分）。

func handlePersonalDataExportRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/accounts/users/me/personal-data-export/")
	path = strings.Trim(path, "/")
	switch path {
	case "request":
		if r.Method == http.MethodPost {
			handlePersonalDataExportRequest(w, r)
			return
		}
	case "status":
		if r.Method == http.MethodGet {
			handlePersonalDataExportStatus(w, r)
			return
		}
	case "download":
		if r.Method == http.MethodGet {
			handlePersonalDataExportDownload(w, r)
			return
		}
	}
	writeErrorDetail(w, r, http.StatusNotFound, "not found")
}

func handlePersonalDataExportRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	row, failures, err := generatePersonalDataExport(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "personal_data_export_generate_failed",
			"user_id", userID, "error", err.Error())
		writeErrorDetail(w, r, http.StatusInternalServerError, "导出生成失败，请稍后重试")
		return
	}
	unavailable := make([]string, 0, len(failures))
	for _, f := range failures {
		unavailable = append(unavailable, f.Section)
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"status":         row.Status,
		"export_id":      fmt.Sprintf("%d", row.ID),
		"generated_at":   row.GeneratedAt.UTC().Format(time.RFC3339),
		"expires_at":     row.ExpiresAt.UTC().Format(time.RFC3339),
		"retention_days": personalDataExportRetentionDays(),
		"sections_unavailable": unavailable,
	})
}

func handlePersonalDataExportStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	payload, err := exportStatusPayload(userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "personal_data_export_status_failed",
			"user_id", userID, "error", err.Error())
		writeErrorDetail(w, r, http.StatusInternalServerError, "状态查询失败")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func handlePersonalDataExportDownload(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	row, err := getLatestPersonalDataExport(userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "personal_data_export_download_failed",
			"user_id", userID, "error", err.Error())
		writeErrorDetail(w, r, http.StatusInternalServerError, "下载失败")
		return
	}
	if row == nil {
		writeErrorDetail(w, r, http.StatusNotFound, "尚未生成导出，请先发起导出请求")
		return
	}
	if !row.downloadable() {
		slog.InfoContext(r.Context(), "personal_data_export_download_expired",
			"user_id", userID, "export_id", row.ID)
		writeErrorMap(w, r, http.StatusGone, map[string]interface{}{
			"error":  "export_expired",
			"detail": "导出已过期，请重新发起导出请求",
		})
		return
	}
	filename := fmt.Sprintf("personal-data-%d.json", row.ID)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(row.Content))
	slog.InfoContext(r.Context(), "personal_data_export_downloaded",
		"user_id", userID, "export_id", row.ID, "status", row.Status)
}
