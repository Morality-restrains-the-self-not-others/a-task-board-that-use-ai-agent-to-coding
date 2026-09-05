-- 030: GitLab 区域（region）支持 — 多区域资源隔离 + 管理员开通流程
-- 背景：
--   1. 不同区域的 GitLab 实例磁盘空间和流量独立，不共享
--   2. 用户下单购买某区域资源后，需管理员后台开通实施
--   3. 当前实例命名为 "腾讯上海五区" (tencent-shanghai-5)
-- 2026-08-01
--
-- 幂等设计：使用存储过程 guarded_add_column / guarded_exec 防止列已存在或操作重复导致迁移失败。

-- Step 0: 幂等列添加辅助存储过程
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

-- Step 1: billing_tenant_gitlab_resource 新增 region 列和 provisioning_status 列（幂等）
CALL guarded_add_column('billing_tenant_gitlab_resource', 'region', 'VARCHAR(64) NOT NULL DEFAULT ''tencent-shanghai-5'' AFTER tenant_id');
CALL guarded_add_column('billing_tenant_gitlab_resource', 'provisioning_status', 'VARCHAR(32) NOT NULL DEFAULT ''active'' AFTER traffic_used_gb');

-- Step 2: 重建主键 — 从 (tenant_id) 升级为 (tenant_id, region)（幂等，仅当旧主键存在时执行）
DELIMITER //
DROP PROCEDURE IF EXISTS guarded_migrate_pk;
CREATE PROCEDURE guarded_migrate_pk()
BEGIN
  DECLARE has_single_pk INT DEFAULT 0;
  SELECT COUNT(*) INTO has_single_pk FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_tenant_gitlab_resource'
      AND INDEX_NAME = 'PRIMARY' AND SEQ_IN_INDEX = 2;
  IF has_single_pk = 0 THEN
    -- 当前仍为单列主键 (tenant_id)，升级为复合主键
    ALTER TABLE billing_tenant_gitlab_resource
      DROP PRIMARY KEY,
      ADD PRIMARY KEY (tenant_id, region);
  END IF;
END //

DELIMITER ;
CALL guarded_migrate_pk();
DROP PROCEDURE IF EXISTS guarded_migrate_pk;

-- Step 3: 创建 billing_gitlab_region 区域元数据表
CREATE TABLE IF NOT EXISTS billing_gitlab_region (
  id           BIGINT NOT NULL PRIMARY KEY,
  name         VARCHAR(128) NOT NULL COMMENT '区域展示名称，如"腾讯上海五区"',
  slug         VARCHAR(64) NOT NULL UNIQUE COMMENT '区域标识，如 tencent-shanghai-5',
  description  VARCHAR(512) DEFAULT '' COMMENT '区域描述',
  gitlab_api_base VARCHAR(256) NOT NULL DEFAULT 'http://127.0.0.1:8012' COMMENT 'GitLab API 地址',
  gitlab_web_url  VARCHAR(256) NOT NULL DEFAULT 'https://gitlab.daydaymoney.com' COMMENT 'GitLab Web 地址',
  is_active    TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  sort_order   INT NOT NULL DEFAULT 0 COMMENT '排序',
  created_at   VARCHAR(32) NOT NULL DEFAULT '',
  updated_at   VARCHAR(32) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Step 4: 插入默认区域 "腾讯上海五区"
INSERT INTO billing_gitlab_region (id, name, slug, description, gitlab_api_base, gitlab_web_url, is_active, sort_order, created_at, updated_at)
VALUES (1, '腾讯上海五区', 'tencent-shanghai-5', '当前默认 GitLab 实例',
        'http://127.0.0.1:8012', 'https://gitlab.daydaymoney.com',
        1, 0, NOW(), NOW())
ON DUPLICATE KEY UPDATE name = VALUES(name);

DROP PROCEDURE IF EXISTS guarded_add_column;
