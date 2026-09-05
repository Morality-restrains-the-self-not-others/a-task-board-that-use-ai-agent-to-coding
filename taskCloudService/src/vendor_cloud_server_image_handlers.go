package main

import (
	"net/http"
	"strings"
)

func handleVendorCloudServerImageRoutes(w http.ResponseWriter, r *http.Request) {
	vendorAuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/vendor/cloud-server-images")
		path = strings.Trim(path, "/")

		switch path {
		case "regions":
			handleVendorCloudRegions(w, r)
		case "images":
			handleVendorCloudImages(w, r)
		case "instance-types":
			handleVendorCloudInstanceTypes(w, r)
		default:
			writeErrorMapJSON(w, r, http.StatusNotFound, map[string]interface{}{
				"status":  "error",
				"message": "not found",
			})
		}
	})(w, r)
}

type vendorCloudCredential struct {
	secretID  string
	secretKey string
}

func resolveVendorCloudCredentialForQuery(w http.ResponseWriter, r *http.Request) (platformType string, cred vendorCloudCredential, ok bool) {
	platformType = strings.TrimSpace(r.URL.Query().Get("platform_type"))
	if platformType == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "缺少 platform_type 参数",
		})
		return "", vendorCloudCredential{}, false
	}

	vendorID := getVendorID(r)
	secretID, secretKey, err := loadVendorCloudCredentialSecrets(vendorID, platformType)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "请先在「云平台测试密钥」Tab 配置并测试通过该云平台的 AccessKey",
			"code":    "vendor_cloud_credential_missing",
		})
		return "", vendorCloudCredential{}, false
	}

	if platformType != "aliyun" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "不支持的云平台类型: " + platformType,
		})
		return "", vendorCloudCredential{}, false
	}

	return platformType, vendorCloudCredential{secretID: secretID, secretKey: secretKey}, true
}

func handleVendorCloudRegions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorMapJSON(w, r, http.StatusMethodNotAllowed, map[string]interface{}{
			"status":  "error",
			"message": "method not allowed",
		})
		return
	}

	_, cred, ok := resolveVendorCloudCredentialForQuery(w, r)
	if !ok {
		return
	}

	var regions []map[string]string
	var requestID string
	var err error
	if useInMemoryCloud() {
		regions = mockCloudRegions()
	} else {
		regions, requestID, err = aliyunDescribeRegions(cred.secretID, cred.secretKey)
		if err != nil {
			writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "获取地域列表失败: " + err.Error(),
			})
			return
		}
	}

	setCloudRequestIDHeader(w, requestID)
	regions = filterChinaRegions(formatRegionsIDName(regions))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"regions": regions,
	})
}

func handleVendorCloudImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorMapJSON(w, r, http.StatusMethodNotAllowed, map[string]interface{}{
			"status":  "error",
			"message": "method not allowed",
		})
		return
	}

	_, cred, ok := resolveVendorCloudCredentialForQuery(w, r)
	if !ok {
		return
	}

	regionID := strings.TrimSpace(r.URL.Query().Get("region_id"))
	if regionID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "缺少 region_id 参数",
		})
		return
	}

	var images []map[string]interface{}
	var requestID string
	var err error
	if useInMemoryCloud() {
		images = mockCloudImages()
	} else {
		images, requestID, err = aliyunDescribeImages(cred.secretID, cred.secretKey, regionID)
		if err != nil {
			writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "获取镜像列表失败: " + err.Error(),
			})
			return
		}
	}

	setCloudRequestIDHeader(w, requestID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"images": images,
	})
}

func handleVendorCloudInstanceTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorMapJSON(w, r, http.StatusMethodNotAllowed, map[string]interface{}{
			"status":  "error",
			"message": "method not allowed",
		})
		return
	}

	_, cred, ok := resolveVendorCloudCredentialForQuery(w, r)
	if !ok {
		return
	}

	regionID := strings.TrimSpace(r.URL.Query().Get("region_id"))
	if regionID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "缺少 region_id 参数",
		})
		return
	}

	imageID := strings.TrimSpace(r.URL.Query().Get("image_id"))
	if imageID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "缺少 image_id 参数，请先选定云平台镜像以便按镜像支持与可用资源过滤实例规格",
		})
		return
	}

	zoneID := strings.TrimSpace(r.URL.Query().Get("zone_id"))

	var instanceTypes []map[string]interface{}
	var requestID string
	meta := map[string]interface{}{}
	var err error
	if useInMemoryCloud() {
		instanceTypes, meta = mockVendorInstanceTypes()
	} else {
		instanceTypes, requestID, meta, err = aliyunVendorInstanceTypes(
			cred.secretID, cred.secretKey, regionID, imageID, zoneID,
		)
		if err != nil {
			writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
				"status":  "error",
				"message": "获取实例类型列表失败: " + err.Error(),
			})
			return
		}
	}

	setCloudRequestIDHeader(w, requestID)
	resp := map[string]interface{}{
		"status":          "success",
		"instance_types":  instanceTypes,
	}
	for k, v := range meta {
		resp[k] = v
	}
	writeJSON(w, http.StatusOK, resp)
}
