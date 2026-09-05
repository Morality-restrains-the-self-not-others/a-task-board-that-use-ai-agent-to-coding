-- ═══════════════════════════════════════════════════════════════
-- taskTenantService: 资源→小组分配 (v63 design, 源自 v62)
-- 租户把项目/云资源/任务分配给小组，组内成员自动获得访问权
-- ═══════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS tenant_resource_group_assignment (
    id VARCHAR(64) PRIMARY KEY,
    company_id VARCHAR(64) NOT NULL,
    resource_type ENUM('project','cloud','task','workspace') NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    group_id VARCHAR(64) NOT NULL,
    permission VARCHAR(32) NOT NULL DEFAULT 'view',  -- view | manage
    assigned_by VARCHAR(64) NOT NULL,
    assigned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_resource_group (resource_type, resource_id, group_id),
    CONSTRAINT fk_trga_group FOREIGN KEY (group_id) REFERENCES tenant_company_group(id),
    INDEX idx_group (group_id),
    INDEX idx_resource (resource_type, resource_id),
    INDEX idx_company (company_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
