-- ═══════════════════════════════════════════════════════════════
-- taskAuth RBAC: 角色 + 权限 + 关联表 (v63 merged design §5.1)
-- 5 内置角色 (含 group_admin) + 23 权限码种子
-- 自定义租户角色: auth_role.company_id 非 NULL = 该租户自定义角色
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS auth_role (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    level ENUM('platform','tenant') NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    is_system TINYINT NOT NULL DEFAULT 0,
    company_id VARCHAR(64) DEFAULT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_name_company (name, company_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_permission (
    id VARCHAR(64) PRIMARY KEY,
    codename VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL,
    level ENUM('platform','tenant') NOT NULL,
    description TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_role_permission (
    id VARCHAR(64) PRIMARY KEY,
    role_id VARCHAR(64) NOT NULL,
    permission_id VARCHAR(64) NOT NULL,
    UNIQUE KEY uk_rp (role_id, permission_id),
    CONSTRAINT fk_rp_role FOREIGN KEY (role_id) REFERENCES auth_role(id),
    CONSTRAINT fk_rp_perm FOREIGN KEY (permission_id) REFERENCES auth_permission(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_user_role (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    role_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) DEFAULT NULL,
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME DEFAULT NULL,
    UNIQUE KEY uk_urc (user_id, role_id, company_id),
    CONSTRAINT fk_aur_user FOREIGN KEY (user_id) REFERENCES auth_user(id),
    CONSTRAINT fk_aur_role FOREIGN KEY (role_id) REFERENCES auth_role(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ═══ 种子: 5 内置系统角色 (is_system=1, 锁定) ═══
INSERT IGNORE INTO auth_role (id, name, display_name, level, priority, is_system, company_id, description) VALUES
('role-super-admin',  'super_admin',  '超级管理员', 'platform', 100, 1, NULL, '平台全局管理、员工管理、系统配置'),
('role-employee',     'employee',     '平台员工',   'platform', 50,  1, NULL, '日常运维、客服、租户支持'),
('role-tenant-admin', 'tenant_admin', '租户管理员', 'tenant',   100, 1, NULL, '公司成员/组/项目/云资源/计费管理'),
('role-group-admin',  'group_admin',  '小组管理员', 'tenant',   75,  1, NULL, '管理指定小组内成员、查看组关联资源'),
('role-member',       'member',       '成员',       'tenant',   50,  1, NULL, '协作使用 SaaS 功能');

-- ═══ 种子: 23 权限码 (6 platform + 14 tenant 资源 + 3 tenant 组) ═══
INSERT IGNORE INTO auth_permission (id, codename, name, resource_type, action, level) VALUES
-- Platform (6)
('perm-plat-manage',   'platform:manage',   '平台全局管理',   'platform',  'manage',      'platform'),
('perm-tenant-audit',  'tenant:audit',      '跨租户审计',     'platform',  'audit',       'platform'),
('perm-impersonate',   'user:impersonate',  '模拟用户登录',   'user',      'impersonate', 'platform'),
('perm-sys-config',    'system:config',     '系统配置管理',   'system',    'manage',      'platform'),
('perm-bill-audit',    'billing:audit',     '跨租户账单审计', 'billing',   'audit',       'platform'),
('perm-emp-manage',    'employee:manage',   '员工管理',       'employee',  'manage',      'platform'),
-- Tenant resources (14)
('perm-company-manage','company:manage',    '公司信息管理',   'company',   'manage',      'tenant'),
('perm-company-view',  'company:view',      '公司信息查看',   'company',   'view',        'tenant'),
('perm-member-manage', 'member:manage',     '成员管理',       'member',    'manage',      'tenant'),
('perm-member-view',   'member:view',       '成员列表查看',   'member',    'view',        'tenant'),
('perm-group-manage',  'group:manage',      '分组管理',       'group',     'manage',      'tenant'),
('perm-project-manage','project:manage',    '项目管理',       'project',   'manage',      'tenant'),
('perm-project-view',  'project:view',      '项目查看',       'project',   'view',        'tenant'),
('perm-task-manage',   'task:manage',       '任务管理',       'task',      'manage',      'tenant'),
('perm-task-view',     'task:view',         '任务查看',       'task',      'view',        'tenant'),
('perm-cloud-manage',  'cloud:manage',      '云资源管理',     'cloud',     'manage',      'tenant'),
('perm-cloud-view',    'cloud:view',        '云资源查看',     'cloud',     'view',        'tenant'),
('perm-billing-manage','billing:manage',    '计费充值管理',   'billing',   'manage',      'tenant'),
('perm-billing-view',  'billing:view',      '账单查看',       'billing',   'view',        'tenant'),
('perm-workspace-mng', 'workspace:manage',  '工作空间管理',   'workspace', 'manage',      'tenant'),
-- Tenant groups (3)
('perm-grp-members',   'group-members:manage',  '组内成员管理', 'group',    'members:manage', 'tenant'),
('perm-grp-res-view',  'group-resources:view',  '查看组资源',   'group',    'resources:view', 'tenant'),
('perm-grp-res-mng',   'group-resources:manage','管理组资源',   'group',    'resources:manage','tenant');

-- ═══ 种子: 角色→权限关联 ═══
-- super_admin: 全部 platform 码
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-sa-', id), 'role-super-admin', id FROM auth_permission WHERE level='platform';

-- employee: platform 中除 platform:manage, system:config, employee:manage
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-emp-', id), 'role-employee', id FROM auth_permission
WHERE level='platform' AND codename NOT IN ('platform:manage','system:config','employee:manage');

-- tenant_admin: 全部 tenant 码 (14 资源 + 3 组)
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-ta-', id), 'role-tenant-admin', id FROM auth_permission WHERE level='tenant';

-- group_admin: 组内成员管理 + 查看组资源 (组范围判定在 handler 层)
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-ga-', id), 'role-group-admin', id FROM auth_permission
WHERE codename IN ('group-members:manage','group-resources:view');

-- member: 全部 view 码 + task:manage
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
SELECT CONCAT('rp-mb-', id), 'role-member', id FROM auth_permission WHERE level='tenant' AND action='view';
INSERT IGNORE INTO auth_role_permission (id, role_id, permission_id)
VALUES ('rp-mb-task-manage', 'role-member', 'perm-task-manage');
