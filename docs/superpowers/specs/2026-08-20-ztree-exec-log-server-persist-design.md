# ztree 执行日志服务端持久化（容器关闭后仍可访问）

- **Date:** 2026-08-20
- **Status:** accepted（用户批准：写架构并进入下一步；范围锁定不存克隆日志）
- **Iteration:** ztree-exec-log-server-persist
- **User lock:** **不存克隆日志**（`container-clone-log` / exec-stream clone 仍仅容器内存；关容器后克隆进度/文本允许空白）
- **python_api_approval:** not_applicable（全 Go + Vue；无新增 Python endpoint）
- **Architecture change:** 是（层图快照 DataObject + Cloud hydrate；目标 **v89 target**，based_on **v88 current**；v86/v87 正交积压。v88 已被 CCB 启动日志分表占用，故本迭代升号）
- **Page:** 任务详情 ztree 任务关联 Tab + 选中节点的 **job 执行步骤**（不包含「项目克隆」条）

---

## 根据当前架构，系统现状

- 共有 2 个 **current** 架构视图：`enterprise-landscape` v88、`application-integration` v88（评论启动日志按 workspace_id 哈希分表；v85 多区域 gitService 已 archived）
- **业务层：** Developer / Task Management / Cloud Resource / IDE Workspace
- **应用层：** taskFE、APISIX、taskCloudService、taskContainerGateway、taskSSE、taskEvents、comment-scoped container（onlineServiceJS）
- **技术层：** MySQL（task_cloud，utf8mb4）、Kafka、Redis/SSE、Tencent COS（目前用于厂商证照，非执行日志）、Grafana/Loki（现网采集面为空，见 Trace 节）
- 上次 **current** 版本是 **v88**；积压 target：**v86**（PIPL 注销）、**v87**（taskFE nginx 静态常驻）。本次在 v88 基线上设计 **v89**，与 v86/v87 正交。

📋 架构版本历史（最近）：

- v88 (2026-08-20) ✅ current — 评论启动日志按 workspace_id 哈希分表
- v87 (2026-08-20) 🎯 target — taskFE Docker nginx 静态常驻
- v86 (2026-08-19) 🎯 target — 个人账号注销（PIPL）
- v85 (2026-08-18) 📦 archived — 可插拔多区域 gitService

本次迭代将在 **v88 current** 基础上设计 **v89**。

---

## 🔍 Trace 日志分析 (traceId: `9cc3bd8a21180182f9e36238`)

页面可见文本「启动 TraceId：9cc3bd8a21180182f9e36238」是**容器启动 TraceId**（`data-testid=comment-execution-start-trace-id`），不是前端报错节点。仍按技能强制检索 Loki。

- **Grafana Trace Dashboard:** http://10.2.150.68:3000/d/distributed-trace-view?var-trace_id=9cc3bd8a21180182f9e36238&var-tempo_trace_id=9cc3bd8a21180182f9e36238
- **Grafana 日志搜索:** http://10.2.150.68:3000/explore?orgId=1&left={"datasource":"loki","queries":[{"refId":"A","expr":"{job=~\".+\"} |= \"9cc3bd8a21180182f9e36238\"","queryType":"range"}]}
- **Loki:** `http://10.2.150.68:3100` — `/ready` = `ready`

### 查询尝试

| 步骤 | 查询条件 | 时间范围 | 结果 |
|------|---------|---------|------|
| 主查询 | `{job=~".+"} \| json \| trace_id` | 1h | 0 条 |
| 回退 1 | `{job=~".+"} \|= "<ID>"` | 24h | 0 条 |
| 回退 2 | `{job=~".+"} \| logfmt \| trace_id` | 24h | 0 条 |
| D1 扩大时间 | `{job=~".+"} \|= "<ID>"` | 7d | 0 条 |
| D2 任意日志 | `{job=~".+"}` | 1h / 7d | **0 条；job 标签 0 个** |
| D3 格式变体 | `(?i)` / `traceId=` | 7d | 0 条 |
| D3 前缀 8 位 | `9cc3bd8a` | 7d | 0 条 |

### 根因判定

- **根因:** Loki 采集管道无数据（`/ready` 健康但无 job、无任意日志），不是「该 TraceId 未打点」。
- **证据:** D2 任意 `{job=~".+"}` 7 天 limit=1 仍空；label `job` values 为空数组。
- **对本次设计的影响:** 无法用该启动 Trace 重建克隆/层图时间线；设计以代码路径为准。运行时「容器关闭后日志消失」由源码证实（见下），不依赖本次 Loki。
- **短期:** 不阻塞本设计。长期：修复 promtail/fluent-bit 采集（独立运维项，不纳入本迭代实现范围）。

---

## 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | `.code-review-graph/graph.db` 存在；`status`：108 nodes / 937 edges / 17 files；语言 javascript, typescript, python, bash；branch `main` |
| 关键发现 | 图覆盖面偏前端/脚本，**未索引** `taskCloudService` / `taskContainerGateway` Go 符号；对 `handleContainerJobExecutionLogFromDB`、`job_stream_inbound` 的 search/impact 为空 |
| 决策影响 | 爆炸半径改由静态检索：Cloud GET hydrate、Gateway L0 代理、容器 `execStream` 内存 Map、FE `refreshZTreeExecutionLog` |
| skip 理由 | 部分 `unavailable`（Go 服务未入图）；结论不依赖 CRG |

---

## 1. 问题

用户在任务详情评论执行细节要求 ztree **执行日志**服务端存储、按 workspace/task 分区、容器关闭后仍可访问，并分析 JSON 文件。**2026-08-20 用户锁定：不存克隆日志。**

一期范围因此是：

1. **层图快照**（ztree 树：layers/jobs）落到服务端
2. **Agent job 步骤**继续走已有 `cloud_job_execution_event`，关容器后可读
3. **按工作空间 ID / 任务 ID** 作为访问与分片键
4. **不采用**宿主机 JSON 文件当权威存储
5. **明确不做：** 克隆进度条、`container-clone-log`、exec-stream clone 分片的服务端落库

### 1.1 现状（代码）

| 数据 | 今日权威存储 | 容器关闭后 |
|------|----------------|------------|
| Agent 步骤 / job 生命周期 | `cloud_job_execution_event`（023，按 `created_at` RANGE 分区；GET 已由 Cloud hydrate） | **理论上可读**（依赖 PUSH 已发生；chunk 明文不落库） |
| 层图（ztree 树：layers/jobs） | 容器内存 + `layer-graph-push` **只发 SSE**，Cloud **不落库** | **丢失**（刷新后空树） |
| 克隆日志 / exec-stream | 容器进程内 `Map`（`execStream.mjs`），GET `container-clone-log` **经 Gateway 代理容器** | **丢失** |
| 绑定启动时间线 | `cloud_comment_container_binding_logs` | 可读（非 ztree 执行日志） |

前端 `refreshZTreeExecutionLog` 同时打：

- `GET .../container-clone-log/?layer_id=` → Gateway → 容器（**本期保持如此，关容器后允许失败/空白**）
- `GET .../container-job-execution-log/?job_id=` → Cloud 读库

「容器关了还能看 ztree」失败的主因是 **层图只 SSE、不落库**；job 步骤已有表。克隆日志按用户锁定 **继续绑在容器生命周期**。

### 1.2 与已有意图的关系

`docs/intents/backend/container_job_stream_kafka_persist.intent.md` 已覆盖 **job 步骤** 落库。本设计 **不替换** 该表；只补 **层图快照**，并规定 **workspace_id / task_id 为访问与分片键**。

---

## 2. JSON 文件方案分析（用户点名）

「直接存 JSON 文件」有三种完全不同的含义，必须拆开：

| 方案 | 含义 | 分区形态 | 结论 |
|------|------|----------|------|
| **A. 宿主机/应用盘裸 JSON** | `data/exec-logs/{workspace_id}/{task_id}/....json` | 目录即分区 | **否决作为权威存储** |
| **B. 对象存储上的 JSON/JSONL** | COS key：`exec-logs/{workspace_id}/{task_id}/{comment_id}/...` | key 前缀即分区 | **大 blob 升级路径，一期不引入** |
| **C. MySQL 中的 JSON 文档** | 表内存 JSON/MEDIUMTEXT；查询键含 workspace/task | 列 + 时间 RANGE；预留 HASH(workspace) | **一期采纳** |

### 2.1 为何否决 A（本地 JSON 文件当 SSOT）

1. **容器内 JSON/内存已经是这种模式**：`execStream` 在容器进程里；容器释放后文件/内存一起没。把同样的文件放到「某一台 SaaS 宿主机」只是把 SPOF 从容器换到单机磁盘。
2. **多实例 / 精准编译重启 / 换机**：文件不在共享存储则读不到；与 runAll 多进程、Cloud 无状态重启冲突。
3. **无索引、无事务、并发 append 需自研锁/fsync/轮转**；utf8mb4、备份、审计、表前缀/owner 规范全部旁路。
4. **冷热与分片元规则**：时间累积日志要求分区/归档方案；裸文件没有 `TRUNCATE PARTITION`、没有 Prometheus 分区指标。
5. **单库单表所有权**：文件目录不是 `dataMigrate` 可迁移资产，owner 边界比表更糊。

### 2.2 为何一期用 C 而不是 B

- 层图快照通常 **远小于 1MB**（layers + jobs 数组），MySQL JSON 列足够；**不存克隆日志后更没有大 blob 理由去上 COS。**
- 项目已有 Cloud MySQL + 023 job 表；快照 UPSERT 与 job hydrate 同一 owner。
- **升级触发：** 单评论 `graph_json` 中位 > 1MB 或快照表需水平分库时，再考虑 COS key `.../{workspace_id}/{task_id}/layer-graph.json`。

### 2.3 「JSON」仍然用在载荷里

一期 **采用 JSON 作为快照文档格式**（`graph_json` 列），**不采用本地 JSON 文件作为权威介质**。回答「是否直接存 JSON 文件」：**不要用宿主机文件；表里存 JSON 文档即可。**

---

## 3. 选定方案

### 3.1 两类产物、一个 owner

**Owner：`taskCloudService`（库 `task_cloud`，表前缀 `cloud_`）。** 他服务只经 Cloud API / 已有 Kafka SSE 路径，禁止直连表。

| 产物 | 表 | 形状 | 分区 / 分片 |
|------|----|------|-------------|
| 层图最新快照 | `cloud_layer_graph_snapshot` | 每评论一行 last-write-wins；`graph_json` JSON | 访问键 `(workspace_id, task_id, comment_id)`；小表，**HASH 预留、一期不分片** |
| Job 步骤 | 已有 `cloud_job_execution_event` | 不改语义 | 已 RANGE(time)；hydrate **必须**用 path 上的 workspace/task，禁止只靠 job_id 全表扫 |

stdout **chunk 仍不落库**。克隆日志 **不落库**。

### 3.2 写路径（无进程内 poll）

```text
容器 onlineServiceJS
  └─ layer-graph-push（已有）──► Cloud handleLayerGraphPush
        1) UPSERT snapshot（workspace/task/comment）
        2) 发布 SSE_MESSAGE status=container_layer_graph（保持直播）
        3) 发布领域事件 LayerGraphSnapshotPersisted
```

禁止 Gateway/Cloud `ticker` 去容器拉日志。容器关闭后 **不再有 PUSH**，层图与 job 步骤读只走库。克隆日志仍代理容器（关则无）。

### 3.3 读路径（容器关闭仍 200）

| 浏览器 GET | 行为 |
|------------|------|
| `container-layer-graph` | **Cloud 先读 snapshot**；无行则空图（关闭后不得因 Gateway 502 挡整页 ztree） |
| `container-job-execution-log` | 保持 Cloud hydrate（已实现） |
| `container-clone-log` | **不变**：仍 Gateway→容器；关容器后失败/空白为预期 |

前端：`containerEndpointRegistered=false` 时仍 GET 层图快照与 job 步骤并渲染；克隆区允许空。

### 3.4 路径与分片键（NFR 预览，供 /5-nfr）

| 路径 | 已带 ID | 分片键判定 |
|------|---------|------------|
| `GET/POST /api/cloud/compute/.../tenant_id/{t}/workspace_id/{ws}/task_id/{task}/comment_id/{cmt}/...` | tenant, workspace, task, comment | **workspace_id 适合租户级分库；task_id 对齐本页访问**。comment_id 是实体键但基数更高，作查询后缀而非一级分片。 |
| Kafka `SSE_MESSAGE` 今日多按 `task_id` | task | 保持；payload 必须带 `workspace_id` 供消费者写表 |
| 对象存储升级 key | 须 `.../{workspace_id}/{task_id}/...` | 禁止只按 job_id 扁平堆 |

禁止新增「只有 job_id / layer_id、没有 workspace+task」的公网读接口。

### 3.5 幂等（NFR 预览，供 /5-nfr）

| 写路径 | 重复边界 | 幂等键 | 重放语义 |
|--------|----------|--------|----------|
| layer-graph-push | 同一评论最新树 | `(workspace_id, task_id, comment_id)` UPSERT | 后写覆盖；L2 |
| job-stream-push | 已有 | `(task_id, job_id, seq)` | 保持；查询侧加上 workspace 过滤 |

禁止用单独 `tenant_id`/`user_id` 当快照或 job 事件幂等键。

### 3.6 冷热

- 快照：跟评论生命周期；容器释放后 **仍保留**；归档与评论/任务保留策略对齐（默认热 90 天）。
- Job 事件表：沿用既有按月 RANGE。

### 3.7 明确不做（一期）

- **克隆日志 / 项目克隆进度服务端存储**（用户锁定）
- 本地 JSON 文件 SSOT、共享 NFS 目录、COS 执行日志桶
- stdout chunk 写入 MySQL；`cloud_exec_stream_segment` 表
- 持久化完整工作区文件树 / 文件内容
- 新建微服务；不新增 Python 接口
- 修复 Loki 采集管道（运行时债，非本功能验收）

---

## 4. 方案对比（落地选型）

| # | 方案 | 优点 | 缺点 | 推荐 |
|---|------|------|------|------|
| 1 | 宿主机 JSON 文件按 ws/task 目录 | 实现快、分区直观 | SPOF、无 HA、旁路迁移/owner | ❌ |
| 2 | 仅 COS JSON | 天然按 key 分区、大日志便宜 | 小快照过重；无现成 Cloud COS 执行日志桶 | 二期升级 |
| 3 | 023 job 表 + 一张快照表（JSON 列） | 关容器仍见 ztree 与步骤；无克隆 blob | 克隆进度关容器后空白（已锁定） | ⭐ 一期 |
| 4 | 继续 Gateway 代理容器 | 零开发 | **不满足关容器仍访问** | ❌ |

---

## 5. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 容器上报层图 | LayerGraphSnapshotPersisted | taskCloudService `handleLayerGraphPush` 落库后 | SSE 直播（既有 `container_layer_graph`） | — |
| 容器上报 job 步骤 | SSE_MESSAGE（既有 `container_job_stream`） | 既有 job-stream-push | `2_persist_job_execution_event` | 本迭代不改契约 |
| 页面拉取历史层图/步骤 | — | GET Cloud | 读库 | 纯查询 |
| 页面拉取克隆日志 | — | GET Gateway→容器 | — | 纯查询；**无服务端事实、不落库**（用户锁定） |

---

## Domain Concept Inventory

- **Bounded Contexts:** Cloud / Container runtime（容器为临时执行者，非日志 owner）
- **Key Entities:** CommentContainerBinding、LayerGraphSnapshot、JobExecution
- **Candidate Aggregates:** LayerGraphSnapshot 根 = `(workspace_id, task_id, comment_id)`
- **Domain Events:** 见上表
- **业务意图 → 事件：** 见上表

---

## Value Stream Impact

现有 `conf/value-stream.yaml` 已有：

- `taskEvents.sse_message.persist_job_execution_event`
- `taskCloudService.job_execution_log.read_db`

本需求：

- **影响流：** 云平台与资源 / 任务协作（任务详情评论执行细节、ztree）
- **新步骤（建议 `/4-value-stream` 落地）：** `layer-graph-snapshot-persist`、`layer-graph-hydrate-from-db`
- **字段（三段式）：**
  - `task-cloud-service.cloud_layer_graph_snapshot.graph_json`
  - 已有 `task-cloud-service.cloud_job_execution_event.*`
- **测试：** Cloud snapshot UPSERT/GET；FE：endpoint 未注册时仍渲染 ztree 与 job 步骤；克隆区允许空
- **Status：** 新步骤 `planned` → 交付后 `active`

Looking at the existing value streams, this change extends job-execution persist and adds layer-graph snapshot hydrate（不含克隆日志）；full slicing in `/4-value-stream`.

---

## 🏛️ 架构变更影响

- **迭代版本:** v89 🎯 target
- **迭代名称:** ztree 执行日志服务端持久化
- **作者:** cursor
- **设计日期:** 2026-08-20 15:49
- **每个视图已交付四类文件:**
  - 🆕 `docs/architecture/v89-enterprise-landscape-20260820-1549-cursor.puml`
  - 🆕 `docs/architecture/v89-application-integration-20260820-1549-cursor.puml`
  - 🆕 各视图 `.diff.archimate`（增量：v88→v89 变更 + Plateau/Gap/WP）
  - 🆕 各视图 `.full.archimate`（全量拓扑：v88 CCB 分表 + 层图快照）
  - 🆕 各视图 `.mermaid.md`
- **已有文件（未修改）:** v88 current；v86/v87 正交积压
- **变更明细:**
  - 🟢 [NEW] `cloud_layer_graph_snapshot`
  - 🟢 [NEW] `LayerGraphSnapshotPersisted`
  - 🟡 [MODIFIED] taskCloudService `handleLayerGraphPush` 落库；`container-layer-graph` GET hydrate
  - 🟡 [MODIFIED] taskFE 关容器仍展示层图与 job 步骤
  - 无标记 — 023 job 表、clone-log 仍代理容器、Kafka job-stream、CCB 16 分表

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v88 → Gap「层图仅 SSE 不落库」→ WP → Plateau v89；Container→Cloud snapshot→FE GET |
| **`.full.archimate`** | v88 全量（含 CCB 分表）+ 新 DataObject 与 hydrate 流 |

---

## 实施切片（批准后 /8-build，测试先行）

1. Characterization：layer-graph-push 不写库；关容器后 GET 层图失败。
2. DDL `031_cloud_layer_graph_snapshot.sql`（utf8mb4、前缀 `cloud_`；030 已被 v88 CCB 分表占用）。
3. Cloud UPSERT snapshot + GET hydrate；单测无容器仍 200。
4. FE：endpoint 未就绪仍渲染快照与 job 步骤；克隆区不强制有文本。
5. 经 9999 初始化库；精准编译重启 Cloud/FE。旧容器从未 PUSH 的树不可还原。

---

## 验收标准

1. 层图 PUSH 后 **停止/释放容器**，硬刷新：ztree 仍能展开（快照）；选中节点 job 步骤仍来自 023。
2. 「项目克隆」关容器后允许空白；`container-clone-log` 不改为读库。
3. 新表/API 路径含 `workspace_id`+`task_id`；无本地 `*.json` SSOT。
4. 同评论层图 UPSERT 不插出行。
5. 无新增 Python HTTP 接口；无业务进程 ticker。

---

## 7. 权限影响分析

见 `.claude/skills/2-role-permission/permission-analysis-2026-08-20.md`。无新角色。写路径沿用容器 access_token + URL scope；读路径沿用认证用户 compute。查询必须带 `workspace_id`+`task_id`+`comment_id`。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| layer-graph-push | 容器 token | Comment | write | token + pathMatchesScope | ✅ | 落库键取 CSC，禁止 body 改 task |
| GET container-layer-graph | 工作区成员 | Comment | read | 会话 + 路径 ID | 实施时强制三键 WHERE | 无行返回空图 200 |
| GET job-execution-log | 工作区成员 | Comment | read | 已有 | ✅ | 不改 |
| GET clone-log | 工作区成员 | 容器 | read | Gateway | ✅ 不改 | 关容器空白 |

## 8. 角色与权限建模

不适用。不新增角色、region、page ACL。

## 9. 安全审查结论

- IDOR：三键定位；缺键 400；跨 workspace 不得读到快照。
- 无 user_id 注入；无新敏感删除接口。
- 空图 200，避免用 404 探测评论是否有树。
- 结论：绿灯（实施须带 IDOR 单测）。
