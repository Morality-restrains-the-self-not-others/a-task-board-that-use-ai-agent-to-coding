-- OPT-052 跟进: github-credential-status/approve Go 重新实现（Django saas-backend
-- retired 2026-07-30，原绑定存 Django 库已随退役删除）。
-- 表：cloud_task_repo_github_bindings（任务级 GitHub PR 授权账号绑定）
--   对称既有 task_task 库 task_repo_identities（repo → git_identity_id）模式；
--   本表位于 task_cloud 库（与 cloud_comment_container_bindings 同类任务级绑定，
--   由 taskCloudService 直接读写，避免跨服务 HTTP 往返）。

CREATE TABLE IF NOT EXISTS cloud_task_repo_github_bindings (
  id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  repo_url VARCHAR(512) NOT NULL,
  repo_slug VARCHAR(255) NOT NULL,
  github_user_id VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_ctrgb_task_repo (task_id, repo_url),
  KEY idx_ctrgb_task (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
