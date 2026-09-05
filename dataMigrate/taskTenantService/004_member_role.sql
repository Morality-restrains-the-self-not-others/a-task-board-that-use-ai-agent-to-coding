-- ═══════════════════════════════════════════════════════════════
-- taskTenantService: 租户成员直接角色 (v63 design §5.3)
-- member_id ↔ 角色名 (内置或自定义角色均按名引用)
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS tenant_member_role (
    id VARCHAR(64) PRIMARY KEY,
    member_id VARCHAR(64) NOT NULL,
    role_name VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_member_role (member_id, role_name),
    CONSTRAINT fk_tmr_member FOREIGN KEY (member_id) REFERENCES tenant_company_member(id),
    INDEX idx_company (company_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 硬切换回填: is_admin=1 → tenant_admin 角色 (成员行存在时)
INSERT IGNORE INTO tenant_member_role (id, member_id, role_name, company_id, assigned_by, assigned_at)
SELECT CONCAT('bf-tad-', m.id), m.id, 'tenant_admin', m.company_id, 'system', NOW()
FROM tenant_company_member m
WHERE m.is_admin = 1;

-- 硬切换回填: 公司创建者 (creator_id) → tenant_admin（即使 is_admin=0，消除 creator 特判依赖）
INSERT IGNORE INTO tenant_member_role (id, member_id, role_name, company_id, assigned_by, assigned_at)
SELECT CONCAT('bf-cre-', m.id), m.id, 'tenant_admin', m.company_id, 'system', NOW()
FROM tenant_company_member m
JOIN tenant_company c ON c.id = m.company_id
WHERE c.creator_id = m.user_id
  AND NOT EXISTS (SELECT 1 FROM tenant_member_role r WHERE r.member_id = m.id AND r.role_name = 'tenant_admin');
