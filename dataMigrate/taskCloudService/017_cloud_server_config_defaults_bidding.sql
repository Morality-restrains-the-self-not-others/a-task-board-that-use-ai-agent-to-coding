-- 017: cloud_server_config_defaults 持久化竞价策略与 IoOptimized
-- 背景：实例筛选区已支持 IoOptimized/竞价策略，但「设置默认服务器启动配置」未采集这两项，
--       应用模版后仍用面板默认值。补充 io_optimized（'optimized'/'none'，默认 optimized）
--       与 spot_strategy（''/NoSpot/SpotWithPriceLimit/SpotAsPriceGo）列。OPT-20260812-020。

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_config_defaults' AND COLUMN_NAME = 'io_optimized') = 0, 'ALTER TABLE cloud_server_config_defaults ADD COLUMN io_optimized VARCHAR(16) NOT NULL DEFAULT \'optimized\'', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_config_defaults' AND COLUMN_NAME = 'spot_strategy') = 0, 'ALTER TABLE cloud_server_config_defaults ADD COLUMN spot_strategy VARCHAR(64) NOT NULL DEFAULT \'\'', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
