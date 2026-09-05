-- 镜像市场全局设置：厂商申请审核开关（默认开启，兼容现网审核流）。
-- 单行配置表（id=1），非时间累积型，无需分区。
CREATE TABLE IF NOT EXISTS ai_provider_marketplace_settings (
  id TINYINT NOT NULL PRIMARY KEY,
  vendor_application_review_enabled TINYINT(1) NOT NULL DEFAULT 1,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO ai_provider_marketplace_settings (id, vendor_application_review_enabled)
VALUES (1, 1);
