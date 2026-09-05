-- ═══════════════════════════════════════════════════════════════
-- taskAuth 逻辑资源组 (v72 / ADR-0003)
-- 页面组(page) = 载体；UI 组件区域(ui_region) = 默认可授予资源组
-- 叶子 ui/api 仅挂在 ui_region；角色绑 page 时由 PDP 展开为子 region
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS auth_resource_group (
    id VARCHAR(64) PRIMARY KEY,
    group_key VARCHAR(128) NOT NULL,
    kind ENUM('page','ui_region') NOT NULL,
    parent_id VARCHAR(64) DEFAULT NULL,
    display_name VARCHAR(255) NOT NULL,
    route_prefix VARCHAR(255) DEFAULT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    is_system TINYINT NOT NULL DEFAULT 1,
    company_id VARCHAR(64) DEFAULT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_group_key (group_key),
    KEY idx_parent (parent_id),
    KEY idx_kind (kind)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_resource_member (
    id VARCHAR(64) PRIMARY KEY,
    resource_group_id VARCHAR(64) NOT NULL,
    member_kind ENUM('ui','api') NOT NULL,
    member_key VARCHAR(512) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_rg_member (resource_group_id, member_kind, member_key),
    KEY idx_member_key (member_key(191))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_role_resource_group (
    id VARCHAR(64) PRIMARY KEY,
    role_id VARCHAR(64) NOT NULL,
    resource_group_id VARCHAR(64) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_role_rg (role_id, resource_group_id),
    KEY idx_rg (resource_group_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ═══ 种子: 控制台页面组（group_key 对齐 TENANT_CONSOLE_NAV.key）═══
INSERT IGNORE INTO auth_resource_group (id, group_key, kind, parent_id, display_name, route_prefix, sort_order, is_system) VALUES
('rg-page-nav-projects', 'nav.projects', 'page', NULL, '项目列表', '/projects', 10, 1),
('rg-page-nav-work-panel', 'nav.work_panel', 'page', NULL, '工作面板', '/work-panel', 20, 1),
('rg-page-nav-image-market', 'nav.image_market', 'page', NULL, '镜像市场', '/image-market', 30, 1),
('rg-page-people-invite', 'people.invite', 'page', NULL, '邀请人', '/people/invite/', 40, 1),
('rg-page-people-manage', 'people.manage', 'page', NULL, '管理人员', '/people/manage/', 50, 1),
('rg-page-people-groups', 'people.groups', 'page', NULL, '管理分组', '/people/groups/', 60, 1),
('rg-page-people-access', 'people.access', 'page', NULL, '访问管理', '/people/access/', 70, 1),
('rg-page-settings-company', 'settings.company', 'page', NULL, '公司设置', '/settings/company/', 80, 1),
('rg-page-settings-cloud', 'settings.cloud', 'page', NULL, '云平台绑定', '/settings/cloud-platform/', 90, 1),
('rg-page-settings-gitlab', 'settings.gitlab', 'page', NULL, 'GitLab', '/settings/gitlab-connection/', 100, 1),
('rg-page-settings-task-panel', 'settings.task_panel', 'page', NULL, '工作空间管理', '/settings/task-panel/', 110, 1),
('rg-page-settings-feature-params', 'settings.feature_params', 'page', NULL, '智能体资源配置', '/settings/feature-params/', 120, 1),
('rg-page-settings-deliverable', 'settings.deliverable', 'page', NULL, '交付物体系设置', '/deliverable-systems/', 130, 1),
('rg-page-settings-status', 'settings.status', 'page', NULL, '进度体系设置', '/settings/status/', 140, 1),
('rg-page-billing-overview', 'billing.overview', 'page', NULL, '概览', '/billing/', 150, 1),
('rg-page-billing-orders', 'billing.orders', 'page', NULL, '订单列表', '/billing/orders/', 160, 1),
('rg-page-billing-transactions', 'billing.transactions', 'page', NULL, '交易流水', '/billing/transactions/', 170, 1),
('rg-page-billing-usage', 'billing.usage', 'page', NULL, '使用明细', '/billing/usage/', 180, 1);

-- 每个页面默认一个「整页」ui_region（P1 粒度；访问管理页再细分）
INSERT IGNORE INTO auth_resource_group (id, group_key, kind, parent_id, display_name, route_prefix, sort_order, is_system) VALUES
('rg-reg-nav-projects-main', 'nav.projects.main', 'ui_region', 'rg-page-nav-projects', '项目列表主区域', NULL, 1, 1),
('rg-reg-nav-work-panel-main', 'nav.work_panel.main', 'ui_region', 'rg-page-nav-work-panel', '工作面板主区域', NULL, 1, 1),
('rg-reg-nav-image-market-main', 'nav.image_market.main', 'ui_region', 'rg-page-nav-image-market', '镜像市场主区域', NULL, 1, 1),
('rg-reg-people-invite-main', 'people.invite.main', 'ui_region', 'rg-page-people-invite', '邀请人主区域', NULL, 1, 1),
('rg-reg-people-manage-main', 'people.manage.main', 'ui_region', 'rg-page-people-manage', '管理人员主区域', NULL, 1, 1),
('rg-reg-people-groups-main', 'people.groups.main', 'ui_region', 'rg-page-people-groups', '管理分组主区域', NULL, 1, 1),
('rg-reg-people-access-subject', 'people.access.subject_list', 'ui_region', 'rg-page-people-access', '主体列表', NULL, 1, 1),
('rg-reg-people-access-matrix', 'people.access.region_matrix', 'ui_region', 'rg-page-people-access', '资源区域勾选', NULL, 2, 1),
('rg-reg-people-access-save', 'people.access.save_actions', 'ui_region', 'rg-page-people-access', '保存操作', NULL, 3, 1),
('rg-reg-settings-company-main', 'settings.company.main', 'ui_region', 'rg-page-settings-company', '公司设置主区域', NULL, 1, 1),
('rg-reg-settings-cloud-main', 'settings.cloud.main', 'ui_region', 'rg-page-settings-cloud', '云平台绑定主区域', NULL, 1, 1),
('rg-reg-settings-gitlab-main', 'settings.gitlab.main', 'ui_region', 'rg-page-settings-gitlab', 'GitLab 主区域', NULL, 1, 1),
('rg-reg-settings-task-panel-main', 'settings.task_panel.main', 'ui_region', 'rg-page-settings-task-panel', '工作空间管理主区域', NULL, 1, 1),
('rg-reg-settings-feature-params-main', 'settings.feature_params.main', 'ui_region', 'rg-page-settings-feature-params', '智能体资源配置主区域', NULL, 1, 1),
('rg-reg-settings-deliverable-main', 'settings.deliverable.main', 'ui_region', 'rg-page-settings-deliverable', '交付物体系主区域', NULL, 1, 1),
('rg-reg-settings-status-main', 'settings.status.main', 'ui_region', 'rg-page-settings-status', '进度体系主区域', NULL, 1, 1),
('rg-reg-billing-overview-main', 'billing.overview.main', 'ui_region', 'rg-page-billing-overview', '计费概览主区域', NULL, 1, 1),
('rg-reg-billing-orders-main', 'billing.orders.main', 'ui_region', 'rg-page-billing-orders', '订单列表主区域', NULL, 1, 1),
('rg-reg-billing-transactions-main', 'billing.transactions.main', 'ui_region', 'rg-page-billing-transactions', '交易流水主区域', NULL, 1, 1),
('rg-reg-billing-usage-main', 'billing.usage.main', 'ui_region', 'rg-page-billing-usage', '使用明细主区域', NULL, 1, 1);

-- people.access API/UI 成员（P1 关键路径）
INSERT IGNORE INTO auth_resource_member (id, resource_group_id, member_kind, member_key) VALUES
('rm-pa-subject-ui', 'rg-reg-people-access-subject', 'ui', 'PeopleAccess.SubjectList'),
('rm-pa-subject-api-members', 'rg-reg-people-access-subject', 'api', 'GET /api/tenant/members/'),
('rm-pa-subject-api-groups', 'rg-reg-people-access-subject', 'api', 'GET /api/tenant/groups/'),
('rm-pa-matrix-ui', 'rg-reg-people-access-matrix', 'ui', 'PeopleAccess.RegionMatrix'),
('rm-pa-matrix-api-catalog', 'rg-reg-people-access-matrix', 'api', 'GET /api/auth/resource-groups/'),
('rm-pa-save-ui', 'rg-reg-people-access-save', 'ui', 'PeopleAccess.SaveButton'),
('rm-pa-save-api-put-rg', 'rg-reg-people-access-save', 'api', 'PUT /api/auth/roles/role_id/{rid}/resource-groups/'),
('rm-pa-save-api-get-rg', 'rg-reg-people-access-save', 'api', 'GET /api/auth/roles/role_id/{rid}/resource-groups/'),
('rm-pa-save-api-roles', 'rg-reg-people-access-save', 'api', 'POST /api/auth/roles/'),
('rm-pa-save-api-member-role', 'rg-reg-people-access-save', 'api', 'PUT /api/tenant/member-role/'),
('rm-pa-save-api-group-role', 'rg-reg-people-access-save', 'api', 'PUT /api/tenant/group-role/');

-- tenant_admin 绑定全部系统 page（PDP 展开子 region；亦可用 PDP 特判，双保险）
INSERT IGNORE INTO auth_role_resource_group (id, role_id, resource_group_id)
SELECT CONCAT('rrg-ta-', g.id), 'role-tenant-admin', g.id
FROM auth_resource_group g
WHERE g.is_system = 1 AND g.kind = 'page' AND g.company_id IS NULL;
