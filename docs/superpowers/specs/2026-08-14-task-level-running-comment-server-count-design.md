# 任务级只标记「运行中机器数 + 运行中容器数」

- **日期**: 2026-08-14
- **作者**: cursor
- **迭代**: task-level-running-comment-server-count-v80
- **状态**: accepted（2026-08-14 总体设计审批通过；任务级两计数；2026-08-14 修订：清空旧数据、不兼容存量任务）
- **ADR**: ADR-0007
- **意图**: `docs/intents/backend/task_level_running_comment_server_count.intent.md`

## 1. 问题背景

评论已各自拥有 CSC（`UNIQUE(workspace_id, task_id, comment_id)`）。运行态 API 已按 `comment_id` 读取。但任务级行（`comment_id=''`）仍被写成「这台任务的服务器」：

- `last_runtime_status` 按 `task_id` 整表刷（所有评论 + 模板行一起变）
- 启动成功无 `comment_id` 时把 `instance_id` 写到任务级空行
- 前端 `cloudRuntimeStatus` 仍是任务级一份 Starting/Running

结果：一台评论的云主机已起来，任务级仍显示「启动中 / 云实例创建中」，或把一份状态盖到所有评论上。

用户变更：**任务级不再存服务器生命周期；只标记这个任务有多少台评论机器在运行、多少个评论容器在运行。**

## 2. 对当前架构的理解（设计前确认）

根据 `docs/architecture/` current：

- 共有 2 个 current 视图：`enterprise-landscape` v79、`application-integration` v79
- 业务层：Cloud Resource Service / Task Management / IDE Workspace
- 应用层：`taskCloudService` 拥有 `cloud_server_configs`；`taskFE` 看板与任务详情消费运行态
- 技术层：MySQL `task_cloud`（utf8mb4）
- 上次 shipped：v79 厂商证照 COS；相关 target：v78 评论级容器令牌

📋 架构版本历史（节选）：

- v79 (2026-08-14) ✅ current — 厂商证照 COS 预签名
- v78 🎯 target — 评论级容器令牌（同一任务多评论隔离）
- v69 及更早 — 评论级 CSC / 去掉本地 mock 启动

本次需求在 v79 current 上改 **taskCloudService 数据语义**（不新增服务）。批准后写 **v80** 四类伴生文件。

## 3. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 任务级 CSC 不作为运行实例：不写、不读 `instance_id`/`last_runtime_status` 当「这台任务的服务器」 | 单测：无 comment 的 persist/import 不得把 instance 落到 `comment_id=''`；runtime-status 无 comment → 400 |
| S2 | 任务级只存/只暴露两个计数：运行中机器数、运行中容器数 | 列 `running_machine_count` + `running_container_count`；任务详情/看板可读到两个 N |
| S3 | 机器运行中 = 评论 CSC 满足 `machineRuntimeCountsAsStarted`（Running 或 `mock-`）；容器运行中 = `containerReachabilityCountsAsRunning`（已启动且 `server_url` 非空）。Starting **不计入机器数** | 单测：Starting → 机器 0 容器 0；Running 无 URL → 机器 1 容器 0；有 URL → 皆 1；Released → 皆 0 |
| S4 | 评论 CSC 仍是 instance / last_runtime_status / IP / URL 的唯一权威 | 停机、Describe 只改对应评论行；**禁止**从任务级回填评论行 |
| S5 | 看板 `machine_running` = 机器数>0，`container_running` = 容器数>0；响应带两计数 | indicators 扫描忽略任务级行的 instance/status |
| S6 | 前端任务壳不再用一份 Starting/Running 冒充所有评论 | 无评论回退面板展示数量或空；评论卡走评论级 |
| S7 | 不兼容存量：任务级旧运行态直接清空，不 heal 到评论行 | DDL 后任务级 instance/status/IP/URL 为空；无 `healCommentCSCInstanceFromTaskLevel`；仅任务级有 instance 的任务显示 0/0 |

## 4. 方案决策

| # | 决策 | 理由 |
|---|------|------|
| D1 | 任务级行继续存在，职责缩成 **硬件/平台模板 + 运行中计数** | 评论 CSC 仍从模板拷 platform/region/auth；不另开表 |
| D2 | 任务级行两列：`running_machine_count`、`running_container_count`（`INT NOT NULL DEFAULT 0`）；评论行保持 0 | 用户要求同时标记机器与容器；写入时用看板同一谓词重算 |
| D3 | 机器谓词 = `comment_id!=''` 且 `machineRuntimeCountsAsStarted`；容器谓词 = 同上且 `containerReachabilityCountsAsRunning`（需非空 `server_url`） | 与 `workspaceMachineSnapshot` 对齐；不把 Starting 算进机器数 |
| D4 | `setCloudServerLastRuntimeStatus` 必须带 `comment_id` 或 `instance_id`，禁止 `WHERE task_id=?` 全刷 | 修串台根因 |
| D5 | `persistStartVmInstanceBinding` 无 `csc_id`/`comment_id` 时 **no-op + 告警日志**，不再写任务级 | 杜绝新的任务级 instance |
| D6 | **不兼容存量任务**。DDL 同事务清空任务级行的 `instance_id` / `last_runtime_status` / `public_ip` / `server_url`；**删除** `healCommentCSCInstanceFromTaskLevel`，禁止再从任务级领养 instance | 用户确认可清空旧数据；仅有任务级 instance、评论行仍空的任务视为未启动，需重新启动 |
| D7 | 不新增 HTTP 路径；扩展既有 `workspace-runtime-indicators` JSON；任务详情用现有 comment 作用域 API + 计数字段（可挂在 ui-context 或 indicators） | 无 Python 新接口；Go 落点 `taskCloudService` |
| D8 | 计数更新在同库同事务重算，**不发 Kafka** | 同服务投影，非跨边界副作用；见意图豁免 |
| D9 | 批准后架构 v80 + ADR-0007 | 数据所有权语义变更，不是纯 bugfix |

### 拒绝的方案

| 方案 | 拒绝原因 |
|------|----------|
| 只读时 COUNT、不落列 | 用户要求「标记」；列表/任务头需要稳定字段 |
| 任务级仍存 Starting/Running 再派生数量 | 正是当前串台来源 |
| 启动中也计入 | 用户明确「处于运行中」 |
| 删除任务级 CSC 行 | 模板（platform/region/auth）仍要给新评论克隆 |
| 保留一轮 heal 再清空任务级 instance | 用户确认可清空旧数据、不兼容存量任务；heal 会把误写的任务级 instance 领养到评论行，延长双读窗口 |

## 5. 数据模型

```
cloud_server_configs
  comment_id = ''     → 模板 + running_machine_count + running_container_count
  comment_id = cmt_*  → 唯一运行实例（instance_id, last_runtime_status, public_ip, server_url, …）
```

- DDL：`dataMigrate/taskCloudService/NNN_task_running_counts.sql`（**不兼容存量任务，可清空旧数据**）
  - `ADD COLUMN running_machine_count INT NOT NULL DEFAULT 0`
  - `ADD COLUMN running_container_count INT NOT NULL DEFAULT 0`
  - **清空**任务级行（`comment_id=''`）的 `instance_id`、`last_runtime_status`、`public_ip`、`server_url`
  - 两计数仅按**当前评论行**重算（已启动 / 已启动且 `server_url` 非空）；任务级旧 instance **不**参与回填、**不** heal 到评论行
- 伸缩：两列小整数，非时间累积；冷热/分片 **L0**（升级触发：单租户 CSC 年增量 > 100 万再评估按 tenant 分库）
- 禁止任务级行再写入：`instance_id`、`last_runtime_status`、`public_ip`、`server_url`（清空后保持空）

## 6. 写路径

评论 CSC 的 instance / last_runtime_status 变更后（启动回填、Describe、停机、释放）：

1. 只 UPDATE 该评论行
2. `recomputeTaskRunningCounts(tenant, workspace, task)`：一次重算两列  
   - `running_machine_count` = COUNT 评论行且已启动  
   - `running_container_count` = COUNT 评论行且容器可达（已启动 + server_url）
3. 结构化日志：`event=task_running_counts_updated task_id=… machines=N containers=M`

## 7. 读路径

| 消费者 | 行为 |
|--------|------|
| `GET server-runtime-status` | 继续强制 `comment_id`；只读评论 CSC |
| `GET workspace-runtime-indicators` | **忽略** `comment_id=''` 的 instance/status；`machine_running = machines>0`，`container_running = containers>0`；响应增加两计数 |
| 工作区机器摘要「已启动 N」 | 按 **instance_id 去重** 的评论级已启动集合（与今日 snapshot 一致），不把模板行算进去 |
| 任务详情壳 | 展示 `N 台机器 / M 个容器运行中`，不用任务级 Starting/Running |
| 评论卡 | 评论 CSC + binding 生命周期 |

## 8. API / 前端

- **不新增 path**。`workspace-runtime-indicators` 每项增加：
  ```json
  { "task_id": "…", "machine_running": true, "machine_starting": true, "container_running": true,
    "running_machine_count": 2, "running_container_count": 1 }
  ```
  `machine_starting` 仍可从评论行派生（不落任务级列）。
- 任务详情：从 indicators 或 ui-context 读 count；`cloudRuntimeStatus` 不再作为跨评论权威。
- 路径均已带 `tenant_id` + `workspace_id`（+ 详情 `task_id`/`comment_id`）。分片键 **tenant_id** 合适。

## 9. 路径分片键审视（NFR 预览，供 /5-nfr）

| 路径 | 已有 ID | 分片键判定 | 可伸缩性 | 动作 |
|------|---------|------------|----------|------|
| `GET .../workspace-runtime-indicators/tenant_id/{t}/workspace_id/{w}` | tenant + workspace | tenant 合适 | L1 | 扫描已按 tenant+workspace；计数列避免二次语义 |
| `GET .../server-runtime-status/...?task_id&comment_id` | tenant + workspace + task + comment | tenant 分片；comment 隔离 | L2 | 禁止回退任务级 |
| 任务详情路由 `.../task-detail/{taskId}/` | tenant + workspace + task | tenant 合适 | L1 | 壳上只渲染 count |
| 计数重算 SQL | 同库 WHERE task_id | 非跨服务 | L0 | 同任务评论数 ≪ 千 |

## 10. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 评论机器/容器进入或离开运行中并更新任务计数 | TaskRunningCountsRecomputed | `recomputeTaskRunningCounts` | 同函数写两列 | 证据豁免：同服务同库投影，不跨边界；登记 `_meta/publish_evidence_exempt.yaml` |
| 查询运行中数量 | — | indicators / 详情 | — | 纯查询 |

## 11. Domain Concept Inventory（供 /6-ddd）

| 概念 | 说明 |
|------|------|
| Bounded Context | 云资源 / 任务协作（taskCloudService） |
| 实体 | `CommentCloudServer`（评论 CSC）；`TaskCloudServerTemplate`（任务级行） |
| 聚合 | 评论 CSC 为运行一致性边界；任务级计数为评论集合的投影 |
| 领域事件（逻辑） | `CommentServerBecameRunning` / `CommentServerLeftRunning` → 投影重算（进程内） |

## 12. 价值流影响

现有流：`comment-binding-recover-after-start-success`、工作区机器摘要 / runtime-indicators、任务详情启动面板。

- 影响字段：`running_machine_count`、`running_container_count`（新）；任务级 `instance_id`/`last_runtime_status` 不再作运行态
- 测试：`comment_csc_start_persist_test.go`、`compute_workspace_runtime_indicators_test.go`、`machine_runtime_persist` 新测、`workPanelRuntimeIndicators.test.js`
- 新流步骤建议（/4-value-stream）：`task-running-comment-server-count`
- 不新增跨流依赖

## 13. 🕸️ Code Review Graph 分析

- 图存在（`.code-review-graph/graph.db`，108 nodes / 17 files，**仅 JS/TS/Python/bash**）
- `search cloudRuntimeStatus` / `workspace-runtime-indicators`：**0 节点**（Go `taskCloudService` 未入图）
- **CRG unavailable for Go blast radius**：设计以源码检索为准
- 已核对写点：`setCloudServerLastRuntimeStatus`、`persistStartVmInstanceBinding`、`UpsertAfterStart`、`computeWorkspaceMachineSnapshot`、FE `cloudRuntimeStatus` / `workPanelRuntimeIndicators`
- **删除** `healCommentCSCInstanceFromTaskLevel`（及「从任务级领养 instance」单测）；不再保留兼容路径

## 14. 🏛️ 架构变更影响（批准后写入）

- **迭代版本**: v80 🎯 target
- **需要更新的视图**: `application-integration`（数据语义 + 看板读 count）；`enterprise-landscape`（Cloud Resource 数据对象标注）
- **新增文件**（每个视图四类，已生成）:
  - 🆕 `docs/architecture/v80-application-integration-20260814-1437-cursor.puml`
  - 🆕 `docs/architecture/v80-enterprise-landscape-20260814-1437-cursor.puml`
  - 🆕 各视图 `.diff.archimate` / `.full.archimate` / `.mermaid.md`
- **变更明细（预告）**:
  - 🟡 [MODIFIED] taskCloudService — 任务级 CSC 仅模板+count；运行态仅评论 CSC
  - 🟡 [MODIFIED] `cloud_server_configs.running_machine_count` / `running_container_count`
  - 🟡 [MODIFIED] taskFE 看板/任务壳读 count
  - 🔴 [DEPRECATED] 任务级 `last_runtime_status`/`instance_id` 作为运行实例

## 15. 实施切片（批准后 /8-build）

1. Red：计数重算、禁止任务级写 instance/status、indicators 忽略模板行
2. DDL：加两列 + **清空**任务级运行态字段；两计数只按评论行重算
3. 收窄 persist / last_runtime_status / snapshot；**删除**任务级→评论 heal
4. FE indicators + 任务壳

## 17. 权限影响分析

见 `docs/superpowers/specs/2026-08-14-task-level-running-comment-server-count-permission-analysis.md`。不新增 HTTP 路径；复用工作区成员读云资源。评级绿灯。

## 16. 风险

| 风险 | 缓解 |
|------|------|
| 旧进程仍整任务刷 status | 精准编译重启 task-cloud-service + taskFE |
| 仅有任务级 instance、评论行仍空 | **接受**：清空后该任务显示 0 台 / 0 容器，需重新启动；不 heal、不双读 |
| 两计数与 snapshot 漂移 | 写入重算 + snapshot 用同一谓词；单测对照 |
| 看板头部「已启动 N」按 instance 去重 vs 任务标记按评论台数 | 头部是实例去重；任务两列是评论台数。共驻同机两评论：机器 count=2，头部实例摘要=1 |
