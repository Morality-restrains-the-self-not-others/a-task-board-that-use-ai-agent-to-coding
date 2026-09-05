-- Migration 005: git_oauth_appaccesstokenuseaudit.provider → site
-- 访问令牌使用审计记录的站点键与凭据表 provider 列语义分离：本表记录「在哪个 Git 站点换发」，
-- 列名改为 site。仅改本表；git_oauth_appusercredential / git_oauth_taskcredentialaudit 仍用 provider。
-- 幂等：已有 site 或尚无 provider 时 no-op。

SET @stmt = IF(
  EXISTS(
    SELECT 1 FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'git_oauth_appaccesstokenuseaudit'
       AND COLUMN_NAME = 'provider'
  )
  AND NOT EXISTS(
    SELECT 1 FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND TABLE_NAME = 'git_oauth_appaccesstokenuseaudit'
       AND COLUMN_NAME = 'site'
  ),
  'ALTER TABLE `git_oauth_appaccesstokenuseaudit` CHANGE COLUMN `provider` `site` varchar(64) NOT NULL',
  'SELECT 1'
);
PREPARE s FROM @stmt;
EXECUTE s;
DEALLOCATE PREPARE s;
