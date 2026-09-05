# 设计：任务状态变更事件 → 终态自动释放服务器

- 日期：2026-07-13
- 状态：已采纳（goal-mode 自动决策，跳过确认门）
- 相关 URL：`/tenant/{t}/workspace/{w}/task-detail/{taskId}/` 进度列变更
- 架构版本：v17 🎯 target

## 1. 问题

任务详情页 / 看板变更任务进度列（`progress_column_id`）时：

1. **无领域事件**：`taskTaskService.handleUpdateTask` 写库后不发 MQ；预留的 `TASK_COMPLETED` 从未被生产。
2. **终态不释放资源**：任务进入「已完成 / 已取消」后，关联的云 ECS / relay / mock 容器仍可能运行，需用户手动 stop。

## 2. 目标与成功标准

1. 任务 `progress_column_id` 或遗留字段 `completed` 发生实际变化时，发布 `TASK_STATUS_CHANGED` 到 Kafka（topic `task-status-changed`）。
2. taskEvents 新 intent 消费该事件；若新状态为终态（已完成或已取消），检查该任务是否有运行中服务器。
3. 若有运行中资源：编排调用既有释放路径（ECS → `CLOUD_SERVER_STOPPED`；relay/mock → 对应 stop + `clear-after-stop`）。
4. 幂等：重复消费不重复破坏；无运行中资源时 no-op 成功。
5. 不新增 Python HTTP 接口；不改前端（状态变更仍走既有 PATCH todos）。

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 取舍 |
|------|------|------|
| **A（采纳）** | `taskTaskService` 发 `TASK_STATUS_CHANGED`；新 taskEvents intent 编排释放；复用 stop-vm / relay stop / mock stop | 与现有事件总线一致；状态 SSOT 在 Go；无新 Python API |
| B | 复用空壳 `TASK_COMPLETED`，仅 completed=true 时发 | 漏掉「已取消」；语义不覆盖 progress_column |
| C | 前端在改状态后同步调 stop API | 不可靠（多入口看板/API）；耦合 UI；漏掉非前端变更 |

## 4. 设计要点

### 4.1 事件：`TASK_STATUS_CHANGED`

- **Producer**：`taskTaskService`（Kafka，对齐 `taskCloudService.publishDomainEvent`）
- **触发**：`handleUpdateTask` 成功写库后，若 `progress_column_id` 或 `completed` 相对旧值变化
- **Topic**：`task-status-changed`（注册到 Django `KAFKA_TOPICS`、`taskEvents` `EventTopic`、`DOMAIN_EVENTS.md`）
- **Payload（data）**：

```json
{
  "task_id": "...",
  "tenant_id": "...",
  "workspace_id": "...",
  "company_id": "...",
  "previous_progress_column_id": "...",
  "progress_column_id": "...",
  "previous_completed": false,
  "completed": false,
  "trace_id": "...",
  "span_id": "..."
}
```

发布失败：记 ERROR 日志，**不回滚**已成功的任务更新（最终一致；可依赖重试/补偿运维）。

### 4.2 终态判定

Consumer 内解析 workspace 进度列名称（调 Django 已有 `progress-system` 或内部只读查询；**不新增**公网 Python endpoint；优先复用已有 internal/只读路径；若仅有公网路由则经 gateway 服务凭据拉取）。

终态规则（满足任一即可）：

| 条件 | `terminal_kind` |
|------|-----------------|
| 列名 ∈ {`已完成`, `Completed`, `completed`}（大小写不敏感） | `completed` |
| 列名 ∈ {`已取消`, `Cancelled`, `canceled`, `cancelled`} | `cancelled` |
| `completed` 布尔从 false→true（遗留路径） | `completed` |

非终态：ack 成功，不做释放。

### 4.3 释放编排（新 intent）

**Intent**：`task_status_changed/1_release_servers_on_terminal`  
**Port**：18040（顺延现有 intent 端口）  
**Group**：`task-events-task-status-changed-1-release-servers-on-terminal`

流程：

1. 解析 envelope → 非终态则 return success。
2. 经 `cloudconfig` 客户端加载任务 `CloudServerConfig`（taskCloudService SSOT）。
3. **ECS**：若 `instance_id` 非空 → 组装并 `publishDomainEvent(CLOUD_SERVER_STOPPED, …)`（与 stop-vm 同构），由既有 `cloudserverstopped` handler 删实例 + clear-after-stop + SSE。`stop_reason` = `task_status_completed` | `task_status_cancelled`。
4. **Relay / Mock 容器**：若 `server_url` / `business_api_endpoint` 表明本地运行态仍可达，且非纯 ECS 场景 → HTTP 调 taskContainerGateway 既有 stop（`relay-to-trae/stop` 或 `mock-run-container/stop`），失败记日志并 `DispatchRetry`（可重试错误）。
5. 无配置 / 已释放：success no-op。
6. 结构化日志：task_id、terminal_kind、释放分支、trace_id。

### 4.4 与空壳 `TASK_COMPLETED` 的关系

- **本期不生产** `TASK_COMPLETED`（避免双事件语义混乱）。
- 保留 18032 consumer 空壳；后续若有「完成业务副作用」可另开 intent 订阅 `TASK_STATUS_CHANGED` 且 `terminal_kind=completed`。

### 4.5 权限与安全

- 状态变更仍走既有 todos PATCH 鉴权（租户/工作区/任务变更权限）。
- 释放为服务间异步副作用，不暴露新公网 API。
- Consumer 调 stop / 发 `CLOUD_SERVER_STOPPED` 使用服务凭据；payload 须带 tenant/workspace/task 边界字段，禁止跨任务释放。

### 4.6 Python 新增接口

**不新增** Python HTTP 接口 → 不触发 Python 接口专项审批。  
若需解析进度列名，仅复用已有 Django 路由或在 Go 侧缓存/查询。

### 4.7 Domain 概念清单（供 Step 6）

| 概念 | 说明 |
|------|------|
| Bounded Context | Task（状态 SSOT）、Cloud Runtime（服务器生命周期）、Domain Events |
| Entity | Task（progress_column_id, completed） |
| Domain Event | `TASK_STATUS_CHANGED`（新增）；复用 `CLOUD_SERVER_STOPPED` / `RELAY_STOP_*` |
| Domain Service | `ReleaseServersOnTerminal`（consumer 编排） |
| Value Object | `TerminalKind` = completed \| cancelled |

### 4.8 价值流影响（供 Step 4）

- 影响流：任务协作 / task-management（progress_column 变更）
- 新增步骤：状态变更发事件 → 终态释放服务器
- 字段：`task-task-service.tasks.progress_column_id`；云配置 `cloud.cloud_server_config.instance_id` / `server_url`
- 测试：Go 单测（publish + consumer）；可选 Playwright 终态后 runtime 变为 Released

## 5. 非目标

- 不在前端新增「释放确认」弹窗。
- 不自动启动服务器。
- 不修改进度列体系模型（不加 `is_terminal` 列）；本期用列名约定。
- 不实现通用 outbox（发布失败不回滚任务）；后续可增强。

## 6. 🏛️ 架构变更影响

- **迭代版本**: v17 🎯 target
- **迭代名称**: 任务状态变更事件与终态释放服务器
- **作者**: claude
- **设计日期**: 2026-07-13 14:28
- **新增文件**（每个视图三类伴生格式）:
  - 🆕 `docs/architecture/v17-application-integration-20260713-1428-claude.puml`
  - 🆕 `docs/architecture/v17-application-integration-20260713-1428-claude.archimate`（含 Plateau/Gap/WP）
  - 🆕 `docs/architecture/v17-application-integration-20260713-1428-claude.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] `TASK_STATUS_CHANGED` 事件流 + topic `task-status-changed`
  - 🟢 [NEW] taskEvents intent `task_status_changed/1_release_servers_on_terminal`
  - 🟡 [MODIFIED] taskTaskService — PATCH 成功后发事件
  - 🟡 [MODIFIED] 复用 CLOUD_SERVER_STOPPED / container stop 路径

### .archimate 架构变迁要点

| 元素类型 | 内容 |
|----------|------|
| **Plateau v16** | 多入口同源 API |
| **Plateau v17** | 任务终态自动释放运行时 |
| **Gap** | 任务终态不发事件、不释放服务器 |
| **WorkPackage** | WP-v17-task-status-release-servers |
| **视图** | `架构变迁 v16→v17` + `v17 Target — 状态变更释放数据流`（含 sourceConnection） |

## 7. 测试意图摘要

- 单元：column 变化 → 发事件；无变化 → 不发；终态判定；无服务器 no-op；有 instance_id → 发 CLOUD_SERVER_STOPPED。
- 集成：consumer 消费终态事件后 clear reachability / stop 被调用。
- 回归：非终态变更不触发释放；手动 stop-vm 路径不变。
