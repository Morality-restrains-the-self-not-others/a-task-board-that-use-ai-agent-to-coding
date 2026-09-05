-- 043: seed 可插拔区域 tencent-sh-1（上海独立 GitLab CE）
-- 现网 tencent-shanghai-5 保留。本行 is_active=1 可售；admin token 由 SystemAdmin 后填。
-- api_base 必须从 INFRA（taskBill）可达。SH GitLab 不在 10.2.150.68 本机，
-- 禁止用 127.0.0.1:8014（那是 SH 宿主机 loopback）。公网 HTTPS 与 web_url 同主机。

INSERT INTO billing_gitlab_region (
  id, name, slug, description,
  gitlab_api_base, gitlab_web_url,
  is_active, sort_order,
  total_disk_gb, total_traffic_gb,
  allocated_disk_gb, allocated_traffic_gb,
  admin_private_token, cloud_provider,
  created_at, updated_at
) VALUES (
  871590000000000001,
  '腾讯上海一区',
  'tencent-sh-1',
  '上海机独立 GitLab CE（gitlab-tencent-sh-1）；精简内存；SSH :2223',
  'https://gitlab-tencent-sh-1.daydaymoney.com',
  'https://gitlab-tencent-sh-1.daydaymoney.com',
  1, 10,
  50, 500,
  0, 0,
  '',
  'tencent',
  NOW(), NOW()
) ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  gitlab_api_base = VALUES(gitlab_api_base),
  gitlab_web_url = VALUES(gitlab_web_url),
  cloud_provider = VALUES(cloud_provider);
