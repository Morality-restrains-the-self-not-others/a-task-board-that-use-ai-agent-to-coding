package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"tracelog"
)

// handleSystemAdminGitlabRegions is the public system-admin endpoint for region capacity management.
// Authentication is handled by the API gateway (requires system-admin role).
func handleSystemAdminGitlabRegions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimRight(r.URL.Path, "/")
	// /api/system_admin/gitlab-regions/{slug}/capacity/ → update capacity
	// /api/system_admin/gitlab-regions/ → list regions with capacity
	parts := strings.Split(strings.Trim(path, "/"), "/")
	// parts: ["api", "system_admin", "gitlab-regions", ...]
	var regionSlug string
	for i, p := range parts {
		if p == "gitlab-regions" && i+1 < len(parts) {
			regionSlug = parts[i+1]
			break
		}
	}

	if regionSlug != "" && regionSlug != "capacity" {
		// Sub-path like /gitlab-regions/{slug}/capacity/
		if strings.HasSuffix(path, "/capacity") {
			handleSystemAdminUpdateRegionCapacity(w, r, regionSlug)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		if regionSlug != "" {
			handleSystemAdminGetRegionBySlug(w, r, regionSlug)
		} else {
			handleSystemAdminListRegionsFull(w, r)
		}
	case http.MethodPost:
		handleSystemAdminCreateRegion(w, r)
	case http.MethodPut:
		if regionSlug != "" {
			handleSystemAdminUpdateRegionMetadata(w, r, regionSlug)
		} else {
			writeErrorJSON(w, http.StatusBadRequest, "region slug required for PUT", tracelog.TraceIDFromContext(r.Context()))
		}
	case http.MethodDelete:
		if regionSlug != "" {
			handleSystemAdminDeleteRegion(w, r, regionSlug)
		} else {
			writeErrorJSON(w, http.StatusBadRequest, "region slug required for DELETE", tracelog.TraceIDFromContext(r.Context()))
		}
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

// handleSystemAdminGetRegionBySlug returns a single region by slug.
func handleSystemAdminGetRegionBySlug(w http.ResponseWriter, r *http.Request, slug string) {
	region, err := getAnyGitlabRegionBySlug(slug)
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	view := gitlabRegionCapacityView(region)
	view["region"] = region
	writeJSON(w, http.StatusOK, view)
}

// handleSystemAdminCreateRegion creates a new GitLab region.
func handleSystemAdminCreateRegion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid json", "trace_id": tracelog.TraceIDFromContext(ctx)})
		return
	}
	name := strings.TrimSpace(stringField(body, "name"))
	slug := strings.TrimSpace(stringField(body, "slug"))
	if name == "" || slug == "" {
		writeErrorJSON(w, http.StatusBadRequest, "name and slug are required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	desc := strings.TrimSpace(stringField(body, "description"))
	apiBase := strings.TrimSpace(stringField(body, "gitlab_api_base"))
	if apiBase == "" {
		apiBase = "http://127.0.0.1:8012"
	}
	webURL := strings.TrimSpace(stringField(body, "gitlab_web_url"))
	if webURL == "" {
		webURL = "https://gitlab.daydaymoney.com"
	}
	token := strings.TrimSpace(stringField(body, "admin_private_token"))
	cloudProvider := strings.TrimSpace(stringField(body, "cloud_provider"))
	isActive := int64(1)
	if v, ok := body["is_active"]; ok && v != nil {
		if b, ok2 := v.(bool); ok2 && !b {
			isActive = 0
		}
	}
	sortOrder := int64(0)
	if v, ok := body["sort_order"]; ok && v != nil {
		if n, err2 := parseNonNegInt64(v, "sort_order"); err2 == nil {
			sortOrder = n
		}
	}
	disk := int64(0)
	if v, ok := body["total_disk_gb"]; ok && v != nil {
		if n, err2 := parseNonNegInt64(v, "total_disk_gb"); err2 == nil {
			disk = n
		}
	}
	traffic := int64(0)
	if v, ok := body["total_traffic_gb"]; ok && v != nil {
		if n, err2 := parseNonNegInt64(v, "total_traffic_gb"); err2 == nil {
			traffic = n
		}
	}
	totalBw := int64(0)
	if v, ok := body["total_bandwidth_mbps"]; ok && v != nil {
		if n, err2 := parseNonNegInt64(v, "total_bandwidth_mbps"); err2 == nil {
			totalBw = n
		}
	}
	remainBw := totalBw
	if v, ok := body["remaining_bandwidth_mbps"]; ok && v != nil {
		if n, err2 := parseNonNegInt64(v, "remaining_bandwidth_mbps"); err2 == nil {
			remainBw = n
		}
	}
	remainBw = clampRemainingBandwidth(totalBw, remainBw)
	bwShared, _ := parseOptionalBool(body, "bandwidth_shared")
	accessMode, errMode := ParseGitlabRegionAccessMode(stringField(body, "access_mode"))
	if errMode != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": errMode.Error(), "trace_id": tracelog.TraceIDFromContext(ctx)})
		return
	}
	infraStatus, errInfra := ParseGitlabRegionInfraStatus(stringField(body, "infra_status"))
	if errInfra != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": errInfra.Error(), "trace_id": tracelog.TraceIDFromContext(ctx)})
		return
	}
	id := generateSnowflakeID()
	now := utcNow()
	_, err = db.Exec(`
		INSERT INTO billing_gitlab_region (id, name, slug, description, gitlab_api_base, gitlab_web_url,
			admin_private_token, cloud_provider, is_active, sort_order, total_disk_gb, total_traffic_gb,
			allocated_disk_gb, allocated_traffic_gb, bandwidth_shared, total_bandwidth_mbps, remaining_bandwidth_mbps,
			access_mode, infra_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, slug, desc, apiBase, webURL, token, cloudProvider, isActive, sortOrder, disk, traffic,
		bwShared, totalBw, remainBw, accessMode, infraStatus, now, now)
	if err != nil {
		slog.ErrorContext(ctx, "region_create_failed", "level", "error", "error", err.Error(), "slug", slug)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error(), "trace_id": tracelog.TraceIDFromContext(ctx)})
		return
	}
	slog.InfoContext(ctx, "region_created", "level", "info", "name", name, "slug", slug, "access_mode", accessMode)
	_ = publishEvent(ctx, "GitlabRegionAccessModeChanged", map[string]interface{}{
		"region_slug": slug,
		"from":        "",
		"to":          accessMode,
	}, slug)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id": id, "name": name, "slug": slug, "created": true, "access_mode": accessMode,
		"trace_id": tracelog.TraceIDFromContext(ctx),
	})
}

// handleSystemAdminUpdateRegionMetadata updates a region's metadata fields.
func handleSystemAdminUpdateRegionMetadata(w http.ResponseWriter, r *http.Request, regionSlug string) {
	ctx := r.Context()
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid json", "trace_id": tracelog.TraceIDFromContext(ctx)})
		return
	}
	setClauses := []string{}
	args := []interface{}{}

	if _, ok := body["name"]; ok {
		args = append(args, strings.TrimSpace(stringField(body, "name")))
		setClauses = append(setClauses, "name = ?")
	}
	if _, ok := body["description"]; ok {
		args = append(args, strings.TrimSpace(stringField(body, "description")))
		setClauses = append(setClauses, "description = ?")
	}
	if _, ok := body["gitlab_api_base"]; ok {
		args = append(args, strings.TrimSpace(stringField(body, "gitlab_api_base")))
		setClauses = append(setClauses, "gitlab_api_base = ?")
	}
	if _, ok := body["gitlab_web_url"]; ok {
		args = append(args, strings.TrimSpace(stringField(body, "gitlab_web_url")))
		setClauses = append(setClauses, "gitlab_web_url = ?")
	}
	if _, ok := body["admin_private_token"]; ok {
		args = append(args, strings.TrimSpace(stringField(body, "admin_private_token")))
		setClauses = append(setClauses, "admin_private_token = ?")
	}
	if _, ok := body["cloud_provider"]; ok {
		args = append(args, strings.TrimSpace(stringField(body, "cloud_provider")))
		setClauses = append(setClauses, "cloud_provider = ?")
	}
	if v, ok := body["is_active"]; ok && v != nil {
		active := int64(1)
		if b, ok2 := v.(bool); ok2 && !b {
			active = 0
		}
		args = append(args, active)
		setClauses = append(setClauses, "is_active = ?")
	}
	if v, ok := body["sort_order"]; ok && v != nil {
		if n, err2 := parseNonNegInt64(v, "sort_order"); err2 == nil {
			args = append(args, n)
			setClauses = append(setClauses, "sort_order = ?")
		}
	}
	if v, ok := body["total_disk_gb"]; ok && v != nil {
		if n, err2 := parseNonNegInt64(v, "total_disk_gb"); err2 == nil {
			args = append(args, n)
			setClauses = append(setClauses, "total_disk_gb = ?")
		}
	}
	if v, ok := body["total_traffic_gb"]; ok && v != nil {
		if n, err2 := parseNonNegInt64(v, "total_traffic_gb"); err2 == nil {
			args = append(args, n)
			setClauses = append(setClauses, "total_traffic_gb = ?")
		}
	}
	if v, ok := body["total_bandwidth_mbps"]; ok && v != nil {
		if n, err2 := parseNonNegInt64(v, "total_bandwidth_mbps"); err2 == nil {
			args = append(args, n)
			setClauses = append(setClauses, "total_bandwidth_mbps = ?")
		}
	}
	if v, ok := body["remaining_bandwidth_mbps"]; ok && v != nil {
		if n, err2 := parseNonNegInt64(v, "remaining_bandwidth_mbps"); err2 == nil {
			args = append(args, n)
			setClauses = append(setClauses, "remaining_bandwidth_mbps = ?")
		}
	}
	if n, ok := parseOptionalBool(body, "bandwidth_shared"); ok {
		args = append(args, n)
		setClauses = append(setClauses, "bandwidth_shared = ?")
	}
	var nextAccessMode string
	if _, ok := body["access_mode"]; ok {
		mode, errMode := ParseGitlabRegionAccessMode(stringField(body, "access_mode"))
		if errMode != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": errMode.Error(), "trace_id": tracelog.TraceIDFromContext(ctx)})
			return
		}
		nextAccessMode = mode
		args = append(args, mode)
		setClauses = append(setClauses, "access_mode = ?")
	}
	var nextInfraStatus string
	if _, ok := body["infra_status"]; ok {
		st, errSt := ParseGitlabRegionInfraStatus(stringField(body, "infra_status"))
		if errSt != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": errSt.Error(), "trace_id": tracelog.TraceIDFromContext(ctx)})
			return
		}
		nextInfraStatus = st
		args = append(args, st)
		setClauses = append(setClauses, "infra_status = ?")
	}

	if len(setClauses) == 0 {
		writeErrorJSON(w, http.StatusBadRequest, "no fields to update", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	prev, _ := getAnyGitlabRegionBySlug(regionSlug)
	args = append(args, regionSlug)
	_, err = db.Exec(fmt.Sprintf("UPDATE billing_gitlab_region SET %s, updated_at = NOW() WHERE slug = ?",
		strings.Join(setClauses, ", ")), args...)
	if err != nil {
		slog.ErrorContext(ctx, "region_update_failed", "level", "error", "error", err.Error(), "slug", regionSlug)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error(), "trace_id": tracelog.TraceIDFromContext(ctx)})
		return
	}
	if nextAccessMode != "" {
		from := gitlabRegionAccessRelease
		if prev != nil {
			from = NormalizeGitlabRegionAccessMode(prev.AccessMode)
		}
		if from != nextAccessMode {
			slog.InfoContext(ctx, "region_access_mode_changed",
				"level", "info", "slug", regionSlug, "from", from, "to", nextAccessMode)
			_ = publishEvent(ctx, "GitlabRegionAccessModeChanged", map[string]interface{}{
				"region_slug": regionSlug,
				"from":        from,
				"to":          nextAccessMode,
			}, regionSlug)
		}
	}
	if nextInfraStatus == gitlabRegionInfraReady {
		from := gitlabRegionInfraReady
		if prev != nil {
			from = NormalizeGitlabRegionInfraStatus(prev.InfraStatus)
		}
		if from != gitlabRegionInfraReady {
			slog.InfoContext(ctx, "gitlab_region_infra_marked_ready",
				"level", "info", "slug", regionSlug, "from", from)
			_ = gitlabRegionEventPublisher(ctx, "GitlabRegionInfraMarkedReady", map[string]interface{}{
				"region_slug": regionSlug,
				"from":        from,
				"to":          gitlabRegionInfraReady,
			}, regionSlug)
		}
	}
	slog.InfoContext(ctx, "region_updated", "level", "info", "slug", regionSlug)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"updated": true, "region": regionSlug,
		"trace_id": tracelog.TraceIDFromContext(ctx),
	})
}

// handleSystemAdminDeleteRegion soft-deletes (deactivates) a region.
func handleSystemAdminDeleteRegion(w http.ResponseWriter, r *http.Request, regionSlug string) {
	ctx := r.Context()
	_, err := db.Exec(`UPDATE billing_gitlab_region SET is_active = 0, updated_at = NOW() WHERE slug = ?`, regionSlug)
	if err != nil {
		slog.ErrorContext(ctx, "region_delete_failed", "level", "error", "error", err.Error(), "slug", regionSlug)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"error": err.Error(), "trace_id": tracelog.TraceIDFromContext(ctx)})
		return
	}
	slog.InfoContext(ctx, "region_deactivated", "level", "info", "slug", regionSlug)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deleted": true, "region": regionSlug, "note": "region deactivated",
		"trace_id": tracelog.TraceIDFromContext(ctx),
	})
}

func handleSystemAdminUpdateRegionCapacity(w http.ResponseWriter, r *http.Request, regionSlug string) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if regionSlug == "" {
		writeErrorJSON(w, http.StatusBadRequest, "region required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	ctx := r.Context()
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "invalid json", "trace_id": tracelog.TraceIDFromContext(ctx),
		})
		return
	}
	region, err := applyGitlabRegionCapacity(ctx, regionSlug, body)
	if err != nil {
		writeJSON(w, capacityUpdateHTTPStatus(err), map[string]interface{}{
			"error": err.Error(), "trace_id": tracelog.TraceIDFromContext(ctx),
		})
		return
	}
	view := gitlabRegionCapacityView(region)
	view["trace_id"] = tracelog.TraceIDFromContext(ctx)
	writeJSON(w, http.StatusOK, view)
}

// handleAdminUpdateRegionCapacity allows system admins to set a region's total capacity.
// (internal service-to-service variant — requires internal secret)
func handleAdminUpdateRegionCapacity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"detail":   "internal secret required",
			"trace_id": tracelog.TraceIDFromContext(r.Context()),
		})
		return
	}
	// Extract region slug from URL path: /api/internal/taskbill/gitlab-regions/{slug}/capacity/
	path := strings.TrimRight(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	var regionSlug string
	for i, p := range parts {
		if p == "gitlab-regions" && i+1 < len(parts) {
			regionSlug = parts[i+1]
			break
		}
	}
	if regionSlug == "" || regionSlug == "capacity" {
		writeErrorJSON(w, http.StatusBadRequest, "region slug required", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	region, err := applyGitlabRegionCapacity(r.Context(), regionSlug, body)
	if err != nil {
		writeErrorJSON(w, capacityUpdateHTTPStatus(err), err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, gitlabRegionCapacityView(region))
}

// handleSystemAdminListRegionsFull returns all regions (including inactive) with capacity
// and deploy-recipe paths (service_process / config_file / data_dir) for SystemAdmin cards.
func handleSystemAdminListRegionsFull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	regions, err := listAllGitlabRegions()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	out := attachDeployInfo(regions)
	writeJSON(w, http.StatusOK, map[string]interface{}{"regions": out, "total": len(out)})
}
