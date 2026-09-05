-- 048_dedupe_platform_user_roles.sql
-- MySQL UNIQUE KEY uk_urc (user_id, role_id, company_id) 将 NULL 视为互不相等，
-- 导致 ensurePlatformRoleRows / INSERT IGNORE 每次写入新主键时堆叠平台角色行。
-- GET /api/auth/user-roles/ 曾对同一 super_admin 返回数十条重复。
-- 本迁移：折叠存量 NULL company_id 重复行，并用生成列把 NULL 归一成 '' 后加唯一键。

DELETE FROM auth_user_role
WHERE
    company_id IS NULL
    AND id NOT IN (
        SELECT id FROM (
            SELECT MIN(id) AS id
            FROM auth_user_role
            WHERE
                company_id IS NULL
            GROUP BY user_id, role_id
        ) AS keep_ids
    );

ALTER TABLE auth_user_role
ADD COLUMN company_id_key VARCHAR(64)
GENERATED ALWAYS AS (COALESCE(company_id, '')) STORED,
ADD UNIQUE KEY uk_user_role_company_key (user_id, role_id, company_id_key);

ALTER TABLE auth_user_role DROP INDEX uk_urc;
