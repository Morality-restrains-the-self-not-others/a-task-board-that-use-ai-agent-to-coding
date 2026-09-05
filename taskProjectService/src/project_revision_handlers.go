package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

func parseRevisionPage(r *http.Request) (limit, offset int) {
	limit = 20
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func handleProjectRevisionRoutes(w http.ResponseWriter, r *http.Request, tenantID, projectID string, rest []string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	detail, err := loadProjectDetail(projectID)
	if err == sql.ErrNoRows || detail == nil {
		writeError(w, r, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if rejectIfProjectNotInTenant(w, r, detail, tenantID) {
		logProjectRevisionAuthDenied(tracelog.TraceIDFromContext(r.Context()), getAuthUser(r), projectID, "tenant_mismatch")
		return
	}
	revID := ""
	if len(rest) > 0 {
		revID = strings.TrimSpace(rest[0])
	}
	if revID == "" {
		handleListProjectRevisions(w, r, tenantID, projectID)
		return
	}
	handleGetProjectRevision(w, r, tenantID, projectID, revID)
}

func handleListProjectRevisions(w http.ResponseWriter, r *http.Request, tenantID, projectID string) {
	limit, offset := parseRevisionPage(r)
	rows, total, err := listProjectRevisions(projectID, tenantID, limit, offset)
	if err != nil {
		log.Printf("[taskProjectService] event=project_revision_list_failed project_id=%s err=%v", projectID, err)
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	results := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		results = append(results, projectRevisionListJSON(row))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func handleGetProjectRevision(w http.ResponseWriter, r *http.Request, tenantID, projectID, revisionID string) {
	row, err := getProjectRevision(projectID, tenantID, revisionID)
	if err == sql.ErrNoRows || row == nil {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		log.Printf("[taskProjectService] event=project_revision_get_failed project_id=%s revision_id=%s err=%v", projectID, revisionID, err)
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, projectRevisionDetailJSON(*row))
}
