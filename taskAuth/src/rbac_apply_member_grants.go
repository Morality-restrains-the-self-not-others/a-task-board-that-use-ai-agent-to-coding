package main

import (
	"authz"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type applyGrantSpec struct {
	id     string
	key    string
	kind   string
	effect string
}

// handleApplyMemberGrants — POST /api/internal/authz/apply-member-grants/
// Creates a tenant custom role, binds resource-group grants, dual-writes coarse perms.
// Auth: X-Internal-Secret (service-to-service from taskTenantService on join).
func handleApplyMemberGrants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	var body struct {
		CompanyID   string `json:"company_id"`
		DisplayName string `json:"display_name"`
		Grants      []struct {
			GroupKey string `json:"group_key"`
			Effect   string `json:"effect"`
		} `json:"grants"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid body")
		return
	}
	companyID := strings.TrimSpace(body.CompanyID)
	displayName := strings.TrimSpace(body.DisplayName)
	if companyID == "" || displayName == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "company_id and display_name required")
		return
	}
	if len(body.Grants) == 0 {
		writeErrorDetail(w, r, http.StatusBadRequest, "grants required")
		return
	}
	if len(body.Grants) > 200 {
		writeErrorDetail(w, r, http.StatusBadRequest, "grants too many")
		return
	}

	seen := map[string]string{} // id → effect
	var specs []applyGrantSpec
	for _, g := range body.Grants {
		key := strings.TrimSpace(g.GroupKey)
		if key == "" {
			writeErrorDetail(w, r, http.StatusBadRequest, "grants.group_key required")
			return
		}
		effect := authz.NormalizeGrantEffect(g.Effect)
		var id, kind string
		err := db.QueryRow(`SELECT id, kind FROM auth_resource_group WHERE group_key=?`, key).Scan(&id, &kind)
		if err != nil {
			writeErrorDetail(w, r, http.StatusBadRequest, "unknown resource group: "+key)
			return
		}
		if prev, ok := seen[id]; ok {
			seen[id] = authz.MaxGrantEffect(prev, effect)
			continue
		}
		seen[id] = effect
		specs = append(specs, applyGrantSpec{id: id, key: key, kind: kind, effect: effect})
	}
	for i := range specs {
		specs[i].effect = seen[specs[i].id]
	}

	perms := coarsePermsFromGrantKeys(specs)
	if len(perms) == 0 {
		perms = []string{"member:view"}
	}

	name := "custom_" + companyID + "_" + randSuffix()
	id := "role-custom-" + randSuffix()
	if len([]rune(displayName)) > 64 {
		displayName = string([]rune(displayName)[:64])
	}
	if _, err := db.Exec(`INSERT INTO auth_role (id, name, display_name, level, priority, is_system, company_id, description)
		VALUES (?, ?, ?, 'tenant', 50, 0, ?, ?)`, id, name, displayName, companyID, "邀请预授访问角色"); err != nil {
		if isDuplicateKey(err) {
			writeErrorDetail(w, r, http.StatusConflict, "display_name 已存在")
			return
		}
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if err := replaceRolePermissions(id, perms); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	tx, err := db.Begin()
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "tx begin failed")
		return
	}
	bound := 0
	for _, spec := range specs {
		if _, err := tx.Exec(
			`INSERT INTO auth_role_resource_group (id, role_id, resource_group_id, effect) VALUES (?,?,?,?)`,
			"rrg-"+randSuffix(), id, spec.id, spec.effect,
		); err != nil {
			_ = tx.Rollback()
			writeErrorDetail(w, r, http.StatusInternalServerError, "bind grant failed")
			return
		}
		bound++
	}
	if err := tx.Commit(); err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "tx commit failed")
		return
	}
	publishRoleChanged(companyID)
	log.Printf("[taskAuth] event=rbac_apply_member_grants status=ok company_id=%s role_id=%s role=%s count=%d",
		companyID, id, name, bound)
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": name, "bound": bound})
}

// coarsePermsFromGrantKeys maps page/region keys to legacy coarse permission codes (bridge).
func coarsePermsFromGrantKeys(specs []applyGrantSpec) []string {
	set := map[string]struct{}{}
	add := func(code string) {
		if code != "" {
			set[code] = struct{}{}
		}
	}
	for _, s := range specs {
		key := s.key
		switch {
		case key == "people.access" || strings.HasPrefix(key, "people.access."):
			add("member:manage")
		case key == "people.invite" || strings.HasPrefix(key, "people.invite.") ||
			key == "people.manage" || strings.HasPrefix(key, "people.manage."):
			add("member:manage")
		case key == "people.groups" || strings.HasPrefix(key, "people.groups."):
			add("group:manage")
		case strings.HasPrefix(key, "billing."):
			add("billing:view")
		case strings.HasPrefix(key, "settings.cloud") || strings.HasPrefix(key, "nav.image_market"):
			add("cloud:manage")
		case strings.HasPrefix(key, "settings.company") || strings.HasPrefix(key, "settings.gitlab") ||
			strings.HasPrefix(key, "settings.deliverable") || strings.HasPrefix(key, "settings.status"):
			add("company:view")
		case strings.HasPrefix(key, "settings.task_panel"):
			add("workspace:manage")
		case strings.HasPrefix(key, "nav.projects"):
			add("project:view")
		case strings.HasPrefix(key, "nav.work_panel"):
			add("task:view")
		default:
			add("member:view")
		}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	return out
}
