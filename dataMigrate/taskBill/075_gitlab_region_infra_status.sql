-- 075: GitLab 区域 infra_status + seed 阿里云可售待建节点
-- pending_node：购买页可选，禁止 Admin API，直至运维建机挂载并标记 ready。
-- 存量腾讯云行 DEFAULT ready。重复执行不覆盖已改为 ready 的 infra_status。

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

CALL guarded_add_column(
  'billing_gitlab_region',
  'infra_status',
  'VARCHAR(32) NOT NULL DEFAULT ''ready'' COMMENT ''ready=实例已部署 pending_node=可售待人工建节点'''
);

DROP PROCEDURE IF EXISTS guarded_add_column;

INSERT INTO billing_gitlab_region (
  id, name, slug, description,
  gitlab_api_base, gitlab_web_url,
  is_active, sort_order,
  total_disk_gb, total_traffic_gb,
  allocated_disk_gb, allocated_traffic_gb,
  admin_private_token, cloud_provider, access_mode, infra_status,
  created_at, updated_at
) VALUES
  (871592000000000001, '阿里云杭州（华东1）', 'aliyun-cn-hangzhou', '阿里云 cn-hangzhou；节点待人工创建并挂载后开通', '', '', 1, 100, 0, 0, 0, 0, '', 'aliyun', 'release', 'pending_node', NOW(), NOW()),
  (871592000000000002, '阿里云上海（华东2）', 'aliyun-cn-shanghai', '阿里云 cn-shanghai；节点待人工创建并挂载后开通', '', '', 1, 101, 0, 0, 0, 0, '', 'aliyun', 'release', 'pending_node', NOW(), NOW()),
  (871592000000000003, '阿里云青岛（华北1）', 'aliyun-cn-qingdao', '阿里云 cn-qingdao；节点待人工创建并挂载后开通', '', '', 1, 102, 0, 0, 0, 0, '', 'aliyun', 'release', 'pending_node', NOW(), NOW()),
  (871592000000000004, '阿里云北京（华北2）', 'aliyun-cn-beijing', '阿里云 cn-beijing；节点待人工创建并挂载后开通', '', '', 1, 103, 0, 0, 0, 0, '', 'aliyun', 'release', 'pending_node', NOW(), NOW()),
  (871592000000000005, '阿里云张家口（华北3）', 'aliyun-cn-zhangjiakou', '阿里云 cn-zhangjiakou；节点待人工创建并挂载后开通', '', '', 1, 104, 0, 0, 0, 0, '', 'aliyun', 'release', 'pending_node', NOW(), NOW()),
  (871592000000000006, '阿里云深圳（华南1）', 'aliyun-cn-shenzhen', '阿里云 cn-shenzhen；节点待人工创建并挂载后开通', '', '', 1, 105, 0, 0, 0, 0, '', 'aliyun', 'release', 'pending_node', NOW(), NOW()),
  (871592000000000007, '阿里云广州（华南3）', 'aliyun-cn-guangzhou', '阿里云 cn-guangzhou；节点待人工创建并挂载后开通', '', '', 1, 106, 0, 0, 0, 0, '', 'aliyun', 'release', 'pending_node', NOW(), NOW()),
  (871592000000000008, '阿里云成都（西南1）', 'aliyun-cn-chengdu', '阿里云 cn-chengdu；节点待人工创建并挂载后开通', '', '', 1, 107, 0, 0, 0, 0, '', 'aliyun', 'release', 'pending_node', NOW(), NOW()),
  (871592000000000009, '阿里云香港', 'aliyun-cn-hongkong', '阿里云 cn-hongkong；节点待人工创建并挂载后开通', '', '', 1, 108, 0, 0, 0, 0, '', 'aliyun', 'release', 'pending_node', NOW(), NOW()),
  (871592000000000010, '阿里云新加坡', 'aliyun-ap-southeast-1', '阿里云 ap-southeast-1；节点待人工创建并挂载后开通', '', '', 1, 109, 0, 0, 0, 0, '', 'aliyun', 'release', 'pending_node', NOW(), NOW())
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  cloud_provider = VALUES(cloud_provider),
  is_active = VALUES(is_active),
  sort_order = VALUES(sort_order);
