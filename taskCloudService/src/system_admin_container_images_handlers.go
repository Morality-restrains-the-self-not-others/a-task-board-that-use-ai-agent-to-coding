package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"authz"
	"gatewayauth"
)

// --- System-Admin Container Images (OPT-20260806-046) ---
//
// 孤儿页面 SystemAdminContainerImages.vue 的后端补齐：容器镜像 CRUD +
// 与厂商服务器镜像的关联（set-cloud-server-images）。
// 路由：/api/system-admin/cloud/container-images/（网关已把 /api/system-admin/cloud/*
// 转发到本服务，前端去 uid 段后直接可达）。
// 认证：gatewayauth.ApplyGatewayUser（网关 forward-auth 已认证系统管理员）。

func handleSystemAdminContainerImagesRoutes(w http.ResponseWriter, r *http.Request) {
	gatewayauth.ApplyGatewayUser(r, cfg.GatewayInternalSecret)
	// OPT-20260806-052: 系统管理员后台资源，仅平台角色（super_admin/employee）可读写
	// （网关 forward-auth 注入 X-User-Roles；authz.IsPlatformStaff 统一判定）
	if !authz.IsPlatformStaff(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "需要系统管理员权限")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/system-admin/cloud/container-images")
	path = strings.Trim(path, "/")
	parts := []string{}
	if path != "" {
		parts = strings.Split(path, "/")
	}
	id := int64(0)
	action := ""
	if len(parts) >= 1 && parts[0] != "" {
		parsed, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			writeErrorJSON(w, r, 400, "invalid container image id")
			return
		}
		id = parsed
		if len(parts) >= 2 {
			action = strings.Trim(parts[1], "/")
		}
	}

	switch {
	case id == 0 && r.Method == http.MethodGet:
		handleSystemAdminListContainerImages(w, r)
	case id == 0 && r.Method == http.MethodPost:
		handleSystemAdminCreateContainerImage(w, r)
	case id > 0 && action == "" && r.Method == http.MethodGet:
		handleSystemAdminGetContainerImage(w, r, id)
	case id > 0 && action == "" && (r.Method == http.MethodPut || r.Method == http.MethodPatch):
		handleSystemAdminUpdateContainerImage(w, r, id)
	case id > 0 && action == "" && r.Method == http.MethodDelete:
		handleSystemAdminDeleteContainerImage(w, r, id)
	case id > 0 && (action == "cloud-server-image-associations" || action == "cloud-server-image-association") && r.Method == http.MethodGet:
		handleSystemAdminContainerImageAssociations(w, r, id)
	case id > 0 && action == "set-cloud-server-images" && r.Method == http.MethodPost:
		handleSystemAdminSetContainerImageAssociations(w, r, id)
	default:
		writeErrorJSON(w, r, 405, "method not allowed")
	}
}

func handleSystemAdminListContainerImages(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, name, COALESCE(description,''), image_url, size, status, created_at, updated_at
		FROM system_admin_container_images ORDER BY updated_at DESC`)
	if err != nil {
		log.Printf("[taskCloudService] list container images: %v", err)
		writeErrorJSON(w, r, 500, "list failed")
		return
	}
	defer rows.Close()
	items := []map[string]interface{}{}
	for rows.Next() {
		var id, size int64
		var name, desc, imageURL, status, createdAt, updatedAt string
		if err := rows.Scan(&id, &name, &desc, &imageURL, &size, &status, &createdAt, &updatedAt); err != nil {
			log.Printf("[taskCloudService] scan container image: %v", err)
			continue
		}
		items = append(items, map[string]interface{}{
			"id": strconv.FormatInt(id, 10), "name": name, "description": desc,
			"image_url": imageURL, "size": size, "status": status,
			"created_at": createdAt, "updated_at": updatedAt,
		})
	}
	writeJSON(w, 200, items)
}

func handleSystemAdminCreateContainerImage(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, r, 400, "invalid json")
		return
	}
	name := strings.TrimSpace(strField(body, "name"))
	imageURL := strings.TrimSpace(strField(body, "image_url"))
	if name == "" || imageURL == "" {
		writeErrorJSON(w, r, 400, "name 与 image_url 不能为空")
		return
	}
	desc := strField(body, "description")
	size, _ := body["size"].(float64)
	res, err := db.Exec(`INSERT INTO system_admin_container_images (name, description, image_url, size, status, created_by)
		VALUES (?, ?, ?, ?, 'draft', ?)`, name, desc, imageURL, int64(size), r.Header.Get("X-User-Id"))
	if err != nil {
		log.Printf("[taskCloudService] create container image: %v", err)
		writeErrorJSON(w, r, 500, "create failed")
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, 201, map[string]interface{}{"id": strconv.FormatInt(id, 10), "status": "draft"})
}

func handleSystemAdminGetContainerImage(w http.ResponseWriter, r *http.Request, id int64) {
	var name, desc, imageURL, status, createdAt, updatedAt string
	var size int64
	err := db.QueryRow(`SELECT name, COALESCE(description,''), image_url, size, status, created_at, updated_at
		FROM system_admin_container_images WHERE id = ?`, id).
		Scan(&name, &desc, &imageURL, &size, &status, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		writeErrorJSON(w, r, 404, "not found")
		return
	}
	if err != nil {
		writeErrorJSON(w, r, 500, "get failed")
		return
	}
	writeJSON(w, 200, map[string]interface{}{
		"id": strconv.FormatInt(id, 10), "name": name, "description": desc,
		"image_url": imageURL, "size": size, "status": status,
		"created_at": createdAt, "updated_at": updatedAt,
	})
}

func handleSystemAdminUpdateContainerImage(w http.ResponseWriter, r *http.Request, id int64) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, r, 400, "invalid json")
		return
	}
	var name, desc, imageURL string
	var size int64
	if v, ok := body["name"].(string); ok {
		name = strings.TrimSpace(v)
	}
	if v, ok := body["description"].(string); ok {
		desc = v
	}
	if v, ok := body["image_url"].(string); ok {
		imageURL = strings.TrimSpace(v)
	}
	if v, ok := body["size"].(float64); ok {
		size = int64(v)
	}
	if name == "" && imageURL == "" && desc == "" && size == 0 {
		writeErrorJSON(w, r, 400, "无有效更新字段")
		return
	}
	if _, err := db.Exec(`UPDATE system_admin_container_images SET
		name = CASE WHEN ? <> '' THEN ? ELSE name END,
		description = CASE WHEN ? <> '' THEN ? ELSE description END,
		image_url = CASE WHEN ? <> '' THEN ? ELSE image_url END,
		size = CASE WHEN ? > 0 THEN ? ELSE size END
		WHERE id = ?`,
		name, name, desc, desc, imageURL, imageURL, size, size, id); err != nil {
		log.Printf("[taskCloudService] update container image %d: %v", id, err)
		writeErrorJSON(w, r, 500, "update failed")
		return
	}
	handleSystemAdminGetContainerImage(w, r, id)
}

func handleSystemAdminDeleteContainerImage(w http.ResponseWriter, r *http.Request, id int64) {
	if _, err := db.Exec(`DELETE FROM system_admin_container_images WHERE id = ?`, id); err != nil {
		writeErrorJSON(w, r, 500, "delete failed")
		return
	}
	_, _ = db.Exec(`DELETE FROM system_admin_container_image_server_assoc WHERE container_image_id = ?`, id)
	w.WriteHeader(204)
}

func handleSystemAdminContainerImageAssociations(w http.ResponseWriter, r *http.Request, id int64) {
	rows, err := db.Query(`SELECT platform_type, cloud_server_image_id
		FROM system_admin_container_image_server_assoc WHERE container_image_id = ?`, id)
	if err != nil {
		writeErrorJSON(w, r, 500, "associations query failed")
		return
	}
	defer rows.Close()
	assocs := []map[string]interface{}{}
	for rows.Next() {
		var platform string
		var csiID int64
		if err := rows.Scan(&platform, &csiID); err != nil {
			continue
		}
		assocs = append(assocs, map[string]interface{}{
			"platform_type":       platform,
			"cloud_server_image": map[string]interface{}{"id": csiID},
		})
	}
	writeJSON(w, 200, assocs)
}

func handleSystemAdminSetContainerImageAssociations(w http.ResponseWriter, r *http.Request, id int64) {
	var body struct {
		Associations []struct {
			PlatformType      string `json:"platform_type"`
			CloudServerImage  int64  `json:"cloud_server_image"`
		} `json:"associations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorJSON(w, r, 400, "invalid json")
		return
	}
	tx, err := db.Begin()
	if err != nil {
		writeErrorJSON(w, r, 500, "tx begin failed")
		return
	}
	if _, err := tx.Exec(`DELETE FROM system_admin_container_image_server_assoc WHERE container_image_id = ?`, id); err != nil {
		tx.Rollback()
		writeErrorJSON(w, r, 500, "assoc reset failed")
		return
	}
	for _, a := range body.Associations {
		if a.CloudServerImage <= 0 {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO system_admin_container_image_server_assoc (container_image_id, platform_type, cloud_server_image_id)
			VALUES (?, ?, ?)`, id, a.PlatformType, a.CloudServerImage); err != nil {
			tx.Rollback()
			writeErrorJSON(w, r, 500, "assoc insert failed")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeErrorJSON(w, r, 500, "tx commit failed")
		return
	}
	writeJSON(w, 200, map[string]interface{}{"ok": true})
}
