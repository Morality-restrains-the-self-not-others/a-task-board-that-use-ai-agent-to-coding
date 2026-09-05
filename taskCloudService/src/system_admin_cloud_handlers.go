package main

import (
	"net/http"
	"strings"

	"gatewayauth"
)

// --- System-Admin Cloud Handlers ---
//
// These handlers serve /api/system-admin/cloud/* endpoints previously served by
// Django cloudSystemAdmin. They use vendor cloud credentials (platform-level)
// instead of tenant cloud auth, matching the OPT-037 migration.

// loadSystemAdminCloudSecrets loads the first active vendor credential for the
// given platform type. This matches the Django pattern where system admin views
// read from settings.CLOUD_PLATFORM_AUTHORIZATION.
func loadSystemAdminCloudSecrets(platformType string) (secretID, secretKey string, err error) {
	row := db.QueryRow(`SELECT secret_id,secret_key FROM cloud_vendor_platform_credentials
		WHERE platform_type=? AND is_active=1 LIMIT 1`, platformType)
	if err := row.Scan(&secretID, &secretKey); err != nil {
		return "", "", err
	}
	return secretID, secretKey, nil
}

func handleSystemAdminCloudRoutes(w http.ResponseWriter, r *http.Request) {
	gatewayauth.ApplyGatewayUser(r, cfg.GatewayInternalSecret)

	path := strings.TrimPrefix(r.URL.Path, "/api/system-admin/cloud/")
	path = strings.Trim(path, "/")

	if path == "" || path == "regions" || strings.HasPrefix(path, "regions/") {
		handleSystemAdminCloudRegions(w, r)
		return
	}
	if path == "instance-types" || strings.HasPrefix(path, "instance-types/") {
		handleSystemAdminCloudInstanceTypes(w, r)
		return
	}
	if path == "images" || strings.HasPrefix(path, "images/") {
		handleSystemAdminCloudImages(w, r)
		return
	}
	// OPT-20260806-046: 容器镜像 CRUD + 关联（孤儿页面后端补齐）
	if path == "container-images" || strings.HasPrefix(path, "container-images/") {
		handleSystemAdminContainerImagesRoutes(w, r)
		return
	}
	writeErrorJSON(w, r, 404, "not found")
}

func handleSystemAdminCloudRegions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	platformType := strings.TrimSpace(r.URL.Query().Get("platform_type"))
	if platformType == "" {
		platformType = "aliyun"
	}

	secretID, secretKey, err := loadSystemAdminCloudSecrets(platformType)
	if err != nil {
		cloudQueryError(w, 400, "未找到平台 "+platformType+" 的厂商凭证，请先在厂商云凭证页面配置")
		return
	}

	if useInMemoryCloud() {
		regions := mockCloudRegions()
		writeJSON(w, 200, map[string]interface{}{
			"status":  "success",
			"regions": formatRegionsIDName(regions),
		})
		return
	}

	regions, rid, err := aliyunDescribeRegions(secretID, secretKey)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	setCloudRequestIDHeader(w, rid)
	writeJSON(w, 200, map[string]interface{}{
		"status":  "success",
		"regions": formatRegionsIDName(regions),
	})
}

func handleSystemAdminCloudInstanceTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	platformType := strings.TrimSpace(r.URL.Query().Get("platform_type"))
	if platformType == "" {
		platformType = "aliyun"
	}
	regionID := strings.TrimSpace(r.URL.Query().Get("region_id"))

	secretID, secretKey, err := loadSystemAdminCloudSecrets(platformType)
	if err != nil {
		cloudQueryError(w, 400, "未找到平台 "+platformType+" 的厂商凭证，请先在厂商云凭证页面配置")
		return
	}

	instanceTypes, err := aliyunDescribeInstanceTypes(secretID, secretKey, regionID, nil)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, instanceTypes)
}

func handleSystemAdminCloudImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, 405, "method not allowed")
		return
	}
	platformType := strings.TrimSpace(r.URL.Query().Get("platform_type"))
	if platformType == "" {
		platformType = "aliyun"
	}
	regionID := strings.TrimSpace(r.URL.Query().Get("region_id"))

	secretID, secretKey, err := loadSystemAdminCloudSecrets(platformType)
	if err != nil {
		cloudQueryError(w, 400, "未找到平台 "+platformType+" 的厂商凭证，请先在厂商云凭证页面配置")
		return
	}

	if useInMemoryCloud() {
		writeJSON(w, 200, map[string]interface{}{
			"status": "success",
			"images": []map[string]interface{}{},
		})
		return
	}

	images, rid, err := aliyunDescribeImages(secretID, secretKey, regionID)
	if err != nil {
		cloudQueryError(w, 500, err.Error())
		return
	}
	setCloudRequestIDHeader(w, rid)
	writeJSON(w, 200, map[string]interface{}{
		"status": "success",
		"images": images,
	})
}
