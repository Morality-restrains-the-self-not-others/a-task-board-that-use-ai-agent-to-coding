-- 031: billing_gitlab_region 扩展 — 容量管理 + 管理员密钥 + 区域 CRUD 支持
-- 背景：
--   1. billing_gitlab_region 表缺 total_disk_gb / total_traffic_gb / allocated_disk_gb / allocated_traffic_gb 列
--   2. 需支持区域级 GitLab Admin Private Token 配置（用于不同区域不同密钥）
--   3. 管理后台需要完整的区域 CRUD 能力（创建/编辑/删除/启停）
-- 2026-08-02
--
-- 幂等设计：使用存储过程 guarded_add_column 防止列已存在导致迁移失败（→ 启动 crash loop）。

-- 幂等列添加辅助存储过程
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

-- 防御：若 billing_gitlab_region 尚未创建（如 030 部分失败），则先创建
CREATE TABLE IF NOT EXISTS billing_gitlab_region (
  id           BIGINT NOT NULL PRIMARY KEY,
  name         VARCHAR(128) NOT NULL COMMENT '区域展示名称',
  slug         VARCHAR(64) NOT NULL UNIQUE COMMENT '区域标识',
  description  VARCHAR(512) DEFAULT '' COMMENT '区域描述',
  gitlab_api_base VARCHAR(256) NOT NULL DEFAULT 'http://127.0.0.1:8012' COMMENT 'GitLab API 地址',
  gitlab_web_url  VARCHAR(256) NOT NULL DEFAULT 'https://gitlab.daydaymoney.com' COMMENT 'GitLab Web 地址',
  is_active    TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
  sort_order   INT NOT NULL DEFAULT 0 COMMENT '排序',
  created_at   VARCHAR(32) NOT NULL DEFAULT '',
  updated_at   VARCHAR(32) NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 幂等添加列
CALL guarded_add_column('billing_gitlab_region', 'total_disk_gb',    'BIGINT NOT NULL DEFAULT 0 COMMENT ''磁盘总容量 (GB)''');
CALL guarded_add_column('billing_gitlab_region', 'total_traffic_gb', 'BIGINT NOT NULL DEFAULT 0 COMMENT ''流量总容量 (GB)''');
CALL guarded_add_column('billing_gitlab_region', 'allocated_disk_gb',    'BIGINT NOT NULL DEFAULT 0 COMMENT ''已分配磁盘 (GB)''');
CALL guarded_add_column('billing_gitlab_region', 'allocated_traffic_gb', 'BIGINT NOT NULL DEFAULT 0 COMMENT ''已分配流量 (GB)''');
CALL guarded_add_column('billing_gitlab_region', 'admin_private_token', 'VARCHAR(256) NOT NULL DEFAULT '''' COMMENT ''GitLab 管理员 Private Token''');
CALL guarded_add_column('billing_gitlab_region', 'cloud_provider', 'VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''云服务商''');

DROP PROCEDURE IF EXISTS guarded_add_column;

-- 初始化现有区域（幂等）
UPDATE billing_gitlab_region SET
  total_disk_gb    = 100,
  total_traffic_gb = 1000,
  cloud_provider   = 'tencent'
WHERE slug = 'tencent-shanghai-5' AND total_disk_gb = 0;
