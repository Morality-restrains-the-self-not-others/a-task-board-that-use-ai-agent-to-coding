-- ═══════════════════════════════════════════════════════════════
-- taskTenantService: 组→角色继承 (v63 design §5.3)
-- 组内成员自动继承组角色
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS tenant_group_role (
    id VARCHAR(64) PRIMARY KEY,
    group_id VARCHAR(64) NOT NULL,
    role_name VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_group_role (group_id, role_name),
    CONSTRAINT fk_tgr_group FOREIGN KEY (group_id) REFERENCES tenant_company_group(id),
    INDEX idx_company (company_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
