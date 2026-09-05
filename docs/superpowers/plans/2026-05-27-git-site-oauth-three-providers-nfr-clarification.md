# NFR 澄清: Git 网站授权页多 service_provider

> 输入: 设计文档 + `docs/superpowers/plans/2026-05-27-git-site-oauth-three-providers-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | providers 目录 GET P95 ≤ 200ms；切换站点后 connection GET P95 ≤ 500ms |
| 可用性 | L2 | 配置 3 站点时 UI 展示 3 按钮；Playwright 回归通过 |
| 安全性 | L2 | providers/connection/start 均需 IsAuthenticated；目录不暴露 secret |
| 可维护性 | L2 | pytest + Playwright 防回归；目录与 `GIT_OAUTH_PROVIDER_CONFIGS` 同源 |

## 质量场景

### QS-01: 设置页站点数量与配置一致

| 要素 | 内容 |
|------|------|
| 刺激 | 登录后打开 `/user/{id}/profile/git-site-oauth/` |
| 响应 | 切换按钮数 = `port_config.gitOauth` 条目数 |
| 度量 | `GitSiteOAuth.provider-tabs-count.playwright.test.js` |

## 领域模型影响

复用既有 `OAuthProviderKey`、`GIT_OAUTH_PROVIDER_CONFIGS`；新增只读目录查询，不新增持久化实体。
