package main

import (
	"authz"
	"encoding/json"
	"net/http"
	"strings"

	"snowflake"
)

// ═══════════════════════════════════════════════════════════════
// 资源→小组分配 API (v63 design §6 taskTenant 21-24)
// ═══════════════════════════════════════════════════════════════

var allowedResourceTypes = map[string]bool{"project": true, "cloud": true, "task": true, "workspace": true}

// handleAssignResourceToGroup — POST /api/tenant/resource-group/company_id/{cid}/
// body: {"resource_type": "project", "resource_id": "p1", "group_id": "g1", "permission": "view|manage"}
func handleAssignResourceToGroup(w http.ResponseWriter, r *http.Request, cid string) {
	if !authz.RequirePerm(w, r, authz.PermGroupResourcesMng, cid) {
		return
	}
	var body struct {
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		GroupID      string `json:"group_id"`
		Permission   string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid body")
		return
	}
	if !allowedResourceTypes[body.ResourceType] || body.ResourceID == "" || body.GroupID == "" {
		writeError(w, r, http.StatusBadRequest, "resource_type/resource_id/group_id required")
		return
	}
	if body.Permission != "manage" && body.Permission != "view" {
		body.Permission = "view"
	}
	if err := ensureGroupInCompany(body.GroupID, cid); err != nil {
		writeError(w, r, http.StatusNotFound, "小组不存在")
		return
	}
	if _, err := db.Exec(`INSERT INTO tenant_resource_group_assignment
		(id, company_id, resource_type, resource_id, group_id, permission, assigned_by, assigned_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE permission = VALUES(permission), assigned_by = VALUES(assigned_by)`,
		snowflake.GenerateIDString(), cid, body.ResourceType, body.ResourceID, body.GroupID, body.Permission, getAuthUser(r)); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	// 事件失效: 资源授权变化影响组内全部成员（group-res 伪码）→ 逐一 incr rev
	incrGroupMembersRev(body.GroupID)
	publishTenantRoleChanged(cid)
	writeJSON(w, http.StatusOK, map[string]any{"detail": "ok"})
}

// handleRevokeResourceFromGroup — DELETE /api/tenant/resource-group/company_id/{cid}/assignment_id/{aid}/
func handleRevokeResourceFromGroup(w http.ResponseWriter, r *http.Request, cid, aid string) {
	if !authz.RequirePerm(w, r, authz.PermGroupResourcesMng, cid) {
		return
	}
	// 回收前取 group_id：组资源变化影响组内全部成员 → 逐一 incr rev
	var gid string
	_ = db.QueryRow(`SELECT group_id FROM tenant_resource_group_assignment WHERE id=? AND company_id=?`, aid, cid).Scan(&gid)
	if _, err := db.Exec(`DELETE FROM tenant_resource_group_assignment WHERE id=? AND company_id=?`, aid, cid); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	incrGroupMembersRev(gid)
	publishTenantRoleChanged(cid)
	writeJSON(w, http.StatusOK, map[string]any{"detail": "ok"})
}

// handleListGroupResources — GET /api/tenant/resource-group/company_id/{cid}/group_id/{gid}/
func handleListGroupResources(w http.ResponseWriter, r *http.Request, cid, gid string) {
	if !authz.RequireTenantMember(w, r, cid) {
		return
	}
	if err := ensureGroupInCompany(gid, cid); err != nil {
		writeError(w, r, http.StatusNotFound, "小组不存在")
		return
	}
	rows, err := db.Query(`SELECT id, resource_type, resource_id, permission, assigned_at
		FROM tenant_resource_group_assignment WHERE group_id=? AND company_id=?`, gid, cid)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, rtype, rid, perm, at string
		if err := rows.Scan(&id, &rtype, &rid, &perm, &at); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, map[string]any{"id": id, "resource_type": rtype, "resource_id": rid, "permission": perm, "assigned_at": at})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleListResourceGroups — GET /api/tenant/resource-group/company_id/{cid}/resource_type/{type}/resource_id/{rid}/
func handleListResourceGroups(w http.ResponseWriter, r *http.Request, cid, rtype, rid string) {
	if !authz.RequireTenantMember(w, r, cid) {
		return
	}
	rows, err := db.Query(`SELECT a.id, a.group_id, g.name, a.permission
		FROM tenant_resource_group_assignment a
		LEFT JOIN tenant_company_group g ON g.id = a.group_id
		WHERE a.company_id=? AND a.resource_type=? AND a.resource_id=?`, cid, rtype, rid)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, gid string
		var gname, perm string
		var gnameNull interface{}
		if err := rows.Scan(&id, &gid, &gnameNull, &perm); err != nil {
			writeError(w, r, http.StatusInternalServerError, err.Error())
			return
		}
		gname, _ = gnameNull.(string)
		if gname == "" {
			gname = strings.TrimSpace(gname)
		}
		out = append(out, map[string]any{"id": id, "group_id": gid, "group_name": gname, "permission": perm})
	}
	writeJSON(w, http.StatusOK, out)
}
