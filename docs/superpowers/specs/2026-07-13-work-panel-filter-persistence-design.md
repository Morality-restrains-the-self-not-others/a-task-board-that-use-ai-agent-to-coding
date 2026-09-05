# 设计：工作面板过滤选项按工作空间持久化

- 日期：2026-07-13
- 状态：已采纳（goal-mode 自动决策，跳过确认门）
- 相关 URL：`https://www.daydaymoney.com/tenant/{tenantId}/work-panel/`
- 架构版本：v20 🎯 target（基于 v19 current；**纠正**早期草稿中的 Django 归属）
- `python_api_approval`: scoped-down（**零新增 Python HTTP 接口**；全部落 Go）

## 1. 问题

1. 工作面板交付物过滤栏（`deliverableFilterBars`）仅存内存；刷新后丢失，用户需重新搭栏。
2. 切换工作空间时强制 `defaultDeliverableFilterBars()`，无法按工作空间恢复各自布局。
3. 用户明确要求：**持久化与 API 放在 Go 服务**（非 Django saas-backend）。

## 2. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 过滤栏按 **用户 × 租户 × 工作空间** 持久化 | SQLite 唯一键；同用户不同工作空间互不影响 |
| S2 | 打开 work-panel 自动拉取并应用 | init 后栏数量/路径与上次保存一致 |
| S3 | 增删改过滤栏自动保存 | debounce PUT；刷新后仍在 |
| S4 | 切换工作空间加载对应偏好 | 不再无脑 reset 为默认；先 GET 再应用 |
| S5 | 实现在 Go `taskProjectService` | 表在 `task_project.db`；无新 Python API |
| S6 | Swagger 可见 | `openapi.yaml` + `/api/schema/`；网关 docs 可聚合 |

**范围外（本迭代）**：旧版模态 `filterOptions`、机器运行态 `machineRuntimeFilter`、分区折叠 `sectionCollapsed`。

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 取舍 |
|------|------|------|
| **A（采纳）** | `taskProjectService` 新表 + `GET/PUT .../workspaces/{id}/work-panel-filters/` | 工作空间 SSOT 已在该服务；网关已有 `/workspaces/*` 通配；符合「放 Go」 |
| B | Django `accounts` 表（早期草稿） | 与用户要求冲突；用户 UI 偏好虽可放 saas，但本需求强制 Go |
| C | `taskCloudService` | 云/机器域，与 UI 过滤无关 |
| D | 仅 localStorage | 跨设备/清缓存丢失；无法服务端审计 |

## 4. 领域概念

| 概念 | 说明 |
|------|------|
| **WorkPanelFilterPreference** | 聚合根：某用户在某工作空间下的 work-panel 过滤偏好 |
| **DeliverableFilterBar** | 值对象：`{id, path[]}`；path 段为 root / category / task |
| **FilterPayload** | `{ version: 1, deliverable_filter_bars: [...] }` |

### 规范化规则

- 空或非法 payload → 返回/存为默认：`[{ id: "bar-0", path: [{ type: "root" }] }]`
- bar `id` 去空白；缺省时生成 `bar-{n}`
- path 必须非空且首段可为 root；category/task 须有非空 `id`
- 最多 20 条过滤栏（防滥用）；超限 400
- `user_id` **仅**来自网关鉴权头，禁止客户端指定他人

## 5. 数据模型（task_project.db）

```sql
CREATE TABLE IF NOT EXISTS user_workspace_work_panel_filters (
  user_id TEXT NOT NULL,
  company_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  payload_json TEXT NOT NULL DEFAULT '{"version":1,"deliverable_filter_bars":[{"id":"bar-0","path":[{"type":"root"}]}]}',
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, company_id, workspace_id)
);
CREATE INDEX IF NOT EXISTS idx_uw_wpf_ws
  ON user_workspace_work_panel_filters(company_id, workspace_id);
```

登记：`db/table_ownership.yaml` → `task-project` / `task-project-service`。

## 6. API（Go / taskProjectService :8016）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/tenant/{tenant}/workspaces/{workspace_id}/work-panel-filters/` | 无行则返回默认 payload |
| PUT | 同上 | upsert；body 为 FilterPayload |

响应：

```json
{
  "status": "success",
  "version": 1,
  "deliverable_filter_bars": [ { "id": "bar-0", "path": [{ "type": "root" }] } ],
  "updated_at": "2026-07-13T12:00:00Z"
}
```

权限：

1. 网关 `auth_mode: token` 注入 `X-Auth-User-Id` / tenant
2. `workspace.company_id == tenant`
3. 读写仅作用于 **当前认证用户** 自己的行（天然防 IDOR）
4. 401：无 user；404：工作空间不存在或不属于租户；400：校验失败

网关：现有 `task-project-service` 的 `/api/tenant/*/workspaces/*` 已覆盖；**无需新路由条目**。启用 upstream `docs` 指向本服务 OpenAPI。

Swagger：新增 `taskProjectService/openapi.yaml`，嵌入并暴露 `GET /api/schema/` + Swagger UI `/api/swagger/`。

## 7. 前端

- 新工具：`workPanelFilterPersistence.js`（normalize / default / apply / API URL）
- `WorkPanel.vue`：
  - init：拿到 `currentWorkspace.id` 后 GET → 设置 `deliverableFilterBars`
  - `watch(deliverableFilterBars, debounce 400ms PUT)`（跳过加载中）
  - `handleWorkspaceSwitched`：改为 GET 该 workspace 偏好（不再无脑默认）
- 失败：GET 失败落默认并 `console.warn`；PUT 失败不打断 UI

## 8. 日志

- GET/PUT info：`tenant_id`, `workspace_id`, `user_id`, `bars_count`, `ok|error`
- 禁止整段 path dump（label 可能含业务标题，可记 count + bar ids）

## 9. 测试

- Go：`work_panel_filters_test.go` — 默认 GET、PUT 回显、多 workspace 隔离、非法 payload、跨租户 404、用户隔离
- 前端：`workPanelFilterPersistence.test.js` — normalize / 默认 / 序列化
- 可选 Playwright：增栏 → 刷新 → 栏数不变（若环境可用）

## 10. 架构交付物

| 文件 | 说明 |
|------|------|
| `docs/architecture/v20-application-integration-20260713-2005-claude.puml` | 目标拓扑 |
| 同名 `.archimate` / `.mermaid.md` | 伴生格式 + 架构变迁视图 |
| `VERSION_HISTORY.md` | v20 target 条目 |

> 早期 `v20-...-1956-claude.puml`（Django）作废，改为 archived 说明并由本版替代。

## 11. 关键决策摘要

| 决策 | 理由 |
|------|------|
| 归属 taskProjectService | 用户要求 Go；工作空间实体已在此；网关通配已通 |
| 仅持久化 deliverableFilterBars | 用户描述的「过滤栏」即多栏面包屑；其它过滤可后续加 version 字段 |
| debounce PUT | 拖拽/连续点选时减少写放大 |
| 无行返回默认 | 与「从未保存」UX 一致，前端无需分支 |
