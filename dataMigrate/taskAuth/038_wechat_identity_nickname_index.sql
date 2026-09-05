-- 038: wechat_identity.nickname 等值查询索引（超管按微信关联账号查单）
-- 幂等: taskAuth runDataMigrateFromDir 以 data_migrate_log 按文件名去重。

CREATE INDEX wechat_identity_nickname_idx
  ON wechat_identity (nickname);
