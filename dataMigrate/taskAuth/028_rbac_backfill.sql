-- ═══════════════════════════════════════════════════════════════
-- taskAuth RBAC 硬切换回填 (v63 design §5.2)
-- is_superuser → super_admin；is_staff → employee
-- 回填前备份旧值到归档表；回填后代码层删除 is_admin/is_superuser 判断
-- ═══════════════════════════════════════════════════════════════

-- 1. 归档旧值 (可逆性保障，硬切换无双轨)
CREATE TABLE IF NOT EXISTS auth_user_legacy_archive (
    user_id VARCHAR(64) PRIMARY KEY,
    is_superuser TINYINT NOT NULL DEFAULT 0,
    is_staff TINYINT NOT NULL DEFAULT 0,
    archived_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO auth_user_legacy_archive (user_id, is_superuser, is_staff)
SELECT id, is_superuser, is_staff FROM auth_user WHERE is_superuser = 1 OR is_staff = 1;

-- 2. 回填 is_superuser → super_admin (platform, company_id=NULL)
INSERT IGNORE INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
SELECT CONCAT('bf-sa-', id), id, 'role-super-admin', NULL, 'system', NOW()
FROM auth_user WHERE is_superuser = 1;

-- 3. 回填 is_staff 且非 superuser → employee
INSERT IGNORE INTO auth_user_role (id, user_id, role_id, company_id, assigned_by, assigned_at)
SELECT CONCAT('bf-emp-', id), id, 'role-employee', NULL, 'system', NOW()
FROM auth_user WHERE is_staff = 1 AND is_superuser = 0;

-- 4. 校验: 回填遗漏检查 (应返回 0 行)
-- SELECT id FROM auth_user WHERE is_superuser = 1
--   AND id NOT IN (SELECT user_id FROM auth_user_role WHERE role_id = 'role-super-admin');
-- SELECT id FROM auth_user WHERE is_staff = 1 AND is_superuser = 0
--   AND id NOT IN (SELECT user_id FROM auth_user_role WHERE role_id = 'role-employee');
