package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"authz"
	"tracelog"
)

func requireFeedbackPlatformAdmin(w http.ResponseWriter, r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
		return false
	}
	if !authz.IsPlatformStaff(r) {
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
		return false
	}
	return true
}

func handleSystemAdminFeedbackResourceKinds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !requireFeedbackPlatformAdmin(w, r) {
		return
	}
	kinds, err := listFeedbackResourceKinds()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": kinds})
}

func handleSystemAdminFeedbackLinkGroupsRouter(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/api/system-admin/feedback-link-groups")
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		handleSystemAdminFeedbackLinkGroupsCollection(w, r)
		return
	}
	id := strings.TrimSuffix(p, "/")
	handleSystemAdminFeedbackLinkGroupDetail(w, r, id)
}

func handleSystemAdminFeedbackLinkGroupsCollection(w http.ResponseWriter, r *http.Request) {
	if !requireFeedbackPlatformAdmin(w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		groups, err := listFeedbackGroups()
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		out := make([]map[string]interface{}, 0, len(groups))
		for i := range groups {
			out = append(out, feedbackGroupAdminMap(&groups[i]))
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": out})
	case http.MethodPost:
		handleFeedbackGroupWrite(w, r, "", http.StatusCreated, "FEEDBACK_LINK_GROUP_CREATED")
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

func handleSystemAdminFeedbackLinkGroupDetail(w http.ResponseWriter, r *http.Request, id string) {
	if !requireFeedbackPlatformAdmin(w, r) {
		return
	}
	id = strings.TrimSpace(id)
	if id == "" {
		writeErrorJSON(w, http.StatusBadRequest, "missing id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	switch r.Method {
	case http.MethodGet:
		g, err := loadFeedbackGroup(id)
		if err == sql.ErrNoRows {
			writeErrorJSON(w, http.StatusNotFound, "group not found", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, feedbackGroupAdminMap(g))
	case http.MethodPut:
		handleFeedbackGroupWrite(w, r, id, http.StatusOK, "FEEDBACK_LINK_GROUP_UPDATED")
	case http.MethodDelete:
		handleFeedbackGroupDelete(w, r, id)
	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}

func replayFeedbackIdempotency(w http.ResponseWriter, key string) bool {
	if key == "" {
		return false
	}
	code, body, ok, err := loadFeedbackIdempotency(key)
	if err != nil || !ok {
		return false
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if len(body) > 0 && code != http.StatusNoContent {
		_, _ = w.Write(body)
	}
	return true
}

func handleFeedbackGroupWrite(w http.ResponseWriter, r *http.Request, existingID string, successStatus int, eventType string) {
	ikey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if ikey == "" {
		writeErrorJSON(w, http.StatusBadRequest, "Idempotency-Key required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if replayFeedbackIdempotency(w, ikey) {
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid body", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	name, sortOrder, enabled, ths, links, err := parseFeedbackSaveBody(body)
	if err != nil {
		status := http.StatusBadRequest
		writeErrorJSON(w, status, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	g, err := saveFeedbackGroup(existingID, name, sortOrder, enabled, ths, links)
	if err == sql.ErrNoRows {
		writeErrorJSON(w, http.StatusNotFound, "group not found", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	payload := feedbackGroupAdminMap(g)
	_ = saveFeedbackIdempotency(ikey, r.Method, g.ID, successStatus, payload)
	_ = feedbackEventPublisher(r.Context(), eventType, map[string]interface{}{
		"group_id":    g.ID,
		"name":        g.Name,
		"updated_at":  g.UpdatedAt,
		"operator_id": strings.TrimSpace(r.Header.Get(authz.HeaderUserID)),
	}, g.ID+"|"+eventType+"|"+g.UpdatedAt)
	writeJSON(w, successStatus, payload)
}

func handleFeedbackGroupDelete(w http.ResponseWriter, r *http.Request, id string) {
	ikey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if ikey == "" {
		writeErrorJSON(w, http.StatusBadRequest, "Idempotency-Key required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if replayFeedbackIdempotency(w, ikey) {
		return
	}
	g, err := loadFeedbackGroup(id)
	if err == sql.ErrNoRows {
		writeErrorJSON(w, http.StatusNotFound, "group not found", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if err := deleteFeedbackGroup(id); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	empty := map[string]interface{}{}
	_ = saveFeedbackIdempotency(ikey, r.Method, id, http.StatusNoContent, empty)
	_ = feedbackEventPublisher(context.Background(), "FEEDBACK_LINK_GROUP_DELETED", map[string]interface{}{
		"group_id":    id,
		"name":        g.Name,
		"updated_at":  g.UpdatedAt,
		"operator_id": strings.TrimSpace(r.Header.Get(authz.HeaderUserID)),
	}, id+"|FEEDBACK_LINK_GROUP_DELETED|"+g.UpdatedAt)
	w.WriteHeader(http.StatusNoContent)
}

func handleTenantFeedbackLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.RequireRegionView(w, r, feedbackTenantRegionKey, formatID(tid)) {
		return
	}
	snap, err := loadTenantFeedbackConsumption(tid)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	groups, err := listFeedbackGroups()
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	out := make([]map[string]interface{}, 0)
	for i := range groups {
		g := &groups[i]
		if !g.Enabled {
			continue
		}
		ths := make([]FeedbackThreshold, 0, len(g.Thresholds))
		for _, t := range g.Thresholds {
			ths = append(ths, FeedbackThreshold{ResourceKind: t.ResourceKind, MinQuantity: t.MinQuantity})
		}
		if !groupVisible(ths, snap) {
			continue
		}
		out = append(out, feedbackGroupTenantMap(g))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"groups": out})
}
