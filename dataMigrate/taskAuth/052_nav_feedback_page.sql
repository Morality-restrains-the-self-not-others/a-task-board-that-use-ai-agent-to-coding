-- 租户控制台「意见与建议」page ⊃ ui_region（v72 / ADR-0003）
-- 内置三角色默认可见；访问管理可关掉。

INSERT IGNORE INTO auth_permission (id, codename, name, resource_type, action, level) VALUES
('perm-feedback-view', 'feedback:view', '意见与建议查看', 'feedback', 'view', 'tenant');

INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT 'rp-ta-perm-feedback-view', 'role-tenant-admin', 'perm-feedback-view'
FROM DUAL WHERE EXISTS (SELECT 1 FROM auth_role WHERE id='role-tenant-admin');
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT 'rp-mb-perm-feedback-view', 'role-member', 'perm-feedback-view'
FROM DUAL WHERE EXISTS (SELECT 1 FROM auth_role WHERE id='role-member');
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT 'rp-ga-perm-feedback-view', 'role-group-admin', 'perm-feedback-view'
FROM DUAL WHERE EXISTS (SELECT 1 FROM auth_role WHERE id='role-group-admin');

INSERT IGNORE INTO auth_resource_group (id, group_key, kind, parent_id, display_name, route_prefix, sort_order, is_system) VALUES
('rg-page-nav-feedback', 'nav.feedback', 'page', NULL, '意见与建议', NULL, 25, 1);

INSERT IGNORE INTO auth_resource_group (id, group_key, kind, parent_id, display_name, route_prefix, sort_order, is_system) VALUES
('rg-reg-nav-feedback-main', 'nav.feedback.main', 'ui_region', 'rg-page-nav-feedback', '意见与建议链接', NULL, 1, 1);

INSERT IGNORE INTO auth_resource_member (id, resource_group_id, member_kind, member_key) VALUES
('rm-fb-main-ui', 'rg-reg-nav-feedback-main', 'ui', 'TenantConsoleFeedbackNav.Main'),
('rm-fb-main-api', 'rg-reg-nav-feedback-main', 'api', 'GET /api/tenant/{tenantId}/billing/feedback-links/');

INSERT IGNORE INTO auth_role_resource_group (id, role_id, resource_group_id)
SELECT 'rrg-ta-rg-page-nav-feedback', 'role-tenant-admin', 'rg-page-nav-feedback'
FROM DUAL WHERE EXISTS (SELECT 1 FROM auth_role WHERE id='role-tenant-admin');
INSERT IGNORE INTO auth_role_resource_group (id, role_id, resource_group_id)
SELECT 'rrg-mb-rg-page-nav-feedback', 'role-member', 'rg-page-nav-feedback'
FROM DUAL WHERE EXISTS (SELECT 1 FROM auth_role WHERE id='role-member');
INSERT IGNORE INTO auth_role_resource_group (id, role_id, resource_group_id)
SELECT 'rrg-ga-rg-page-nav-feedback', 'role-group-admin', 'rg-page-nav-feedback'
FROM DUAL WHERE EXISTS (SELECT 1 FROM auth_role WHERE id='role-group-admin');
