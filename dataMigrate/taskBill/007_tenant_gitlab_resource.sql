-- 租户系统内建 GitLab 资源配额（磁盘 GB + 流量预购 GB）
CREATE TABLE IF NOT EXISTS billing_tenant_gitlab_resource (
  tenant_id INT AUTO_INCREMENT PRIMARY KEY,
  disk_gb INTEGER NOT NULL DEFAULT 0,
  traffic_prepaid_gb INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
