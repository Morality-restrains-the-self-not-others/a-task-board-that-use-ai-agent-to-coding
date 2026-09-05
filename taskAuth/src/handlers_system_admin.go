package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// --- System-Admin Dashboard & User Management ---
//
// OPT-036: System-admin dashboard and user management, previously served by
// Django cloudSystemAdmin, now moved to taskAuth (which already owns user
// auth and has superadmin infrastructure).
//
// Authentication: gateway token → superuser check via loadUserAuthFlags.

// handleSystemAdminDashboard returns aggregate stats for the superadmin dashboard.
// GET /api/system-admin/dashboard/
func handleSystemAdminDashboard(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperuser(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	stats := map[string]interface{}{}
	var count int64

	// Total users
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_user`).Scan(&count); err != nil {
		stats["total_users"] = nil
	} else {
		stats["total_users"] = count
	}

	// Online admins: count active superusers (last_login within 1 hour)
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_user WHERE is_superuser = 1 AND is_active = 1`).Scan(&count); err != nil {
		stats["online_admins"] = "0"
	} else {
		stats["online_admins"] = count
	}

	// Cloud authorizations — cross-service stat (TODO: add count when available)
	stats["cloud_authorizations"] = nil

	// Deliverable systems — cross-database query to task_project
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM task_project.project_deliverable_systems WHERE is_system=1`,
	).Scan(&count); err != nil {
		stats["deliverable_systems"] = nil
	} else {
		stats["deliverable_systems"] = count
	}

	systemInfo := map[string]interface{}{
		"version":          serviceVersion(),
		"last_deploy_time": "",
		"status":           "running",
	}

	writeJSON(w, 200, map[string]interface{}{
		"stats":       stats,
		"system_info": systemInfo,
	})
}

// handleSystemAdminUsers routes system-admin user management requests.
// GET  /api/system-admin/users/         → list users (paginated)
// POST /api/system-admin/users/create/  → create user
// PATCH /api/system-admin/users/{id}/   → update flags and phone/email identifiers flags and phone/email identifiers
func handleSystemAdminUsers(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/system-admin/users")
	path = strings.Trim(path, "/")
	if path != "" && r.Method == http.MethodPost {
		parts := strings.SplitN(path, "/", 2)
		userID := strings.TrimSpace(parts[0])
		sub := ""
		if len(parts) == 2 {
			sub = strings.Trim(parts[1], "/")
		}
		if userID != "" && userID != "create" && sub == "impersonate" {
			handleStartImpersonation(w, r, userID)
			return
		}
	}

	if _, ok := requireSuperuser(w, r); !ok {
		return
	}

	// POST /api/system-admin/users/create/
	if path == "create" || path == "create/" {
		if r.Method != http.MethodPost {
			writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleSystemAdminCreateUser(w, r)
		return
	}

	// PATCH /api/system-admin/users/{id}/ — including archive/unarchive/deactivate
	if path != "" {
		userID := strings.SplitN(path, "/", 2)[0]
		userID = strings.TrimSpace(userID)
		if userID == "" {
			writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
			return
		}
		subAction := strings.TrimPrefix(path, userID)
		subAction = strings.Trim(subAction, "/")

		switch {
		case subAction == "login-history" || subAction == "login-history/":
			handleSystemAdminUserLoginHistory(w, r, userID)
			return
		case subAction == "archive" || subAction == "archive/":
			if r.Method != http.MethodPost {
				writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			_, _ = db.Exec(`UPDATE auth_user SET is_archived = 1 WHERE id = ?`, userID)
			payload, err := buildUserDetailJSON(userID)
			if err == sql.ErrNoRows {
				writeErrorDetail(w, r, 404, "user not found")
				return
			}
			if err != nil {
				writeErrorDetail(w, r, 500, "db error")
				return
			}
			writeJSON(w, 200, payload)
			return
		case subAction == "unarchive" || subAction == "unarchive/":
			if r.Method != http.MethodPost {
				writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			// OPT-20260824-089: 解档前检测该用户仍持有的未作废标识是否被活跃用户占用，
			// 避免恢复绑定造成双活占用/登录歧义。冲突则 409 并说明需先解绑或换号。
			conflicts, err := findActiveIdentifierConflicts(userID)
			if err != nil {
				writeErrorDetail(w, r, 500, "db error")
				return
			}
			if len(conflicts) > 0 {
				slog.InfoContext(r.Context(), "system_admin_unarchive_identifier_conflict",
					"user_id", userID, "conflict_count", len(conflicts))
				writeErrorDetail(w, r, http.StatusConflict, describeIdentifierConflict(conflicts))
				return
			}
			_, _ = db.Exec(`UPDATE auth_user SET is_archived = 0 WHERE id = ?`, userID)
			payload, err := buildUserDetailJSON(userID)
			if err == sql.ErrNoRows {
				writeErrorDetail(w, r, 404, "user not found")
				return
			}
			if err != nil {
				writeErrorDetail(w, r, 500, "db error")
				return
			}
			writeJSON(w, 200, payload)
			return
		case subAction == "delete" || subAction == "delete/":
			if r.Method != http.MethodDelete && r.Method != http.MethodPost {
				writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			_, _ = db.Exec(`UPDATE auth_user SET is_active = 0 WHERE id = ?`, userID)
			writeJSON(w, 200, map[string]string{"status": "deactivated"})
			return
		default:
			// PATCH update user fields
			if r.Method == http.MethodPatch || r.Method == http.MethodPut {
				// Delegate to internal patchUser logic
				patchUserAsAdmin(w, r, userID)
				return
			}
			if r.Method == http.MethodGet {
				payload, err := buildUserDetailJSON(userID)
				if err == sql.ErrNoRows {
					writeErrorDetail(w, r, 404, "user not found")
					return
				}
				if err != nil {
					writeErrorDetail(w, r, 500, "db error")
					return
				}
				writeJSON(w, 200, payload)
				return
			}
			writeErrorDetail(w, r, 405, "method not allowed")
			return
		}
	}

	// GET /api/system-admin/users/ — list users (public admin version)
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	handleSystemAdminListUsers(w, r)
}

// serviceVersion returns the service version string for dashboard display.
func serviceVersion() string {
	return "v56"
}

// handlePublicUpsertProfile is the public (token-authenticated) wrapper for
// upserting user profile data (username display cache). Replaces the Django
// users/profile endpoint (OPT-043: profile → taskAuth).
// POST /api/accounts/users/profile/
func handlePublicUpsertProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	username := strField(body, "username")
	if err := upsertUserProfile(userID, username); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

// --- 插件 OIDC 白名单管理 ---
//
// OPT-20260808-026: GET/PUT /api/system-admin/oidc-extension/（requireSuperuser）。
// chrome-extension 行的 redirect_uris 为 chrome-extension://<32位ID>/oauth-callback.html
// JSON 数组；PUT 一次即置位 managed_by='admin'（幂等接管，防 bootstrap seed 自愈覆盖
// 竞态——OPT-20260808-025 admin 托管语义）。

const (
	oidcExtensionClientID = "chrome-extension"
	oidcExtensionPath     = "/oauth-callback.html"
	oidcExtensionMaxIDs   = 20
)

// oidcExtensionIDRe — Chrome 扩展 ID：32 位小写 a-p。
var oidcExtensionIDRe = regexp.MustCompile(`^[a-p]{32}$`)

func oidcExtensionRedirectURI(id string) string {
	return "chrome-extension://" + id + oidcExtensionPath
}

// oidcExtensionIDsFromURIs 从 redirect_uris 提取扩展 ID（非法/重复项跳过）。
func oidcExtensionIDsFromURIs(uris []string) []string {
	seen := map[string]bool{}
	var ids []string
	for _, u := range uris {
		rest, ok := strings.CutPrefix(u, "chrome-extension://")
		if !ok {
			continue
		}
		id, _, ok := strings.Cut(rest, "/")
		if !ok || !oidcExtensionIDRe.MatchString(id) || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// normalizeOidcExtensionIDs 服务端权威校验（前端同规则预校验）：
// 正则 ^[a-p]{32}$、非空、去重、≤20、≥1。返回去重后的列表。
func normalizeOidcExtensionIDs(ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, errors.New("extension_ids required")
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			return nil, errors.New("extension_ids contains empty id")
		}
		if !oidcExtensionIDRe.MatchString(id) {
			return nil, fmt.Errorf("invalid extension id: %q", id)
		}
		if seen[id] {
			continue // 去重
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, errors.New("extension_ids required")
	}
	if len(out) > oidcExtensionMaxIDs {
		return nil, fmt.Errorf("extension_ids too many: max %d", oidcExtensionMaxIDs)
	}
	return out, nil
}

// handleSystemAdminOidcExtension routes GET/PUT for the extension whitelist.
// GET  → {client_id, name, managed_by, extension_ids, raw_redirect_uris}
// PUT  → body {extension_ids: [...]} → 更新后 {client_id, managed_by, extension_ids, raw_redirect_uris}
func handleSystemAdminOidcExtension(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireSuperuser(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		row, err := loadOidcClient(oidcExtensionClientID)
		if err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		if row == nil {
			writeErrorDetail(w, r, http.StatusNotFound, "oidc client not found")
			return
		}
		var uris []string
		if err := json.Unmarshal([]byte(row.RedirectURIs), &uris); err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "invalid client config")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"client_id":         row.ClientID,
			"name":              row.Name,
			"managed_by":        row.ManagedBy,
			"extension_ids":     oidcExtensionIDsFromURIs(uris),
			"raw_redirect_uris": uris,
		})
	case http.MethodPut:
		var body struct {
			ExtensionIDs []string `json:"extension_ids"`
		}
		dec := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
		if err := dec.Decode(&body); err != nil {
			writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
			return
		}
		ids, err := normalizeOidcExtensionIDs(body.ExtensionIDs)
		if err != nil {
			writeErrorDetail(w, r, http.StatusBadRequest, err.Error())
			return
		}
		// client 不存在（loadOidcClient 判空）→ 404；写库前检查，避免 UPDATE 0 行歧义
		row, err := loadOidcClient(oidcExtensionClientID)
		if err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		if row == nil {
			writeErrorDetail(w, r, http.StatusNotFound, "oidc client not found")
			return
		}
		uris := make([]string, 0, len(ids))
		for _, id := range ids {
			uris = append(uris, oidcExtensionRedirectURI(id))
		}
		urisJSON, _ := json.Marshal(uris)
		now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
		// 幂等接管：一次 PUT 即置位 admin（防 bootstrap seed 自愈覆盖竞态）
		if _, err := db.Exec(`UPDATE auth_oidc_client SET redirect_uris = ?, managed_by = 'admin', updated_at = ? WHERE client_id = ?`,
			string(urisJSON), now, oidcExtensionClientID); err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"client_id":         oidcExtensionClientID,
			"managed_by":        "admin",
			"extension_ids":     ids,
			"raw_redirect_uris": uris,
		})
	default:
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}
