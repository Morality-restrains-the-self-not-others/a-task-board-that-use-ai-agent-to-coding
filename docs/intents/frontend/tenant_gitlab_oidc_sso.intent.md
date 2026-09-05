# 功能意图：gitlab-connection 平台 SSO 配置区块

- **日期**: 2026-08-25
- **状态**: 实现中

## 背景与目标

租户管理员在同一 GitLab 设置页配置「用平台账号登录自建 GitLab」，并看到链路 A Application 操作说明。

## 范围

- `WorkspaceSettingsGitlabOidcSso.vue` + Path A 帮助
- 路由仍为 `/tenant/:tenant/settings/gitlab-connection/`

## 约束

- 写按钮 `hasRegion('settings.gitlab.main')` + `createClickGuard` + `Idempotency-Key`
- secret 一次展示，可复制，刷新后消失
- 真实 `<a href>` 外链文档不拦截（Anti-Replay-OK: 只读导航）
- 凭证区须告知小白：值写进自建 GitLab 的 `/etc/gitlab/gitlab.rb`（`identifier` / `issuer` / `secret`），不是填在本页

## 验收标准

1. 启用成功后出现 OmniAuth 片段与 client_id（端点含 `/api/oidc/{tenantId}/`）
2. GET 加载不出现 secret
3. 无 operate 权限时写按钮不可用
4. Path A Base URL 为公网 HTTP 时展示 `gitlab-oidc-sso-http-warning`（明文回调提醒，不阻断），签发按钮可用并发 PUT
5. 凭证区有步骤化「放到哪里」说明：`/etc/gitlab/gitlab.rb`、`identifier`/`issuer`/`secret`、`gitlab-ctl reconfigure`；签发后一次性密钥写入展示片段

## 业务意图 → 事件对照

前端不直接发 MQ；由后端 PUT/POST/DELETE 投递（见 backend 意图）。

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-25 | 初稿 |
| 2026-08-26 | 公网 HTTP GitLab 预检提示，不再展示英文 `redirect_uri must be https` |
| 2026-08-26 | 产品确认：公网 HTTP 可签发；预检改为非阻断 warn |
| 2026-08-26 | 凭证区增加小白向 gitlab.rb 放置说明（identifier / issuer / secret） |
