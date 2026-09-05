package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	maxWorkPanelFilterBars = 20
	workPanelFilterVersion = 2
)

type pathSegment struct {
	Type  string `json:"type"`
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
}

type filterBar struct {
	ID   string        `json:"id"`
	Path []pathSegment `json:"path"`
}

// accessFilterPref is the persisted person/group selection (memberIds recomputed client-side).
type accessFilterPref struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

type filterPayload struct {
	Version               int               `json:"version"`
	DeliverableFilterBars []filterBar       `json:"deliverable_filter_bars"`
	AccessFilter          *accessFilterPref `json:"access_filter"`
}

func defaultFilterPayload() filterPayload {
	return filterPayload{
		Version: workPanelFilterVersion,
		DeliverableFilterBars: []filterBar{
			{ID: "bar-0", Path: []pathSegment{{Type: "root"}}},
		},
		AccessFilter: nil,
	}
}

func normalizeAccessFilterPref(raw *accessFilterPref) *accessFilterPref {
	if raw == nil {
		return nil
	}
	kind := strings.TrimSpace(strings.ToLower(raw.Kind))
	id := strings.TrimSpace(raw.ID)
	if (kind != "person" && kind != "group") || id == "" {
		return nil
	}
	return &accessFilterPref{
		Kind:  kind,
		ID:    id,
		Label: strings.TrimSpace(raw.Label),
	}
}

func normalizeFilterPayload(raw filterPayload) (filterPayload, error) {
	out := filterPayload{Version: workPanelFilterVersion, AccessFilter: normalizeAccessFilterPref(raw.AccessFilter)}
	bars := raw.DeliverableFilterBars
	if len(bars) == 0 {
		def := defaultFilterPayload()
		def.AccessFilter = out.AccessFilter
		return def, nil
	}
	if len(bars) > maxWorkPanelFilterBars {
		return filterPayload{}, fmt.Errorf("deliverable_filter_bars exceeds max %d", maxWorkPanelFilterBars)
	}
	for i, bar := range bars {
		id := strings.TrimSpace(bar.ID)
		if id == "" {
			id = fmt.Sprintf("bar-%d", i)
		}
		path, err := normalizeFilterPath(bar.Path)
		if err != nil {
			return filterPayload{}, fmt.Errorf("bar %s: %w", id, err)
		}
		out.DeliverableFilterBars = append(out.DeliverableFilterBars, filterBar{ID: id, Path: path})
	}
	if raw.Version > 0 {
		out.Version = raw.Version
	}
	return out, nil
}

func normalizeFilterPath(path []pathSegment) ([]pathSegment, error) {
	if len(path) == 0 {
		return []pathSegment{{Type: "root"}}, nil
	}
	out := make([]pathSegment, 0, len(path))
	for _, seg := range path {
		typ := strings.TrimSpace(strings.ToLower(seg.Type))
		switch typ {
		case "root":
			out = append(out, pathSegment{Type: "root"})
		case "category", "task":
			id := strings.TrimSpace(seg.ID)
			if id == "" {
				return nil, fmt.Errorf("segment type %s requires id", typ)
			}
			out = append(out, pathSegment{
				Type:  typ,
				ID:    id,
				Label: strings.TrimSpace(seg.Label),
			})
		default:
			return nil, fmt.Errorf("unknown segment type %q", seg.Type)
		}
	}
	return out, nil
}

func loadWorkPanelFilterPayload(userID, companyID, workspaceID string) (filterPayload, string, error) {
	var raw, updated string
	err := db.QueryRow(
		`SELECT payload_json, updated_at FROM project_user_workspace_work_panel_filters
		 WHERE user_id=? AND company_id=? AND workspace_id=?`,
		userID, companyID, workspaceID,
	).Scan(&raw, &updated)
	if err == sql.ErrNoRows {
		return defaultFilterPayload(), "", nil
	}
	if err != nil {
		return filterPayload{}, "", err
	}
	var p filterPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return defaultFilterPayload(), updated, nil
	}
	norm, err := normalizeFilterPayload(p)
	if err != nil {
		return defaultFilterPayload(), updated, nil
	}
	return norm, updated, nil
}

func upsertWorkPanelFilterPayload(userID, companyID, workspaceID string, p filterPayload) (string, error) {
	now := time.Now().UTC()
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	_, err = db.Exec(
		`INSERT INTO project_user_workspace_work_panel_filters
			(user_id, company_id, workspace_id, payload_json, updated_at)
		 VALUES (?,?,?,?,?)
		 ON DUPLICATE KEY UPDATE
			payload_json=VALUES(payload_json),
			updated_at=VALUES(updated_at)`,
		userID, companyID, workspaceID, string(b), now,
	)
	if err != nil {
		return "", err
	}
	return now.Format(time.RFC3339), nil
}

func filterPayloadResponse(p filterPayload, updatedAt string) map[string]interface{} {
	out := map[string]interface{}{
		"status":                  "success",
		"version":                 p.Version,
		"deliverable_filter_bars": p.DeliverableFilterBars,
		"access_filter":           p.AccessFilter,
	}
	if updatedAt != "" {
		out["updated_at"] = updatedAt
	}
	return out
}

func handleWorkPanelFilters(w http.ResponseWriter, r *http.Request, tenantID, workspaceID string) {
	traceID := r.Header.Get("X-Trace-Id")
	userID := strings.TrimSpace(getAuthUser(r))
	if userID == "" {
		logWarn(fmt.Sprintf("work-panel-filters unauthorized tenant=%s workspace=%s", tenantID, workspaceID), traceID)
		writeError(w, r, http.StatusUnauthorized, "missing user")
		return
	}
	workspaceID = strings.TrimSpace(workspaceID)
	tenantID = strings.TrimSpace(tenantID)
	if workspaceID == "" || tenantID == "" {
		writeError(w, r, http.StatusBadRequest, "tenant_id and workspace_id required")
		return
	}
	if err := verifyWorkspaceInTenant(workspaceID, tenantID); err != nil {
		logWarn(fmt.Sprintf("work-panel-filters workspace missing tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
		writeError(w, r, http.StatusNotFound, "workspace not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		p, updated, err := loadWorkPanelFilterPayload(userID, tenantID, workspaceID)
		if err != nil {
			logError(fmt.Sprintf("work-panel-filters GET error tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusInternalServerError, "load failed")
			return
		}
		afKind := ""
		if p.AccessFilter != nil {
			afKind = p.AccessFilter.Kind
		}
		logInfo(fmt.Sprintf("work-panel-filters GET ok tenant=%s workspace=%s user=%s bars_count=%d access_kind=%s", tenantID, workspaceID, userID, len(p.DeliverableFilterBars), afKind), traceID)
		writeJSON(w, http.StatusOK, filterPayloadResponse(p, updated))
	case http.MethodPut:
		body, err := readJSONBody(r)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid JSON")
			return
		}
		rawBytes, _ := json.Marshal(body)
		var raw filterPayload
		if err := json.Unmarshal(rawBytes, &raw); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid payload")
			return
		}
		norm, err := normalizeFilterPayload(raw)
		if err != nil {
			logWarn(fmt.Sprintf("work-panel-filters PUT validate fail tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		updated, err := upsertWorkPanelFilterPayload(userID, tenantID, workspaceID, norm)
		if err != nil {
			logError(fmt.Sprintf("work-panel-filters PUT error tenant=%s workspace=%s user=%s err=%v", tenantID, workspaceID, userID, err), traceID)
			writeError(w, r, http.StatusInternalServerError, "save failed")
			return
		}
		afKind := ""
		if norm.AccessFilter != nil {
			afKind = norm.AccessFilter.Kind
		}
		logInfo(fmt.Sprintf("work-panel-filters PUT ok tenant=%s workspace=%s user=%s bars_count=%d access_kind=%s", tenantID, workspaceID, userID, len(norm.DeliverableFilterBars), afKind), traceID)
		writeJSON(w, http.StatusOK, filterPayloadResponse(norm, updated))
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}
