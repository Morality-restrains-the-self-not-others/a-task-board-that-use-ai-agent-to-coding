# 评论容器启动 TraceId 须作为绑定字段持久化

- **Status:** accepted
- **Date:** 2026-08-13
- **Iteration:** comment-start-traceid-persist
- **Based on:** OPT-20260812-013（启动 TraceId 展示）；`032_server_runtime_tab_container_meta`（两 Tab 共享元信息）
- **Architecture impact:** **否** — 不增删服务/事件；在既有 `cloud_comment_container_bindings` 增列 + 既有 list/start-vm 回写
- **ADR:** No-ADR: trivial tech choice, no architectural impact
- **python_api_approval:** not_applicable（无新 Python 接口；扩展既有 Go list/start-vm）
- **Decision:** 方案 A — binding 表一等字段 `start_trace_id`；**每次启机独立 TraceId，禁止用 `task_id` 冒充**；同任务多评论互不共享；list API 回传；前端以该字段为 SSOT

---

## 0. 问题

页面：https://www.daydaymoney.com/tenant/875588283562749952/workspace/ws_-2740859684112864748/task-detail/task_15742467311115154867/

`comment-execution-container-meta` 可见：

- 容器名 `task_15742467311115154867_cmt_15742473712028559869`
- CSC `csc_-2704251625846902750`
- **没有「启动 TraceId」行**

产品期望：与容器名、CSC 并列展示启动 TraceId，便于复制到 Grafana/Loki。

无错误节点 `data-traceId`（展示缺口，不是请求失败）。**不查 Loki。** 先前同页曾短暂出现「启动 TraceId」且值为 **任务 ID**（`bindStartVmTraceContext` 把 `task_id` 归一成 run trace），刷新后消失，与「只活在 SSE/内存」一致。

## 1. 架构理解（本需求不改架构）

根据 current 架构设计稿：

- 视图：`application-integration` current **v75**；`enterprise-landscape` current **v13**
- 积压 target：v74 / v76 / v77，本缺口不叠加、**不创建 v78**
- 应用层（既有）：taskFE 发评论 → taskTaskService → Kafka `TASK_COMMENT_IMAGE_MENTIONED` → taskEvents `1_start_vm_for_at_mention` → taskCloudService `start-vm` / `start-vm-auto` → SSE `SSE_MESSAGE` → taskFE 评论执行细节

本次只补 **评论绑定上的启动 TraceId 真源**，不改服务边界。

📋 架构版本历史（节选）：

- v75 ✅ current — 租户角色管理
- v76 / v77 🎯 target 积压（订单评论 / MEMBER_JOINED Git 身份）

## 2. 根因：启动会生成 TraceId，但没有一等存储

**结论：不是「容器启动完全没有 TraceId」；是「没有作为绑定/CSC 字段持久化」，冷打开读不回来。**

### 2.1 表上没有 start_trace_id

`cloud_comment_container_bindings` 列：`id, company_id, task_id, comment_id, execution_mode, depends_on_comment_id, status, mock_container_name, csc_id, created_at, updated_at`。

**没有** `start_trace_id` / `trace_id`。CSC 行同样不存启机 trace。

### 2.2 现有「持久化」只是日志正文后缀

`publishTaskSSE` → `logServerSchedulingToBindingBestEffort`：仅当 SSE `statusData.message` 非空时，把 `trace_id=<id>` **追加到** `cloud_comment_container_binding_logs.message`（VARCHAR 512）。

前端冷打开：`extractStartTraceIdFromBindingLogs(b.logs)` 用正则 `\btrace_id=...` 还原。

因此：

| 路径 | 是否写入带 `trace_id=` 的调度日志 | 冷打开能否还原 |
|------|-----------------------------------|----------------|
| `handleStartVmNative` / `start-vm-auto` 主路径 `publishTaskSSE`（有 message + ctx trace） | 会 | 能（若后缀没被 512 截断） |
| 闲置复用 / 本任务附着 `publishSSEMessage`（**不**走 `publishTaskSSE`） | **不会** | **不能** |
| 阶段日志 `logCommentContainerBindingStageBestEffort`（pending/starting/cscAllocated/running） | 不含 trace | 不能 |
| 浏览器未开、仅 Kafka 消费者调 start-vm | 依赖上表；复用路径必丢 | 常不能 |

复用/附着代码（`workspace_machine_idle_reuse.go`）明确调用 `publishSSEMessage` 而非 `publishTaskSSE`：SSE 可能带 ctx 里的 `trace_id`，但 **不注入 `comment_id`、不写 binding 调度日志**。页面当时若开着，任务级 `statusTraceId` 能显示（常等于 **task_id**）；刷新后内存清空，list 日志里没有 `trace_id=`，UI 因 `startTraceId===''` 整行不渲染。

这与当前快照吻合：**CSC 已有、容器名已有、TraceId 行没有**（复用/附着或冷打开丢内存）。

### 2.3 HTTP 受理响应也不带回 TraceId

`handleStartVmNative` 成功 JSON：`status/message/event_id`，**无 `trace_id`**。`@镜像` 启机由 Kafka 消费者调 start-vm，浏览器根本收不到该 HTTP。前端 `buildStartVmAcceptedStatusUpdate` 读 `_traceId` 只覆盖「页面自己 POST start-vm」。

`bindStartVmTraceContext` **当前错误地把 run trace 设成 `task_id`**（覆盖入站 `X-Trace-Id`）。同任务多条评论各自启容器时会撞上同一个 ID，无法在 Grafana/Loki 区分链路。产品要求：**每个评论的每次启动都有独立 TraceId，永远不要用 `task_id`。**

### 2.4 前端展示门闩

`TaskDetailCommentExecutionDetails`：`displayStartTraceId` 为空则不渲染行。数据源：`bindingStartTraceIdFor(comment.id) || statusTraceId`。两者都空 → 「缺少 traceId」。

## 3. 决策（方案 A）

1. **DDL**：`cloud_comment_container_bindings.start_trace_id VARCHAR(128) NOT NULL DEFAULT ''`（`dataMigrate/taskCloudService/018_binding_start_trace_id.sql`）。
2. **独立 TraceId**：`bindStartVmTraceContext` 优先保留入站 `X-Trace-Id`；若缺失或等于 `task_id` 则 `tracelog.NewTraceID()`。**禁止** `NormalizeTraceID(taskID)` 覆盖。同任务两次 start-vm 必须得到两个不同 ID。
3. **写入**：仅当请求带 `comment_id` 时，把该次 `runTraceID` 写入**该评论** binding 的 `start_trace_id`（含闲置复用、本任务附着、冷启动、失败 SSE）。**无 `comment_id` 时不得把同一 trace 扇出到本任务全部活跃 binding。** `task_id` 不得写入该列。
4. **复用路径改为 `publishTaskSSE(ctx, taskID, commentID, …)`**，禁止再用无 comment 的 `publishSSEMessage` 作为唯一通知。
5. **list API** `commentContainerBindingToJSON` 增加 `start_trace_id`。
6. **HTTP 200** start-vm / start-vm-auto / 复用成功体增加 `trace_id`（服务端分配值，与 Loki 一致）。
7. **FE**：`refreshBindings` 优先 `b.start_trace_id`；日志后缀仅存量回填；**禁止**用任务级 `statusTraceId`（常等于 `task_id`）冒充评论启动 TraceId。空串仍不展示行。

| 状态 | 容器名 | CSC | 启动 TraceId |
|------|--------|-----|--------------|
| 已启动且 binding 有 start_trace_id | 有 | 有 | 有（冷打开也有） |
| 尚未启动 | 推导名 | 尚未分配文案 | 无行 |
| 存量 binding 无列值、日志也无后缀 | 有 | 有 | 无行（不伪造） |

## 4. 方案对比

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| A. binding 一等字段 + 复用改 publishTaskSSE | 冷打开稳定；截断无关 | 需 DDL | **采用** |
| B. 只把复用改成 publishTaskSSE，继续靠日志后缀 | 无 DDL | 512 截断；阶段日志仍无 trace；契约仍隐含 | 拒绝 |
| C. 空 startTraceId 时用 taskId 冒充 | 永远有字 | 同任务多评论撞 ID；非真实请求 trace；误导 Grafana | **拒绝（含禁止 `bindStartVmTraceContext` 用 task_id）** |

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 记录本评论容器启动 TraceId | — | start-vm 写 binding 列 | — | 同一次启机的附属持久化，不新增领域事实；启机事件仍为既有 `CLOUD_SERVER_STARTED` / SSE |

无新 Python 接口。无新 Kafka 事件。

## 6. Domain Concept Inventory

无新聚合。`CommentContainerBinding` 增加属性 `StartTraceId`（启机相关 ID，SSOT 在 taskCloudService）。

## 7. 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `code-review-graph status`：Nodes 108 / Edges 937 / Files 17；Languages: javascript, typescript, python, bash；Last updated 2026-08-12 |
| 关键发现 | CRG unavailable for Go：图未索引 `taskCloudService` start-vm；以源码检索为准 |
| 决策影响 | 爆炸半径：`compute_start_vm_native.go`、`compute_start_vm_auto.go`、`workspace_machine_idle_reuse.go`、`comment_container_bindings_store.go`、`events.go`、`useCommentContainerBindings.js`、list JSON、dataMigrate |
| skip 理由 | Go 符号不在图内，不阻断 |

## 8. 价值流影响

- 展示侧：任务详情评论执行细节（`task-detail`）
- 测点：Go 独立 TraceId（≠ task_id、同任务两评论不同）；binding 列写入 + list JSON；vitest 冷打开 `start_trace_id`；复用路径 publishTaskSSE；FE 不用 task_id 冒充
- 字段：`taskCloudService.cloud_comment_container_bindings.start_trace_id`（新增列）
- 不新增 stream

## 9. 🏛️ 架构变更影响

不创建新架构 target。current 仍为 v75 / v13。老 `.puml` 不修改。

## 10. 实施计划

1. `dataMigrate/taskCloudService/018_binding_start_trace_id.sql`：幂等 `ALTER TABLE ... ADD start_trace_id VARCHAR(128) NOT NULL DEFAULT ''`
2. `bindStartVmTraceContext`：入站合法且 ≠ task_id 则保留，否则 `NewTraceID()`
3. store：list/load Scan + JSON 读写该列；`persistCommentBindingStartTraceID(taskID, commentID, runTrace)` 仅写单条评论
4. start-vm / auto / reuse / inflight：`publishTaskSSE` + 写列；HTTP 200 带服务端 `trace_id`；无 comment 的调度日志扇出**不得**附带同一 `trace_id=`
5. FE：`refreshBindings` 优先 `start_trace_id`；忽略等于 `taskId` 的值；测例覆盖两评论独立 ID、task_id 不展示
6. 登记精准编译重启：`task-cloud-service`、`taskFE`；DDL 走 9999 初始化
7. 存量：不回填（日志有独立后缀的仍走 extract）；新启动必写列
