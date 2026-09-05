-- OPT-20260818-015 owner DB 幂等兜底（gap c）
-- cloud_access_key_iam_associations 原先无 UNIQUE，导入方 REPLACE INTO 以新 snowflake
-- 重复插行；现按自然键 (cloud_platform_auth_id, access_key) 建唯一索引，
-- 使重放事件在 owner 侧 upsert 而非产生重复行。
--
-- 注意：须先去重再建唯一索引，否则存量重复行会使 ALTER 失败。
-- 保留每组自然键下 id 最小的一行（其余为重复授权事件产生的冗余行）。

DELETE a FROM cloud_access_key_iam_associations a
JOIN cloud_access_key_iam_associations b
  ON a.cloud_platform_auth_id = b.cloud_platform_auth_id
 AND a.access_key = b.access_key
 AND a.id > b.id;

ALTER TABLE cloud_access_key_iam_associations
  ADD UNIQUE INDEX uq_akia_auth_access_key (cloud_platform_auth_id, access_key);
