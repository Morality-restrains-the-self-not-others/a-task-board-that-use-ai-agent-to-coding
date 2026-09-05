-- 019: 任务级只标记运行中机器数/容器数；清空任务级运行态（不兼容存量）
-- 评论 CSC 仍是 instance / last_runtime_status / URL 的唯一权威。

SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_configs' AND COLUMN_NAME = 'running_machine_count') = 0, 'ALTER TABLE cloud_server_configs ADD COLUMN running_machine_count INT NOT NULL DEFAULT 0', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;
SET @stmt = IF((SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'cloud_server_configs' AND COLUMN_NAME = 'running_container_count') = 0, 'ALTER TABLE cloud_server_configs ADD COLUMN running_container_count INT NOT NULL DEFAULT 0', 'SELECT 1'); PREPARE s FROM @stmt; EXECUTE s; DEALLOCATE PREPARE s;

UPDATE cloud_server_configs
SET instance_id = '',
    last_runtime_status = '',
    public_ip = '',
    server_url = '',
    business_api_endpoint = '',
    container_vscode_url = '',
    launch_request_id = ''
WHERE TRIM(COALESCE(comment_id, '')) = '';

UPDATE cloud_server_configs AS t
INNER JOIN (
  SELECT company_id, workspace_id, task_id,
    SUM(CASE
      WHEN TRIM(COALESCE(comment_id, '')) = '' THEN 0
      WHEN TRIM(COALESCE(instance_id, '')) LIKE 'mock-%' THEN 1
      WHEN TRIM(COALESCE(instance_id, '')) = '' THEN 0
      WHEN LOWER(TRIM(COALESCE(last_runtime_status, ''))) IN ('starting', 'pending', 'initializing', 'stopped', 'stopping', 'shutting-down', 'shuttingdown', 'released', 'terminated') THEN 0
      WHEN TRIM(COALESCE(last_runtime_status, '')) = '' OR LOWER(TRIM(last_runtime_status)) = 'running' THEN 1
      ELSE 0
    END) AS machines,
    SUM(CASE
      WHEN TRIM(COALESCE(comment_id, '')) = '' THEN 0
      WHEN TRIM(COALESCE(server_url, '')) = '' THEN 0
      WHEN TRIM(COALESCE(instance_id, '')) LIKE 'mock-%' THEN 1
      WHEN TRIM(COALESCE(instance_id, '')) = '' THEN 0
      WHEN LOWER(TRIM(COALESCE(last_runtime_status, ''))) IN ('starting', 'pending', 'initializing', 'stopped', 'stopping', 'shutting-down', 'shuttingdown', 'released', 'terminated') THEN 0
      WHEN TRIM(COALESCE(last_runtime_status, '')) = '' OR LOWER(TRIM(last_runtime_status)) = 'running' THEN 1
      ELSE 0
    END) AS containers
  FROM cloud_server_configs
  GROUP BY company_id, workspace_id, task_id
) AS s
  ON t.company_id = s.company_id
 AND t.workspace_id = s.workspace_id
 AND t.task_id = s.task_id
SET t.running_machine_count = s.machines,
    t.running_container_count = s.containers
WHERE TRIM(COALESCE(t.comment_id, '')) = '';
