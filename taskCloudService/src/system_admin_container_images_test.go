package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── OPT-20260806-046: 系统管理员容器镜像 CRUD + 关联 ─────────────────────

func containerImagesRequest(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "test-admin")
	// OPT-20260806-052: 平台角色（super_admin）— 网关 forward-auth 注入 X-User-Roles
	req.Header.Set("X-User-Roles", "super_admin")
	w := httptest.NewRecorder()
	handleSystemAdminContainerImagesRoutes(w, req)
	return w
}

// TestSystemAdminContainerImagesForbiddenWithoutStaffRole — OPT-20260806-052：
// 无平台角色（普通用户）调用返回 403。
func TestSystemAdminContainerImagesForbiddenWithoutStaffRole(t *testing.T) {
	setupCloudTestDB(t)
	var buf bytes.Buffer
	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/cloud/container-images/", &buf)
	req.Header.Set("X-User-Id", "regular-user")
	// 无 X-User-Roles 头（普通登录用户）
	w := httptest.NewRecorder()
	handleSystemAdminContainerImagesRoutes(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("regular user status = %d, want 403 (body: %s)", w.Code, w.Body.String())
	}
}

func TestSystemAdminContainerImagesCRUD(t *testing.T) {
	setupCloudTestDB(t)

	// 空列表
	w := containerImagesRequest(t, http.MethodGet, "/api/system-admin/cloud/container-images/", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list empty: status = %d", w.Code)
	}
	var list []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}

	// 创建
	w = containerImagesRequest(t, http.MethodPost, "/api/system-admin/cloud/container-images/", map[string]interface{}{
		"name": "ubuntu-24.04", "description": "Ubuntu 24.04 容器", "image_url": "registry.example.com/ubuntu:24.04", "size": 1024,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", w.Code, w.Body.String())
	}
	var created map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	id := fmt.Sprint(created["id"])
	if id == "" {
		t.Fatal("create did not return id")
	}

	// 列表含 1 条
	w = containerImagesRequest(t, http.MethodGet, "/api/system-admin/cloud/container-images/", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 || list[0]["name"] != "ubuntu-24.04" {
		t.Fatalf("list after create = %v", list)
	}

	// 详情
	w = containerImagesRequest(t, http.MethodGet, "/api/system-admin/cloud/container-images/"+id+"/", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get: status = %d", w.Code)
	}

	// 更新
	w = containerImagesRequest(t, http.MethodPut, "/api/system-admin/cloud/container-images/"+id+"/", map[string]interface{}{
		"name": "ubuntu-24.04-lts", "size": 2048,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update: status = %d, body = %s", w.Code, w.Body.String())
	}
	var updated map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &updated)
	if updated["name"] != "ubuntu-24.04-lts" {
		t.Fatalf("updated name = %v", updated["name"])
	}

	// 关联（设置运行环境）
	w = containerImagesRequest(t, http.MethodPost, "/api/system-admin/cloud/container-images/"+id+"/set-cloud-server-images/", map[string]interface{}{
		"associations": []map[string]interface{}{
			{"platform_type": "aliyun", "cloud_server_image": 1001},
			{"platform_type": "tencent", "cloud_server_image": 2002},
		},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("set assoc: status = %d, body = %s", w.Code, w.Body.String())
	}

	// 读关联
	w = containerImagesRequest(t, http.MethodGet, "/api/system-admin/cloud/container-images/"+id+"/cloud-server-image-associations/", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get assoc: status = %d", w.Code)
	}
	var assocs []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &assocs)
	if len(assocs) != 2 {
		t.Fatalf("assocs = %d, want 2", len(assocs))
	}

	// 删除
	w = containerImagesRequest(t, http.MethodDelete, "/api/system-admin/cloud/container-images/"+id+"/", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d", w.Code)
	}

	// 删除后列表空 + 关联清理
	w = containerImagesRequest(t, http.MethodGet, "/api/system-admin/cloud/container-images/", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Fatalf("list after delete = %v", list)
	}
	var rowCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM system_admin_container_image_server_assoc WHERE container_image_id = ?`, id).Scan(&rowCount)
	if rowCount != 0 {
		t.Fatalf("assoc rows after delete = %d, want 0", rowCount)
	}
}

func TestSystemAdminContainerImagesValidation(t *testing.T) {
	setupCloudTestDB(t)

	// 缺 name → 400
	w := containerImagesRequest(t, http.MethodPost, "/api/system-admin/cloud/container-images/", map[string]interface{}{
		"image_url": "registry.example.com/x:1",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("missing name: status = %d", w.Code)
	}

	// 非法 id → 400
	w = containerImagesRequest(t, http.MethodGet, "/api/system-admin/cloud/container-images/abc/", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid id: status = %d", w.Code)
	}

	// 不存在 id 详情 → 404
	w = containerImagesRequest(t, http.MethodGet, "/api/system-admin/cloud/container-images/999999/", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing id get: status = %d", w.Code)
	}
}
