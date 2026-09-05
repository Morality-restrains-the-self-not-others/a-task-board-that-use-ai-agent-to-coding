# 意图：访问令牌审计表 site 存域名+端口

- **日期**: 2026-08-20
- **状态**: 已实施
- **变更记录**:
  - 2026-08-19：列名 `provider` → `site`（005）。
  - 2026-08-20：`site` 语义改为 Git 站点 `域名+端口`（`url.Host`）；
    旧 provider_key 行由 006 直接删除，不回填。

## 背景与目标

`git_oauth_appaccesstokenuseaudit.site` 曾写入 provider key
（如 `gitlab:tencent-sh-1`），与凭据表 `provider` 混淆，也无法直接
对照 Git 站点。改为存储 website 的 `host[:port]`，例如
`github.com`、`localhost:8012`、`gitlab-tencent-sh-1.daydaymoney.com`。

默认端口不写 `:443`/`:80`（与 `url.Parse(...).Host` 一致）。
解析失败不得回退写成 provider key。

## 范围与边界

- 范围内：写入路径（OAuth 回调、token-use-report、access-for-user）；
  `006_access_audit_clear_legacy_site.sql` 清空旧行并将 `site` 扩至
  `varchar(255)`。
- 范围外：凭据表与 task 审计表的 `provider` 不改。无新 API、无新领域事件。

## 验收标准

1. 新写入的 `site` 为 `域名+端口`，不再是 `gitlab:default` 等 key。
2. 已部署库执行 006 后该表无旧 provider_key 行；列为 `varchar(255)`。
3. website 解析不到时：token-use-report 失败；回调不阻断 OAuth，跳过审计并打 warn。

## 业务意图 → 事件对照

| 业务意图 | 事件 | 例外理由 |
|---------|------|---------|
| 审计 site 语义变更 | 无 | 审计副作用，无新业务事件 |
