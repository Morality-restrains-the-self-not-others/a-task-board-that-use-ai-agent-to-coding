-- Migration 006: git_oauth_appaccesstokenuseaudit.site 改为「域名+端口」语义
-- 005 仅将列名 provider → site，存量值仍是 provider_key（如 gitlab:tencent-sh-1）。
-- 新写入存 website 的 host[:port]（url.Host）。旧行无法可靠回填，按产品确认直接删除。
-- 禁止 TRUNCATE（隐式提交，破坏 apply_datamigrate.sh 事务）。
-- 本 step 由 data_migrate_log 防重入；勿改 checksum 以免二次清空新数据。

DELETE FROM git_oauth_appaccesstokenuseaudit;

ALTER TABLE git_oauth_appaccesstokenuseaudit
  MODIFY COLUMN `site` varchar(255) NOT NULL;
