package main

import (
	"net/http"
	"regexp"
	"strings"
)

var (
	exportEnvRe = regexp.MustCompile(`^export\s+([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)
	dashEEnvRe  = regexp.MustCompile(`(?:^|\s)(?:-e|--env)\s+([A-Za-z_][A-Za-z0-9_]*)=([^\s]+)`)
)

// mergeInstalledImageRef builds image_url:version like enrichStartVmPayloadFromInstalledImage.
func mergeInstalledImageRef(img *TenantInstalledImage) string {
	if img == nil {
		return ""
	}
	ref := strings.TrimSpace(img.ImageURL)
	if ref == "" {
		return ""
	}
	version := strings.TrimSpace(img.Version)
	if version != "" && !strings.Contains(ref, ":") && !strings.Contains(ref, "@") {
		ref = ref + ":" + version
	}
	return ref
}

func heuristicExtractEnvFromUserdata(userdataText string) map[string]string {
	out := map[string]string{}
	if userdataText == "" {
		return out
	}
	for _, line := range strings.Split(userdataText, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if m := exportEnvRe.FindStringSubmatch(s); len(m) == 3 {
			key := m[1]
			value := strings.TrimSpace(m[2])
			value = strings.Trim(value, `"'`)
			out[key] = value
		}
		for _, m2 := range dashEEnvRe.FindAllStringSubmatch(s, -1) {
			if len(m2) != 3 {
				continue
			}
			key := m2[1]
			value := strings.TrimSpace(m2[2])
			value = strings.Trim(value, `"'`)
			out[key] = value
		}
	}
	return out
}

// handleInternalImageResolve implements GET /api/internal/image/resolve —
// resolves installed_image_id → image_url:version for relay container start.
// (Migrated from /api/internal/mock-run/resolve-image after mock-run removal;
// used by taskContainerGateway relay start path.)
func handleInternalImageResolve(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]any{"status": "error", "message": "forbidden"})
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"status": "error", "message": "method not allowed"})
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	imageID := strings.TrimSpace(r.URL.Query().Get("installed_image_id"))
	if tenantID == "" || imageID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"status": "error", "message": "tenant_id and installed_image_id required",
		})
		return
	}
	img, err := getInstalledImage(tenantID, imageID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": "error", "message": err.Error()})
		return
	}
	if img == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"status": "error", "message": "所选镜像不存在或不属于当前租户",
		})
		return
	}
	ref := mergeInstalledImageRef(img)
	if ref == "" {
		writeJSON(w, http.StatusConflict, map[string]any{
			"status": "error", "message": "镜像缺少 image_url",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"image":              ref,
		"installed_image_id": img.ID,
		"external_image_id":  img.ExternalImageID,
	})
}
