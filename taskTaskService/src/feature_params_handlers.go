package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

var featureParamsSources = map[string]struct{}{
	"company": {}, "workspace": {}, "personal": {}, "none": {},
}

func handleFeatureParams(w http.ResponseWriter, r *http.Request) {
	taskID := r.Header.Get("X-Task-Id")
	if taskID == "" {
		writeError(w, r, http.StatusBadRequest, "task id required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleGetFeatureParams(w, taskID)
	case http.MethodPatch, http.MethodPut:
		handlePatchFeatureParams(w, r, taskID)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleFeatureParamsSnapshots(w http.ResponseWriter, r *http.Request) {
	taskID := r.Header.Get("X-Task-Id")
	if taskID == "" {
		writeError(w, r, http.StatusBadRequest, "task id required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleListFeatureParamsSnapshots(w, taskID)
	case http.MethodPost:
		handleCreateFeatureParamsSnapshot(w, r, taskID)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleGetFeatureParams(w http.ResponseWriter, taskID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows {
		writeError(w, nil, http.StatusNotFound, "not found")
		return
	}
	var params string
	err = db.QueryRow(`SELECT COALESCE(params,'{}') FROM task_feature_params WHERE task_id=?`, taskID).Scan(&params)
	if err == sql.ErrNoRows {
		params = "{}"
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"task_id":                           taskID,
		"feature_params_source":             t.FeatureParamsSource,
		"personal_feature_params_config_id": nilIfEmpty(t.PersonalFeatureParamsConfigID),
		"params":                            jsonRaw(params),
	})
}

func handlePatchFeatureParams(w http.ResponseWriter, r *http.Request, taskID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	source := strField(body, "feature_params_source")
	if source != "" {
		if _, ok := featureParamsSources[source]; !ok {
			writeError(w, r, http.StatusBadRequest, "invalid feature_params_source")
			return
		}
		t.FeatureParamsSource = source
	}
	pfpc := t.PersonalFeatureParamsConfigID
	if _, ok := body["personal_feature_params_config_id"]; ok {
		pfpc = strField(body, "personal_feature_params_config_id")
	}
	db.Exec(`UPDATE task_tasks SET feature_params_source=?, personal_feature_params_config_id=?, updated_at=? WHERE id=?`,
		t.FeatureParamsSource, pfpc, time.Now().UTC(), taskID)
	if rawParams, ok := body["params"]; ok {
		b, _ := json.Marshal(rawParams)
		db.Exec(`INSERT INTO task_feature_params(id,task_id,params,updated_at) VALUES(?,?,?,?)
			ON DUPLICATE KEY UPDATE params=VALUES(params), updated_at=VALUES(updated_at)`,
			genID("tfp"), taskID, string(b), time.Now().UTC())
	}
	handleGetFeatureParams(w, taskID)
}

func handleListFeatureParamsSnapshots(w http.ResponseWriter, taskID string) {
	if _, err := loadTask(taskID); err == sql.ErrNoRows {
		writeError(w, nil, http.StatusNotFound, "not found")
		return
	}
	rows, _ := db.Query(`SELECT id,COALESCE(params,'{}'),created_at FROM task_feature_params_snapshots WHERE task_id=? ORDER BY created_at DESC`, taskID)
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, params string
		var ca time.Time
		rows.Scan(&id, &params, &ca)
		out = append(out, map[string]interface{}{
			"id": id, "params": jsonRaw(params), "created_at": ca.UTC().Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func handleCreateFeatureParamsSnapshot(w http.ResponseWriter, r *http.Request, taskID string) {
	if _, err := loadTask(taskID); err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "not found")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	params := "{}"
	if raw, ok := body["params"]; ok {
		b, _ := json.Marshal(raw)
		params = string(b)
	}
	id := genID("snap")
	now := time.Now().UTC()
	db.Exec(`INSERT INTO task_feature_params_snapshots(id,task_id,params,created_at) VALUES(?,?,?,?)`, id, taskID, params, now)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id": id, "params": jsonRaw(params), "created_at": now.Format(time.RFC3339Nano),
	})
}

func jsonRaw(s string) json.RawMessage {
	if s == "" {
		s = "{}"
	}
	return json.RawMessage(s)
}
