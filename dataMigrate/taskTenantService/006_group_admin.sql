-- ═══════════════════════════════════════════════════════════════
-- taskTenantService: 小组管理员指派 (v63 design, 源自 v62)
-- group_id ↔ user_id: 谁管理哪个小组
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS tenant_group_admin (
    id VARCHAR(64) PRIMARY KEY,
    group_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    company_id VARCHAR(64) NOT NULL,
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_group_admin (group_id, user_id),
    CONSTRAINT fk_tga_group FOREIGN KEY (group_id) REFERENCES tenant_company_group(id),
    INDEX idx_company (company_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
