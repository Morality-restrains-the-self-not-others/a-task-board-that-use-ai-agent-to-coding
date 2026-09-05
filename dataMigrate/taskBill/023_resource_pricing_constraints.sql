-- 023: billing_unit 资源定价约束（不向后兼容：重置种子数据 + 清理脏数据）
-- min_consumption_cents：购买前租户累计核销消费最低金额（分），0=无门槛
-- requires_unit_type：购买前必须先持有的 billing_unit.unit_type，NULL=无前置依赖

-- Step 1: 添加新列（幂等：使用 guarded_add_column 防止列已存在导致迁移失败）
DELIMITER //
DROP PROCEDURE IF EXISTS guarded_add_column;
CREATE PROCEDURE guarded_add_column(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND COLUMN_NAME = col
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` ADD COLUMN `', col, '` ', col_def);
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END //

DELIMITER ;

CALL guarded_add_column('billing_unit', 'min_consumption_cents', 'bigint NOT NULL DEFAULT 0');
CALL guarded_add_column('billing_unit', 'requires_unit_type', 'varchar(20) NULL');

DROP PROCEDURE IF EXISTS guarded_add_column;

-- Step 2: 删除三个资源定价行，重新插入（确保约束字段干净）
DELETE FROM billing_unit WHERE unit_type IN ('server_start', 'gitlab_disk', 'gitlab_traffic');

INSERT INTO billing_unit (id, unit_type, name, price, unit, is_active, min_consumption_cents, requires_unit_type, created_at, updated_at)
VALUES
  (990000000000000001, 'server_start',   '任务',            33,   '次',    1, 0,     NULL,            NOW(), NOW()),
  (990000000000000002, 'gitlab_disk',    'GitLab 磁盘',     800,  'GB/月', 1, 0,     NULL,            NOW(), NOW()),
  (990000000000000003, 'gitlab_traffic', 'GitLab 流量费',   100,  'GB',    1, 0,     'gitlab_disk',  NOW(), NOW());

-- Step 3: 清理违反新约束的脏数据
-- 流量费须先有磁盘：移除仅有流量无磁盘的租户 GitLab 资源记录中的流量配额
UPDATE billing_tenant_gitlab_resource
SET traffic_prepaid_gb = 0, updated_at = NOW()
WHERE traffic_prepaid_gb > 0 AND (disk_gb IS NULL OR disk_gb = 0);
