# 指令完成后闲置回收（任务详情字段 + 容器倒计时）

- **日期:** 2026-08-22
- **状态:** accepted（总体设计已批准；架构 v98 target）
- **作者:** cursor
- **ADR:** [ADR-0031](../../adr/0031-instruction-idle-recycle-no-sts-in-comments.md)（accepted）
- **页面:** 任务详情（容器消费 `task-detail`）；工作空间设置「闲置自动回收」已存在，本迭代不改配置 UI
- **python_api_approval:** scoped-down（零新增 Python HTTP 接口）

## 当前架构理解

根据 current 架构（v96 shipped）与既有实现：

- **业务层:** 租户成员在任务评论向镜像下指令；工作空间可配「闲置自动回收分钟」（0=关闭）
- **应用层:** taskFE、taskCredentialService（容器 `task-detail`）、taskCloudService（CSC / stop-vm / `request-machine-release`）、taskEvents `workspace_machine_idle/1_recycle_idle_nodes`、onlineServiceJS（容器内 job）
- **技术层:** MySQL `task_cloud.cloud_workspace_machine_policies.idle_recycle_minutes`；阿里云 ECS DeleteInstance 只用 CPA 长期 AK（容器从不持有阿里云密钥）
- **上次版本:** v96 📦 archived — 管理员订单分账只读
- **当前版本:** v98 ✅ current — 指令闲置回收

📋 架构版本历史（摘）：

- v19 ✅ — 工作空间机器闲置策略：policy + 卸载后 `idle_since` + timer 回收
- ADR-0013 ✅ — 去掉跨任务闲置复用；回收机制保留
- v96 📦 archived — 与本需求无关的分账展示
- v98 ✅ current — 指令闲置回收

本次在此基础上把「闲置」从「容器已卸载」扩展为「**本评论容器上一条指令已交付成功且无新指令**」。

## 🕸️ Code Review Graph 分析

`code-review-graph status` 可用，但图仅索引 17 个 JS/TS/Python 文件，**未覆盖 Go 服务**。按源码调用链设计（`CRG: graph incomplete for Go — proceeded from grep`）。

既有链：

- 容器 `POST …/server-container-token/task-detail/` → `taskCredentialService.FetchTaskDetail` → `ContainerTaskDetail`（无策略字段）
- 工作空间策略 SSOT：`cloud_workspace_machine_policies.idle_recycle_minutes`；GET `workspace-machine-policy/` / summary
- 现网回收：`idle_since` 在 **server_url 清空** 时写入；timer 调 `recycle-idle-machines` → stop-vm。`server_url` 仍在（容器还挂着）**不会**回收
- 容器已有释放口：`request-machine-release` → `CLOUD_SERVER_STOPPED`；`taskLifecycleShutdown`（终态）已走 interrupt → layer-graph → release
- 新指令：同评论 CSC 上 `POST /jobs` 或 instruct 流；`createJob` **不会**全局抢占其它 running job
- 交付：`finalizeJobCloseSideEffects` 对 mounted complete / auto_run delivery 目前 **soft-fail**

## 问题

1. 容器 `task-detail` 看不到工作空间 `idle_recycle_minutes`，无法按策略自己倒计时。
2. 现网回收只覆盖「容器卸载后机器空转」。指令跑完、容器仍注册时，机器会一直开着。
3. 用户要求：指令完成后按策略倒计时；到期无新指令则释放；倒计时结束前（以及旧指令仍在跑时）收到新指令则中断旧指令、执行新指令。
4. 顾虑：容器连不上平台时无法 `request-machine-release`；指令结果提交失败时若先拆机则丢结果。

## 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 容器 `task-detail` 带上本工作空间 `idle_recycle_minutes`（0=关闭） | 契约单测；与 policy GET 同值 |
| S2 | 指令**进程结束且结果已成功提交平台**后开始倒计时 | 交付失败不启动倒计时；有重试 |
| S3 | 倒计时到期无新指令 → 释放本评论 CSC 对应机器 | 复用 `request-machine-release`；reason=`instruction_idle` |
| S4 | 倒计时内或旧 job 仍 running 时收到新指令 → 中断旧 job、取消倒计时、跑新指令 | 仅**同一评论容器**内；不影响其它评论的独立机器 |
| S5 | 结果未交付成功 → 禁止释放（含禁止 STS 拆机） | 单测：delivery 失败不调 release / 不调阿里云 |
| S6 | 平台对容器不可达时仍能回收「已交付、已闲置」的机器 | 服务端 `instruction_idle_since` + 现有 timer；heartbeat 仅作辅助 |
| S7 | 零新增 Python 接口 | 全部 Go + onlineServiceJS |

## 决策（锁定草案）

### 1. 闲置时钟（已与用户确认）

```
新指令到达 ──(若有 running/pending)──► interrupt 旧 job（不交付）
        │
        ▼
   执行新指令
        │
        ▼
   本地 job 结束
        │
        ├─ 交付平台失败 ──► 重试交付；不倒计时；不释放
        │
        └─ 交付成功 ──► 标记 instruction_idle；开始 N 分钟倒计时
                              │
                              ├─ 新指令 ──► 取消倒计时；回到顶部
                              └─ 到期无新指令 ──► 释放机器
```

- **N** = 工作空间 `idle_recycle_minutes`。`0` 仍表示关闭（不倒计时、不因指令闲置释放）。
- 「交付成功」：本 job 需要对平台做的收尾都成功（mounted `complete`/`fail`、auto_run/edit_run delivery、layer-graph 快照至少一次成功）。无交付义务的 job（普通非 auto_run）在进程结束后即视为可闲置。
- 中断的旧 job **不**走 complete 交付（与现 `finalizeJobCloseSideEffects` interrupted 语义一致）。

### 2. 任务详情字段（需求 1）

`POST …/server-container-token/task-detail/` 增加：

```json
{
  "idle_recycle_minutes": 5,
  "instruction_idle": {
    "enabled": true,
    "minutes": 5
  }
}
```

- `idle_recycle_minutes` 与设置页同一 SSOT（`cloud_workspace_machine_policies`；无行则默认 5，与现 GET policy 一致）。
- `instruction_idle.enabled` = `minutes > 0`。
- **taskCredentialService** 通过 **internal GET** 读 taskCloudService（policy 表 owner 在 Cloud；禁止 Credential 直连 `task_cloud`）。
- Cloud 新增只读 internal：`GET /api/internal/cloud/workspace-machine-policy/?company_id=&workspace_id=`（供 Credential 组装 task-detail）。**不是** Python 接口。
- 评论正文 / `context_pack` **不**承载该字段（避免评论存储与日志扩散）；容器以 task-detail 为准，bootstrap 后已有拉取。

### 3. 释放路径：平台优先，STS 不进评论（回应网络顾虑）

**禁止**把阿里云 STS/AK 写入评论、`context_pack`、用户可见 SSE。

三层释放，按序：

| 层 | 何时 | 谁 | 覆盖的故障 |
|----|------|----|------------|
| **L1 容器主动** | 倒计时到期 | 容器 `request-machine-release`（已有），`reason=instruction_idle` | 正常路径 |
| **L2 平台 timer** | CSC 上 `instruction_idle_since + N` 已过，且无新指令心跳 | 现有 `recycle-idle-machines` **扩展**（仍由 taskEvents 叫醒，无业务进程内 ticker） | 容器交付后崩溃；容器当时连不上平台 |
| **L3 可选 STS** | L1 失败（平台 HTTP 不可达）**且**交付已成功 **且** CPA 配置了回收专用 RAM Role | 容器用 **task-detail 或 inbound 签发的短时 STS** 调 `DeleteInstance`（Resource 仅本 `instance_id`） | 平台宕机、阿里云仍可达；结果已在平台 |

**对「跑着跑着连不上平台」的精确回答：**

- 指令**还在跑**、结果尚未交付：容器连不上平台 ⇒ 平台看不到 `instruction_idle_since`。此时 **不**按 N 分钟拆机（避免网络抖动杀掉长任务、避免丢未提交结果）。依赖现有 heartbeat / orphan 的**更长**不可达窗口（本迭代不缩短为 N）。
- 指令**已交付成功**后连不上平台：交付时平台已写入 `instruction_idle_since`，L2 timer 用 CPA 密钥 DeleteInstance，**不依赖**容器再连回来。
- 平台整体宕机、交付已成功、L1 失败：才允许 L3 STS。

**对「提交结果失败怎么办」的精确回答：**

- 交付失败 = 未闲置。本地保留 output，指数退避重试（上限与现 outbound 日志一致）。
- 重试期间收到新指令：中断旧 job（旧结果仍尽量 fail/complete 一次 best-effort），执行新指令。
- **在 `delivery_pending` 时禁止 L1/L3 释放。** L2 也不会有 `instruction_idle_since`。

STS 签发（仅 L3，且 CPA 有 `sts_release_role_arn`）：

- `AssumeRole` + session policy：`ecs:DeleteInstance` / `ecs:DescribeInstances`，Resource = 该实例 ARN。
- TTL = `min(3600, (idle_recycle_minutes + 15) * 60)` 秒；每次 task-detail 或专用 inbound 可刷新。
- 字段放在 task-detail 的 **`machine_release_sts`（可选，可空）**，不进评论。无 Role 则字段省略，L3 不可用。
- 这是架构决策，见 ADR-0031。

### 4. 服务端状态（L2）

`cloud_server_configs` 增列（评论级 CSC 行）：

- `instruction_idle_since DATETIME NULL` — 本容器最近一次「指令已交付、进入闲置」的 UTC 时间
- 新指令入站（heartbeat 带 `busy=true`，或 Cloud 收到对该 comment 的 instruct/start-job 代理）时清空

容器在交付成功后调用已有 `heartbeat` 或扩展 payload：

```json
{ "access_token": "...", "instruction_idle": true }
```

Cloud 写入 `instruction_idle_since=now`（若尚空或刷新为 now）。新 job 开始时 heartbeat `instruction_idle: false` 清空列。

回收扫描（扩展现 `recycleIdleMachinesForWorkspace`，同一 timer，不新建 ticker）：

- 原条件保留：`server_url` 空 + 旧 `idle_since`（卸载闲置）
- **新条件：** `instruction_idle_since` 非空且 `now - instruction_idle_since >= N`，即使 `server_url` 非空也回收本 CSC 实例（sole busy 门禁与 `request-machine-release` 相同）

事件：

| 业务意图 | 事件名 | 发布点 | 消费者 |
|---------|--------|--------|--------|
| 指令交付成功进入闲置 | `CONTAINER_INSTRUCTION_IDLE_MARKED` | taskCloudService heartbeat/inbound | 审计；L2 扫描读 DB 即可，消费者可无 |
| 新指令取消闲置 | `CONTAINER_INSTRUCTION_IDLE_CLEARED` | 同上清空列时 | 审计 |
| 闲置到期释放 | `CLOUD_SERVER_STOPPED`（已有） | stop 编排 | 现有 clear-after-stop |

`CONTAINER_INSTRUCTION_IDLE_*` 为状态事实，须投递 MQ（不得只改列不发事件）。L2 扫描本身是 timer 一次性 API，释放成功仍发 `CLOUD_SERVER_STOPPED`。

### 5. 容器 onlineServiceJS

- 读 task-detail 的 `idle_recycle_minutes`；0 则不启动 timer。
- **禁止**业务服务进程内 ticker 元规则约束的是 **SaaS 业务 HTTP 进程**；容器内一次性 `setTimeout` 倒计时是对「本机闲置」的外界触发等价物，允许（与本地倒计时例外同类）。到期只触发一次 release，新指令 `clearTimeout`。
- 新指令入口（`createJob` / instruct）：先 `interruptJob` 所有 running/pending，再 cancel idle timer，再跑新 job。
- 到期：先 `postRequestMachineRelease({ reason: 'instruction_idle' })`；失败且存在 `machine_release_sts` 才走阿里云 DeleteInstance。
- 镜像契约：更新 `docs/skills/saas-container/saas-machine-container.md` task-detail 字段；`trae-agent` 提交后须 `DOCKER_PUSH=1 ./buildDocker.sh`。

### 6. 与现网卸载闲置的关系

两者并存：

- **卸载闲置**（v19）：容器 unregister，`idle_since`，timer 回收
- **指令闲置**（本迭代）：容器仍注册，但指令已交付且无新指令

不恢复跨任务闲置复用（ADR-0013 不变）。

## 领域概念（给 /6-ddd）

| 概念 | 说明 |
|------|------|
| Bounded Context | Cloud（机器生命周期）、Credential（容器引导契约）、Container Runtime（onlineServiceJS） |
| Entity | WorkspaceMachinePolicy、Comment-scoped CSC、InstructionIdleWindow |
| Aggregate | CSC（评论级）是释放一致性边界；Policy 是工作空间配置根 |
| Domain Events | 见上表 |

## 价值流影响

- 现有 stream `workspace-machine-idle-policy`：扩展字段 `instruction_idle_since`；timer 行为扩展
- 现有 `task-detail-repo-clone-credentials-contract` / bootstrap-task-detail：task-detail 增只读策略字段
- 新 stream（建议 Step 4 切片）：`instruction-idle-recycle`（域：云平台与资源 + 任务协作）
- 测试：Credential task-detail 单测、Cloud recycle 单测、onlineServiceJS timer/preempt/delivery-gate 单测；不改设置页 Playwright（字段 SSOT 已有）

字段名（三段）：

- `task-cloud-service.cloud_workspace_machine_policies.idle_recycle_minutes`（已有）
- `task-cloud-service.cloud_server_configs.instruction_idle_since`（新）
- `task-cloud-service.cloud_cloudplatformauthorization.sts_release_role_arn`（新，可选）

## 🐍 Python 新增接口清单与 Go 替代评估

### 拟新增接口

无。

### 选型结论

- **最终选择:** Go（taskCredentialService 读组装、taskCloudService owner、taskEvents timer）+ 容器 Node
- **python_api_approval:** scoped-down

## 非目标

- 跨评论抢占（每评论独立 CSC/机器）
- 恢复 `prefer_idle_reuse`
- 把 STS 写入评论或用户可见日志
- 指令执行中因短时心跳失败按 N 分钟拆机
- 前端展示倒计时 UI（可后续）

## 🏛️ 架构变更影响

- **迭代版本**: v98 ✅ current
- **迭代名称**: instruction-idle-recycle
- **作者**: cursor
- **设计日期**: 2026-08-22 15:55
- **新增文件**（每个视图四类伴生格式）:
  - 🆕 `docs/architecture/v98-enterprise-landscape-20260822-1555-cursor.puml`
  - 🆕 `docs/architecture/v98-application-integration-20260822-1555-cursor.puml`
  - 🆕 `docs/architecture/v98-enterprise-landscape-20260822-1555-cursor.diff.archimate`
  - 🆕 `docs/architecture/v98-application-integration-20260822-1555-cursor.diff.archimate`
  - 🆕 `docs/architecture/v98-enterprise-landscape-20260822-1555-cursor.full.archimate`
  - 🆕 `docs/architecture/v98-application-integration-20260822-1555-cursor.full.archimate`
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**:
  - `docs/architecture/v96-*-20260822-1405-cursor.puml` (archived)
- **变更明细**:
  - 🟢 [NEW] task-detail `idle_recycle_minutes` / 可选 `machine_release_sts`
  - 🟢 [NEW] CSC `instruction_idle_since`；事件 `CONTAINER_INSTRUCTION_IDLE_MARKED` / `_CLEARED`
  - 🟡 [MODIFIED] recycle timer 识别指令闲置；onlineServiceJS 倒计时 + 抢占
  - 🟡 [MODIFIED] taskCredentialService 调 Cloud internal policy
  - ⚪ 不改 Django

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | 增量 — Plateau v96 → Gap → WP → Plateau v98；变更数据流（task-detail / idle heartbeat / recycle / DeleteInstance） |
| **`.full.archimate`** | 全量 — 变迁后本切片完整拓扑（OSJS / Credential / Cloud / Events / CSC / policy / ECS） |
