-- 026: CSC 持久化实例带宽，授权缺失回退仍可展示带宽行
-- 背景：DescribeInstances 已映射带宽并展示；auth_missing 回退只拼 Status/公网 IP，
--       CSC 不存带宽，授权失效时评论运行态网格缺流量带宽。
-- 存量数据无需回填：下次任一成功 Describe 即自动持久化。

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_configs' AND COLUMN_NAME = 'internet_charge_type') = 0, 'ALTER TABLE cloud_server_configs ADD COLUMN internet_charge_type VARCHAR(32) NOT NULL DEFAULT \'\'', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_configs' AND COLUMN_NAME = 'internet_max_bandwidth_out') = 0, 'ALTER TABLE cloud_server_configs ADD COLUMN internet_max_bandwidth_out INT NOT NULL DEFAULT 0', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
