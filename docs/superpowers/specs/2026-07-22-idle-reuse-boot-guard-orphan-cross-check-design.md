# 设计：闲置复用启动保护 + 孤儿删除交叉校验

- **日期**: 2026-07-22
- **状态**: approved（goal-mode 自动采用）
- **迭代名**: `idle-reuse-boot-guard-orphan-cross-check`
- **作者**: claude
- **选型**: 用户确认 **A + C**
- **现场案例**:
  - 源任务 `task_13823780785377101867`
  - Fork 任务 `task_13827276062684361869`
  - 实例 `i-j6ci4sjmjfq40pr6mxop`
- **相关设计**:
  - `2026-07-13-workspace-machine-idle-policy-design.md`（闲置复用）
  - `2026-07-15-terminal-hard-release-container-migrate-design.md`（复用后解绑源）
  - `2026-07-18-ecs-orphan-double-start-guard-design.md`（孤儿 InstanceName 对账）
- **python_api_approval**: n/a（零新增 Python HTTP；仅改 Go `taskCloudService` 内部判定）

---

## 1. 问题

Fork 本身不复制 CSC。但 Fork + `auto_run` → `start-vm-auto` 会走工作区 `prefer_idle_reuse`。

现场时间线（2026-07-22）：

| 时间 | 事件 |
|------|------|
| 18:03:49 | 源任务 ECS 启动成功 `i-j6ci4sjmjfq40pr6mxop` |
| 18:04:13–18:05:02 | 源任务仍在 `boot-progress`（`server_url` 空） |
| 18:05:12 | 新任务 `idle_reuse_bound` 抢绑同一实例并 `idle_reuse_unbound_source` |
| 18:05:20 | 源任务侧 `orphan_reconcile_delete` 把该实例当孤儿删除 |

根因：

1. **A 缺口**：闲置判定 = `started && server_url==""`。刚启动、容器未 register 也算「闲置」，可被其他任务复用。
2. **C 缺口**：孤儿对账只看「本 task CSC.keep vs InstanceName=本 task id（兼 legacy task-{id}）」，不解绑后实例已被其他 CSC 持有，仍 DeleteInstance。

---

## 2. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 机器处于启动保护期（未真正闲置）时不可被 idle reuse | Go 单测：Running + 空 server_url + `idle_since IS NULL` → 不复用 |
| S2 | 仅「曾有容器后清空 server_url」的真正闲置机可复用 | Go 单测：`idle_since` 已设且无 busy → 可复用 |
| S3 | 孤儿删除前若同 workspace 其他 CSC 持有该 `instance_id`，跳过删除 | Go 单测：解绑源后 fork 持有 → 不 delete |
| S4 | 不新增公网/Python HTTP；无新 Django 路由 | 代码审查 |
| S5 | 回归：既有真正闲置复用、终态释放、手动 stop-vm 行为不变 | 既有 `workspace_machine_*` / orphan 单测全绿 |

---

## 3. 方案（已确认 A+C）

### A. 启动保护期（收紧 idle 候选）

在 `findIdleMachineForReuseExcluding` 中，对按 `instance_id` 聚合的候选增加：

1. **真正闲置证据**：该 instance 上**所有**参与绑定的 CSC 均须 `idle_since IS NOT NULL`（`idle_since` 仅在 `server_url` 从非空→空时由 `maybeMarkIdleOnServerURLClear` 写入）。
2. **过渡态排除**：任一绑定 `last_runtime_status ∈ {Starting, Pending, Initializing}` → 不可复用（与「已启动」汇总一致）。
3. **保留既有**：`server_url` 非空、未 started、`terminal_released`、`forbid_instance_id` 仍排除。

效果：刚 RunInstances 成功、正在 boot-progress、尚未 register-reachability 的机器**不会**进入复用池；Fork+auto_run 将走新建 ECS，而不是抢源任务机器。

> 不采用 B（fork 血缘硬隔离）与 D（Fork 禁用 reuse）：A 已覆盖本现场，且不削弱「真正闲置机跨任务复用」的成本收益。

### C. 孤儿删除交叉校验

在 `reconcileOrphanInstancesByName` 删除前：

1. 预加载本 `company_id+workspace_id` 下所有非空 `cloud_server_configs.instance_id` 集合 `owned`。
2. 对每个候选 `cloudID`：若 `cloudID ∈ owned`（任意任务 CSC 持有，含他任务），**跳过删除**，打日志：
   `event=orphan_reconcile_skip_owned task_id=… orphan_instance_id=… owner_hint=cross_csc`
3. 仅当云上 `InstanceName=task_id`（或 legacy `task-{task_id}`）的实例**不被任何 CSC 持有**且不等于本 task `keep` 时，才 DeleteInstance。

效果：idle reuse 解绑源后，即使 A 偶发漏网，孤儿对账也不会误杀已绑到目标任务的实例。

---

## 4. 非目标

- 不改 Fork API / `fork_from` 语义
- 不改 `prefer_idle_reuse` 默认值或工作区策略 UI
- 不改 InstanceName 命名规则
- 不做跨 workspace 复用

---

## 5. 领域概念（轻量，供 /6-ddd）

| 概念 | 说明 |
|------|------|
| **BootProtectedMachine** | 已有 instance 且尚未形成真正闲置（无 `idle_since` 或过渡态） |
| **TrueIdleMachine** | started、全绑定无 `server_url`、均有 `idle_since` |
| **CrossCSCOwnership** | 同 workspace 内其他任务 CSC 持有某 `instance_id` |

### 业务意图 → 事件对照

**无对应新事件**：本变更是 `taskCloudService` 内部判定收紧与删除门闩，不新增领域事件类型；仍挂在既有 `start-vm(-auto)` / runtime reconcile 路径。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 闲置复用启动保护 | — | — | — | 无对应事件：纯内部候选过滤，不改系统对外事实契约 |
| 孤儿删除交叉校验 | — | — | — | 无对应事件：删除门闩；删机仍走既有 orphan/stop 路径 |

---

## 6. 实现落点（Go only）

| 文件 | 变更 |
|------|------|
| `taskCloudService/src/workspace_machine_idle_reuse.go` | SELECT 增加 `idle_since`；候选聚合应用 A |
| `taskCloudService/src/orphan_instance_reconcile.go` | 删除前 workspace `owned` 集合校验 C |
| `*_test.go` | S1–S3 单测；回归既有 reuse/orphan |

日志（既有风格）：

- `event=idle_reuse_skip_boot_guard instance_id=… reason=no_idle_since|starting`
- `event=orphan_reconcile_skip_owned …`

---

## 7. 价值流影响

影响既有流（`conf/value-stream.yaml`）：

- `cloud-integration` / `workspace-machine-idle-policy`：复用候选收紧（行为增强，字段不变）
- 孤儿对账步骤（若已映射）：增加「跨 CSC 持有则跳过」

预计新增/更新测试文件：

- `taskCloudService/src/workspace_machine_idle_reuse_test.go`（或扩展现有）
- `taskCloudService/src/orphan_instance_reconcile_test.go`

完整切片由 `/4-value-stream` 负责。

---

## 8. 🏛️ 架构变更影响

- **迭代版本**: v50 🎯 target
- **视图**: `application-integration`（enterprise-landscape 无新组件，未改）
- **新增文件**:
  - 🆕 `docs/architecture/v50-application-integration-20260722-1815-claude.puml`
  - 🆕 `docs/architecture/v50-application-integration-20260722-1815-claude.archimate`（含 Plateau/Gap/WP 架构变迁视图）
  - 🆕 `docs/architecture/v50-application-integration-20260722-1815-claude.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] `taskCloudService` — idle 候选需 `idle_since`；orphan 删除前 cross-CSC 校验
  - 🎯 Plateau v50；Gap：boot 误闲置 + orphan 误删已复用实例

### .archimate 架构变迁要点

| 元素类型 | 内容 |
|----------|------|
| **Plateau v48** | Current — 超管充值消费总览基线 |
| **Plateau v50** | Target — idle boot-guard + orphan cross-check |
| **Gap** | boot 误闲置 + orphan 误删已复用实例 |
| **WorkPackage** | WP-v50-idle-reuse-boot-guard-orphan-cross-check |
| **视图** | `架构变迁 v48→v50 — idle-reuse-boot-guard`（含 sourceConnection） |

---

## 9. 验收计划（摘要）

1. 单测复现场景：源 Running + 空 URL + 无 idle_since → 新任务 start-vm-auto **不 reuse**。
2. 单测：源曾 busy 后 idle_since 设置 → 可 reuse；reuse 后源 orphan reconcile **不删**目标持有实例。
3. `go test ./…`（taskCloudService）。
4. 可选手工：Fork 源任务 boot 中点 auto_run → 新任务应新建机，源机不被抢。

---

## 10. 审批记录

- **方向**: A+C（用户 2026-07-22 确认）
- **总体设计**: approved（2026-07-22 goal-mode 覆盖 USER GATE）
