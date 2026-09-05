-- 053: GitLab 赠送批次补 region 列，供账单页按区拆分「赠送/购买」
-- billing_resource_grant 历史 gitlab_disk/gitlab_traffic 赠送批次无区域归属，
-- 只能经 order_item.region 回填；仅当该订单同一资源类型唯一区域时回填，避免多区域同批次归属歧义。

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

DROP PROCEDURE IF EXISTS guarded_add_index;
CREATE PROCEDURE guarded_add_index(IN tbl VARCHAR(64), IN idx VARCHAR(64), IN cols VARCHAR(255))
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND INDEX_NAME = idx
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` ADD INDEX `', idx, '` (', cols, ')');
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END //
DELIMITER ;

CALL guarded_add_column(
  'billing_resource_grant',
  'region',
  'VARCHAR(64) NOT NULL DEFAULT '''' COMMENT ''GitLab 区域 slug；仅 gitlab_disk/gitlab_traffic 批次使用'''
);

CALL guarded_add_index('billing_resource_grant', 'bill_res_grant_region', 'tenant_id, resource_type, source_kind, region');

-- 回填：只处理「该订单同一资源类型唯一区域」的遗留赠送批次（含数量拆行但区域一致的情形）
UPDATE billing_resource_grant g
JOIN (
  SELECT order_id, resource_type, MAX(region) AS region
  FROM billing_resource_order_item
  WHERE resource_type IN ('gitlab_disk', 'gitlab_traffic') AND region != ''
  GROUP BY order_id, resource_type
  HAVING COUNT(DISTINCT region) = 1
) oi ON oi.order_id = g.order_id AND oi.resource_type = g.resource_type
SET g.region = oi.region
WHERE g.region = ''
  AND g.order_id IS NOT NULL
  AND g.resource_type IN ('gitlab_disk', 'gitlab_traffic');

DROP PROCEDURE IF EXISTS guarded_add_column;
DROP PROCEDURE IF EXISTS guarded_add_index;
