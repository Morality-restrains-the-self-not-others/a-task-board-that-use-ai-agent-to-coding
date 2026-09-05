// Package saastest provides httptest doubles for saas-backend intent APIs.
package saastest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
)

// IntentMux is an in-memory mock for handler tests, covering
// taskProjectService, taskTenantService, taskCloudService, taskBill,
// and taskAuth HTTP APIs.
type IntentMux struct {
	mu sync.Mutex

	Companies          map[string]Company
	DeliverableTenants map[string]bool
	ProgressTenants    map[string]bool
	Workspaces         map[string]Workspace
	WorkspaceHandled   int

	// taskProjectService mocks
	WorkspaceAccesses []WorkspaceAccessEntry // accumulated access rows

	// taskTenantService mocks
	CompanyMembers  []CompanyMemberEntry // accumulated member rows
	CompanySeq      int64                // auto-increment company ID
	FailMembersPost bool                 // 模拟成员行创建失败（OPT-20260806-013 重试路径）

	// taskAuth platform roles mocks (user_id → roles)
	PlatformRoles map[string][]string
	// taskAuth personal nicknames (user_id → username/个人昵称)
	ProfileNicknames map[string]string
}

type Company struct {
	ID   int64
	Name string
}

type Workspace struct {
	ID   int64
	Name string
}

// WorkspaceAccessEntry represents a workspace_accesses row for mock tracking.
type WorkspaceAccessEntry struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	UserID      string `json:"user_id"`
	GroupID     string `json:"group_id,omitempty"`
	Permission  string `json:"permission"`
}

// CompanyMemberEntry represents an tenant_company_member row for mock tracking.
type CompanyMemberEntry struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	CompanyID  string `json:"company_id"`
	IsAdmin    bool   `json:"is_admin"`
	MemberName string `json:"member_name"`
}

// SetServiceEnv sets environment variables so that Repository HTTP clients
// target the mock server instead of real services. Call before creating handlers.
// Returns a cleanup function to restore the original values.
func (m *IntentMux) SetServiceEnv(baseURL string) func() {
	restore := make(map[string]string)
	for _, k := range []string{
		"TASK_TENANT_SERVICE_INTERNAL_URL",
		"TENANT_SERVICE_INTERNAL_SECRET",
		"TASK_PROJECT_SERVICE_URL",
		"TASK_AUTH_INTERNAL_URL",
		"TASK_CLOUD_SERVICE_BASE_URL",
		"TASK_BILL_INTERNAL_URL",
		"TASK_BILL_INTERNAL_SECRET",
		"TASK_EVENTS_INTERNAL_SECRET",
	} {
		restore[k] = os.Getenv(k)
		os.Setenv(k, baseURL)
	}
	return func() {
		for k, v := range restore {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}
}

func NewIntentMux() *IntentMux {
	return &IntentMux{
		Companies:          map[string]Company{},
		DeliverableTenants: map[string]bool{},
		ProgressTenants:    map[string]bool{},
		Workspaces:         map[string]Workspace{},
		WorkspaceAccesses:  []WorkspaceAccessEntry{},
		CompanyMembers:     []CompanyMemberEntry{},
		CompanySeq:         1000,
		PlatformRoles:      map[string][]string{},
		ProfileNicknames:   map[string]string{},
	}
}

func (m *IntentMux) Server() *httptest.Server {
	mux := http.NewServeMux()
	// taskProjectService internal API
	mux.HandleFunc("/api/internal/workspaces/", m.serveTaskProjectInternal)
	// taskProjectService tenant-scoped API (legacy /api/tenant/{tid}/...)
	mux.HandleFunc("/api/tenant/", m.serveTaskProjectTenant)
	// taskProjectService tenant-scoped API (convention /api/projects/<action>/.../tenant_id/{tid}/...)
	mux.HandleFunc("/api/projects/", m.serveTaskProjectConvention)
	// System-admin API (used by setDefaultProgressDirect for progress system list)
	mux.HandleFunc("/api/system-admin/", m.serveSystemAdmin)
	// taskTenantService internal API
	mux.HandleFunc("/api/internal/tenant/", m.serveTaskTenantInternal)
	// taskCloudService internal API
	mux.HandleFunc("/api/internal/cloud/", m.serveTaskCloudInternal)
	// taskBill internal API
	mux.HandleFunc("/api/internal/taskbill/", m.serveTaskBillInternal)
	// taskAuth internal API (markUserTenant etc.)
	mux.HandleFunc("/api/internal/", m.serveGenericInternal)
	return httptest.NewServer(mux)
}

// serveTaskProjectInternal handles /api/internal/workspaces/* (taskProjectService internal API).
func (m *IntentMux) serveTaskProjectInternal(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/workspaces")
	path = strings.Trim(path, "/")
	m.mu.Lock()
	defer m.mu.Unlock()

	// GET /api/internal/workspaces/{wid} — verify workspace exists
	if r.Method == http.MethodGet && path != "" {
		writeJSON(w, 200, map[string]interface{}{"id": path, "name": "用户的工作空间"})
		return
	}
	http.NotFound(w, r)
}

// serveTaskProjectTenant handles /api/tenant/* (taskProjectService tenant-scoped API)
// and also /api/system-admin/* (system-admin progress systems).
func (m *IntentMux) serveTaskProjectTenant(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/tenant/")
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	m.serveTaskProjectResource(w, r, parts[0], parts[1], tenantResourceSub(parts))
}

// serveTaskProjectConvention 处理 convention 路径 /api/projects/<action>/.../tenant_id/{tid}/...。
// 与 taskProjectService 的 gatewayauth.ParseConventionPath 语义一致：action 前缀在前，
// tenant_id/{tid} 等 key=value 路径参数可出现在前缀之后的任意位置。
func (m *IntentMux) serveTaskProjectConvention(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	// 找到第一个已识别 key（tenant_id/workspace_id/task_id/user_id），
	// 之前的段为 action 前缀；从 key 处开始解析 key=value 对（未知 key 停止）。
	// 与 gatewayauth.ParseConventionPath 语义一致（2026-08-07 修复）：
	// key=value 对之后的「位置段」保留在 action 中，不丢弃。
	action := parts
	tid := ""
parseLoop:
	for i, p := range parts {
		switch p {
		case "tenant_id", "workspace_id", "task_id", "user_id":
			kvEnd := i
		kvLoop:
			for j := i; j+1 < len(parts); j += 2 {
				switch parts[j] {
				case "tenant_id":
					tid = parts[j+1]
					kvEnd = j + 2
				case "workspace_id":
					r.Header.Set("X-Workspace-Id", parts[j+1])
					kvEnd = j + 2
				case "task_id":
					r.Header.Set("X-Task-Id", parts[j+1])
					kvEnd = j + 2
				case "user_id":
					r.Header.Set("X-Auth-User-Id", parts[j+1])
					kvEnd = j + 2
				default:
					// 未识别 key — 仅停止 kv 对解析，剩余段保留为位置后缀
					break kvLoop
				}
			}
			action = append(append([]string{}, parts[:i]...), parts[kvEnd:]...)
			break parseLoop
		}
	}
	if len(action) == 0 {
		// 无 action 前缀（如 /api/projects/tenant_id/{tid}/...）→ 项目列表
		if r.Method == http.MethodGet {
			writeJSON(w, 200, []map[string]interface{}{})
			return
		}
		http.NotFound(w, r)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	switch {
	case strings.Join(action, "/") == "workspace-access/workspace-permissions":
		// 真实服务仅支持 GET 列表（创建走 set-permission）
		if r.Method != http.MethodGet {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		m.serveWorkspaceAccess(w, r, "")
	case strings.Join(action, "/") == "workspace-access/set-permission":
		// 真实服务仅支持 POST 创建/更新
		if r.Method != http.MethodPost {
			writeJSON(w, 405, map[string]string{"error": "method not allowed"})
			return
		}
		m.serveWorkspaceAccess(w, r, "")
	case action[0] == "workspaces" && len(action) == 1:
		m.serveWorkspacesTenant(w, r, "", tid)
	case action[0] == "workspaces" && len(action) == 3 && action[2] == "progress-system":
		m.serveWorkspaceProgress(w, r, action[1])
	case action[0] == "deliverable-systems" && len(action) == 1:
		m.serveDeliverableSystems(w, r, "", tid)
	case action[0] == "deliverable-systems" && len(action) == 3 && action[2] == "set-default":
		m.DeliverableTenants[tid] = true
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	case strings.Join(action, "/") == "settings/default-progress-system":
		m.serveSettings(w, r, "default-progress-system", tid)
	case action[0] == "progress-systems" && len(action) == 1:
		writeJSON(w, 200, []map[string]interface{}{
			{"id": "ps_default", "name": "默认进度体系", "is_default": true, "is_system": true},
		})
	case action[0] == "system-deliverable-systems" && len(action) == 1:
		writeJSON(w, 200, []map[string]interface{}{
			{"id": "ds_default_global", "name": "默认交付体系", "is_default": true, "is_system": true},
		})
	default:
		http.NotFound(w, r)
	}
}

// tenantResourceSub 提取 resource 之后的子路径（去掉首尾斜杠）。
func tenantResourceSub(parts []string) string {
	if len(parts) > 2 {
		return strings.Trim(parts[2], "/")
	}
	return ""
}

func (m *IntentMux) serveTaskProjectResource(w http.ResponseWriter, r *http.Request, tid, resource, sub string) {

	m.mu.Lock()
	defer m.mu.Unlock()

	switch resource {
	case "workspace-access":
		m.serveWorkspaceAccess(w, r, sub)
	case "workspaces":
		m.serveWorkspacesTenant(w, r, sub, tid)
	case "settings":
		m.serveSettings(w, r, sub, tid)
	case "deliverable-systems":
		m.serveDeliverableSystems(w, r, sub, tid)
	case "company-deliverable-systems", "system-deliverable-systems":
		if r.Method == http.MethodGet {
			writeJSON(w, 200, []map[string]interface{}{
				{"id": "ds_default_global", "name": "默认交付体系", "is_default": true, "is_system": true},
			})
			return
		}
		http.NotFound(w, r)
	case "progress-systems":
		if r.Method == http.MethodGet {
			writeJSON(w, 200, []map[string]interface{}{
				{"id": "ps_default", "name": "默认进度体系", "is_default": true, "is_system": true},
			})
			return
		}
		http.NotFound(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (m *IntentMux) serveWorkspaceAccess(w http.ResponseWriter, r *http.Request, sub string) {
	_ = sub
	wsID := r.URL.Query().Get("workspace_id")
	switch r.Method {
	case http.MethodGet:
		result := []WorkspaceAccessEntry{}
		for _, a := range m.WorkspaceAccesses {
			if wsID == "" || a.WorkspaceID == wsID {
				result = append(result, a)
			}
		}
		writeJSON(w, 200, result)
	case http.MethodPost:
		var body WorkspaceAccessEntry
		_ = json.NewDecoder(r.Body).Decode(&body)
		// set-permission 语义：同一 workspace+user 已存在则更新权限，否则新增
		updated := false
		for i := range m.WorkspaceAccesses {
			if m.WorkspaceAccesses[i].WorkspaceID == body.WorkspaceID && m.WorkspaceAccesses[i].UserID == body.UserID {
				m.WorkspaceAccesses[i].Permission = body.Permission
				updated = true
				break
			}
		}
		if !updated {
			if body.ID == "" {
				body.ID = fmt.Sprintf("wa_mock_%d", len(m.WorkspaceAccesses)+1)
			}
			m.WorkspaceAccesses = append(m.WorkspaceAccesses, body)
		}
		writeJSON(w, 200, body)
	default:
		http.NotFound(w, r)
	}
}

func (m *IntentMux) serveWorkspacesTenant(w http.ResponseWriter, r *http.Request, sub string, tid string) {
	if strings.HasSuffix(sub, "/progress-system") || strings.Contains(sub, "/progress-system/") {
		m.serveWorkspaceProgress(w, r, sub)
		return
	}
	switch r.Method {
	case http.MethodGet:
		wsList := []map[string]interface{}{}
		for _, ws := range m.Workspaces {
			wsList = append(wsList, map[string]interface{}{
				"id": fmt.Sprintf("%d", ws.ID), "name": ws.Name, "is_default": true,
			})
		}
		writeJSON(w, 200, wsList)
	case http.MethodPost:
		// Echo the posted name (default 用户的工作空间) so tests can assert
		// the workspace-naming contract (OPT-20260827-020: company name).
		name := "用户的工作空间"
		var body struct {
			Name string `json:"name"`
		}
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
			if body.Name != "" {
				name = body.Name
			}
		}
		ws := Workspace{ID: 5000 + int64(len(m.Workspaces)), Name: name}
		m.Workspaces[tid] = ws
		m.WorkspaceHandled++
		writeJSON(w, 201, map[string]interface{}{
			"id": fmt.Sprintf("%d", ws.ID), "name": ws.Name,
			"company_id": tid, "is_default": true, "created_at": "2026-07-31T00:00:00Z",
		})
	default:
		http.NotFound(w, r)
	}
}

func (m *IntentMux) serveDeliverableSystems(w http.ResponseWriter, r *http.Request, sub string, tid string) {
	_ = tid
	if strings.HasSuffix(sub, "/set-default") || strings.Contains(sub, "/set-default/") {
		m.DeliverableTenants[tid] = true
		writeJSON(w, 200, map[string]interface{}{"ok": true})
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, 200, []map[string]interface{}{
			{"id": "ds_default_global", "name": "默认交付体系", "is_default": false, "is_system": true},
		})
		return
	}
	http.NotFound(w, r)
}

func (m *IntentMux) serveWorkspaceProgress(w http.ResponseWriter, r *http.Request, sub string) {
	_ = sub
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 404, map[string]string{"error": "progress system not set"})
	case http.MethodPost:
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	default:
		http.NotFound(w, r)
	}
}

func (m *IntentMux) serveSettings(w http.ResponseWriter, r *http.Request, sub string, tid string) {
	if sub == "default-progress-system" {
		if r.Method == http.MethodGet {
			writeJSON(w, 200, map[string]interface{}{"progress_system": nil})
			return
		}
		if r.Method == http.MethodPost {
			m.ProgressTenants[tid] = true
			writeJSON(w, 200, map[string]interface{}{"ok": true})
			return
		}
	}
	http.NotFound(w, r)
}

// serveSystemAdmin handles /api/system-admin/* (progress systems list for setDefaultProgress).
func (m *IntentMux) serveSystemAdmin(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/system-admin/")
	m.mu.Lock()
	defer m.mu.Unlock()

	if path == "progress-systems" || strings.HasPrefix(path, "progress-systems") {
		if r.Method == http.MethodGet {
			writeJSON(w, 200, map[string]interface{}{
				"project_progress_systems": []map[string]interface{}{
					{"id": "ps_default", "name": "默认进度体系", "is_default": true},
				},
			})
			return
		}
	}
	http.NotFound(w, r)
}

// serveTaskCloudInternal handles /api/internal/cloud/* (taskCloudService internal API).
func (m *IntentMux) serveTaskCloudInternal(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/cloud/")
	m.mu.Lock()
	defer m.mu.Unlock()

	switch {
	case path == "init-tenant-feature-params" || strings.HasPrefix(path, "init-tenant-feature-params"):
		writeJSON(w, 200, map[string]interface{}{
			"id":         fmt.Sprintf("fp_mock_%d", len(m.CompanyMembers)+1),
			"company_id": strings.TrimSpace(r.URL.Query().Get("company_id")),
		})
	case path == "access-key-iam-associations" || strings.HasPrefix(path, "access-key-iam-associations"):
		writeJSON(w, 201, map[string]interface{}{"ok": true})
	default:
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	}
}

// serveTaskBillInternal handles /api/internal/taskbill/* (taskBill internal API).
func (m *IntentMux) serveTaskBillInternal(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/taskbill/")
	_ = path
	// All known routes just need 200 with a valid JSON body.
	writeJSON(w, 200, map[string]interface{}{
		"ok":           true,
		"task_count":   100,
		"quota_after":  100,
		"quota_before": 0,
	})
}

// serveGenericInternal handles /api/internal/* catch-all (taskAuth markUserTenant, etc.).
func (m *IntentMux) serveGenericInternal(w http.ResponseWriter, r *http.Request) {
	// taskAuth platform-roles lookup: GET /api/internal/users/id/{uid}/platform-roles/
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/internal/users/id/") {
		rest := strings.TrimPrefix(r.URL.Path, "/api/internal/users/id/")
		rest = strings.Trim(rest, "/")
		uid := strings.TrimSuffix(rest, "/platform-roles")
		m.mu.Lock()
		defer m.mu.Unlock()
		roles := m.PlatformRoles[uid]
		if roles == nil {
			roles = []string{}
		}
		writeJSON(w, 200, map[string]interface{}{"user_id": uid, "roles": roles})
		return
	}
	// taskAuth batch user details (personal nickname): POST /api/internal/users/batch/details/
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/internal/users/batch/details") {
		var body struct {
			UserIDs []string `json:"user_ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		m.mu.Lock()
		defer m.mu.Unlock()
		results := map[string]interface{}{}
		for _, uid := range body.UserIDs {
			results[uid] = map[string]string{
				"user_id":  uid,
				"email":    "",
				"username": m.ProfileNicknames[uid],
			}
		}
		writeJSON(w, 200, map[string]interface{}{"results": results})
		return
	}
	// Accept any internal PUT/PATCH/POST as successful
	switch r.Method {
	case http.MethodPatch, http.MethodPut, http.MethodPost:
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	case http.MethodGet:
		writeJSON(w, 200, map[string]interface{}{"ok": true})
	default:
		http.NotFound(w, r)
	}
}

// serveTaskTenantInternal handles /api/internal/tenant/* (taskTenantService internal API).
func (m *IntentMux) serveTaskTenantInternal(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/tenant/")
	parts := strings.SplitN(path, "/", 2)
	resource := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	switch resource {
	case "companies":
		if sub == "by-creator" && r.Method == http.MethodGet {
			uid := r.URL.Query().Get("creator_id")
			if c, ok := m.Companies[uid]; ok {
				writeJSON(w, 200, []map[string]interface{}{
					{"id": fmt.Sprintf("%d", c.ID), "name": c.Name, "creator_id": uid},
				})
				return
			}
			writeJSON(w, 200, []map[string]interface{}{})
			return
		}
		if sub == "upsert" && r.Method == http.MethodPost {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			uid := body["creator_id"]
			c, exists := m.Companies[uid]
			if !exists {
				m.CompanySeq++
				c = Company{ID: m.CompanySeq, Name: body["name"]}
				m.Companies[uid] = c
			}
			writeJSON(w, 200, map[string]interface{}{
				"id":         fmt.Sprintf("%d", c.ID),
				"name":       c.Name,
				"creator_id": uid,
				"created_at": "2026-07-31T00:00:00",
			})
			return
		}
		http.NotFound(w, r)
	case "members":
		if r.Method == http.MethodPost {
			if m.FailMembersPost {
				writeJSON(w, 500, map[string]string{"error": "mock member failure"})
				return
			}
			var body CompanyMemberEntry
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.ID == "" {
				body.ID = fmt.Sprintf("mem_mock_%d", len(m.CompanyMembers)+1)
			}
			m.CompanyMembers = append(m.CompanyMembers, body)
			writeJSON(w, 201, map[string]string{"id": body.ID})
			return
		}
		if r.Method == http.MethodGet {
			uid := r.URL.Query().Get("user_id")
			cid := r.URL.Query().Get("company_id")
			if uid != "" {
				result := []CompanyMemberEntry{}
				for _, mem := range m.CompanyMembers {
					if mem.UserID == uid {
						result = append(result, mem)
					}
				}
				writeJSON(w, 200, result)
				return
			}
			result := []CompanyMemberEntry{}
			for _, mem := range m.CompanyMembers {
				if cid == "" || mem.CompanyID == cid {
					result = append(result, mem)
				}
			}
			writeJSON(w, 200, result)
			return
		}
		http.NotFound(w, r)
	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
