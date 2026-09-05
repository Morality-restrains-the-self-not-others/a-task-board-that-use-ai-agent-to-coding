package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"snowflake"
	"strings"
	"time"
)

func installedImagesSubPath(r *http.Request, tenantID string) string {
	base := "/api/tenant/" + tenantID + "/installed-images"
	path := strings.TrimSpace(r.URL.Path)
	if path == base {
		return ""
	}
	prefix := base + "/"
	if strings.HasPrefix(path, prefix) {
		return strings.Trim(strings.TrimPrefix(path, prefix), "/")
	}
	return ""
}

func ensureTenantAdmin(w http.ResponseWriter, r *http.Request, tenantID string) bool {
	if !ensureTenantMember(w, r, tenantID) {
		return false
	}
	_, isAdmin, err := resolveUserMember(r, tenantID, effectiveUserID(r))
	if err != nil || !isAdmin {
		writeErrorJSON(w, r, http.StatusForbidden, "需要租户管理员权限")
		return false
	}
	return true
}

// handleInstalledImages 镜像市场租户 API（Go 主路径）
func handleInstalledImages(w http.ResponseWriter, r *http.Request) {
	tenantID := getAuthTenant(r)
	if tenantID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "缺少租户")
		return
	}
	if !ensureTenantMember(w, r, tenantID) {
		return
	}

	sub := strings.Trim(strings.TrimSuffix(installedImagesSubPath(r, tenantID), "/"), "/")
	switch {
	case sub == "catalog":
		handleInstalledImageCatalog(w, r, tenantID)
	case sub == "dev-catalog":
		handleInstalledImageDevCatalog(w, r, tenantID)
	case sub == "resolve-target-architectures":
		handleInstalledImageResolveTargetArchitectures(w, r)
	case sub == "":
		handleInstalledImageCollection(w, r, tenantID)
	default:
		handleInstalledImageDetail(w, r, tenantID, sub)
	}
}

func handleInstalledImageCollection(w http.ResponseWriter, r *http.Request, tenantID string) {
	switch r.Method {
	case http.MethodGet:
		rows, err := listInstalledImages(tenantID)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]map[string]interface{}, 0, len(rows))
		for _, row := range rows {
			out = append(out, installedImageToJSON(row))
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		if !ensureTenantAdmin(w, r, tenantID) {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
			return
		}
		imageData, externalImageID, isDevMode, err := imageDataFromInstallBody(body)
		if err != nil {
			status := http.StatusBadRequest
			msg := err.Error()
			if strings.Contains(msg, "未找到") || strings.Contains(msg, "运行环境") {
				status = http.StatusNotFound
			} else if strings.Contains(msg, "镜像服务") {
				status = http.StatusBadGateway
			}
			writeErrorJSON(w, r, status, msg)
			return
		}
		exists, err := installedImageExists(tenantID, externalImageID)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		if exists {
			writeErrorJSON(w, r, http.StatusBadRequest, "该镜像已安装")
			return
		}
		vendorID, vendorName := vendorInfoFromImage(imageData)
		img := TenantInstalledImage{
			ID:                  snowflake.GenerateIDString(),
			TenantID:            tenantID,
			ExternalImageID:     externalImageID,
			Name:                strField(imageData, "name"),
			Description:         strField(imageData, "description"),
			Version:             strField(imageData, "version"),
			ImageURL:            strField(imageData, "image_url"),
			IconURL:             catalogIconURL(imageData),
			TargetArchitectures: architecturesFromImage(imageData),
			Size:                sizeFromImage(imageData),
			IsDevMode:           isDevMode,
			VendorID:            vendorID,
			VendorName:          vendorName,
			InstalledByID:       effectiveUserID(r),
			InstalledAt:         time.Now(),
			// 目录/厂商镜像的 updated_at 安装时快照，供前端展示镜像更新时间（038 迁移）
			UpdatedAt:                 parseImageUpdatedAt(strField(imageData, "updated_at")),
			AutoRunStepsMd:            strField(imageData, "auto_run_steps_md"),
			AutoRunStepsExtractStatus: strField(imageData, "auto_run_steps_extract_status"),
			AutoRunStepsDigest:        strField(imageData, "auto_run_steps_digest"),
		}
		img.ImageSkillsJSON, img.ImageSkillsExtractStatus, img.ImageSkillsDigest = skillsSnapshotFromCatalog(
			imageData, imageSkillIDSeed(externalImageID, img.ID))
		img.SaasInboundSkillVersion = normalizeInboundSkillVersion(strField(imageData, "saas_inbound_skill_version"))
		if err := createInstalledImage(img); err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		// OPT-20260820-035：快照在 catalog 抽取完成前拷贝导致 auto_run_steps_md/skills 为空时，
		// 从 image_url 再抽一次回填（best-effort，失败不阻断安装）。OPT-20260821-018：
		// 两字段同缺时单次 OCI walk 同时回填，避免重复拉层。
		if err := ensureInstalledImageExtracts(&img, tenantID); err != nil {
			log.Printf("[taskCloudService] event=installed_image_re_extract_failed tenant_id=%s image_id=%s err=%v",
				tenantID, img.ID, err)
		}
		writeJSON(w, http.StatusCreated, installedImageToJSON(img))
	default:
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleInstalledImageDetail(w http.ResponseWriter, r *http.Request, tenantID, sub string) {
	parts := strings.SplitN(sub, "/", 2)
	imageID := parts[0]
	if imageID == "" {
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
		return
	}
	if len(parts) == 2 && strings.Trim(parts[1], "/") == "regions" {
		handleInstalledImageRegions(w, r, tenantID, imageID)
		return
	}
	switch r.Method {
	case http.MethodGet:
		img, err := getInstalledImage(tenantID, imageID)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		if img == nil {
			writeErrorJSON(w, r, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, installedImageToJSON(*img))
	case http.MethodDelete:
		if !ensureTenantAdmin(w, r, tenantID) {
			return
		}
		ok, err := deleteInstalledImage(tenantID, imageID)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErrorJSON(w, r, http.StatusNotFound, "not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodPatch:
		if !ensureTenantAdmin(w, r, tenantID) {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
			return
		}
		templateID := strField(body, "userdata_template_id")
		img, err := patchInstalledImageUserdata(tenantID, imageID, templateID)
		if err != nil {
			writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		if img == nil {
			writeErrorJSON(w, r, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, http.StatusOK, installedImageToJSON(*img))
	default:
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleInstalledImageCatalog(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	catalog, err := fetchAIPublicCatalog()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"detail":   err.Error(),
			"trace_id": r.Header.Get("X-Trace-Id"),
		})
		return
	}
	installedIDs, err := installedExternalIDSet(tenantID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	for _, img := range catalog {
		id := stringifyID(img["id"])
		img["is_installed"] = installedIDs[id]
	}
	writeJSON(w, http.StatusOK, catalog)
}

func handleInstalledImageDevCatalog(w http.ResponseWriter, r *http.Request, tenantID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID := effectiveUserID(r)
	if userID == "" {
		writeErrorJSON(w, r, http.StatusUnauthorized, "未认证")
		return
	}
	devList, err := fetchAIPublicDevCatalog(userID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"detail":   err.Error(),
			"trace_id": r.Header.Get("X-Trace-Id"),
		})
		return
	}
	installedIDs, err := installedExternalIDSet(tenantID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	for _, img := range devList {
		id := stringifyID(img["id"])
		img["is_installed"] = installedIDs[id]
	}
	writeJSON(w, http.StatusOK, devList)
}

func handleInstalledImageRegions(w http.ResponseWriter, r *http.Request, tenantID, imageID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	img, err := getInstalledImage(tenantID, imageID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if img == nil {
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
		return
	}
	envs, err := fetchAIPublicImageRuntimeEnvironments(img.ExternalImageID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusServiceUnavailable, err.Error())
		return
	}
	if len(envs) == 0 {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	platformType := strings.TrimSpace(r.URL.Query().Get("platform_type"))
	seen := map[string]bool{}
	regions := []map[string]string{}
	for _, env := range envs {
		if platformType != "" && strField(env, "platform_type") != platformType {
			continue
		}
		region := strField(env, "region")
		if region == "" || seen[region] {
			continue
		}
		seen[region] = true
		regionName := strField(env, "region_name")
		if regionName == "" {
			regionName = region
		}
		regions = append(regions, map[string]string{
			"region_id":   region,
			"region_name": regionName,
		})
	}
	writeJSON(w, http.StatusOK, regions)
}

func handleInstalledImageResolveTargetArchitectures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	imageURL := strField(body, "image_url")
	if imageURL == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "image_url is required")
		return
	}
	metadata, err := resolveContainerImageMetadata(imageURL)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "镜像地址") || strings.Contains(msg, "缺少仓库") || strings.Contains(msg, "不能为空") || strings.Contains(msg, "无法触及") {
			writeErrorJSON(w, r, http.StatusBadRequest, msg)
			return
		}
		writeErrorJSON(w, r, http.StatusBadGateway, "解析镜像架构失败: "+msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"target_architectures": metadata["target_architectures"],
		"size":                 metadata["size"],
	})
}

func stringifyID(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return t.String()
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", t))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}
