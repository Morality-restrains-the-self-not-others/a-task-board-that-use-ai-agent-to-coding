package main

import (
	"authz"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// ═══════════════════════════════════════════════════════════════
// Logical resource groups (v72 / ADR-0003)
// page = carrier；ui_region = grantable resource group
// ═══════════════════════════════════════════════════════════════

type resourceGroupRow struct {
	ID          string              `json:"id"`
	GroupKey    string              `json:"group_key"`
	Kind        string              `json:"kind"`
	ParentID    *string             `json:"parent_id,omitempty"`
	DisplayName string              `json:"display_name"`
	RoutePrefix *string             `json:"route_prefix,omitempty"`
	SortOrder   int                 `json:"sort_order"`
	Children    []resourceGroupRow  `json:"children,omitempty"`
	Members     []resourceMemberRow `json:"members,omitempty"`
}

type resourceMemberRow struct {
	MemberKind string `json:"member_kind"`
	MemberKey  string `json:"member_key"`
}

func requireAccessManageQuiet(w http.ResponseWriter, r *http.Request, companyID string) bool {
	if authz.HasRegionView(r, companyID, "people.access.save_actions") ||
		authz.HasRegionView(r, companyID, "people.access.region_matrix") ||
		authz.HasRegionView(r, companyID, "people.access.subject_list") ||
		authz.HasPage(r, companyID, "people.access") ||
		authz.HasPerm(r, companyID, authz.PermMemberManage) {
		return true
	}
	writeErrorDetail(w, r, http.StatusForbidden, "权限不足，需要访问管理区域或 member:manage")
	return false
}

// handleListResourceGroups — GET /api/auth/resource-groups/?company_id=
func handleListResourceGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	if companyID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "company_id required")
		return
	}
	if !requireAccessManageQuiet(w, r, companyID) {
		return
	}
	rows, err := db.Query(`
SELECT id, group_key, kind, COALESCE(parent_id,''), display_name, COALESCE(route_prefix,''), sort_order
FROM auth_resource_group
WHERE company_id IS NULL AND is_system=1
ORDER BY sort_order, group_key`)
	if err != nil {
		log.Printf("[taskAuth] event=rbac_resource_group_list status=error err=%v", err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "加载资源组失败")
		return
	}
	defer rows.Close()
	byID := map[string]*resourceGroupRow{}
	var pages []*resourceGroupRow
	for rows.Next() {
		var g resourceGroupRow
		var parentID, route string
		if err := rows.Scan(&g.ID, &g.GroupKey, &g.Kind, &parentID, &g.DisplayName, &route, &g.SortOrder); err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "扫描资源组失败")
			return
		}
		if parentID != "" {
			p := parentID
			g.ParentID = &p
		}
		if route != "" {
			rp := route
			g.RoutePrefix = &rp
		}
		cp := g
		byID[g.ID] = &cp
		if g.Kind == "page" {
			pages = append(pages, byID[g.ID])
		}
	}
	for _, g := range byID {
		if g.Kind != "ui_region" || g.ParentID == nil {
			continue
		}
		if parent, ok := byID[*g.ParentID]; ok {
			parent.Children = append(parent.Children, *g)
		}
	}
	// attach members for regions
	mrows, err := db.Query(`SELECT resource_group_id, member_kind, member_key FROM auth_resource_member`)
	if err == nil {
		defer mrows.Close()
		for mrows.Next() {
			var rgID, mk, mkey string
			if err := mrows.Scan(&rgID, &mk, &mkey); err != nil {
				break
			}
			if g, ok := byID[rgID]; ok {
				g.Members = append(g.Members, resourceMemberRow{MemberKind: mk, MemberKey: mkey})
			}
			// also patch into page children copies — rebuild children with members
		}
		// re-sync children members from byID
		for _, p := range pages {
			for i := range p.Children {
				if src, ok := byID[p.Children[i].ID]; ok {
					p.Children[i].Members = src.Members
				}
			}
		}
	}
	out := make([]resourceGroupRow, 0, len(pages))
	for _, p := range pages {
		out = append(out, *p)
	}
	log.Printf("[taskAuth] event=rbac_resource_group_list status=ok company_id=%s pages=%d", companyID, len(out))
	writeJSON(w, http.StatusOK, map[string]any{"pages": out})
}

// handleRoleResourceGroups — GET/PUT /api/auth/roles/role_id/{rid}/resource-groups/
func handleRoleResourceGroups(w http.ResponseWriter, r *http.Request) {
	rid := r.PathValue("rid")
	if rid == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "role_id required")
		return
	}
	var roleCompany sql.NullString
	var isSystem int
	var roleName string
	err := db.QueryRow(`SELECT name, is_system, company_id FROM auth_role WHERE id=?`, rid).Scan(&roleName, &isSystem, &roleCompany)
	if err == sql.ErrNoRows {
		writeErrorDetail(w, r, http.StatusNotFound, "role not found")
		return
	}
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "query role failed")
		return
	}
	companyID := ""
	if roleCompany.Valid {
		companyID = roleCompany.String
	}
	if companyID == "" {
		// system roles: allow read with query company_id; forbid PUT
		companyID = strings.TrimSpace(r.URL.Query().Get("company_id"))
	}
	if companyID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "company_id required for system role")
		return
	}
	if !requireAccessManageQuiet(w, r, companyID) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(`
SELECT g.id, g.group_key, g.kind, COALESCE(rrg.effect,'operate')
FROM auth_role_resource_group rrg
JOIN auth_resource_group g ON g.id = rrg.resource_group_id
WHERE rrg.role_id=?`, rid)
		if err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "query failed")
			return
		}
		defer rows.Close()
		items := make([]map[string]string, 0)
		for rows.Next() {
			var id, key, kind, effect string
			if err := rows.Scan(&id, &key, &kind, &effect); err != nil {
				writeErrorDetail(w, r, http.StatusInternalServerError, "scan failed")
				return
			}
			items = append(items, map[string]string{
				"id": id, "group_key": key, "kind": kind,
				"effect": authz.NormalizeGrantEffect(effect),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"role_id": rid, "resource_groups": items})
	case http.MethodPut:
		if isSystem == 1 {
			writeErrorDetail(w, r, http.StatusForbidden, "系统角色资源组不可直接修改")
			return
		}
		if !roleCompany.Valid || roleCompany.String != companyID {
			writeErrorDetail(w, r, http.StatusForbidden, "角色不属于该公司")
			return
		}
		// 写路径：subject_list/region_matrix operate，或存量 save_actions，或 member:manage。
		if !authz.HasPeopleAccessWrite(r, companyID) {
			writeErrorDetail(w, r, http.StatusForbidden, "权限不足，需要访问管理区域可编辑执行")
			return
		}
		var body struct {
			GroupKeys []string `json:"group_keys"`
			GroupIDs  []string `json:"group_ids"`
			Grants    []struct {
				GroupKey string `json:"group_key"`
				GroupID  string `json:"group_id"`
				Effect   string `json:"effect"`
			} `json:"grants"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErrorDetail(w, r, http.StatusBadRequest, "invalid body")
			return
		}
		type grantSpec struct {
			id     string
			effect string
		}
		var specs []grantSpec
		seenID := map[string]string{} // id → effect

		addGrant := func(id, effect string) {
			if id == "" {
				return
			}
			effect = authz.NormalizeGrantEffect(effect)
			if prev, ok := seenID[id]; ok {
				seenID[id] = authz.MaxGrantEffect(prev, effect)
				return
			}
			seenID[id] = effect
		}

		for _, g := range body.Grants {
			id := strings.TrimSpace(g.GroupID)
			if id == "" && g.GroupKey != "" {
				_ = db.QueryRow(`SELECT id FROM auth_resource_group WHERE group_key=?`, g.GroupKey).Scan(&id)
			}
			addGrant(id, g.Effect)
		}
		for _, id := range body.GroupIDs {
			addGrant(strings.TrimSpace(id), authz.EffectOperate)
		}
		if len(body.GroupKeys) > 0 {
			ph := inPlaceholders(len(body.GroupKeys))
			args := make([]any, len(body.GroupKeys))
			for i, k := range body.GroupKeys {
				args[i] = k
			}
			krows, err := db.Query(`SELECT id FROM auth_resource_group WHERE group_key IN (`+ph+`)`, args...)
			if err != nil {
				writeErrorDetail(w, r, http.StatusInternalServerError, "resolve keys failed")
				return
			}
			for krows.Next() {
				var id string
				if err := krows.Scan(&id); err != nil {
					krows.Close()
					writeErrorDetail(w, r, http.StatusInternalServerError, "scan key failed")
					return
				}
				addGrant(id, authz.EffectOperate)
			}
			krows.Close()
		}
		for id, eff := range seenID {
			specs = append(specs, grantSpec{id: id, effect: eff})
		}
		tx, err := db.Begin()
		if err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "tx begin failed")
			return
		}
		if _, err := tx.Exec(`DELETE FROM auth_role_resource_group WHERE role_id=?`, rid); err != nil {
			_ = tx.Rollback()
			writeErrorDetail(w, r, http.StatusInternalServerError, "clear failed")
			return
		}
		bound := 0
		for _, spec := range specs {
			var kind string
			if err := tx.QueryRow(`SELECT kind FROM auth_resource_group WHERE id=?`, spec.id).Scan(&kind); err != nil {
				_ = tx.Rollback()
				writeErrorDetail(w, r, http.StatusBadRequest, "unknown resource group: "+spec.id)
				return
			}
			if kind != "page" && kind != "ui_region" {
				_ = tx.Rollback()
				writeErrorDetail(w, r, http.StatusBadRequest, "invalid kind")
				return
			}
			bindID := "rrg-" + rid + "-" + spec.id
			if len(bindID) > 64 {
				bindID = "rrg-" + randSuffix()
			}
			if _, err := tx.Exec(
				`INSERT INTO auth_role_resource_group (id, role_id, resource_group_id, effect) VALUES (?,?,?,?)`,
				bindID, rid, spec.id, spec.effect,
			); err != nil {
				_ = tx.Rollback()
				writeErrorDetail(w, r, http.StatusInternalServerError, "bind failed")
				return
			}
			bound++
		}
		if err := tx.Commit(); err != nil {
			writeErrorDetail(w, r, http.StatusInternalServerError, "commit failed")
			return
		}
		publishRoleChanged(companyID)
		publishRoleResourceGroupsChanged(companyID, rid)
		log.Printf("[taskAuth] event=rbac_role_resource_groups_put status=ok role_id=%s role=%s count=%d", rid, roleName, bound)
		writeJSON(w, http.StatusOK, map[string]any{"role_id": rid, "bound": bound})
	default:
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}
