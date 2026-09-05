-- 011_deliverable_default_tenant.sql — 租户默认交付物体系独立存储
-- 根因（OPT-20260808-004）：project_progress_systems_default_tenant 表 UNIQUE(tenant_id)，
-- 每租户仅一行，被交付物体系默认（company-created intent 1）与进度体系默认（intent 2）
-- 共用。公司创建链上 intent 2 的 upsert 覆盖 intent 1 写入的行 → 租户默认交付物体系
-- 恒丢失，页面恒显示「暂无默认交付物体系」。
-- 修复：新建按租户独立的交付物默认表，与进度默认表隔离存储；并 backfill 存量租户。
-- 幂等：CREATE TABLE IF NOT EXISTS + INSERT IGNORE + 稳定 ID（L1 由 data_migrate_log 保证）。

CREATE TABLE IF NOT EXISTS project_deliverable_systems_default_tenant (
  id varchar(64) NOT NULL,
  tenant_id varchar(64) NOT NULL,
  deliverable_id varchar(64) NOT NULL,
  created_at datetime DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_ddst_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 迁移历史数据：旧表曾以 target_type='deliverable' 存储过交付物默认（若存在）
INSERT IGNORE INTO project_deliverable_systems_default_tenant(id, tenant_id, deliverable_id)
SELECT CONCAT('ddst_', tenant_id), tenant_id, target_id
FROM project_progress_systems_default_tenant
WHERE target_type = 'deliverable' AND target_id IS NOT NULL AND target_id <> '';

-- 存量租户 backfill：为尚无默认交付物体系的租户补齐系统级默认（ds_default_global）。
-- 租户全集 = 有工作空间的租户（列名 company_id）∪ 有公司级交付物体系的公司 ∪ 已有任意默认行的租户。
INSERT IGNORE INTO project_deliverable_systems_default_tenant(id, tenant_id, deliverable_id)
SELECT CONCAT('ddst_', t.tenant_id), t.tenant_id, 'ds_default_global'
FROM (
  SELECT company_id AS tenant_id FROM project_workspace_entries WHERE company_id IS NOT NULL AND company_id <> '' GROUP BY company_id
  UNION
  SELECT company_id AS tenant_id FROM project_deliverable_systems WHERE company_id IS NOT NULL AND company_id <> '' GROUP BY company_id
  UNION
  SELECT tenant_id FROM project_progress_systems_default_tenant WHERE tenant_id IS NOT NULL AND tenant_id <> '' GROUP BY tenant_id
) t;
