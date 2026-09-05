-- ═══════════════════════════════════════════════════════════════
-- taskAuth v75 — 角色管理页 people.roles（page ⊃ ui_region）
-- ═══════════════════════════════════════════════════════════════

INSERT IGNORE INTO auth_resource_group (id, group_key, kind, parent_id, display_name, route_prefix, sort_order, is_system) VALUES
('rg-page-people-roles', 'people.roles', 'page', NULL, '角色管理', '/people/roles/', 75, 1);

INSERT IGNORE INTO auth_resource_group (id, group_key, kind, parent_id, display_name, route_prefix, sort_order, is_system) VALUES
('rg-reg-people-roles-main', 'people.roles.main', 'ui_region', 'rg-page-people-roles', '角色列表与编辑', NULL, 1, 1),
('rg-reg-people-roles-save', 'people.roles.save_actions', 'ui_region', 'rg-page-people-roles', '保存角色与授权', NULL, 2, 1);

INSERT IGNORE INTO auth_resource_member (id, resource_group_id, member_kind, member_key) VALUES
('rm-pr-main-ui', 'rg-reg-people-roles-main', 'ui', 'PeopleRoles.Main'),
('rm-pr-main-api-roles', 'rg-reg-people-roles-main', 'api', 'GET /api/auth/roles/company_id/{cid}/'),
('rm-pr-main-api-catalog', 'rg-reg-people-roles-main', 'api', 'GET /api/auth/resource-groups/'),
('rm-pr-save-ui', 'rg-reg-people-roles-save', 'ui', 'PeopleRoles.SaveActions'),
('rm-pr-save-api-post', 'rg-reg-people-roles-save', 'api', 'POST /api/auth/roles/'),
('rm-pr-save-api-put', 'rg-reg-people-roles-save', 'api', 'PUT /api/auth/roles/role_id/{rid}/'),
('rm-pr-save-api-del', 'rg-reg-people-roles-save', 'api', 'DELETE /api/auth/roles/role_id/{rid}/'),
('rm-pr-save-api-rg', 'rg-reg-people-roles-save', 'api', 'PUT /api/auth/roles/role_id/{rid}/resource-groups/');

-- tenant_admin 绑定新 page（幂等）
INSERT IGNORE INTO auth_role_resource_group (id, role_id, resource_group_id)
SELECT 'rrg-ta-rg-page-people-roles', 'role-tenant-admin', 'rg-page-people-roles'
FROM DUAL
WHERE EXISTS (SELECT 1 FROM auth_role WHERE id='role-tenant-admin');
