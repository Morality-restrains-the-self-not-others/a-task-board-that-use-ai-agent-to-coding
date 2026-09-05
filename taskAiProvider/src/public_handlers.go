package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"taskAiProvider/domain"
	"taskAiProvider/infrastructure"
)

// filterSunsetSkillCatalogItems 过滤公开目录中仍声明 sunset 契约版本的镜像。
// OPT-20260820-027：契约 sunset 后，已上架镜像若仍声明旧版本须从公开目录过滤，
// 避免用户选中已下线实现（write 路径已由 ValidateSaasInboundSkillVersion 拒新写）。
func filterSunsetSkillCatalogItems(items []map[string]any, entries []domain.SkillVersionEntry) []map[string]any {
	if len(entries) == 0 {
		return items
	}
	supported := domain.SupportedSkillVersionSet(entries)
	kept := make([]map[string]any, 0, len(items))
	for _, it := range items {
		if supported[domain.NormalizeSkillVersion(fmt.Sprintf("%v", it["saas_inbound_skill_version"]))] {
			kept = append(kept, it)
		}
	}
	return kept
}

func (a *App) handlePublicCatalog(w http.ResponseWriter, r *http.Request) {
	items, err := a.DB.ListApprovedCatalog()
	if err != nil {
		writeJSON(w, 500, map[string]any{"detail": err.Error()})
		return
	}
	entries, _, err := a.skillCatalog()
	if err == nil {
		items = filterSunsetSkillCatalogItems(items, entries)
	}
	a.scheduleMissingCatalogExtracts(r.Context(), items)
	writeJSON(w, 200, items)
}

func (a *App) handlePublicUserDataTemplates(w http.ResponseWriter, r *http.Request) {
	items, err := a.DB.ListUserDataTemplates(true)
	if err != nil {
		writeJSON(w, 500, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, 200, items)
}

func (a *App) handleImageRuntimeEnvs(w http.ResponseWriter, r *http.Request) {
	imageID := r.URL.Query().Get("image_id")
	if imageID == "" {
		writeJSON(w, 400, []any{})
		return
	}
	id, err := strconv.ParseInt(imageID, 10, 64)
	if err != nil {
		writeJSON(w, 200, []any{})
		return
	}
	items, err := a.DB.RuntimeEnvironments(id)
	if err != nil {
		writeJSON(w, 200, []any{})
		return
	}
	writeJSON(w, 200, items)
}

func (a *App) handleRuntimeUserdata(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform_type")
	region := r.URL.Query().Get("region")
	if platform == "" || region == "" {
		writeJSON(w, 400, map[string]any{"detail": "platform_type 与 region 必填"})
		return
	}
	imageID := r.URL.Query().Get("image_id")
	csiID := r.URL.Query().Get("cloud_server_image_id")
	if imageID == "" && csiID == "" {
		writeJSON(w, 400, map[string]any{"detail": "需提供 image_id 或 cloud_server_image_id"})
		return
	}
	var content string
	if imageID != "" {
		id, _ := strconv.ParseInt(imageID, 10, 64)
		_ = a.DB.SQL.QueryRow(`
SELECT COALESCE(t.content,'') FROM ai_provider_containercloudserverassociation a
JOIN ai_provider_vendorcloudserverimage csi ON csi.id=a.cloud_server_image_id
LEFT JOIN ai_provider_userdatatemplate t ON t.id=csi.userdata_template_id
WHERE a.container_image_id=? AND a.platform_type=? AND a.region=? LIMIT 1`, id, platform, region).Scan(&content)
	} else {
		_ = a.DB.SQL.QueryRow(`
SELECT COALESCE(t.content,'') FROM ai_provider_vendorcloudserverimage csi
LEFT JOIN ai_provider_userdatatemplate t ON t.id=csi.userdata_template_id
WHERE csi.image_id=? AND csi.platform_type=? AND csi.region=? LIMIT 1`, csiID, platform, region).Scan(&content)
	}
	writeJSON(w, 200, map[string]any{"content": content, "platform_type": platform, "region": region})
}

func (a *App) handleVendorDevCatalog(w http.ResponseWriter, r *http.Request) {
	uid, _ := strconv.ParseInt(r.URL.Query().Get("saas_user_id"), 10, 64)
	items, err := a.DB.VendorDevelopmentCatalog(uid)
	if err != nil {
		writeJSON(w, 200, []any{})
		return
	}
	a.scheduleMissingCatalogExtracts(r.Context(), items)
	writeJSON(w, 200, items)
}

func (a *App) handleUnsubmittedImage(w http.ResponseWriter, r *http.Request) {
	vendorIDStr := strings.TrimSpace(r.URL.Query().Get("vendor_id"))
	containerIDStr := strings.TrimSpace(r.URL.Query().Get("container_id"))
	if vendorIDStr != "" && containerIDStr != "" {
		vendorID, err1 := strconv.ParseInt(vendorIDStr, 10, 64)
		containerID, err2 := strconv.ParseInt(containerIDStr, 10, 64)
		if err1 != nil || err2 != nil || vendorID == 0 || containerID == 0 {
			writeJSON(w, 400, map[string]any{"detail": "vendor_id 或 container_id 无效"})
			return
		}
		item, err := a.DB.UnsubmittedImageByIDs(vendorID, containerID)
		if err != nil || item == nil {
			writeJSON(w, 404, map[string]any{"detail": "未找到符合条件的镜像（需为草稿、待审核或已驳回且未上架）"})
			return
		}
		a.scheduleCatalogItemExtract(r.Context(), item)
		writeJSON(w, 200, item)
		return
	}
	uid, _ := strconv.ParseInt(r.URL.Query().Get("saas_user_id"), 10, 64)
	item, _ := a.DB.UnsubmittedImage(uid)
	a.scheduleCatalogItemExtract(r.Context(), item)
	writeJSON(w, 200, item)
}

func (a *App) handleAdminVendors(w http.ResponseWriter, r *http.Request) {
	// OPT-20260806-065：PATCH/POST /admin-vendors/{id}/ 为运营审核操作（approve/reject）
	if r.Method == http.MethodPatch || r.Method == http.MethodPost {
		a.handleAdminVendorReview(w, r)
		return
	}
	// GET .../admin-vendors/{id}/documents/{kind}/ — 运营下载证照
	if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/documents/") {
		a.handleAdminVendorDocument(w, r)
		return
	}
	if _, ok := a.requireStaff(w, r); !ok {
		return
	}
	if id, ok := parsePathID(r.URL.Path, "/api/admin/vendors/"); ok && r.Method == http.MethodGet {
		items, err := a.DB.ListVendors()
		if err != nil {
			writeJSON(w, 500, map[string]any{"detail": err.Error()})
			return
		}
		want := infrastructure.IDStr(id)
		for _, it := range items {
			if it["id"] == want {
				writeJSON(w, 200, it)
				return
			}
		}
		writeJSON(w, 404, map[string]any{"detail": "not found"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]any{"detail": "method not allowed"})
		return
	}
	items, err := a.DB.ListVendors()
	if err != nil {
		writeJSON(w, 500, map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, 200, items)
}

func (a *App) handleUserdataVerify(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/admin/cloud-server-images/userdata-verify/")
	secret := strings.Trim(rest, "/")
	if secret == "" {
		writeJSON(w, 404, map[string]any{"detail": "not found"})
		return
	}
	res, err := a.DB.SQL.Exec(`UPDATE ai_provider_vendorcloudserverimageuserdata SET userdata_run_verified=CURRENT_TIMESTAMP, verification_secret=NULL WHERE verification_secret=?`, secret)
	if err != nil {
		writeJSON(w, 500, map[string]any{"detail": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		writeJSON(w, 404, map[string]any{"detail": "invalid or used secret"})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
