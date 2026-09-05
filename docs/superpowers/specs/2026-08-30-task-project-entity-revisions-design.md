<!-- markdownlint-disable MD013 MD060 -->

# 设计文档：任务与项目主要属性历史版本

- **日期**: 2026-08-30
- **作者**: cursor
- **迭代**: task-project-entity-revisions
- **状态**: accepted（goal-mode 自动采用）
- **拟 ADR**: [ADR-0051](../../adr/0051-task-project-entity-revisions.md)
- **明确非目标**: 不版本化进度/完成态/指派/镜像/仓库/标签；不做一键还原；不给评论/工作空间建历史；不新增 Python 接口

---

## 0. 对当前架构的理解

根据 `docs/architecture/` **current** 与 `VERSION_HISTORY.md`：

- 共有 **2** 个架构视图：`enterprise-landscape`、`application-integration`
- 应用层（本题相关）：`taskFE` → `taskGateway` → `taskTaskService` / `taskProjectService`；事件经 Kafka 到 `taskEvents`
- 数据所有权：`task_tasks` 仅 `taskTaskService`；`project_entries` 仅 `taskProjectService`
- **上次交付的架构版本是 v118 ✅ current**（Git OAuth 资源使用标记）
- v119 🎯 target（多区域镜像仓库副本）尚未交付，与本题正交；本迭代 **v120 基于 v118 current 复制**

📋 架构版本历史（与本题相关）：

| 版本 | 状态 | 摘要 |
|------|------|------|
| v118 | ✅ current | Git OAuth L2 标记 |
| v119 | 🎯 target | 多区域 Registry 副本（正交，未交付） |

本次需求：**任务/项目标题与正文变更时落不可变快照，授权用户可列表与查阅。** 新增 DataObject + 只读 API + 领域事件 → 须写 v120 四类伴生文件。

---

## 🔍 Trace 日志分析

skipped_no_traceid — 用户输入无 `data-traceId` / `trace_id`。

---

## 🕸️ Code Review Graph 分析

- CRG `update --brief`：10 files, 0 nodes（图偏 JS/TS/Python；Go 符号不在 `.code-review-graph`）
- Codegraph CLI：`handleUpdateTask`（`task_handlers_update.go:14`）调用方 `route_handlers.go`；门禁 `hasWorkspaceAccess` + `canMutateTaskFromRequest`；更新后发 `TASK_STATUS_CHANGED`（仅进度/完成态）
- `handleUpdateProject`（`project_handlers.go:160`）门禁 `rejectIfProjectNotInTenant`；按字段零散 UPDATE，无领域事件
- FE：`taskDetailEditing.saveEdit` PATCH 任务（含 title/description）；项目 `ProjectEdit.vue`

---

## 1. 问题与意图

任务帖与项目的标题、正文会随协作改写。当前只保留最新行（`task_tasks.title/description`、`project_entries.name/description`），无法回答「上周标题是什么」。

用户意图：

> 主要属性变更时有所记录，并且能够查阅。

---

## 2. 决策（自动锁定）

| 决策点 | 选择 | 理由 |
|--------|------|------|
| 版本化字段 | 任务 `title`+`description`；项目 `name`+`description` | 「主要属性」= 人读内容；进度/镜像/仓库是运行态 |
| 存储模型 | 追加快照（新状态），含创建时 v1 | 查阅不需拼 delta；当前行仍是热路径 SSOT |
| 触发 | 规范化后字段与上一版不等才插入 | 避免 PATCH 其它字段刷出版本 |
| 写入事务 | 与实体 INSERT/UPDATE 同事务，fail-closed | 历史是本需求的核心保证，不同于登录历史 fail-open |
| 还原 | MVP 不做 | 误触可人工复制；还原当写操作需另审幂等 |
| API 落点 | 扩展现有 Go 服务 | 元规则 20；Python 门禁 not_applicable |
| 事件 | `TASK_REVISION_RECORDED` / `PROJECT_REVISION_RECORDED` | 写意图必须投递 MQ；查询无事件 |

---

## 3. 数据模型

### 3.1 `task_revision`（owner: taskTaskService）

时间累积型。年增量按「每任务每次标题/正文编辑一行」估计 **10万~100万** → MySQL 按月 RANGE 分区（策略 A）。热窗：最近 90 天列表。冷分区由既有 `ensure_partitions.sh` 管理。主键 Snowflake，禁止 AUTO_INCREMENT。

```sql
CREATE TABLE IF NOT EXISTS task_revision (
  id VARCHAR(64) NOT NULL,
  tenant_id VARCHAR(64) NOT NULL,
  workspace_id VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  version_num INT NOT NULL,
  title TEXT NOT NULL,
  description TEXT,
  actor_user_id VARCHAR(64) NOT NULL DEFAULT '',
  changed_fields VARCHAR(64) NOT NULL,
  created_at DATETIME NOT NULL,
  PRIMARY KEY (id, created_at),
  INDEX idx_tr_task_created (task_id, created_at),
  INDEX idx_tr_tenant_created (tenant_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
PARTITION BY RANGE (TO_DAYS(created_at)) ( ... 未来 3 月 + p_future );
```

`changed_fields`：逗号分隔 `title` / `description`（创建时 `title,description`）。`version_num` 应用层 `MAX+1`（同事务），分区表 UNIQUE 须含分区键故不建 `UNIQUE(task_id, version_num)`。

### 3.2 `project_revision`（owner: taskProjectService）

同构：`project_id`、`name`、`description`；无 `workspace_id`；索引 `(project_id, created_at)` + `(tenant_id, created_at)`。`tenant_id` = `project_entries.company_id`。

### 3.3 存量

不回填历史。已有任务/项目在**下一次**版本化字段变更时记 v1=变更后快照（若需「变更前」只能从当时热表读，已丢失）。创建新实体立即写 v1。

---

## 4. API 契约

错误格式沿用各服务既有 `{ error, detail, code }` + `data-traceId`。列表分页 `limit`（默认 20，最大 100）+ `offset`，响应 `{ results, total, limit, offset }`。ID 一律字符串。

### 4.1 任务

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/{taskId}/revisions/` | 与 GET 任务相同：工作空间访问 | 按 `version_num` DESC |
| GET | `/api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/{taskId}/revisions/{revisionId}/` | 同上；revision 须属于该 task+tenant | 单条快照 |

列表项：`id, version_num, title, description`（列表可截断 description 前 200 字）、`actor_user_id, changed_fields, created_at`。详情返回完整 description。

写入：既有 POST 创建 / PATCH 更新成功路径内插入；无独立 POST。

### 4.2 项目

| 方法 | 路径 | 权限 |
|------|------|------|
| GET | `/api/projects/tenant_id/{tid}/{projectId}/revisions/` | 与 GET 项目相同：租户内项目 |
| GET | `/api/projects/tenant_id/{tid}/{projectId}/revisions/{revisionId}/` | revision 属于该 project+tenant |

### 4.3 幂等

- 写：同一 PATCH 若 title/description 未变 → 0 行；Idempotency-Key 仍由既有更新请求携带（FE 已有）
- 读：无副作用 L0

---

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ topic | 发布点 | 消费者 | 例外 |
|---------|--------|----------|--------|--------|------|
| 创建任务并记下初始正文 | TASK_REVISION_RECORDED | task-revision-recorded | `handleCreateTask` 同事务提交后 | `1_observe`（结构化日志，Ack） | — |
| 任务标题/正文实际变更 | TASK_REVISION_RECORDED | 同上 | `handleUpdateTask` | 同上 | 字段未变不发 |
| 创建项目并记下初始正文 | PROJECT_REVISION_RECORDED | project-revision-recorded | `handleCreateProject` | `1_observe` | — |
| 项目名称/正文实际变更 | PROJECT_REVISION_RECORDED | 同上 | `handleUpdateProject` | 同上 | 字段未变不发 |
| 查阅版本列表/详情 | — | — | GET | — | 纯查询 |

Payload（最小）：`tenant_id, workspace_id? , entity_id (task_id|project_id), revision_id, version_num, actor_user_id, changed_fields, trace_id`。**不含**全文 description（避免 Kafka 膨胀与 PII 扩散）。幂等键：`revision_id`（与业务重复边界同粒度）。

Intent 端口：`18072`（task）、`18073`（project）。runAll health 抄同一数字。

---

## 6. 前端

- 任务详情标题行：「历史版本」按钮 → 就地面板（禁止无必要 Teleport）列出版本；点选只读展示该版 title+description
- 项目详情/编辑页同样入口
- 打开面板才 GET 列表（禁止后台轮询）
- 列表加载为只读：`Anti-Replay-OK: GET list`
- 错误节点带 `data-traceId`
- 空历史：「尚无历史版本（创建后或下次修改标题/正文才会记录）」——存量未回填

---

## 7. 权限 / RBAC

不新增角色。读历史 = 读实体。写历史 = 已授权的创建/更新副作用。

登记 `auth_resource_member`：

- `GET .../revisions/` 与 `GET .../revisions/{id}/`（任务）→ `nav.work_panel.main`
- 项目对应 API → `nav.projects.main`

跨租户 IDOR：路径 `tenant_id` 必须与行 `tenant_id` 一致；revision 必须属于 path 中的 task/project。

---

## 8. 可观测性

- 插入成功：`event=entity_revision_recorded` + `entity_kind` + `revision_id` + `version_num` + `trace_id`（无正文）
- 插入失败回滚更新：`event=entity_revision_insert_failed` level=error
- 新 GET：tracelog HTTP 中间件 RED

---

## 9. 价值流影响（Step 4 输入）

- 影响流：`task-management` / `todo-crud`；项目 CRUD 流（`conf/value-stream.yaml` 项目域）
- 新步：`task-entity-revision`、`project-entity-revision`（status: 设计后 active）
- 新字段：`task-task-service.task_revision.title` 等（三段式）

---

## 10. Domain Concept Inventory（Step 6 输入）

| 概念 | 说明 |
|------|------|
| Bounded Context | Task Collaboration / Project Catalog |
| Entity | TaskRevision, ProjectRevision（不可变） |
| Aggregate | Task 根持有 revisions；Project 根持有 revisions；一致性边界=单次更新事务 |
| Domain Event | TaskRevisionRecorded, ProjectRevisionRecorded |

---

## 🐍 Python 新增接口清单与 Go 替代评估

**python_api_approval: not_applicable** — 全部 Go + Vue，无新 Python endpoint。

---

## 🏛️ 架构变更影响

- **迭代版本**: v120 🎯 target
- **迭代名称**: task-project-entity-revisions
- **作者**: cursor
- **设计日期**: 2026-08-30 00:40
- **新增文件**（每个视图四类伴生格式）:
  - `docs/architecture/v120-enterprise-landscape-20260830-0040-cursor.puml`
  - `docs/architecture/v120-application-integration-20260830-0040-cursor.puml`
  - 各视图 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **已有文件（未修改）**: v118 current；v119 正交 target 不动
- **变更明细**: 🟢 `task_revision` / `project_revision` / 两事件 / 查阅 UI；🟡 taskTaskService、taskProjectService、taskFE、taskEvents

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v118 → Gap「无内容历史」→ WP → Plateau v120 + 变更拓扑 |
| **`.full.archimate`** | 本迭代完整拓扑（FE/GW/Task/Project/Events + 两表 + 两事件） |
