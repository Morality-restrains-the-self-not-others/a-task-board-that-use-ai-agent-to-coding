# runAll 链式关闭混合所有权修复设计

**日期:** 2026-05-27  
**状态:** 已实现  
**范围:** `http://localhost:9999/` runAll Web UI — 链式关闭 / 关闭本组

## 1. 问题陈述

在 runAll 控制台对 `git-oauth` 点击「关闭」（默认 `cascade: true`），弹出：

```text
Stop failed: service "taskFE" is owned by session "runall-bootstrap",
actor session "bc5e71cb-be59-44b5-9af5-ec3a44acb575" requires explicit takeover
```

**复现路径：**

1. runAll 启动时 bootstrap 拉起 `taskFE`、`saas-backend` 等 → 所有权为 `runall-bootstrap`
2. 用户在 UI 对 `git-oauth` 点击「启动」→ 所有权变为浏览器 `session_id`（如 `bc5e71cb-...`）
3. 用户对 `git-oauth` 点击「关闭」→ 链式计划先停 `taskFE`，但用 UI session 操作 bootstrap 拥有的服务 → **失败**

`.runall/ownership.json` 典型状态：

| 服务 | OwnerSessionID |
|------|----------------|
| taskFE | runall-bootstrap |
| saas-backend | runall-bootstrap |
| git-oauth | bc5e71cb-...（UI session） |

## 2. 根因分析

链式执行在 `runner.go` 的 `executeLifecyclePlan` 中，对计划内 **每一步使用同一个 `actorSessionID`**（来自 UI 按钮上的 `data-session-id` / `selectActorSessionID`）。

```go
// 当前：全程使用 initiating actor
stepFn(ctx, serviceName, actor)
```

`StopServiceWithActor` 调用 `EnsureOperableBySession`，非 owner 直接拒绝。  
这与 `2026-05-27-runall-cascade-lifecycle-design.md` §6.2「foreign ownership fail-fast」一致，但 **未覆盖 bootstrap + UI 混用这一常见本地开发场景**。

级联关闭的语义是「关闭以 S 为根的依赖子树」，不应要求用户对每个下游服务分别 takeover。

## 3. 目标与非目标

### 目标

1. 混合所有权下，对 `git-oauth` 一次「关闭」能依次停 `taskFE` / `ai-provider` → `saas-backend` → `git-oauth`。
2. 不引入静默 takeover；不放宽单点 `cascade=false` 的所有权校验。
3. 启动链式 / 启动本组 / 关闭本组在相同混用场景下也不因 foreign ownership 失败。
4. 补充单元测试与 value stream 步骤。

### 非目标

1. 不改变 `runAll takeover` CLI 显式接管语义。
2. 不修改 UI 的 `session_id` 生成逻辑。
3. 不做跨 runAll 实例编排。

## 4. 设计方案

### 4.1 级联步骤 Actor 解析（推荐）

在 `Runner` 增加 **纯解析** 方法（不修改 ownership 记录）：

```go
// resolveCascadeStepActor 为级联计划中的单步选择有效 actor。
// 若 initiating actor 已是 owner → 用它；
// 否则若服务有 registered owner → 用该 owner（委托停止/启动，非 takeover）；
// 否则 → initiating actor。
func (r *Runner) resolveCascadeStepActor(serviceName, initiatingActor string) string
```

`executeLifecyclePlan` 改为：

```go
stepActor := r.resolveCascadeStepActor(serviceName, actor)
stepFn(ctx, serviceName, stepActor)
```

**停止链示例**（用户 session = `U`，点关 `git-oauth`）：

| 步骤 | 服务 | 解析后 actor | 说明 |
|------|------|--------------|------|
| 1 | taskFE | runall-bootstrap | 委托 bootstrap owner 停止 |
| 2 | ai-provider | runall-bootstrap | 同上 |
| 3 | saas-backend | runall-bootstrap | 同上 |
| 4 | git-oauth | U | initiating actor 即 owner |

**启动链示例**（用户 session = `U`，点启 `taskFE`，上游 stopped 且无 ownership）：

| 步骤 | 服务 | 解析后 actor |
|------|------|--------------|
| git-oauth | U | 无 ownership → U |
| saas-backend | U | 无 ownership → U |
| taskFE | U | U |

**启动链**（上游 stopped 但残留 bootstrap ownership 记录）：

| 步骤 | 服务 | 解析后 actor |
|------|------|--------------|
| git-oauth | runall-bootstrap | 用原 owner 启动，ownership 保持 bootstrap |

### 4.2 安全边界

| 操作 | 行为 |
|------|------|
| `cascade=true` 链式 / 组级 | 使用 `resolveCascadeStepActor` |
| `cascade=false` 单点 stop/start/restart | **不变**，仍 strict ownership |
| 显式 `takeover` CLI | **不变** |

级联委托仅作用于 **计划内服务**；不对外部服务开放。

### 4.3 为何不是 auto-takeover

`TakeoverService` 会 **改写** ownership 为 initiating actor，属于「显式接管」语义。  
级联关闭只需 **以各服务当前合法 owner 执行 stop**，无需转移所有权，更符合 stability-first 设计。

### 4.4 领域层（可选薄封装）

在 `domain` 增加 `CascadeStepActorResolver`（输入：initiatingActor、serviceOwnership、hasOwnership）便于单测；`Runner` 负责读 repository。

## 5. Domain Concept Inventory

| 类型 | 名称 | 说明 |
|------|------|------|
| Bounded Context | RunAll 服务编排 | 链式启停 |
| Value Object | CascadeStepActor | 单步有效 session |
| Domain Service | CascadeStepActorResolver | 解析规则 |
| Entity | ServiceOwnership | 已有 |
| Domain Event | ServiceCascadeStepDelegated | 可选日志：step 使用了 delegated owner |

## 6. Value Stream Impact

影响 `value-stream.yaml` 中 **`runall-cascade-lifecycle`**：

| 步骤 | 变更 |
|------|------|
| `runall-stop-cascade` | 补充 mixed-ownership 验收场景 |
| `runall-start-cascade-thin-slice` | 补充 upstream 残留 bootstrap ownership 场景 |
| `runall-start-group` / 关闭本组 | 同逻辑，测试矩阵扩展 |

建议新增 thin-slice 测试文档：`view_test/runall-cascade-mixed-ownership.md`

**字段：** 无 DB 变更；可记录 `runall.ui.cascade_step_actor` 于测试文档。

## 7. 测试策略

1. **单元：** `resolveCascadeStepActor` — owner 匹配 / 委托 / 无 ownership 三路径。
2. **Runner 集成：** 模拟 `a→b→c`，`b` owned by `bootstrap`，`c` owned by `ui-session`，`StopServiceCascadeWithActor(c, ui-session)` 成功且顺序 `a,b,c` stopped。
3. **回归：** `cascade=false` + foreign owner 仍返回 takeover 错误（现有测试保持）。
4. **UI/API：** `ui_test.go` 可选端到端 mock ownership store。

## 8. 实施切片

| Slice | 内容 |
|-------|------|
| 1 | `resolveCascadeStepActor` + `executeLifecyclePlan` 接线 |
| 2 | Runner 混用 ownership 集成测试 |
| 3 | value stream / view_test 文档更新 |

## 9. 验收标准

1. bootstrap 拉起 platform 链后，UI 单独启动 `git-oauth`，再点 `git-oauth`「关闭」→ 无 takeover 错误，下游与上游均 `stopped`。
2. `cascade=false` 关闭仍有下游时仍报错（旧语义）。
3. `go test ./runAll/src/...` 通过。

## 10. 风险

| 风险 | 缓解 |
|------|------|
| 级联委托被误解为安全放宽 | 仅 cascade 计划内生效；单点仍 strict |
| 启动链用 bootstrap actor 导致 UI 以为「自己拥有」上游 | 状态 API 已暴露 `session_id`；文档说明 |

---

**请审阅本设计。** 批准后进入 0-auto-flow 后续流水线（价值流 → NFR → DDD → 计划 → 实现 → 审查 → PR）。
