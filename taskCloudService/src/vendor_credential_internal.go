package main

import (
	"net/http"
)

func handleInternalVendorCloudCredentialLookup(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	vendorID := r.URL.Query().Get("vendor_id")
	platformType := r.URL.Query().Get("platform_type")
	if vendorID == "" || platformType == "" {
		writeErrorJSON(w, r, 400, "vendor_id 与 platform_type 为必填项")
		return
	}
	sid, sk, err := loadVendorCloudCredentialSecrets(vendorID, platformType)
	if err != nil {
		writeErrorMapJSON(w, r, 404, map[string]interface{}{
			"detail": "未配置云平台测试密钥",
			"code":   "vendor_cloud_credential_missing",
		})
		return
	}
	writeJSON(w, 200, map[string]string{
		"secret_id":  sid,
		"secret_key": sk,
		"platform_type": platformType,
		"vendor_id": vendorID,
	})
}
