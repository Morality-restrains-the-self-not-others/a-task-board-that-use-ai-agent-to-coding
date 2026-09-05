-- 030_auth_customtoken_composite_index.sql — userId cookie 兜底热路径复合索引（OPT-20260807-003）
-- 用途: userHasLiveToken 的 COUNT 查询（forward-auth 热路径，userId cookie 兜底时每次
--       缓存未命中执行）此前仅命中 content_type_id 单列索引（001_auth_tables.sql），
--       object_id 需回表过滤；多设备/多 token 用户与高流量下放大该路径成本。
-- 幂等: 由 taskAuth runDataMigrateFromDir 以 data_migrate_log 按文件名去重，每库仅执行一次。

CREATE INDEX auth_customtoken_ct_obj_idx
  ON auth_customtoken (content_type_id, object_id);
