# 终态释放对齐评论级 CSC（修复 TASK_STATUS_CHANGED DLT）

- **日期**: 2026-08-14
- **作者**: cursor
- **迭代**: terminal-release-comment-csc-v82
- **状态**: accepted（2026-08-14 总体设计审批通过）
- **范围决策**: 2026-08-14 用户选择「对齐 v80 — 按评论 CSC 逐台释放；无运行资源则 success no-op」
- **触发信封**: `task-status-changed-dlt` / `trace_id=61544b00-9f30-4c30-97d6-c57264e2cda7` / `task_15808657621248946872`
- **相关**: ADR-0007；v80 设计 `2026-08-14-task-level-running-comment-server-count-design.md`；原释放设计 `2026-07-13-task-status-changed-release-servers-design.md`
- **意图**: `docs/intents/backend/cloud/terminal_release_comment_csc.intent.md`

## 1. 对当前架构的理解

根据 `docs/architecture/` current：

- 共有 2 个 current 视图：`enterprise-landscape` v80、`application-integration` v80
- 业务层：Task Management / Cloud Resource
- 应用层：`taskTaskService` 发 `TASK_STATUS_CHANGED`；`taskEvents` intent `1_release_servers_on_terminal` 编排释放；`taskCloudService` 拥有 `cloud_server_configs`
- 技术层：Kafka `task-status-changed` + `-dlt`；MySQL `task_cloud`
- 上次 shipped：v80 — 任务级 CSC 仅模板 + 两计数；运行态只存评论 CSC

📋 架构版本历史（节选）：

- v80 (2026-08-14) ✅ current — 任务级只标记运行中机器/容器数
- v79 ✅ — 厂商证照 COS
- v17 — 首次引入 `TASK_STATUS_CHANGED` → 终态释放

本次在 v80 上补齐 **被漏掉的消费侧数据流**（不新增服务）。
批准后写 **v82** 四类伴生文件（仅 `application-integration`）。
并行已有 v81 target（工作空间任务帖序号），本迭代与之独立、同基于 v80 current。

## 2. 🔍 Trace 日志分析 (traceId: `61544b00-9f30-4c30-97d6-c57264e2cda7`)

- **Grafana Trace Dashboard**: [打开](http://10.2.150.68:3000/d/distributed-trace-view?var-trace_id=61544b00-9f30-4c30-97d6-c57264e2cda7&var-tempo_trace_id=61544b00-9f30-4c30-97d6-c57264e2cda7)
- **Grafana 日志搜索**: Loki Explore，查询
  `{job=~".+"} |= "61544b00-9f30-4c30-97d6-c57264e2cda7"`
- **时间范围**: 2026-08-14 12:45:50Z → 12:46:11Z
- **涉及服务**: task-auth, task-task-service, task-events-task-status-changed-1-release-servers-on-terminal, task-events-task-status-changed-2-fanout-work-panel-sse

### 日志摘要

| UTC | 服务 | 要点 |
|-----|------|------|
| 12:45:51 | task-task-service | 发布 `TASK_STATUS_CHANGED` |
| 12:45:51 | fanout-sse | dispatch ok |
| 12:45:51 | release-servers | `DispatchRetryable` retry=1：`running resource not yet available` |
| 每 ~2s | release-servers | 同一错误 retry 2…10 |
| 12:46:11 | release-servers | retry=11 **exhausted → DLT** |

同任务 12:11–12:23（非本 trace）：评论 `cmt_15808662877358160874` 反复 `comment_csc_start_bootstrap_failed`（`no task-level start event payload`）。**从未形成 instance**。bootstrap 根因不在本次范围。

### 关键发现

- CSC **模板行存在**（否则错误文案会是 `csc row not yet available`）。
- 模板行 `instance_id` 与 `server_url` 皆空 → 命中 `noResourceDecider`。
- v80 后任务级行**永远不会**再填运行态；11 次 × 2s 重试必然 DLT。
- SSE 消费者成功；本次信封**无资源可漏**（评论 VM 从未起来）。

### 根因假设

`LoadForTask` → lookup **不带 `comment_id`** → `loadCloudServerConfig`（注释：禁止当作运行实例）。handler 仍把空模板行当成「资源稍后会注册」。这与 v80 / ADR-0007 冲突，也违反 2026-07-13 设计 4.3.5「无配置 / 已释放 → success no-op」。

## 3. 🕸️ Code Review Graph 分析

`CRG unavailable: .code-review-graph/graph.db 不存在`

静态调用链：`Handler.Dispatch` → `LoadForTask` → `GET /api/internal/cloud-server-config/lookup/?task_id=`（无 comment）→ 空 instance/URL → `noResourceDecider.ShouldRetry`。

同类 `LoadForTask`（**本次不改**，记入后续）：`taskgracefulshutdownawait`、`cloudserverstarted`、`cloudserverstopped`（后者仅在信封缺 `instance_id`/`region_id` 时回退 lookup；本方案发布停机事件时必须带齐这两字段）。

## 4. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 任务级空 instance/URL **不再** `DispatchRetryable` | 单测：模板行空 + 无评论运行态 → `DispatchSuccess`，不发 `CLOUD_SERVER_STOPPED` |
| S2 | 终态释放对象 = 该任务下全部**评论 CSC** 中已启动者 | 单测：两评论各有 instance → 发 2 条 `CLOUD_SERVER_STOPPED`，payload 含对应 `instance_id`/`comment_id`/`region_id` |
| S3 | 释放谓词：**有 instance 且非已停机终态**（含 Starting/Pending）或非空 `server_url`；与 v80「已启动」计数谓词刻意分离 | 单测：Starting 无 instance → no-op；Starting+instance → 发布停机（2026-08-17 修正） |
| S4 | 停机信封自洽，不依赖任务级 lookup 回退 | 每条 `CLOUD_SERVER_STOPPED` 必带 `instance_id`、`region_id`、`comment_id`、`stop_reason` |
| S5 | 本 DLT 场景复现：模板存在、评论无 instance、计数 0 → success | 回归单测名字含 `NoRunningResourceSuccess` |
| S6 | 不新增 Python 接口；不新增 Kafka 事件类型 | 复用 `TASK_STATUS_CHANGED` / `CLOUD_SERVER_STOPPED` |
| S7 | Swagger：新 internal list 路径写入 `taskCloudService/src/openapi-internal.yaml` | 交付阶段可见 |

## 5. 方案决策

| # | 决策 | 理由 |
|---|------|------|
| D1 | 新增 Go internal `GET /api/internal/cloud-server-config/list-by-task/`，返回模板两计数 + 全部评论 CSC（不含 `comment_id=''`） | lookup 无 comment 只回模板；现有 newest-with-instance 只回一台，不够多评论 |
| D2 | `release-servers` 改为 `LoadCommentRuntimesForTask`；**删除**「任务级空 instance → retry」分支 | v80 后该分支必 DLT |
| D3 | 对每个「已启动」评论 CSC：有 `server_url` 先 graceful notify（失败则 hard local stop）；有 `instance_id` 则 `PublishEvent(CLOUD_SERVER_STOPPED)`，key=`{task_id}:{comment_id}` | 复用现有停机链路；多评论并行释放互不覆盖 |
| D4 | Starting **无** instance / 计数为 0 → **success no-op**（不重试） | 本信封即此态；11×2s 等不来 Aliyun 开箱。**2026-08-17 修正**：Starting **有** instance_id 必须发布 `CLOUD_SERVER_STOPPED`（RunInstances 先回 id 再变 Running；进度切完成/取消不得漏杀）。仅 launch_request、尚无 instance 的残留仍交闲置回收 / bootstrap（见 OPT） |
| D5 | 任务级行只读 `running_machine_count` / `running_container_count` 打日志交叉校验，不当运行实例 | 与 ADR-0007 一致 |
| D6 | 无评论 CSC 行（仅模板或 lookup 404）→ success no-op | 恢复 2026-07-13 §4.3.5 |
| D7 | 不改 `cloudserverstopped` / graceful-await / started（用户未选举一反三） | 停机事件带齐 instance/region，不走坏回退 |
| D8 | 不修 `comment_csc_start_bootstrap_failed` | 用户未纳入范围 |
| D9 | 批准后架构 v82（仅 application-integration） | 数据流从「读任务级 instance」改为「读评论 CSC 列表」 |

### 拒绝的方案

| 方案 | 拒绝原因 |
|------|----------|
| 仅把空模板改 no-op、仍不读评论 | 终态无法释放已运行的评论 VM（真泄漏） |
| 继续对空模板 retry | v80 后必 DLT |
| lookup 改成「最新有 instance 的评论」 | 多评论只停一台 |
| Starting 继续 retry | 20s 窗口对云开箱不够，仍 DLT |

## 6. 接口契约（Go / taskCloudService）

`GET /api/internal/cloud-server-config/list-by-task/?tenant_id=&workspace_id=&task_id=`

```json
{
  "task_id": "task_…",
  "running_machine_count": 0,
  "running_container_count": 0,
  "comments": [
    {
      "id": "csc_…",
      "comment_id": "cmt_…",
      "instance_id": "i-…",
      "server_url": "https://…",
      "last_runtime_status": "Running",
      "region": "cn-hongkong",
      "platform": "aliyun",
      "authorization_id": "…"
    }
  ]
}
```

- 鉴权：既有 `X-Internal-Secret`
- 无行：`200` + `comments: []` + 计数 0（避免 consumer 把 404 当 retry）
- 仅模板行：`comments: []`，计数来自模板列
- **不新增公网 path**

`CLOUD_SERVER_STOPPED` data 增量字段（其余与现网一致）：

| 字段 | 必填 | 说明 |
|------|------|------|
| `comment_id` | 是（本路径） | 评论作用域，供日志/SSE/后续 sibling |
| `instance_id` | 是 | 禁止再靠任务级 lookup |
| `region_id` | 是 | 同上 |
| `stop_reason` | 是 | `task_status_completed` / `task_status_cancelled` |

## 7. 消费流程（替换 handler 中段）

```text
TASK_STATUS_CHANGED
  → 非终态：success
  → list-by-task
  → 过滤 releaseable：comment_id≠'' 且 (有 instance 且非 Released/Stopped/… 或 server_url≠'')
  → 空列表：success no-op（log stage=no_running_resource_skip）
  → 对每条 releaseable：
       SetTerminalReleasedFlag(task, comment)（若现接口无 comment 则仍按 task，见实现计划）
       server_url → NotifyShutdown / StopLocal（StopLocal/ClearAfterStop 传 instance_id）
       instance_id → Publish CLOUD_SERVER_STOPPED
  → 全部成功：success
  → 单条失败：DispatchRetryable（整事件重试；已发布停机须幂等）
```

结构化日志：`event=terminal_release_comment_csc task_id=… comment_id=… instance_id=… machines=N containers=M released=K`

## 8. Domain 概念清单（供 /6-ddd）

| 概念 | 说明 |
|------|------|
| Bounded Context | Cloud Runtime（评论 CSC 生命周期）；Task（终态触发） |
| Entity | CommentCloudServerConfig（运行实例权威）；TaskLevelCloudServerConfig（模板+计数，非实例） |
| Aggregate | 任务终态释放编排（根：TaskId；一致性：逐评论释放，允许部分已停） |
| Domain Event | 复用 `TASK_STATUS_CHANGED`、`CLOUD_SERVER_STOPPED`；**不新增**事件类型 |
| Domain Service | `ReleaseCommentServersOnTerminal` |

## 9. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 任务进入已完成/已取消 | TaskStatusChanged | taskTaskService（已有） | release-servers；fanout-sse | — |
| 终态释放一台评论云主机 | CloudServerStopped | release-servers handler | cloudserverstopped：删实例 + clear-after-stop | — |
| 终态无运行中评论资源 | — | — | success no-op | 无状态变更，不发新事件 |
| 查询任务下评论 CSC 列表 | — | list-by-task 只读 | — | 纯查询 |

## 10. 价值流影响（供 /4-value-stream）

- 影响流：`conf/value-stream.yaml` → `task-management`（进度列变更 → 终态释放）
- 不新增 stream；步骤语义从「释放任务级 CSC」改为「释放该任务全部评论 CSC」
- 字段：`taskCloudService.cloud_server_configs.comment_id` / `instance_id` / `server_url` / `running_machine_count`（读）；不改表结构
- 测试：Go 单测（list-by-task + handler 三态：无资源 / 单评论 / 双评论）；不强制本轮 Playwright
- `planned`→`active`：无

## 11. 已知残余与范围外

| 项 | 处理 |
|----|------|
| `comment_csc_start_bootstrap_failed`（本任务从未起 VM） | 范围外；另开迭代 |
| `LoadForTask` 在 graceful-await / started / stopped 回退 | 范围外；stopped 本路径用自洽信封规避 |
| Starting 中完成任务、VM 晚到 | 接受可能泄漏；闲置回收兜底；禁止再用 20s retry 制造 DLT |
| 多评论停机部分成功后整事件 retry | 依赖 `CLOUD_SERVER_STOPPED` / clear-after-stop 幂等（现网已按 instance 停） |

## 12. 🏛️ 架构变更影响

- **迭代版本**: v82 🎯 target
- **迭代名称**: 终态释放对齐评论级 CSC
- **作者**: cursor
- **设计日期**: 2026-08-14 20:59
- **新增文件**（本迭代仅 `application-integration`，四类伴生）:
  - 🆕 `docs/architecture/v82-application-integration-20260814-2059-cursor.puml`
  - 🆕 `docs/architecture/v82-application-integration-20260814-2059-cursor.diff.archimate`
  - 🆕 `docs/architecture/v82-application-integration-20260814-2059-cursor.full.archimate`
  - 🆕 `docs/architecture/v82-application-integration-20260814-2059-cursor.mermaid.md`
- **已有文件（未修改）**: v80 current 两视图
- **变更明细（预告）**:
  - 🟡 [MODIFIED] taskEvents `release-servers-on-terminal` — 读评论 CSC 列表
  - 🟡 [MODIFIED] taskCloudService — 新增 internal list-by-task
  - 🟢 [NEW] Application_Interface `list-by-task`
  - 无 🔴

### .archimate 架构变迁要点（批准后）

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v80 → Gap（任务级 instance 误读）→ WP「评论列表释放」→ Plateau v82；变更数据流 taskEvents↔taskCloudService |
| **`.full.archimate`** | 变迁后完整集成拓扑 + 🟡 标注 |

## 13. 实施计划（供 /7-plans，本步不编码）

1. Characterization：保留现有 handler 单测；新增「空模板 + 无评论」期望从 retry 改为 success 的红灯测试。
2. `list-by-task` + openapi-internal + 单测。
3. `cloudconfig.ListByTask` 客户端。
4. 改 `taskstatuschanged.Handler`：按评论释放；删 `noResourceDecider` 任务级用法（或改为 skip 日志）。
5. 回归：本 DLT 三元组（空模板 / 无 instance / 计数 0）success；双评论双停机事件。

## 变更记录

- 2026-08-14：由 DLT 信封立项；用户确认范围 = 评论级逐台释放 + 无资源 no-op。
