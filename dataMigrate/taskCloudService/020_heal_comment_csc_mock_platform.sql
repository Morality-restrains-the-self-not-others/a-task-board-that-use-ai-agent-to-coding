-- 020: 评论 CSC 非法/空的 platform、region、authorization_id 按字段从任务模板补齐。
-- 不覆盖评论级已有合法值；不改 instance_id / server_url / public_ip。

UPDATE cloud_server_configs AS c
INNER JOIN cloud_server_configs AS t
  ON t.company_id = c.company_id
 AND t.workspace_id = c.workspace_id
 AND t.task_id = c.task_id
 AND TRIM(COALESCE(t.comment_id, '')) = ''
SET
  c.platform = CASE
    WHEN LOWER(TRIM(COALESCE(c.platform, ''))) IN ('', 'mock', 'relay-local')
      OR LOWER(TRIM(COALESCE(c.platform, ''))) LIKE 'relay%'
    THEN t.platform ELSE c.platform END,
  c.region = CASE WHEN TRIM(COALESCE(c.region, '')) = '' THEN t.region ELSE c.region END,
  c.zone_id = CASE WHEN TRIM(COALESCE(c.zone_id, '')) = '' THEN t.zone_id ELSE c.zone_id END,
  c.authorization_id = CASE WHEN TRIM(COALESCE(c.authorization_id, '')) = '' THEN t.authorization_id ELSE c.authorization_id END,
  c.security_group_id = CASE WHEN TRIM(COALESCE(c.security_group_id, '')) = '' THEN t.security_group_id ELSE c.security_group_id END,
  c.vswitch_id = CASE WHEN TRIM(COALESCE(c.vswitch_id, '')) = '' THEN t.vswitch_id ELSE c.vswitch_id END
WHERE TRIM(COALESCE(c.comment_id, '')) != ''
  AND LOWER(TRIM(COALESCE(t.platform, ''))) NOT IN ('', 'mock', 'relay-local')
  AND LOWER(TRIM(COALESCE(t.platform, ''))) NOT LIKE 'relay%'
  AND (
    LOWER(TRIM(COALESCE(c.platform, ''))) IN ('', 'mock', 'relay-local')
    OR LOWER(TRIM(COALESCE(c.platform, ''))) LIKE 'relay%'
    OR TRIM(COALESCE(c.region, '')) = ''
    OR TRIM(COALESCE(c.authorization_id, '')) = ''
  );
