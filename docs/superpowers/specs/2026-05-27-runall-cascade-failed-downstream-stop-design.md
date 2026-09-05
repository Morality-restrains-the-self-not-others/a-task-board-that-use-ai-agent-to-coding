# runAll 链式关闭：failed 下游遗漏修复设计

**日期:** 2026-05-27  
**状态:** 已实现  
**范围:** `http://localhost:9999/` runAll Web UI — 对 `git-oauth` 点击「关闭」无法停进程

---

## 1. 问题陈述

在 runAll 控制台对 **git-oauth** 点击「关闭」（默认 `cascade: true`）后：

- API 立即返回 `{"status":"accepted"}`，UI 刷新
- **git-oauth 仍为 `healthy`，进程未退出**
- 无明确错误弹窗（异步失败仅写 server log）

典型现场状态：

| 服务 | status | pid | session_id |
|------|--------|-----|------------|
| taskFE | stopped | 0 | runall-bootstrap |
| saas-backend | **failed** | **>0** | runall-bootstrap |
| ai-provider | healthy | >0 | runall-bootstrap |
| git-oauth | healthy | >0 | UI session |

---

## 2. 根因分析

### 2.1 链式计划漏掉 failed 下游

`PlanStopCascade` → `filterStoppable` 仅包含 `isBlockingStatus`（healthy/starting/retrying 等），**不含 `failed`**。

对 git-oauth 关闭，计划变为：

```text
taskFE (已 stopped，跳过) → ai-provider → git-oauth
```

**saas-backend（failed + 仍有 PID）被排除**，不会进入级联步骤。

### 2.2 上游停止被 runningDependents 阻断

`stopService` 合并：

- `EvaluateStop`：failed 下游 **不算** blocking
- `runningDependents`：只要 **PID > 0** 就算 active dependent

因此执行到最后一步停 git-oauth 时：

```text
service "git-oauth" has active downstream dependencies: saas-backend
```

级联在 ai-provider 之后、git-oauth 之前 silent fail（async goroutine 仅 log）。

### 2.3 与 mixed-ownership 修复的关系

`2026-05-27-runall-cascade-mixed-ownership-design.md` 已解决 **actor 委托**（bootstrap owner 停 taskFE 等）。

本问题是 **orthogonal**：即使 actor 委托正确，只要 saas-backend 处于 failed 且带 PID，仍会关不掉 git-oauth。

---

## 3. 目标与非目标

### 目标

1. 级联关闭 git-oauth 时，**failed 但仍有进程的下游**（如 saas-backend）纳入停止计划。
2. 计划顺序：先停 healthy 下游 → 再停 failed 下游 → 最后停目标服务。
3. 补充 Go 单元/集成测试 + Playwright 回归（混合所有权 + platform 链）。
4. 不改变 `cascade=false` 单点停止语义（failed 下游有 PID 仍阻断上游，见现有 `TestRunner_StopService_BlocksFailedDependentWithRunningPID`）。

### 非目标

1. 不修复 saas-backend 本身 health failed 的后端根因。
2. 不改为同步 API（仍 `accepted` + async）。
3. 不调整 `runningDependents` 的 PID 语义（单点 stop 安全边界保留）。

---

## 4. 方案（推荐）

### 4.1 扩展级联停止候选状态

在 `domain/service_stop_policy_service.go` 新增：

```go
func isCascadeStopCandidateStatus(status string) bool
```

规则：

| status | 纳入级联停止计划 |
|--------|------------------|
| stopped / skipped / pending | 否 |
| failed | **是** |
| healthy / starting / retrying / … | 是（沿用 isBlockingStatus） |

`filterStoppable` 改用 `isCascadeStopCandidateStatus`。

### 4.2 修复后计划示例

现场状态下关闭 git-oauth：

```text
ai-provider → saas-backend → git-oauth
```

每步 actor 仍走 `resolveCascadeStepActor`（mixed-ownership 委托）。

### 4.3 测试策略

| 层级 | 内容 |
|------|------|
| Domain | `PlanStopCascade_IncludesFailedDownstream` |
| Runner | `StopServiceCascade_StopsFailedDownstreamBeforeUpstream` |
| Playwright | `runall-stop-git-oauth.playwright.test.js`（fixture 混合所有权 platform 链） |

---

## 5. Domain Concept Inventory

| 类型 | 名称 | 说明 |
|------|------|------|
| Bounded Context | RunAll 服务编排 | 级联启停 |
| Value Object | CascadeStopCandidateStatus | 是否应出现在 stop 计划中 |
| Domain Service | ServiceCascadeOrchestrationService | filterStoppable 规则 |
| 已有 | CascadeStepActorResolver | mixed-ownership 委托 |

---

## 6. Value Stream Impact

影响 `value-stream.yaml` → **`runall-cascade-lifecycle`**：

| 步骤 | 变更 |
|------|------|
| `runall-stop-cascade` | 补充 failed 下游 + PID 场景验收 |
| `runall-cascade-mixed-ownership` | Playwright 回归覆盖 git-oauth 关闭 |

---

## 7. 验收标准

1. bootstrap 拉起 platform 链，saas-backend 处于 failed 且 pid>0 时，UI 对 git-oauth 点「关闭」→ git-oauth / saas-backend / ai-provider 均 `stopped`。
2. `go test ./runAll/src/...` 通过。
3. `npx playwright test runall-stop-git-oauth.playwright.test.js` 通过。

---

## 8. 风险

| 风险 | 缓解 |
|------|------|
| failed 服务本无进程，多停一步 | `CanStop` 已允许 failed；无进程时快速置 stopped |
| 与单点 stop 语义不一致 | 仅扩展 **cascade 计划**；单点仍 blocked |

---

**请审阅本设计。** 批准后进入 0-auto-flow 后续流水线。
