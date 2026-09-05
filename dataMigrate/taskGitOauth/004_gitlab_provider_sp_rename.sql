-- Migration 004: GitLab SP 更名 daydaymoney-gitlab → daydaymoney-gitlab
-- 背景（2026-08-07）：SSOT provider http-gitlab-daydaymoney-com.yaml 的 service_provider
-- 从 daydaymoney-gitlab 更名为 daydaymoney-gitlab（品牌 daydaymoney.com → daydaymoney.com），
-- 回调契约同步迁移到 base 域 /redirect/gitsite/<gitsite>/oauth/callback/。
-- provider_key 形如 'gitlab:<service_provider>'，前端连接查询 / 回调落库均以该值为准；
-- 不迁移会显示「?gitlab=ok 但页面未绑定」。无匹配行时 UPDATE 为 no-op，幂等安全。
-- 审计表（taskcredentialaudit）为历史记录，不改写。

UPDATE `git_oauth_appusercredential`
   SET provider = 'gitlab:daydaymoney-gitlab'
 WHERE provider = 'gitlab:daydaymoney-gitlab';
