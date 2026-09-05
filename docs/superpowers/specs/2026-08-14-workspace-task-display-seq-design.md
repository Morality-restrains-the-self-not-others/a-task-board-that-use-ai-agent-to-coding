# 工作空间任务帖人读序号（workspace_seq）

- **日期**: 2026-08-14
- **作者**: cursor
- **迭代**: workspace-task-display-seq-v81
- **状态**: accepted（2026-08-14 总体设计审批通过；存量清空不回填；展示 `#N`）
- **ADR**: ADR-0008
- **意图**: `docs/intents/backend/workspace_task_display_seq.intent.md`

## 1. 问题背景

同一租户、同一工作空间内，看板卡片与「派生自 / 上层交付物」展示的是技术 ID 的**后 6 位**（`taskIdLast6`）。

`taskTaskService.genID("task")` 为 `task_{unixNano*1000+seq}`：后 6 位是纳秒时间戳低位，不是工作空间内递增号。连续建帖后 6 位跳号，人读起来像乱序。

`order_num` 已按工作空间 `MAX+1`，但是**看板拖拽排序**，用户一改顺序编号就变，不能当身份编号。

用户确认：

- 人读编号用 GitHub 风格 `#12`、`#103`（可变位数）
- 存量**清空即可**，不回填历史序号

## 2. 对当前架构的理解（设计前确认）

根据 `docs/architecture/` current：

- 共有 2 个 current 视图：`enterprise-landscape` v80、`application-integration` v80
- 业务层：Task Management / Workspace / Project
- 应用层：`taskTaskService`（:8017）拥有 `task_tasks`；`taskFE` 看板 `TaskCardIdBadge` 截后 6 位
- 技术层：MySQL 任务库（utf8mb4）；主键规范为 Snowflake / 全局唯一字符串，禁止工作空间自增当 PK
- 上次 shipped：v80 任务级运行中机器/容器计数

📋 架构版本历史（节选）：

- v80 (2026-08-14) ✅ current — 任务级运行中机器/容器计数
- v79 📦 archived — 厂商证照 COS
- v78 🎯 target — 评论级容器令牌（积压，与本次无关）

本次在 v80 上改 **taskTaskService 数据模型 + taskFE 展示**（不新增服务）。批准后写 **v81** 四类伴生文件。

## 3. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 同一 `tenant_id + workspace_id` 内新帖人读编号从 1 连续递增 | 连续创建 3 帖 → `#1` `#2` `#3` |
| S2 | 技术主键 `id`（`task_<digits>`）不变；跨服务外键、Kafka、云资源绑定仍用 `id` | 创建响应同时有 `id` 与 `workspace_seq` |
| S3 | `workspace_seq` 创建后不可变；删除不回收 | 删 `#2` 后再建 → `#4`（若已发到 3） |
| S4 | 看板 / 父任务 / 派生自展示 `#N`，不再截技术 ID 后 6 位 | `TaskCardIdBadge`、`formatTaskIdTitleLabel` 读 `workspace_seq` |
| S5 | 单击复制 `#N`；双击复制完整 `task_<digits>` | 与现交互一致，短号改为序号 |
| S6 | 当前工作空间搜索 `#12` / `12` 命中该序号；完整 `task_…` 仍走主键 | 必须带 `tenant_id + workspace_id` |
| S7 | 存量不回填：迁移清空本服务任务帖及相关行后从 1 起号 | 见 §5 清空清单；`task_git_identities` **不清** |
| S8 | 不新增 HTTP 路径 | 只扩展既有任务 JSON |

## 4. 方案决策

| # | 决策 | 理由 |
|---|------|------|
| D1 | 技术 ID 与人读序号分离：新列 `task_tasks.workspace_seq`（`INT UNSIGNED NOT NULL`） | 不违反 Snowflake / 分片主键规范 |
| D2 | 发号表 `task_workspace_seq`（PK `tenant_id, workspace_id`，`next_val`）与 `INSERT task_tasks` **同一事务** `SELECT … FOR UPDATE` | 避免 `MAX+1` 无锁撞号 |
| D3 | `UNIQUE(tenant_id, workspace_id, workspace_seq)` | 工作空间内序号唯一 |
| D4 | 展示 `#N`（可变位数）；JSON 字段名 `workspace_seq`（小整数，可用 number） | 用户选定 GitHub 风格 |
| D5 | **不复用** `order_num` | 看板排序可变 |
| D6 | 删除不回收序号；失败回滚事务则不消耗号 | 人读稳定性优先于无空洞 |
| D7 | 存量清空、不回填。迁移 `010` 先删任务帖域表再加列/建发号表 | 用户确认「清空即可」 |
| D8 | 不新增 HTTP；`TASK_CREATED` payload **增补** `workspace_seq`（不是新事件） | 无 Python 新接口；Go 落点 `taskTaskService` |
| D9 | 评论不做第二套序号 | 本次只做人读**任务帖**编号 |
| D10 | 批准后架构 v81 + ADR-0008 | 人读 ID 与技术 PK 分离是跨迭代约束 |

### 拒绝的方案

| 方案 | 拒绝原因 |
|------|----------|
| 改 `genID` 让后 6 位递增 | 纳秒/Snowflake 低位天然无工作空间序；改 PK 破坏外键与分片 |
| 工作空间 `AUTO_INCREMENT` 当主键 | 违反 Snowflake 元规则与分片「禁止自增 PK」 |
| 前端按 `created_at` 算名次 | 删除/插入后编号漂移，无法复制稳定短号 |
| Redis `INCR` 发号 | 任务库已是真源，双写与故障窗口 |
| `MAX(workspace_seq)+1` 无锁 | 并发创建撞号 |
| 回填存量按 `created_at` 编号 | 用户选择清空 |

## 5. 数据模型

```
task_workspace_seq          — 每工作空间一行发号器（小表，非时间累积，不分片）
  tenant_id     VARCHAR(64)  PK
  workspace_id  VARCHAR(64)  PK
  next_val      INT UNSIGNED NOT NULL   -- 下一个将分配的序号
  updated_at    DATETIME

task_tasks
  id              VARCHAR(64) PK        -- 技术 ID，不变
  tenant_id
  workspace_id
  workspace_seq   INT UNSIGNED NOT NULL -- 人读序号，创建后不变
  order_num                         -- 仍只用于看板排序
  UNIQUE (tenant_id, workspace_id, workspace_seq)
```

**伸缩要素**：`task_workspace_seq` 一行/工作空间，年增量 ≪ 10 万，无冷热分离。`workspace_seq` 是存量表新列，不改变 `task_tasks` 既有分区策略（当前未分区；任务帖按工作空间查询，分片键仍是 `tenant_id`）。

### 5.1 存量清空清单（本服务）

迁移在加 `NOT NULL workspace_seq` 之前，按子表→主表删除（**保留** `task_git_identities`）：

| 表 | 动作 |
|----|------|
| `task_queued_machine_slots` | DELETE ALL |
| `task_queued_auto_run_memberships` | DELETE ALL |
| `task_top_deliverable_schedule_rhythm_windows` | DELETE ALL |
| `task_top_deliverable_schedule_rhythms` | DELETE ALL |
| `task_feature_params_snapshots` | DELETE ALL |
| `task_feature_params` | DELETE ALL |
| `task_repo_identities` | DELETE ALL |
| `task_branch_strategies` | DELETE ALL |
| `task_projects` | DELETE ALL |
| `task_assignees` | DELETE ALL |
| `task_comments` | DELETE ALL |
| `task_tasks` | DELETE ALL |
| `task_git_identities` | **保留**（用户/公司级，非任务帖） |

跨服务残留（`cloud_server_configs`、凭证令牌等）**不在本次级联删除**。须先停任务写入或接受孤儿行，由 9999 / 运维另行清理。设计假定本环境可清空任务帖。

### 5.2 发号伪代码

```
BEGIN
  INSERT INTO task_workspace_seq(tenant_id, workspace_id, next_val, updated_at)
    VALUES (?, ?, 1, NOW())
    ON DUPLICATE KEY UPDATE next_val = next_val  -- 占行
  SELECT next_val FROM task_workspace_seq
    WHERE tenant_id=? AND workspace_id=? FOR UPDATE
  seq := next_val
  UPDATE task_workspace_seq SET next_val = next_val + 1, updated_at=NOW()
    WHERE tenant_id=? AND workspace_id=?
  INSERT task_tasks (..., workspace_seq) VALUES (..., seq)
COMMIT
```

## 6. API / 前端

不新增 path。既有创建/列表/详情 JSON 增加：

```json
{ "id": "task_1723…", "workspace_seq": 12 }
```

| 位置 | 行为 |
|------|------|
| `TaskCardIdBadge` | 展示 `#{{workspace_seq}}` |
| `formatTaskIdTitleLabel` | `#12 标题`；禁止再 `taskIdLast6` |
| 单击复制 | `#12` |
| 双击复制 | 完整 `task_<digits>` |
| 导航搜索 | `#12` / `12` → 当前工作空间 `workspace_seq`；`task_…` → `id` |
| 父任务 / 派生自 | 用被引用任务的 `workspace_seq`（列表缓存或详情字段） |

后端 `normalizeTaskSearchQuery`：去 `#` 后若为纯数字，在 `tenant_id + workspace_id` 下按 `workspace_seq` 查；禁止跨工作空间用序号撞库。

## 7. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 创建任务帖并分配工作空间序号 | TASK_CREATED（已有） | 既有 task-created 契约 | `handleCreateTask` | 既有下游 | payload **增补** `workspace_seq`；不是新事件 |
| 按序号搜索 / 展示 `#N` | — | — | 列表/详情只读 | — | 纯查询，无对应事件 |
| 迁移清空任务帖域 | — | — | dataMigrate `010` | — | 一次性运维 DDL，无在线业务意图 |

## 8. Domain Concept Inventory

| 项 | 内容 |
|----|------|
| Bounded Context | Task（`taskTaskService`） |
| Key Entities | `Task`（增加 `WorkspaceSeq`）；`WorkspaceTaskSeq`（发号器，每工作空间一行） |
| Candidate Aggregates | 聚合根仍是 `Task`；发号器与建帖同一事务，不对外暴露独立 API |
| Domain Events | 沿用 `TASK_CREATED`，契约加 `workspace_seq` |

## 9. 价值流影响

- 影响流：`task-management` / `todo-crud`（创建、列表、搜索、卡片展示）；`todo-fork-from`（派生自文案）
- 新字段：`task-task-service.task_tasks.workspace_seq`
- 新表：`task-task-service.task_workspace_seq.next_val`
- 测试：`taskTaskService` 创建/搜索单测；`taskFE` `taskIdDisplay` / `TaskCardIdBadge` / `navbarTaskSearch` / `parentDeliverableFilter`
- 完整切片交给 `/4-value-stream`

## 10. 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `code-review-graph status`：108 nodes / 937 edges / 17 files；branch `main`；2026-08-14 更新。语言面以 JS/TS/Python/bash 为主 |
| 关键发现 | `search taskIdLast6` / `CreateTask` 返回 0 节点（Go 发号与部分 FE 工具未入当前图） |
| 决策影响 | 爆炸半径以静态检索为准：`task_handlers.go` 创建路径、`task_store.go` JSON、`taskIdDisplay.js`、`TaskCardIdBadge.vue`、`navbarTaskSearch.js`、`parentDeliverableFilter.js`、`useTaskIdentityPanelDisplay.js` |
| skip 理由 | 图覆盖不足，不阻断；设计以源码检索为准 |

## 11. 🐍 Python 新增接口

不触发。无新 Django/Flask path；扩展既有 Go `taskTaskService` JSON。

## 12. 🏛️ 架构变更影响

- **迭代版本**: v81 🎯 target
- **迭代名称**: 工作空间任务帖人读序号
- **作者**: cursor
- **设计日期**: 2026-08-14 20:57
- **新增文件**（每个视图四类伴生格式，**缺一不可**）:
  - 🆕 `docs/architecture/v81-enterprise-landscape-20260814-2057-cursor.puml`
  - 🆕 `docs/architecture/v81-application-integration-20260814-2057-cursor.puml`
  - 🆕 `docs/architecture/v81-enterprise-landscape-20260814-2057-cursor.diff.archimate`
  - 🆕 `docs/architecture/v81-application-integration-20260814-2057-cursor.diff.archimate`
  - 🆕 `docs/architecture/v81-enterprise-landscape-20260814-2057-cursor.full.archimate`
  - 🆕 `docs/architecture/v81-application-integration-20260814-2057-cursor.full.archimate`
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**: `docs/architecture/v80-*-*.puml`（current）
- **变更明细**:
  - 🟢 [NEW] `task_workspace_seq`；`task_tasks.workspace_seq`
  - 🟡 [MODIFIED] `taskTaskService` 事务发号 + JSON；`taskFE` 展示/搜索/复制 `#N`
  - 🔴 无废弃服务；废弃前端「后 6 位当人读编号」行为

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v80 → Gap（后 6 位无序）→ WP → Plateau v81；变更数据流 taskFE ↔ taskTaskService ↔ 发号表/序号列 |
| **`.full.archimate`** | 变迁后本切片全量：Gateway / taskFE / taskTaskService / task DB + 新 DataObject |

## 13. 实施落点（供后续步骤）

1. `dataMigrate/taskTaskService/010_workspace_seq.sql`（utf8mb4；先清空任务帖域再 DDL）
2. `taskTaskService`：`nextWorkspaceSeq` + `taskSelectCols` + `taskToJSON` + 搜索
3. `taskFE`：`taskIdDisplay.js` 改为 `workspace_seq`；更新单测与 Playwright
4. `TASK_CREATED` 契约加字段
5. 经 9999 初始化数据库，**禁止**业务进程启动时迁移
