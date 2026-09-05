# 从 GitLab 同步项目 — OAuth 启动「网络错误」修复

- **日期**: 2026-09-04 14:54
- **作者**: cursor
- **状态**: approved（goal-mode 自动采纳）
- **页面**: `/tenant/:id/projects` →「从 GitLab 同步项目」→「前往 GitLab 授权」
- **可见症状**: `网络错误，无法启动 GitLab 授权`（红框 `role="alert"`，无 `data-traceId`）

## 架构理解（口头确认）

根据当前架构设计稿（v129 ✅ current）：

- 应用层含 taskFE、taskGitOauth、APISIX 网关；GitLab OAuth 启动走 `/api/git-oauth/gitlab-start-from-gateway/`
- **本次为纯前端契约对齐缺陷**，不新增/移除服务组件或数据流 → **不更新** `docs/architecture/` target

📋 架构版本历史最近：v129 runAll 金丝雀平滑重启 ✅ current。本迭代在 v129 基线上做 Bug 修复。

## 🕸️ Code Review Graph 分析

- CRG `update --brief` 已执行（软依赖）
- Codegraph：唯一 `apiFetch(...gitlab-start-from-gateway...)` 调用点在 `useGitlabProjectSync.startGitlabOAuth`
- 对照成功路径：`useProjectDetailGitRepos.startRepoOAuthConnect` 显式传 `Accept: application/json`

## 问题分析

| 项 | 结论 |
|----|------|
| 触发 | 模态内按钮 `@start-oauth` → `startGitlabOAuth` → `apiFetch` 无 `Accept` |
| 后端 | `wantsJSONOAuthStart`：Accept 不含 `application/json` → **302** 到 GitLab `/oauth/authorize` |
| 浏览器 | `fetch` 跟随跨域 302 → CORS 失败 → `catch` 显示「网络错误…」 |
| 对照 | ProjectDetail 同源接口带 `Accept: application/json` → 200 JSON `{authorize_url}` → `location.href` |
| 次要 | 错误节点无 `data-traceId`，违反元规则 24 |

无用户粘贴 `data-traceId`；Loki 排障门禁不适用。curl 未登录仅得 401，佐证需登录态；根因由 Accept 契约与对照代码确认。

## 方案（选定）

对齐 `useProjectDetailGitRepos.startRepoOAuthConnect`：

1. `apiFetch(..., { credentials: 'include', headers: { Accept: 'application/json' }, timeout: 10000 })`
2. 解析 `extractTraceId(response|data|error)`，写入 `errorTraceId`；模态错误节点 `:data-traceId`
3. 非 ok / 缺 `authorize_url` 时展示后端 `detail`，并挂 traceId
4. 按钮：`Anti-Replay-OK: OAuth start navigates away via authorize_url`（只读启动 + 整页跳转）

**拒绝方案**：改后端默认 JSON — 会破坏真实 `<a href>` HTML 302 导航路径。

## 业务意图 → 事件对照

| 业务意图 | 事件 | 例外理由 |
|---------|------|---------|
| 启动 GitLab 浏览器 OAuth | — | 纯前端导航到既有 OAuth 启动 API；无服务端新状态变更 |

## 价值流影响

- 触及：租户项目列表 → GitLab 同步导入（既有流）
- 无新 stream / 无字段变更；测试：`useGitlabProjectSync.test.js` + 模态 traceId 断言

## 🐍 Python 新增接口

not_applicable — 无新增 Python/Go HTTP 接口。

## 🏛️ 架构变更影响

无（Bug 修复，无组件/数据流增删改）。

## 验收

- 点击「前往 GitLab 授权」请求含 `Accept: application/json`，获 `authorize_url` 并跳转
- 失败时 alert 带 `data-traceId`（有则展示）
- 单测覆盖 Accept 头与失败 traceId
- Search：`apiFetch(...start-from-gateway` across taskFE — 仅本处曾缺 Accept，已修
