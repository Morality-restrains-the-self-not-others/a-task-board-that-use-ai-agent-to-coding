package main

import (
	"authz"
)

// fetchRoleResourceGroupCodesFn 接缝：角色 → region:*/page:* 注入码（不含粗码）。
var fetchRoleResourceGroupCodesFn = fetchRoleResourceGroupCodes

// fetchRoleResourceGroupCodes 按角色名解析逻辑资源组注入码（含 v73 effect）。
// tenant_admin → 全部系统 page/region（operate）；其它角色读 auth_role_resource_group，
// 绑 page 时展开为全部子 ui_region 并继承该 page 的 effect；同 key 取 MaxGrantEffect。
func fetchRoleResourceGroupCodes(roleNames []string, companyID string) ([]string, error) {
	if len(roleNames) == 0 || db == nil {
		return nil, nil
	}
	for _, n := range roleNames {
		if n == tenantAdminRole {
			return loadAllSystemRegionPageCodes()
		}
	}
	placeholders := inPlaceholders(len(roleNames))
	args := make([]any, 0, len(roleNames)+1)
	for _, n := range roleNames {
		args = append(args, n)
	}
	args = append(args, companyID)
	q := `
SELECT g.id, g.group_key, g.kind, COALESCE(g.parent_id,''), COALESCE(rrg.effect,'operate')
FROM auth_role r
JOIN auth_role_resource_group rrg ON rrg.role_id = r.id
JOIN auth_resource_group g ON g.id = rrg.resource_group_id
WHERE r.name IN (` + placeholders + `)
  AND (r.company_id IS NULL OR r.company_id = ?)`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type hit struct {
		id, key, kind, parentID, effect string
	}
	var hits []hit
	pageIDs := map[string]string{} // page id → effect
	for rows.Next() {
		var h hit
		if err := rows.Scan(&h.id, &h.key, &h.kind, &h.parentID, &h.effect); err != nil {
			return nil, err
		}
		h.effect = authz.NormalizeGrantEffect(h.effect)
		hits = append(hits, h)
		if h.kind == "page" {
			pageIDs[h.id] = authz.MaxGrantEffect(pageIDs[h.id], h.effect)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	regionEffects := map[string]string{}
	pageEffects := map[string]string{}
	pageKeyByID := map[string]string{}

	for _, h := range hits {
		if h.kind == "ui_region" {
			regionEffects[h.key] = authz.MaxGrantEffect(regionEffects[h.key], h.effect)
			if h.parentID != "" {
				if _, ok := pageIDs[h.parentID]; !ok {
					pageIDs[h.parentID] = h.effect
				} else {
					pageIDs[h.parentID] = authz.MaxGrantEffect(pageIDs[h.parentID], h.effect)
				}
			}
		}
		if h.kind == "page" {
			pageKeyByID[h.id] = h.key
			pageEffects[h.key] = authz.MaxGrantEffect(pageEffects[h.key], h.effect)
		}
	}

	// Expand page bindings → child ui_regions (inherit page effect)
	if len(pageIDs) > 0 {
		ids := make([]string, 0, len(pageIDs))
		for id := range pageIDs {
			ids = append(ids, id)
		}
		ph := inPlaceholders(len(ids))
		idArgs := make([]any, len(ids))
		for i, id := range ids {
			idArgs[i] = id
		}
		cq := `SELECT id, group_key, kind, COALESCE(parent_id,'') FROM auth_resource_group
			WHERE parent_id IN (` + ph + `) OR id IN (` + ph + `)`
		cargs := append(idArgs, idArgs...)
		crows, err := db.Query(cq, cargs...)
		if err != nil {
			return nil, err
		}
		for crows.Next() {
			var id, key, kind, parent string
			if err := crows.Scan(&id, &key, &kind, &parent); err != nil {
				crows.Close()
				return nil, err
			}
			if kind == "page" {
				pageKeyByID[id] = key
				if eff, ok := pageIDs[id]; ok {
					pageEffects[key] = authz.MaxGrantEffect(pageEffects[key], eff)
				}
			}
			if kind == "ui_region" {
				eff := authz.EffectView
				if parent != "" {
					if pe, ok := pageIDs[parent]; ok {
						eff = pe
					}
				}
				// Direct region grant already in regionEffects may be stronger
				regionEffects[key] = authz.MaxGrantEffect(regionEffects[key], eff)
				if parent != "" {
					if _, ok := pageIDs[parent]; !ok {
						pageIDs[parent] = eff
					}
				}
			}
		}
		crows.Close()
	}

	for id, eff := range pageIDs {
		if k, ok := pageKeyByID[id]; ok && k != "" {
			pageEffects[k] = authz.MaxGrantEffect(pageEffects[k], eff)
			continue
		}
		var pk string
		_ = db.QueryRow(`SELECT group_key FROM auth_resource_group WHERE id=? AND kind='page'`, id).Scan(&pk)
		if pk != "" {
			pageEffects[pk] = authz.MaxGrantEffect(pageEffects[pk], eff)
		}
	}

	// Resolve page keys from region parents
	if len(regionEffects) > 0 {
		rks := make([]string, 0, len(regionEffects))
		for k := range regionEffects {
			rks = append(rks, k)
		}
		ph := inPlaceholders(len(rks))
		rargs := make([]any, len(rks))
		for i, k := range rks {
			rargs[i] = k
		}
		rq := `SELECT r.group_key, COALESCE(p.group_key,'')
			FROM auth_resource_group r
			LEFT JOIN auth_resource_group p ON p.id = r.parent_id AND p.kind='page'
			WHERE r.kind='ui_region' AND r.group_key IN (` + ph + `)`
		rrows, err := db.Query(rq, rargs...)
		if err == nil {
			for rrows.Next() {
				var rk, pk string
				if err := rrows.Scan(&rk, &pk); err != nil {
					break
				}
				if pk != "" {
					pageEffects[pk] = authz.MaxGrantEffect(pageEffects[pk], regionEffects[rk])
				}
			}
			rrows.Close()
		}
	}

	return authz.EmitLogicalRGCodes(regionEffects, pageEffects), nil
}

func loadAllSystemRegionPageCodes() ([]string, error) {
	rows, err := db.Query(`SELECT group_key, kind FROM auth_resource_group WHERE is_system=1 AND company_id IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	regionEffects := map[string]string{}
	pageEffects := map[string]string{}
	for rows.Next() {
		var key, kind string
		if err := rows.Scan(&key, &kind); err != nil {
			return nil, err
		}
		switch kind {
		case "ui_region":
			regionEffects[key] = authz.EffectOperate
		case "page":
			pageEffects[key] = authz.EffectOperate
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return authz.EmitLogicalRGCodes(regionEffects, pageEffects), nil
}

// expandResourceGroupHits is a pure helper for unit tests: given granted
// (kind,key,parentPageKey,effect) and a map pageKey→[]regionKey for expansion,
// return sorted region:*/page:* codes including effect-qualified codes.
func expandResourceGroupHits(hits []struct{ Kind, Key, ParentPageKey, Effect string }, pageChildren map[string][]string) []string {
	regionEffects := map[string]string{}
	pageEffects := map[string]string{}
	for _, h := range hits {
		eff := authz.NormalizeGrantEffect(h.Effect)
		switch h.Kind {
		case "page":
			pageEffects[h.Key] = authz.MaxGrantEffect(pageEffects[h.Key], eff)
			for _, rk := range pageChildren[h.Key] {
				regionEffects[rk] = authz.MaxGrantEffect(regionEffects[rk], eff)
			}
		case "ui_region":
			regionEffects[h.Key] = authz.MaxGrantEffect(regionEffects[h.Key], eff)
			if h.ParentPageKey != "" {
				pageEffects[h.ParentPageKey] = authz.MaxGrantEffect(pageEffects[h.ParentPageKey], eff)
			}
		}
	}
	return authz.EmitLogicalRGCodes(regionEffects, pageEffects)
}
