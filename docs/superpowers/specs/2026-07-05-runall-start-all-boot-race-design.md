# runAll「全部启动」与 DAG 自动引导并发冲突 — 设计

**日期:** 2026-07-05  
**状态:** ✅ 已交付（2026-07-05）；Inc 2 回归已修复 — [2026-07-05-runall-start-all-button-perma-disabled-design.md](./2026-07-05-runall-start-all-button-perma-disabled-design.md)  
**范围:** [runAll Status UI](http://183.250.1.132:9999/)、`runAll/src/runner.go`、`runAll/src/status.html`  
**类型:** Bug 修复 + 幂等语义增强（**不涉及**新架构版本）

---

## 1. 问题现象

在 [runAll Status](http://183.250.1.132:9999/) 点击 **「全部启动」** 时，进度面板出现大量错误，典型两条：

```text
saas-backend: service "saas-backend" is healthy, can only start idle services
taskFE: [taskFE] service is retrying, cannot start
```

同类错误可覆盖 platform / taskEvents 消费者等数十个服务；页面可能同时显示 **已启动: 0** 或进度条与真实状态不一致。

---

## 2. 当前架构理解（基线 v5）

| 维度 | 现状 |
|------|------|
| 编排器 | `runAll` Go 单进程，读 `conf/runAll.yaml` DAG |
| 启动路径 A | **自动引导**：`main` → `runner.Run()` → `executeLevel` 按层并行 `startAndCheck` |
| 启动路径 B | **手动/API**：Web UI `POST /api/start-all` → `StartAllWithActor` → `PlanStartAll` → `executeParallelStartPlan` → `StartService` |
| UI 时机 | `startUIServer` **先于** `runner.Run()` 启动（`main.go:248-257`），引导进行中 UI 已可交互 |
| 状态机 | `pending → starting → retrying → healthy`；`IsStartableServiceStatus` 仅允许 `pending/stopped/failed/skipped` |

📋 **架构版本**：v5 ✅ shipped（relay lifecycle）；本问题属 **runAll 编排层**，不新增 v6 架构文件。

---

## 3. 根因分析

### 3.1 并发时间线（主因）

```text
T0  runAll 进程启动，所有服务 store 状态 = pending
T1  UI :9999 已监听 — 用户可点击「全部启动」
T2  runner.Run() 开始 executeLevel（自动 DAG 引导，与 UI 并发）
T3  用户点击「全部启动」
    └─ PlanStartAll() 快照：大量服务仍为 pending → 纳入计划
T4  自动引导先完成部分服务 → 状态变为 healthy / retrying
T5  StartAll 并行调用 StartService
    └─ healthy → CanStart() 拒绝 → "can only start idle services"
    └─ retrying → startAndCheck CAS 失败 → "service is retrying, cannot start"
```

**结论：** 不是「服务坏了」，而是 **两条启动路径对同一服务双重启动**，且计划基于 **过时的 pending 快照**。

### 3.2 代码锚点

| 位置 | 行为 |
|------|------|
| `main.go:248-257` | UI 先于 `Run()` 启动 |
| `runner.go:340-347` | `Run()` 阻塞主线程执行全量 DAG |
| `service_cascade_orchestration_service.go:184-195` | `filterStartable` 仅在 **计划生成时** 过滤一次 |
| `runner.go:1788-1804` | `startService` 对 `healthy` 硬拒绝，无幂等 skip |
| `runner.go:504-527` | `startAndCheck` 对 `retrying` 直接报错，不等待引导中的 health check |
| `runner.go:2883-2893` | 已有 `bootStillInProgress()`，但 **StartAll 未使用** |

### 3.3 与既有设计文档的偏差

`2026-05-27-runall-cascade-lifecycle-design.md` §4.1 规定链式启动应对 `healthy` **幂等跳过**；`StartService` / `StartAll` 当前实现 **未对齐**该语义。

### 3.4 taskFE 特例

`taskFE` 依赖链末端，自动引导 health check 期间常处于 `retrying`。手动 StartAll 若将其纳入计划，会与引导 goroutine 争抢 CAS，产生 `[taskFE] service is retrying, cannot start`。

---

## 4. 设计目标

1. **幂等**：服务已 healthy（探活通过）时，「全部启动」计为 **成功跳过**，不记失败。
2. **互斥**：自动引导未完成时，拒绝或排队 StartAll，避免双路径并发。
3. **可观测**：进度区分 `started` / `skipped (already running)` / `failed`。
4. **最小 diff**：复用现有 `bootStillInProgress`、`waitForServiceStart`、端口+health 跳过逻辑（`startAndCheck` L550-559 已有先例）。

### 非目标

- 不改变 `depends_on` 配置格式。
- 不重写全量并行启动调度器。
- 不做跨 runAll 实例编排。

---

## 5. 推荐方案（A + B 组合）

### 方案 A — `StartService` 执行期幂等（核心）

在 `startService` **CanStart 之前** 增加：

```go
// skipIfAlreadyHealthy: probe health_check；通过则 Update(healthy)、startMonitoring、return nil
```

逻辑与 `startAndCheck` 中「端口已监听且 health OK → skip start」保持一致。

对 `starting` / `retrying`（由 **其他 goroutine** 引导产生）：

```go
if status in (starting, retrying, restarting) {
    return waitForServiceStart(ctx, svc)  // 已有 30s 超时
}
```

**效果：** 即使计划 stale，也不会把 healthy/retrying 误报为失败。

### 方案 B — StartAll 与引导互斥（防护）

`StartAllWithActor` 入口：

```go
if r.bootStillInProgress() {
    return fmt.Errorf("runAll is still booting managed services; wait for DAG boot to finish or retry in ~30s")
}
```

可选增强：暴露 `GET /api/boot-status` → UI 禁用按钮并展示「自动引导进行中…」。

### 方案 C — 每层执行前再过滤（补强）

在 `executeParallelStartPlan` 每个 level 开始前：

```go
level = filterStillStartable(level)  // 复用 IsStartableServiceStatus + skipIfAlreadyHealthy 快速探针
```

减少无效 goroutine 与错误日志噪音。

### 方案对比

| 方案 | 优点 | 缺点 |
|------|------|------|
| 仅 A | 用户随时可点，最终一致 | 仍可能有短暂并发 CAS |
| 仅 B | 彻底避免并发 | 引导慢时 UX 阻塞 |
| **A+B**（推荐） | 正常路径互斥 + 边界幂等 | 略增代码 |
| 仅 C | 减少计划 stale | 不能单独解决 healthy 硬拒绝 |

---

## 6. API / UI 变更

### 6.1 进度事件扩展（向后兼容）

`StartAllProgressEvent` 增加可选字段：

```json
{
  "started": 12,
  "skipped": 25,
  "failed": 1,
  "phase": "done"
}
```

UI 文案：

- `skipped > 0` → 「已跳过 N 个已在运行」
- `boot blocked` → 「自动引导进行中，请稍后再试」

### 6.2 UI 行为

| 状态 | 「全部启动」按钮 |
|------|------------------|
| `bootStillInProgress()` | disabled + tooltip |
| 全部 healthy | enabled；点击后 instant done（0 待启动） |
| 部分 stopped/failed | enabled；只启动需启动的 |

---

## 7. 实现任务（批准后）

| # | 任务 | 文件 |
|---|------|------|
| 1 | `skipIfAlreadyHealthy` + starting/retrying wait | `runner.go` |
| 2 | `StartAllWithActor` boot 互斥 | `runner.go` |
| 3 | `executeParallelStartPlan` 层前再过滤 + skipped 计数 | `runner.go`, `progress.go` |
| 4 | UI 禁用 + skipped 展示 | `status.html` |
| 5 | 测试：boot 并发 StartAll、全 healthy 幂等、retrying wait | `runner_test.go` |

---

## 8. 验收标准

1. runAll 刚启动、DAG 引导进行中点击「全部启动」→ **明确提示等待引导**，或 **0 failed**（幂等 skip）。
2. 全部服务已 healthy 时再点「全部启动」→ **0 failed**，进度 instant done。
3. `taskFE` 在 retrying 时不会被记为 failed（等待或 skip）。
4. 停止全部 → 再「全部启动」→ 仅 stopped 服务被拉起，顺序正确。
5. `go test ./...` in `runAll/` 通过。

---

## 9. 价值流影响

| 流 | 影响 |
|----|------|
| `runall-lifecycle`（云平台与资源） | 「全部启动」步骤语义从 fail-on-healthy 改为幂等 |
| 测试 | 新增 `runAll/src/runner_test.go` 并发场景 |

---

## 10. 域概念（DDD 轻量清单）

| 类别 | 概念 |
|------|------|
| Bounded Context | runAll 服务编排 |
| Entity | `ManagedService` — 运行时状态 + 可否启动/停止 |
| 领域规则 | **启动幂等**：已 healthy 且探活通过 → no-op 成功 |
| 领域事件 | （无新增 Kafka 事件；lifecycle log 追加 skip 原因即可） |

---

## 11. 架构变更影响

**无 v6 架构变更** — 纯 runAll 编排语义修复，与 v5 application-integration 无关。

---

## 12. 待决问题（默认决策）

| 问题 | 默认 |
|------|------|
| 引导进行中是否完全拒绝 StartAll？ | **是**（方案 B）；若用户坚持可点，A 兜底 |
| healthy skip 是否写 lifecycle log？ | **是**，一行 `start skipped: already healthy` |
| StartGroup 是否同样幂等？ | **是**，共用 `startService` 即可 |

---

## 13. 用户操作指引（短期 workaround）

在修复部署前：

1. 打开 [runAll Status](http://183.250.1.132:9999/) 等待 **1–3 分钟**，直至各服务行变绿（healthy）。
2. **不要**在页面刚打开、服务仍灰/黄时点击「全部启动」。
3. 若需手动拉起：等服务列表加载完成后，仅对 **stopped/failed** 行点「启动」，或使用「启动本组」。
