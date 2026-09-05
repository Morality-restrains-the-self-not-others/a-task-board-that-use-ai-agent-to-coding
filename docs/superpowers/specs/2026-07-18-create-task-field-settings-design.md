# 创建设计：创建任务可选字段工作区显隐设置

**日期**: 2026-07-18  
**状态**: 已采用（goal-mode 自动采用）  
**迭代**: create-task-field-settings  
**范围**: settings/task-panel 配置 → work-panel 创建/编辑任务表单字段显隐

## 1. 目标与成功标准

工作区管理员在 `settings/task-panel` 打开/关闭「创建任务可选输入项」后，`work-panel` 创建任务表单仅展示已开启的可选字段。

| # | 标准 | 验收 |
|---|------|------|
| S1 | 工作区可持久化字段显隐配置 | `GET/PUT .../create-task-field-settings/` |
| S2 | 创建任务按配置显隐可选字段 | CreateTaskModal 子区块 `v-if` |
| S3 | 隐藏字段不参与创建必填门禁 | feature_params 隐藏时跳过环境变量拦截 |
| S4 | 默认：`code_lang` / `structured_fields` 关闭，其余开启 | 无配置时这两项为 `false`，其余为 `true` |
| S5 | Swagger 可见 | openapi.yaml 同步 |

## 2. 非目标

- 不改 Chrome 插件创建任务字段（可另开 OPT）
- 不隐藏核心必填：标题、进度状态、交付物类别
- 不隐藏「上层交付物」（业务门禁驱动，非 settings）
- 不引入 MQ 业务事件（纯工作区配置 CRUD，与 task-kind-options 同级）

## 3. 方案选型（自动采用）

| 方案 | 说明 | 结论 |
|------|------|------|
| A. 复用 taskProjectService options 模式 | 新表 + GET/PUT JSON map | **采用** |
| B. 塞进 Django workspace PATCH | 违反 Go-first；与现有 options 分叉 | 否 |
| C. 前端 localStorage | 非跨成员、非持久权威 | 否 |

**架构理解（基于 v34 current）**：工作区配置已由 `taskProjectService` 承载（task-kind / code-lang / work-panel-filters）；网关 `/api/tenant/*/workspaces/*` 通配，无需新公网 path 条目。本次在同组件上新增配置资源，不新增微服务。

## 4. 字段键与默认值

```json
{
  "description": true,
  "task_kind": true,
  "code_lang": false,
  "structured_fields": false,
  "project_branch": true,
  "container_image": true,
  "feature_params": true,
  "priority": true,
  "due_date": true,
  "auto_run": true,
  "owner": true,
  "assignees": true
}
```

未知键忽略；缺失键补对应默认（`code_lang` / `structured_fields` 为 `false`，其余为 `true`）；非 bool 视为 `true`（容错偏展示）。

**变更记录（2026-07-18）**：产品要求「主要编程语言」「结构化任务说明」默认不启用，与初版「默认全开」区分。

## 5. API

```
GET/PUT /api/tenant/{tenant_id}/workspaces/{workspace_id}/create-task-field-settings/
```

**GET 200**:
```json
{ "status": "success", "fields": { "...": true }, "updated_at": "..." }
```

**PUT body**: `{ "fields": { "priority": false, ... } }`（可部分字段；服务端 merge 默认后写全量）

鉴权：与 code-lang-options 相同（gateway user + workspace in tenant）。

## 6. 数据

表 `workspace_create_task_field_settings`：

| 列 | 类型 |
|----|------|
| workspace_id | TEXT PK |
| fields_json | TEXT NOT NULL |
| updated_at | DATETIME |

## 7. 前端

1. **Settings**：`WorkspaceSettingsTaskPanelActions` 增加「创建字段」→ `CreateTaskFieldSettingsModal`（checkbox 列表）
2. **WorkPanel**：`useWorkPanelCreateTaskFieldSettings` 在 init 时 fetch，传入 `CreateTaskModal`
3. **CreateTaskModal / 子组件**：按 `fieldSettings.<key>` 控制 `v-if`
4. **门禁**：`feature_params === false` 时跳过 `resolveCreateTaskFeatureParamsBlockedReason`

## 8. 架构交付物

- `docs/architecture/v35-application-integration-20260718-0100-claude.{puml,archimate,mermaid.md}`
- `VERSION_HISTORY.md` 增加 v35 target（ship 后改 current）

## 9. 测试

- Go：默认 GET、PUT roundtrip、工作区隔离、未知键忽略
- Vitest：normalize + CreateTaskModal 显隐
- Playwright（若时间允许）：settings 关 priority → 创建表单无 `#task-priority`
