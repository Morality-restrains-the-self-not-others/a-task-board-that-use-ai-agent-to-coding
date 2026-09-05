# 设计：任务终态硬释放机器节点 + 镜像容器迁回所属任务

- **日期**: 2026-07-15
- **作者**: claude
- **状态**: approved（goal-mode 自动采纳，跳过确认门）
- **迭代名**: `terminal-hard-release-container-migrate`
- **相关页面**: `https://www.daydaymoney.com/tenant/{t}/workspace/{w}/task-detail/{taskId}/?relayToTrae=true`
- **架构版本**: v27 🎯 target（基于 v25 ✅ current；与 v26 并行 target）
- **python_api_approval**: n/a（**零新增** Python/Django HTTP 接口；全部落 Go）
- **前置**: v17 终态释放服务器；v19 闲置复用 / 回收

---

## 1. 问题 / 意图

| # | 缺口 |
|---|------|
| 1 | 任务进度变为「已完成 / 已取消」后，关联机器节点仍可能进入**闲置复用池**（`prefer_idle_reuse` 从终态任务 CSC 选中同 `instance_id`），与产品「终态后不可再复用、须释放」冲突 |
| 2 | `bindSharedMachineToTask` 复用时**不清空源任务**的 `instance_id`，多任务 CSC 共享同一实例；终态任务触发 `CLOUD_SERVER_STOPPED` 会误杀仍在跑容器的其他任务 |
| 3 | 同实例上若仍有**其他任务的镜像容器**（`server_url` 非空），释放前缺少「迁到所属任务自己的机器节点（无则新建）」编排 |

### 与用户表述对齐

> 当任务进度状态转为完成或者取消时，任务所关联的机器节点将不允许再被重复使用并且需要释放；如果该机器节点上运行了镜像容器，需要把镜像容器转移到其所属任务的机器节点上（如果没有的话，可以新建）。

| 短语 | 本设计解释 |
|------|------------|
| 不允许再被重复使用并且需要释放 | **硬释放**：终态任务解绑并 stop/destroy 实例（在安全前提下）；该绑定**永不**再进入 idle reuse 候选 |
| 镜像容器转移到其所属任务的机器节点 | 同 `instance_id` 上 **其他非终态任务** 的 busy 容器 → 迁到该任务独占机器（新建或合法复用**其他**闲置机，**禁止**复用正被硬释放的实例） |
| 终态任务自身容器 | 所属任务已终态 → **停止即可**，不迁回 |

---

## 2. 成功标准（SMART）

| # | 标准 | 可验证方式 |
|---|------|------------|
| S1 | 任务进入已完成/已取消 → 其 CSC 解绑且实例在无剩余 busy 兄弟后被 stop | Go 单测 + 事件断言 |
| S2 | 终态任务的 `instance_id` **不会**再被 `findIdleMachineForReuse` 选中 | 单测：终态后 start-vm 不 reuse 该实例 |
| S3 | 同实例上其他任务有 `server_url` → 先迁移再释放；迁移目标为所属任务机器（无则 start-vm 新建） | 单测：migrate 后兄弟 CSC 新 instance；原实例可安全 stop |
| S4 | 仅终态任务独占实例且 busy → 停容器 + 硬释放，无 migrate | 单测 no-op migrate |
| S5 | 幂等：重复消费不重复建机、不重复 stop 已释放资源 | 单测 |
| S6 | 无新 Python HTTP；Swagger 对新增/变更 Go 内部 API 可见 | 路由归属 / OpenAPI |
| S7 | 意图文档 + 价值流测试点登记 | intents / value-stream |

---

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 结论 |
|------|------|------|
| **A. 扩展 taskEvents 终态 intent：先 migrate 兄弟容器，再 hard-release；Cloud 侧修复 reuse 解绑 + 排除终态候选** | 与 v17 同事件链；编排在 taskEvents；机器 SSOT 在 taskCloudService | **采纳** |
| B. 仅禁止 idle reuse，终态不 destroy（软释放） | 不满足「需要释放」；费用与安全残留 | 拒 |
| C. 前端改状态后同步调 migrate/stop | 多入口漏触发；违反事件驱动 | 拒 |
| D. 新建独立 machine-migrate 微服务 | 过度设计 | 拒 |

### 自主决策

1. **迁移语义**：容器无法热迁移跨主机 → **停旧 + 在目标机按原镜像/模板重建**（best-effort 保 `container_image_id` / server_run_template）；工作区状态不保证进程内内存连续。
2. **兄弟判定**：同 `company_id+workspace_id+instance_id`，`task_id != 终态任务`，且 `server_url` 非空（busy）或 `last_runtime_status` 表明容器可达。
3. **新建机器**：对兄弟任务调用既有 `start-vm` / `start-vm-auto`，**强制 `forbid_reuse_instance_id=<正释放实例>`**，避免迁回同一台。
4. **源任务解绑**：`bindSharedMachineToTask` 成功后清空源 CSC 的 `instance_id`/IP/runtime（保留 history 关闭语义与现有 clear 一致），从源头减少共享。
5. **释放顺序**：migrate 全部成功（或无可迁移）→ 再对终态任务走既有 stop（ECS `CLOUD_SERVER_STOPPED` / relay·mock local stop + clear-after-stop）。某兄弟 migrate 失败 → `DispatchRetryable`，不硬杀实例。

---

## 4. 领域概念（供 `/6-ddd`）

| 概念 | Bounded Context | 说明 |
|------|-----------------|------|
| **TerminalHardRelease** | Cloud Runtime / Domain Events | 终态后禁止 reuse 并释放节点 |
| **SharedInstanceBinding** | Cloud Runtime | 多 CSC 曾共享同一 `instance_id`（历史缺陷；本期修复 + 迁移补偿） |
| **ForeignContainerMigrate** | Cloud Runtime | 将非终态任务 busy 容器迁到其独占节点 |
| **OwnedMachineNode** | Cloud Runtime | 任务 CSC 绑定的机器；无则新建 |
| **TerminalKind** | Task | completed \| cancelled（复用 v17） |

**事件**

| 事件 | 生产者 | 消费者 |
|------|--------|--------|
| `TASK_STATUS_CHANGED`（已有） | taskTaskService | taskEvents `1_release…`（扩展） |
| `CONTAINER_MIGRATE_REQUESTED`（可选 MVP 内联 HTTP，不强制新 topic） | taskEvents | taskCloudService internal |
| `CLOUD_SERVER_STOPPED`（已有） | taskEvents / Cloud | 既有 stop handler |

MVP：**不强制**新 Kafka topic；taskEvents 内编排调 taskCloudService **internal** API 完成 list-siblings / migrate / 再发 stop。

---

## 5. 详细设计

### 5.1 taskCloudService

#### 5.1.1 修复闲置复用解绑

`bindSharedMachineToTask` 在 upsert target 成功后：

1. 清空 **source** 的 `instance_id`、`public_ip`、`server_url`、`business_api_endpoint`、`container_vscode_url`、`last_runtime_status`（或置 Released）
2. 关闭源任务 open history（复用既有 clear/stop history 辅助函数）
3. 日志：`event=idle_reuse_unbound_source source_task=… target_task=… instance_id=…`

#### 5.1.2 终态 / 硬释放候选排除

`findIdleMachineForReuse`：

- 跳过 `task_id` 对应任务已终态的行（经 internal 查 task 状态，或 CSC 新列 `release_locked=1` / `terminal_released_at`）
- **采纳**：CSC 增列 `terminal_released INTEGER NOT NULL DEFAULT 0`；终态 hard-release 置 1；reuse 查询 `AND terminal_released=0`
- 另：跳过 `forbid` 列表中的 `instance_id`（迁移时传入）

#### 5.1.3 Internal API（Go，服务凭据）

前缀建议：`/api/internal/cloud/`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `instance-bindings?company_id&workspace_id&instance_id` | 列出共享该实例的 CSC（task_id, server_url, …） |
| POST | `migrate-container-off-instance` | body: `{tenant,workspace,owner_task_id,from_instance_id,forbid_reuse_instance_id,trace_id}` → 为 owner 启新机（或合法 idle）、重建容器、更新 CSC、清空其在 from_instance 上的绑定 |
| POST | `mark-terminal-released` | body: `{tenant,workspace,task_id}` → `terminal_released=1` + 清运行字段（若尚未 clear） |

Swagger：`taskCloudService/openapi.yaml` 登记。

`migrate-container-off-instance` 步骤：

1. 读 owner CSC；记录 `container_image` / 模板线索（从 launch 历史或任务项目 `server_run_template`）
2. `start-vm(-auto)` 且 `forbid_reuse_instance_id=from_instance_id`；`prefer_idle_reuse` 仍可用**其他**闲置机
3. **真实 ECS**：migrate 走完整 `start-vm-auto`（镜像取自任务最近 start 事件的 `container_image_id`），由 ECS UserData **自动拉起容器**
4. **Mock / 闲置复用成功**：经 Container Gateway 调用 relay/mock `start`，按原镜像 **重建并拉起容器**
5. 更新 owner CSC 到新 instance；确保不再指向 `from_instance_id`
6. 结构化日志 `event=container_migrated` / `container_migrate_start_triggered`

### 5.2 taskEvents — 扩展 `task_status_changed/1_release_servers_on_terminal`

在既有 `Handler.Dispatch` 终态分支中，**替换「直接 stop」为**：

```text
1. Load CSC(终态 task)
2. 若无 instance_id 且无 server_url → mark-terminal-released；success
3. instance_id 非空 → GET instance-bindings
4. foreign = bindings where task_id≠self AND server_url≠""
5. for each foreign:
     POST migrate-container-off-instance(owner=foreign.task_id, from=instance_id)
     失败 → DispatchRetryable
6. 终态任务自身：若 server_url≠"" → 既有 StopLocal（停本任务容器）
7. 若仍存在其他非终态绑定指向同 instance 且 started（仅 idle 兄弟）→ 仅 clear 终态 CSC + mark-terminal-released，**暂不** CLOUD_SERVER_STOPPED（留给 idle recycle）；可选：idle 兄弟也解绑后统一 stop
8. 否则 → 既有 CLOUD_SERVER_STOPPED / clear-after-stop
9. mark-terminal-released(终态 task)
```

**自主简化（采纳）步骤 7**：终态 hard-release 时，对同实例上**仅 idle** 的其他绑定一并 clear（不 migrate），然后 **一律** `CLOUD_SERVER_STOPPED`（实例销毁）。这样「不允许再被重复使用」最强；idle 兄弟失去实例后下次 start 会新建/复用别的机。

### 5.3 前端

**不改**进度变更 UI；仍 PATCH todos。可选：SSE 已有 server_status_update 展示迁移/释放进度（若 Cloud 发 SSE）——MVP 不强制新文案。

### 5.4 权限

- 状态变更：既有租户/工作区任务写权限
- migrate / mark / bindings：仅服务间 internal；校验 tenant/workspace 边界，禁止跨租户

### 5.5 Python

无新增接口。

---

## 6. 非目标

- 容器进程内存 / 未提交磁盘的热迁移
- 修改进度列模型（不加 `is_terminal` DB 列；仍用列名约定）
- 关闭工作空间级 `prefer_idle_reuse`（仅排除终态已释放节点）
- 新建微服务

---

## 7. 🏛️ 架构变更影响

- **迭代版本**: v27 🎯 target
- **迭代名称**: terminal-hard-release-container-migrate
- **作者**: claude
- **设计日期**: 2026-07-15 10:10
- **新增文件**（每个视图三类伴生格式）:
  - 🆕 `docs/architecture/v27-application-integration-20260715-1010-claude.puml`
  - 🆕 `docs/architecture/v27-application-integration-20260715-1010-claude.archimate`（含 Plateau/Gap/WP + sourceConnection）
  - 🆕 `docs/architecture/v27-application-integration-20260715-1010-claude.mermaid.md`
- **变更明细**:
  - 🟢 [NEW] Constraint：终态机器硬释放、禁止 reuse
  - 🟢 [NEW] taskCloudService internal：instance-bindings / migrate-container / mark-terminal-released；CSC.`terminal_released`
  - 🟡 [MODIFIED] taskEvents `1_release_servers_on_terminal`：先 migrate 再 stop
  - 🟡 [MODIFIED] idle reuse：解绑 source；排除 `terminal_released`
  - 🎯 [NEW] Plateau v27
  - ✅ [CLOSES] Gap：终态误杀共享实例容器；终态节点仍可被 reuse

### .archimate 架构变迁要点

| 元素 | 内容 |
|------|------|
| Plateau v25 | 当前基线（auto SG） |
| Plateau v27 | 终态硬释放 + 容器迁移 |
| Gap | 终态 reuse 残留；共享实例误杀；无容器迁回所属任务 |
| WorkPackage | WP-v27-terminal-hard-release-container-migrate |
| 视图 | 架构变迁 v25→v27；Target 拓扑（Task→Events→Cloud→ECS/relay） |

---

## 8. 测试意图摘要

- 单元：reuse 后源 CSC 无 instance_id；`terminal_released=1` 不进 reuse；bindings 列表；migrate 调用顺序；migrate 失败不发 STOPPED；独占终态直接 stop；`CONTAINER_MIGRATE_AWAIT_READY` Decide（Ready / Wait / TriggerStart / Exhausted）。
- 集成：TASK_STATUS_CHANGED → migrate → STOPPED；幂等二次消费；migrate 后 await Running+heartbeat。
- 回归：非终态不释放；手动 stop-vm；idle recycle 仍可用非终态闲置机。

---

## 8.1 补偿 intent（2026-07-15 增补）

| 项 | 内容 |
|----|------|
| 事件 | `CONTAINER_MIGRATE_AWAIT_READY` / topic `container-migrate-await-ready` |
| Intent | `container_migrate_await_ready/1_await_running_heartbeat`（:18046） |
| 触发 | Cloud `migrateContainerOffInstance` 在 start_vm_auto / Gateway start 成功后发布 |
| 成功 | CSC `last_runtime_status=Running` 且 `server_url` 非空（且非 `pending-start-*`） |
| 重试 | 未就绪则 republish `attempt+1`；Running 无 URL 时再触发 Gateway start |
| 可配置 | env `CONTAINER_MIGRATE_AWAIT_MAX_ATTEMPTS`（默认 36）、`CONTAINER_MIGRATE_AWAIT_RETRY_DELAY`（默认 `5s`，亦支持整秒如 `5`）；runAll 已透出 |
| 失败 | 超限 → DispatchPermanent + SSE error |

---

## 9. 意图文档路径

- `task2app/docs/intents/frontend/task_detail/023_terminal_hard_release_container_migrate.intent.md`
- `task2app/docs/intents/frontend/task_detail/023_terminal_hard_release_container_migrate.test-intent.md`
