# runAll 链式启停（Cascade Lifecycle）设计

**日期:** 2026-05-27  
**状态:** 已批准（0-auto-flow 流水线）  
**范围:** `http://localhost:9999/`（runAll Web UI）及对应 API / Runner

## 1. 背景与问题

runAll 已在配置层用 `depends_on` 表达服务依赖，并在 **批量启动**（`BuildDAG` + `executeLevel`）与 **进程退出关闭**（逆 DAG）中按序执行。但 Web UI 上的 **单行「启动 / 关闭」** 仍是单点操作：

| 操作 | 当前行为 | 用户痛点 |
|------|----------|----------|
| 启动单服务 | 仅启动目标；不拉起上游 | 点 `taskFE` 前需手动启动 `git-oauth`、`saas-backend` |
| 关闭单服务 | 有活跃下游则 **拒绝** | 点 `git-oauth` 前需手动关闭 `taskFE`、`ai-provider` |
| 关闭本组 | 组内逆拓扑批量关闭 | 已有，体验良好 |
| 启动本组 | **不存在** | 无法一键拉起整组（如 `platform`） |

用户诉求：**链式反应**，使一次点击即可完成依赖链上的启停，与 CLI 批量语义一致。

### 1.1 典型依赖链（`runAll/config.yaml`）

```text
docker-infra（无 depends_on）

platform:
  git-oauth
    └── saas-backend
          ├── taskFE
          └── ai-provider

container-stack:
  go-run-container, go-relay（无 depends_on，与 platform 无交叉依赖）
```

## 2. 目标与非目标

### 2.1 目标

1. **启动链式（Start Cascade）**：对目标服务 S，自动按拓扑序启动所有 **已停止** 的传递上游依赖，再启动 S。
2. **关闭链式（Stop Cascade）**：对目标服务 S，自动按逆拓扑序先关闭所有 **仍在阻塞状态** 的传递下游，再关闭 S。
3. **启动本组（Start Group）**：与「关闭本组」对称，组内（及必要的外部上游）按 DAG 正序批量启动。
4. 保持现有 **所有权（session_id）**、**preflight**、**停止策略** 不变，链式操作在同一 `session_id` 下串行执行。
5. 失败时可诊断：明确失败步骤与已完成的步骤。

### 2.2 非目标（YAGNI）

- 不做跨机器 / 跨 runAll 实例编排。
- 不改变 `depends_on` 配置格式。
- 不在本阶段实现「编译链式」「重启链式」（重启仍保持 stop → start 目标服务；可选后续让 restart 走 start cascade）。
- 不实现并行级联（与批量启动的 level 并行不同，UI 链式采用 **串行、可预测**）。

## 3. 方案对比

### 方案 A：改造现有「启动 / 关闭」按钮为默认链式（推荐）

- UI 仍是一个按钮；请求体 `cascade: true`（默认 true）。
- `cascade: false` 保留旧语义（单点、关闭时仍阻断下游）。
- **优点：** 真正「一键」；与 README 中 DAG 叙述一致。
- **缺点：** 习惯旧行为的用户需知 `cascade=false`（可不暴露于 UI）。

### 方案 B：新增「连带启动 / 连带关闭」按钮

- 保留原按钮单点语义。
- **优点：** 零行为回归风险。
- **缺点：** 两对按钮增加认知负担，不符合「一键完成」表述。

### 方案 C：仅增加组级 / 全局按钮

- **优点：** 实现面小。
- **缺点：** 无法解决「只想起 taskFE」的单行场景。

**结论：采用方案 A**，并补充 **启动本组**（方案 C 的组级子集）。全局「启动全部 / 关闭全部」列为 **二期**，实现成本低但非 MVP 阻塞项。

## 4. 行为规格

### 4.1 启动链式 `StartServiceCascade(name)`

**计划生成：**

1. 在 **全配置服务图**（`cfg.Flatten()`）上，对 `name` 做传递闭包：所有 `depends_on` 上游（含跨 group）。
2. 拓扑排序得到 `startOrder`（依赖在前）。
3. 过滤：仅保留当前状态为 `stopped` 的服务（`healthy` / `starting` 等跳过，幂等）。
4. 追加目标 `name`（若未在列表中且为 stopped）。

**执行：**

- 按 `startOrder` **串行** 调用现有 `StartServiceWithActor`（含 preflight、健康检查、ownership）。
- 任一步失败：**立即返回**，附带 `failed_at` / `completed` 列表；不继续后续服务。

**边界：**

- 目标已是 `healthy`：返回成功（no-op）或明确提示「已在运行」（与现 `CanStart` 一致）。
- 上游属于其他 group：仍启动（例如只点 `taskFE` 也会拉起 `git-oauth`）。

### 4.2 关闭链式 `StopServiceCascade(name)`

**计划生成：**

1. 在全图上收集 `name` 的所有传递 **下游**（依赖图中 dependents 闭包）。
2. 逆拓扑排序得到 `stopOrder`（下游在前）。
3. 过滤：仅保留 `ServiceStopPolicyService` 认定的阻塞状态（与现 `isBlockingStatus` 一致）。

**执行：**

- 按 `stopOrder` 串行调用 `StopServiceWithActor`。
- 若 `stopOrder` 包含 `name`，最后关闭 `name`。
- 任一步失败：fail-fast，返回错误与已完成列表。

**与现逻辑关系：**

- 取代「有下游则直接 `CanStop` 报错」作为 **默认 UI 路径**；`cascade=false` 时保留原阻断语义。

### 4.3 启动本组 `StartGroup(group)`

- 收集该 `group` 内所有服务。
- 在 **全图** 上计算组内服务的启动顺序（正序拓扑），且对每条依赖若上游 **不在组内但为 stopped**，纳入计划（与 4.1 一致）。
- 仅启动当前为 `stopped` 的服务。
- 串行执行；失败 fail-fast。

### 4.4 关闭本组（已有）

- 保持 `StopGroup` / `stopOrderForGroup` 行为不变。
- 实现上可复用统一的 `stopOrderForServices([]Service)`  helper，减少与 cascade 重复。

### 4.5 二期（可选）：全局启停

- `StartAll` / `StopAll`：对 `cfg.Flatten()` 全量 DAG 正/逆序执行。
- UI：标题栏「启动全部」「关闭全部」。
- 本设计 **不阻塞 MVP**。

## 5. API 与 UI

### 5.1 API 变更

| 端点 | 变更 |
|------|------|
| `POST /api/start` | 请求体增加 `cascade`（bool，默认 `true`）。`true` → `StartServiceCascadeWithActor`；`false` → 现有 `StartServiceWithActor`。 |
| `POST /api/stop` | 同上 → `StopServiceCascadeWithActor` / `StopServiceWithActor`。 |
| `POST /api/start-group` | **新增**。body: `{ "group", "session_id", "cascade": true }`（组级可始终 cascade）。 |

响应（失败时建议扩展，向后兼容）：

```json
{
  "error": "stop cascade failed on \"saas-backend\": ...",
  "cascade": {
    "completed": ["taskFE", "ai-provider"],
    "failed_at": "saas-backend"
  }
}
```

成功仍返回 `{ "status": "ok" }`。

### 5.2 UI 变更（`status.html`）

1. 单行「启动 / 关闭」：默认 `cascade: true`（用户无感一键）。
2. 组头：在「关闭本组」旁增加 **「启动本组」**（`data-action="start-group"`）。
3. 链式执行中：对涉及服务行展示 `starting` / `stopping`（依赖现有 2s 轮询，无需新 WebSocket）。
4. 失败 `alert` 展示 `failed_at` 与已完成列表（解析 `cascade` 字段）。

## 6. 架构与模块

```text
status.html
  → POST /api/start|stop|start-group (cascade default true)
ui.go
  → Runner.Start/Stop/StartGroup Cascade variants
runner.go
  → planCascadeStart / planCascadeStop (uses cfg.Flatten + topo)
  → reuse StartServiceWithActor / StopServiceWithActor
domain/
  → ServiceLifecyclePlan (value object: ordered []string)
  → ServiceCascadeOrchestrationService (pure plan + filter rules)
  → extend service_operation_events (CascadeStartRequested, ...)
```

### 6.1 领域概念清单（供 `/5-ddd`）

| 类型 | 候选 |
|------|------|
| **Bounded Context** | RunAll 服务编排（Service Orchestration） |
| **Entity** | ManagedService（已有）、StartupSession（已有） |
| **Value Object** | ServiceLifecyclePlan、CascadeExecutionReport |
| **Domain Service** | ServiceCascadeOrchestrationService、ServiceStopPolicyService（已有） |
| **Domain Events** | ServiceCascadeStartRequested、ServiceCascadeStopRequested、ServiceCascadeStepFailed |

### 6.2 与所有权 / 稳定性设计对齐

- 链式每一步仍走 `EnsureOperableBySession`；非本 session 拥有的运行中服务：启动链遇到 foreign ownership 时 **fail-fast** 并提示（与 `2026-05-25-runall-stability-first-design` 一致）。
- 启动链每一步仍执行 `runPreflight`（含端口冲突恢复，若已实现）。

## 7. 价值流影响

参考 `value-stream.yaml`：

| 维度 | 影响 |
|------|------|
| **受影响流** | `runall-log-copy-gitoauth-port-conflict-recovery`（编排 UX 与 git-oauth 启动路径交互） |
| **新流（建议）** | `runall-cascade-lifecycle` @ 云平台与资源 — 描述 UI 一键链式启停 |
| **字段** | 无 DB 字段；可记录 `runall.ui.cascade_plan` / `runall.ui.cascade_result` 于 view_test 文档 |
| **测试** | 新增 `view_test/runall-cascade-lifecycle-thin-slice.md`；扩展 `runAll/src/runner_test.go`、`ui_test.go` |
| **状态** | 新步骤建议 `planned` → 实现后 `active` |

完整切片与 YAML 更新由 `/3-value-stream` 负责。

## 8. 错误处理与可观测性

| 场景 | 行为 |
|------|------|
| 链中某服务启动失败 | 停止后续；已启动上游保持运行；错误信息含服务名与 preflight/health 原因 |
| 链中某服务关闭失败 | 停止后续；已关闭下游保持 stopped |
| 循环依赖 | 配置校验阶段已拒绝；计划生成若检测到环，返回 500 类错误 |
| 用户双击 | 依赖 `CompareAndSwapStatus`；进行中的服务拒绝重复启动 |

日志：`[cascade] start plan: a -> b -> c`、`[cascade] stop failed at b: ...`

## 9. 测试策略

1. **单元：** `planCascadeStart` / `planCascadeStop` — 传递闭包、顺序、过滤 stopped-only。
2. **Runner 集成：** 三服务链 `a→b→c`：仅点 `c` 启动应依次拉起 `a,b,c`；仅点 `a` 关闭应先停 `c,b,a`。
3. **回归：** `cascade=false` 时关闭仍阻断下游；`StopGroup` 顺序不变。
4. **UI/API：** `ui_test.go` 覆盖 `cascade` 默认值、`/api/start-group`、错误体 `cascade` 字段。

## 10. 实施分期

| 阶段 | 交付 |
|------|------|
| **MVP** | Start/Stop cascade + 启动本组 + UI 默认 cascade + API 测试 |
| **二期** | 全局启动/关闭全部 + 标题栏按钮（可选） |

## 11. 验收标准

1. 在 `http://localhost:9999/` 仅点击 `taskFE`「启动」，无需手动启动，`git-oauth` → `saas-backend` → `taskFE` 依次 healthy（或明确失败于某步）。
2. 仅点击 `git-oauth`「关闭」，若 `taskFE` / `ai-provider` 在运行，自动先关下游再关 `git-oauth`。
3. `platform` 组「启动本组」按依赖顺序拉起组内 stopped 服务。
4. `cascade=false` 时行为与现网一致（关闭下游时仍报错）。

## 12. 未决项（实现前可默认）

| 项 | 默认决策 |
|----|----------|
| 用户跳过范围选择 | MVP = 方案 A + 启动本组；全局启停二期 |
| restart 是否链式启动依赖 | 否（本阶段）；restart 仍只重启单服务 |
| 链式执行是否并行 | 否，串行 |

---

**请审阅本设计。** 批准后写入实施计划（`/6-plans` 或 `writing-plans`），再进入 TDD 实现。
