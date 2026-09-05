package saas_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// IntentMux is an in-memory saas-backend intent API for handler tests.
type IntentMux struct {
	mu sync.Mutex

	Companies map[string]company // creator_id -> company
	Profiles  map[string]string
	DeliverableTenants map[string]bool
	ProgressTenants    map[string]bool
	Workspaces         map[string]workspace // company_id -> default ws
	PendingEvents      map[string]pendingEvent
	EventStatus        map[int64]string
	EventData          map[int64]map[string]interface{}
	IAMAssocs          int
}

type company struct {
	ID   int64
	Name string
}

type workspace struct {
	ID   int64
	Name string
}

type pendingEvent struct {
	ID     int64
	Status string
}

func NewIntentMux() *IntentMux {
	return &IntentMux{
		Companies:          map[string]company{},
		Profiles:           map[string]string{},
		DeliverableTenants: map[string]bool{},
		ProgressTenants:    map[string]bool{},
		Workspaces:         map[string]workspace{},
		PendingEvents:      map[string]pendingEvent{},
		EventStatus:        map[int64]string{},
		EventData:          map[int64]map[string]interface{}{},
	}
}

func (m *IntentMux) Server() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/task-events/intents/", m.serve)
	return httptest.NewServer(mux)
}

func (m *IntentMux) serve(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/task-events/intents/")
	path = strings.TrimSuffix(path, "/")
	m.mu.Lock()
	defer m.mu.Unlock()

	switch {
	case path == "company-by-creator" && r.Method == http.MethodGet:
		uid := r.URL.Query().Get("user_id")
		if c, ok := m.Companies[uid]; ok {
			writeJSON(w, 200, map[string]interface{}{"ok": true, "company_id": c.ID, "name": c.Name})
			return
		}
		writeJSON(w, 200, map[string]interface{}{"ok": false})
	case path == "create-company" && r.Method == http.MethodPost:
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		uid := body["user_id"]
		if _, ok := m.Companies[uid]; ok {
			http.Error(w, `{"detail":"company already exists: 1","code":"already_exists"}`, 409)
			return
		}
		id := int64(1000 + len(m.Companies) + 1)
		m.Companies[uid] = company{ID: id, Name: body["username"]}
		writeJSON(w, 201, map[string]interface{}{
			"company_id": id, "name": body["username"], "creator_id": uid,
			"created_at": "2026-07-14T00:00:00.000000",
		})
	case path == "upsert-user-profile" && r.Method == http.MethodPost:
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.Profiles[body["user_id"]] = body["username"]
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	case path == "set-default-deliverable" && r.Method == http.MethodPost:
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.DeliverableTenants[body["company_id"]] = true
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	case path == "set-default-progress" && r.Method == http.MethodPost:
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.ProgressTenants[body["company_id"]] = true
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	case path == "create-default-workspace" && r.Method == http.MethodPost:
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		cid := body["company_id"]
		if ws, ok := m.Workspaces[cid]; ok {
			writeJSON(w, 200, map[string]interface{}{
				"workspace_id": ws.ID, "workspace_name": ws.Name,
				"created_at": "2026-01-01T00:00:00.000000", "already_existed": true,
			})
			return
		}
		ws := workspace{ID: 5000 + int64(len(m.Workspaces)), Name: "用户的工作空间"}
		m.Workspaces[cid] = ws
		writeJSON(w, 200, map[string]interface{}{
			"workspace_id": ws.ID, "workspace_name": ws.Name,
			"created_at": "2026-07-14T00:00:00.000000", "already_existed": false,
		})
	case path == "handle-workspace-created" && r.Method == http.MethodPost:
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	case path == "create-access-key-iam" && r.Method == http.MethodPost:
		m.IAMAssocs++
		writeJSON(w, 201, map[string]interface{}{"ok": true})
	case path == "latest-pending-start-event" && r.Method == http.MethodGet:
		key := r.URL.Query().Get("company_id") + ":" + r.URL.Query().Get("task_id")
		if e, ok := m.PendingEvents[key]; ok {
			writeJSON(w, 200, map[string]interface{}{"id": e.ID, "status": e.Status})
			return
		}
		http.Error(w, `{"detail":"pending start event not found for task t","code":"not_found"}`, 404)
	case path == "update-cloud-server-event-status" && r.Method == http.MethodPost:
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		id, _ := body["event_id"].(string)
		var eid int64
		_, _ = fmtSscan(id, &eid)
		if eid == 0 {
			if f, ok := body["event_id"].(float64); ok {
				eid = int64(f)
			}
		}
		st, _ := body["status"].(string)
		m.EventStatus[eid] = st
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	case path == "update-cloud-server-event-data" && r.Method == http.MethodPost:
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		id, _ := body["event_id"].(string)
		var eid int64
		_, _ = fmtSscan(id, &eid)
		if data, ok := body["event_data"].(map[string]interface{}); ok {
			m.EventData[eid] = data
		}
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fmtSscan(s string, out *int64) (int, error) {
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			continue
		}
		n = n*10 + int64(ch-'0')
	}
	*out = n
	return 1, nil
}
