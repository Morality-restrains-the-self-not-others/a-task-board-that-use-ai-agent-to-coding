-- OPT-20260806-046: 系统管理员容器镜像管理（孤儿页面后端缺失）
-- 表：system_admin_container_images（自维护容器镜像 CRUD 数据）
--      system_admin_container_image_server_assoc（容器镜像 ↔ 厂商服务器镜像关联）
-- 与 ai_provider 库的镜像市场表（vendor 侧）相互独立：本表供主站系统管理员
-- 后台「容器镜像列表」页面使用（上传/编辑/删除/设置运行环境）。

CREATE TABLE IF NOT EXISTS system_admin_container_images (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL DEFAULT '',
  description TEXT,
  image_url TEXT NOT NULL,
  size BIGINT UNSIGNED NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  created_by VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_sa_container_images_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS system_admin_container_image_server_assoc (
  container_image_id BIGINT UNSIGNED NOT NULL,
  platform_type VARCHAR(32) NOT NULL DEFAULT '',
  cloud_server_image_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  PRIMARY KEY (container_image_id, platform_type),
  KEY idx_sa_assoc_csi (cloud_server_image_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
