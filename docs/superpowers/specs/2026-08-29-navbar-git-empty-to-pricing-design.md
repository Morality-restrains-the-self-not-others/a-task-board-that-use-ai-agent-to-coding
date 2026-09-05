# Navbar「代码仓库」无仓库跳转价格页 — 设计文档

- **日期**: 2026-08-29
- **状态**: accepted（/goal 自动采用）
- **迭代**: navbar-git-empty-to-pricing
- **作者**: cursor

## 背景

工作面板顶栏「代码仓库」（`data-testid=nav-git-service`）在租户 **没有已开通/获赠 GitLab 区域**（`resources[]` 为空、无可跳转 `gitlab_web_url`）时，当前把真实 `<a href>` 指到 **当前页**（如 `/tenant/{id}/work-panel?accessCode=…`），点击等于刷新。产品要求改为跳转 **价格页** `https://www.daydaymoney.com/pricing/`（站内路径 `/pricing/`）。

页面：`/tenant/:tenant/work-panel`（标题「云端开发」）。

既有意图 `frontend/navbar_git_service_region_nav` 明确写过「无资源则留在当前页、不跳转 `/pricing/`」——本迭代 **撤回该产品决策**。

## 成功标准

1. 已登录 + GitLab 区域列表 **已成功拉取且 jumpable 为空**：`a[data-testid=nav-git-service]` 的 `href` 为 `/pricing/`，若当前 URL 有 `accessCode` 则保留（与「价格」链接同一套 `pricingHref`）。
2. 已登录 + ≥1 个带 `gitlab_web_url` 的区域：行为不变（button + 下拉，菜单项外链 `target=_blank`）。
3. 未登录：仍不渲染该入口。
4. **拉取失败 / 尚未返回**：href **仍为当前页**（fail-open），避免接口故障把已购用户送到价格页。
5. 真实 `<a href>`，禁止 `@click.prevent` + `router.push`（元规则 49）。
6. VIP1 角标与跳转解耦：无资源已就绪时仍去价格页，角标照常显示。

## 方案（选定）

在 `NavbarGitServiceNav` 空列表分支把 `href` 从 `currentPageHref` 改为：

| `gitResourcesStatus` | jumpable | href |
|----------------------|----------|------|
| `ready` | 0 | `pricingHref`（`/pricing/` + 可选 `accessCode`） |
| `unknown` / `error` | 0 | `currentPageHref` |
| 任意 | ≥1 | 下拉（不变） |

`Navbar.logic.fetchGitResources` 增加状态：无租户视为 `ready`+空列表；HTTP/网络失败为 `error`；成功为 `ready`。

不新增 API、不改计费/开通。

## 拒绝方案

| 方案 | 原因 |
|------|------|
| `@click` 拦截再 `router.push('/pricing/')` | 违反禁止链接点击拦截 |
| 空列表一律 `/pricing/`（含 API 失败） | 故障时误导已购用户 |
| 继续留在当前页 | 与本次产品要求相反 |

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief`：增量 5 files / 0 nodes（无符号图命中）。`.codegraph/` 索引不存在，代码理解回退 Grep。影响面：`NavbarGitServiceNav.vue` ← `Navbar.ui.vue` ← `Navbar.logic.vue`；测例 `Navbar.ui.test.js`、`Navbar.logic.membershipTier.test.js`。

## 🏛️ 架构变更影响

**不更新架构文件。** 无新增/移除服务、无数据流/所有权变更、无新表。纯前端导航 href 策略。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 无仓库时点击代码仓库去价格页 | — | — | — | 纯前端导航，无服务端状态变更 |

## Python 新增接口

不触发（无 Python 新 endpoint）。
