# 意图：容器页面 UI path 含 tenant / workspace / task

- **日期**: 2026-07-13
- **状态**: 已实现
- **关联页面**: 任务详情「打开容器页面」→ `businessapi…/ui/…`

## 背景

启动任务后提示的容器页为 `https://businessapi…/ui/{access_token}`，path 无法从 URL 识别所属租户/工作区/任务。API 已采用 scoped 前缀 `/api/tenant/{t}/workspace/{w}/task/{task}/…`，UI 应对齐。

## 验收标准

- [x] 规范 path：`/ui/tenant/{tenantId}/workspace/{workspaceId}/task/{taskId}/{access_token}`
- [x] `onlineServiceJS` 服务该 path，并注入当前 `ACCESS_TOKEN` 的控制台 HTML
- [x] 旧 `/ui/{token}` 在能解析 scope 时 **302** 到 scoped path（无 scope 时仍可本地开发）
- [x] stale token 自愈、`/api/session/ui-redirect`、`render-hints` 链接均输出 scoped path（有 scope 时）
- [x] SaaS / taskCloudService 生成的 `container_page_url` 使用 scoped path
- [x] 从 URL 提取 access_token 的解析器兼容旧 `/ui/{token}` 与新 scoped path



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：容器 UI path 约定，无新增业务事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 意图：容器页面 UI path 含 tenant / workspace / task | — | — | — | — | 容器 UI path 约定，无新增业务事件 |
## 变更记录

- 2026-07-13：初版 — 相对旧版仅 `/ui/{token}`，增加三 ID 段以对齐 API scoped 约定。
