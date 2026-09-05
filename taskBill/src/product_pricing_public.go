package main

import (
	"database/sql"
	"net/http"
	"tracelog"
)

// handlePublicProductPricing returns legacy-format public pricing data.
// OPT-049: Migrated from Django /api/public/product-pricing/ (retired 2026-07-30).
// Reads from billing_unit table — prices are in cents (分), same as the frontend's
// "points" fields which get divided by 100 client-side to display yuan.
func handlePublicProductPricing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	var taskPoints, gitlabDisk, gitlabTraffic sql.NullInt64

	// Task post price (server_start)
	err := db.QueryRow(`SELECT price FROM billing_unit WHERE unit_type = 'server_start'`).Scan(&taskPoints)
	if err != nil && err != sql.ErrNoRows {
		writeErrorJSON(w, http.StatusInternalServerError, "db error", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// GitLab disk price
	err = db.QueryRow(`SELECT price FROM billing_unit WHERE unit_type = 'gitlab_disk'`).Scan(&gitlabDisk)
	if err != nil && err != sql.ErrNoRows {
		writeErrorJSON(w, http.StatusInternalServerError, "db error", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// GitLab traffic price
	err = db.QueryRow(`SELECT price FROM billing_unit WHERE unit_type = 'gitlab_traffic'`).Scan(&gitlabTraffic)
	if err != nil && err != sql.ErrNoRows {
		writeErrorJSON(w, http.StatusInternalServerError, "db error", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	resp := map[string]interface{}{
		"status": "success",
	}
	if taskPoints.Valid {
		resp["task_points"] = taskPoints.Int64
	} else {
		resp["task_points"] = DefaultTaskPostUnitPriceCents
	}
	// 续存只扣创建帖配额，不另收续存费
	resp["task_renewal_points"] = int64(0)
	resp["task_renewal_points_per_month"] = int64(0)
	if gitlabDisk.Valid {
		resp["gitlab_disk_points_per_gb_per_month"] = gitlabDisk.Int64
	} else {
		resp["gitlab_disk_points_per_gb_per_month"] = nil
	}
	if gitlabTraffic.Valid {
		resp["gitlab_traffic_points_per_gb"] = gitlabTraffic.Int64
	} else {
		resp["gitlab_traffic_points_per_gb"] = nil
	}

	writeJSON(w, http.StatusOK, resp)
}
