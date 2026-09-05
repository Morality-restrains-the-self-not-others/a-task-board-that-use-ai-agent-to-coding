package main

import (
	"net/http"
	"strings"
)

func handleInternalTenantInstalledImages(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	rawRows, _ := body["installed_images"].([]interface{})
	rows := make([]map[string]interface{}, 0, len(rawRows))
	for _, item := range rawRows {
		if m, ok := item.(map[string]interface{}); ok {
			rows = append(rows, m)
		}
	}
	count, err := importInstalledImages(rows)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":       true,
		"imported": count,
	})
}

func writeInstalledImageLookup(w http.ResponseWriter, r *http.Request, img *TenantInstalledImage, err error) bool {
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return true
	}
	if img == nil {
		return false
	}
	writeJSON(w, http.StatusOK, installedImageToJSON(*img))
	return true
}

func handleInternalTenantInstalledImagesLookup(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	imageID := strings.TrimSpace(r.URL.Query().Get("id"))
	imageName := strings.TrimSpace(r.URL.Query().Get("name"))
	if tenantID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "tenant_id required")
		return
	}
	if imageID != "" {
		img, err := getInstalledImage(tenantID, imageID)
		if writeInstalledImageLookup(w, r, img, err) {
			return
		}
		img, err = getInstalledImageByExternalID(tenantID, imageID)
		if writeInstalledImageLookup(w, r, img, err) {
			return
		}
		if imageName != "" {
			img, err = getInstalledImageByUniqueName(tenantID, imageName)
			if img != nil {
				logWarn("installed image lookup rebound by unique name", r.Header.Get("X-Trace-Id"))
			}
			if writeInstalledImageLookup(w, r, img, err) {
				return
			}
		}
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
		return
	}
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
}
