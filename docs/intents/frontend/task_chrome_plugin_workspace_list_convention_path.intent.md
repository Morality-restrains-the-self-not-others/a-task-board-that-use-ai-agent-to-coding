# 意图：Chrome 插件工作空间列表走现行约定路径

## 背景与目标

- 背景：`taskChromePlugin` 悬浮面板工作空间下拉请求旧路径 `/api/tenant/{companyId}/workspaces/`。网关 `/api/tenant/*` 已派发到 `taskTenantService`，工作空间 CRUD 在 `taskProjectService` 的 `/api/projects/workspaces/tenant_id/{tid}`。租户服务对该 URL 返回 `404 not found`，下拉显示「加载失败:not found」。
- 目标：已登录用户打开浮窗/DevTools 面板时，工作空间下拉能列出其租户下可访问的工作空间。

## 范围与边界

- 范围内：插件默认 API 端点（工作空间、项目、建任务、进度列、分支、交付物、已安装镜像、daydaymoney resolve、成员列表）对齐 work-panel / 网关现行约定路径。
- 范围外：不改后端路由；不恢复已下线的 `/api/tenant/*/workspaces/`。

## 约束与风险

- 成员列表仍走 `taskTenantService` 的 `/api/tenant/{cid}/accounts/members/company_members/`（该前缀仍有效）；空 `.../members/` 同样 404 `not found`。
- 非租户管理员调 `company_members` 可能得到空列表（既有服务端策略），不在本次修复范围。

## 验收标准

1. 默认 `workspaces` 端点为 `/api/projects/workspaces/tenant_id/{companyId}`，不再请求 `/api/tenant/{companyId}/workspaces/`。
2. `getWorkspaces(companyId)` 对上述路径发 GET；200 且数组非空时下拉渲染选项。
3. 同类旧 `/api/tenant/` 项目/任务/云资源端点一并切到约定路径。
4. `taskChromePlugin` 下 `npm test` 通过。

## 实施计划

1. 单测锁定默认端点与 `getWorkspaces`/`getMembers` 实际请求 URL。
2. 更新 `lib/api.js` `DEFAULT_ENDPOINTS` 与 README。
3. `getMembers` 解析 `{ members: [] }`。

## 业务意图 → 事件对照

| 业务意图 | 领域事件 | MQ Topic | 说明 |
|----------|----------|----------|------|
| 插件拉取工作空间列表 | （无） | — | **无对应事件**：只读查询，不改变服务端聚合状态 |

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-20 | 默认端点切到 `/api/projects/`、`/api/tasks/`、`/api/cloud/` 约定路径 | 旧 `/api/tenant/*/workspaces/` 经网关落到租户服务 404 |
