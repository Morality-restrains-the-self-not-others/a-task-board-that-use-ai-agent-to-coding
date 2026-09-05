# runAll「全部启动」按钮永久不可用 — 设计（Inc 2）

**日期:** 2026-07-05  
**状态:** ✅ 已交付（2026-07-05）  
**范围:** [runAll Status UI](http://183.250.1.132:9999/)、`runAll/src/runner.go`、`runAll/src/status.html`  
**前置:** [2026-07-05-runall-start-all-boot-race-design.md](./2026-07-05-runall-start-all-boot-race-design.md)（Inc 1 已交付）  
**类型:** Bug 修复（Inc 1 回归），**不涉及**新架构版本

---

## 1. 问题现象

用户反馈：**当某个子服务启动失败后**，[runAll Status](http://183.250.1.132:9999/) 的 **「全部启动」** 按钮**永远无法再使用**（灰色 disabled，或点击后 API 拒绝）。

典型场景：

1. runAll 进程启动，DAG 自动引导部分服务 `failed` / `skipped`
2. 用户想点「全部启动」重试失败服务
3. 按钮一直 disabled，或提示「runAll 正在自动引导服务…」

这与 Inc 1 的设计目标（引导结束后应能幂等重试）**直接冲突**。

---

## 2. 架构上下文（v5 基线，无变更）

| 组件 | 角色 |
|------|------|
| `runner.Run()` | 前台模式下：DAG 引导 → 健康监控 → **阻塞等待 SIGINT** |
| `StartAllWithActor` | 手动全量启动；`PlanStartAll` 会纳入 `failed/stopped/skipped/pending` |
| `dag_boot_in_progress` | Inc 1 新增；UI 据此 disable「全部启动」 |

📋 **架构版本**：v5 ✅ shipped；本迭代仅修正 runAll 编排语义，**不新增 v6 架构文件**。

---

## 3. 根因分析

### 3.1 主因：Inc 1 回归 — `dagBootInProgress` 生命周期过宽

Inc 1 在 `Run()` **入口**置位、**函数返回**才清位：

```go
func (r *Runner) Run(ctx context.Context, daemon bool) error {
    atomic.StoreInt32(&r.dagBootInProgress, 1)
    defer atomic.StoreInt32(&r.dagBootInProgress, 0)  // ← 仅在 Run 完全退出时执行

    for _, level := range r.levels { ... }  // DAG 引导

    if !daemon {
        <-ctx.Done()  // ← 前台模式：在此阻塞直到 runAll 关闭
    }
    return nil
}
```

前台模式（带 Web UI 的默认路径）下，`Run()` 在 DAG 引导结束后**不会返回**，而是长期阻塞在 `<-ctx.Done()`。因此：

- `dagBootInProgress` **在整个 runAll 进程存活期间恒为 `true`**
- UI 每 2s 轮询 `/api/status` → `dag_boot_in_progress: true` → **「全部启动」永久 disabled**
- `StartAllWithActor` 调用 `rejectManualStartDuringDAGBoot()` → **API 永久拒绝**

**与子服务是否失败无必然因果关系**：只要 runAll 完成引导进入「Running. Press Ctrl+C」阶段，按钮就已不可用。用户在有失败服务时更常尝试点「全部启动」，因而**误归因**为「失败导致按钮锁死」。

### 3.2 次因：用户期望 vs 当前语义

| 用户期望 | Inc 1 实际行为 |
|----------|----------------|
| 引导进行中：禁止重复全量启动 | ✅ 意图正确 |
| 引导结束（含部分 failed）：可重试失败项 | ❌ 被永久互斥锁死 |
| 全部 healthy：幂等 skip | ✅ `startService` 已支持（Inc 1 方案 A） |

### 3.3 排除项

| 假设 | 结论 |
|------|------|
| `failed` 状态不可纳入 StartAll 计划 | ❌ `filterStartable` 包含 `failed` |
| 服务卡在 `retrying` 导致 `bootStillInProgress()` | ❌ UI/API **未**使用 `bootStillInProgress`，仅用 `dagBootInProgress` |
| `on_failure: exit` 导致进程退出 | 进程退出后 UI 也不存在；与本问题（UI 仍在线但按钮不可用）不符 |

---

## 4. 设计目标

1. **`dagBootInProgress` 仅覆盖 DAG 引导阶段**（`executeLevel` 循环），引导结束立即清位。
2. 引导结束后（无论 healthy / failed / skipped 混合），**「全部启动」必须可用**，用于重试失败服务。
3. 引导进行中仍禁止 StartAll/StartGroup（Inc 1 方案 B 意图保留）。
4. 最小 diff；不改动 `depends_on` 或状态机定义。

---

## 5. 推荐方案

### 方案 A — 收窄 boot 标志作用域（推荐，必做）

**改动：** 将 `dagBootInProgress` 的置位/清位限制在 DAG 引导循环内，**不要**用 `defer` 绑定整个 `Run()`。

```go
func (r *Runner) Run(ctx context.Context, daemon bool) error {
    resilient := !daemon
    r.logDAGTopology()

    r.setDAGBootInProgress(true)
    for _, level := range r.levels {
        if err := r.executeLevel(ctx, level, resilient); err != nil {
            r.setDAGBootInProgress(false)
            return err
        }
    }
    r.setDAGBootInProgress(false)

    log.Println("[runAll] All services healthy.") // 或 partial — 见方案 B 文案
    // ... monitoring + <-ctx.Done()
}
```

封装 `setDAGBootInProgress(bool)` 便于测试与日志。

**效果：**

- 引导中：`dag_boot_in_progress=true`，按钮 disabled
- 引导后（含 failed）：`false`，按钮 enabled，`StartAll` 可重试 `failed` 服务

### 方案 B — 引导结束日志与 UI 提示（推荐，轻量）

引导结束后若存在 `failed/skipped`，日志改为：

```text
[runAll] DAG boot finished (N healthy, M failed, K skipped). Manual start-all available for retries.
```

UI 无需额外字段；可选在 status 页展示 `boot_summary`（非 MVP）。

### 方案 C — 不用 boot 标志、仅用 `bootStillInProgress()`（不推荐）

`bootStillInProgress()` 检测 `starting/retrying/restarting`。健康检查长尾时窗口更长，且**无法**区分「引导 vs 用户手动单服务启动」。Inc 1 已选显式 boot 阶段标志，应修正其范围而非替换。

---

## 6. API / UI 变更

| 项目 | 变更 |
|------|------|
| `GET /api/status` → `dag_boot_in_progress` | 语义收窄：仅 DAG 引导中为 `true` |
| 「全部启动」disabled | 仅引导中；**引导后即使存在 failed 也可点** |
| `POST /api/start-all` | 引导结束后正常接受；计划含 `failed` 服务 |

**向后兼容：** JSON 字段名不变，仅布尔值语义修正。

---

## 7. 实现任务（批准后）

| # | 任务 | 文件 |
|---|------|------|
| 1 | 收窄 `dagBootInProgress` 至 `executeLevel` 循环 | `runner.go` |
| 2 | 引导 abort（`on_failure: exit`）路径清位 | `runner.go` |
| 3 | 测试：模拟引导结束后 `IsDAGBootInProgress()==false` 且 `StartAll` 可接受 | `runner_test.go` |
| 4 | 测试：引导进行中仍拒绝 `StartAll` | `runner_test.go`（已有，需配合新置位范围） |
| 5 | 更新 Inc 1 设计文档 §12 待决 / 验收 | `2026-07-05-runall-start-all-boot-race-design.md` |

---

## 8. 验收标准

1. runAll 启动并完成 DAG 引导（**含部分服务 failed**）后，`/api/status` 返回 `dag_boot_in_progress: false`。
2. 「全部启动」按钮可点击；`StartAll` 仅启动 `failed/stopped/skipped/pending` 服务，healthy 计为 skipped。
3. runAll **刚启动、引导尚未结束**时，按钮仍 disabled，API 仍拒绝。
4. `go test ./...` in `runAll/` 通过。

---

## 9. 价值流影响

| 流 | 影响 |
|----|------|
| runAll 生命周期 | 「全部启动」在引导后恢复可用，支持失败重试 |
| 测试 | 补充 boot 标志生命周期用例 |

---

## 10. 架构变更影响

**无** — 纯 runAll 编排 bugfix，不更新 `docs/architecture/`。

---

## 11. 与 Inc 1 的关系

| Inc | 内容 | 状态 |
|-----|------|------|
| Inc 1 | 引导互斥 + 启动幂等 + skipped 计数 | ✅ 已交付（含本回归） |
| **Inc 2** | **修正 `dagBootInProgress` 生命周期** | 本文档 |

Inc 1 的 **方案 A（幂等 skip）** 与 **方案 B（引导互斥）** 设计正确；实现上方案 B 的标志位**绑错了生命周期**，Inc 2 仅修正这一点。

---

## 12. 短期 workaround（修复部署前）

1. 刷新页面无法解决（标志在服务端进程内）。
2. **重启 runAll** 后，在 DAG 引导完成前快速点「全部启动」仍会被 Inc 1 互斥拦截；应**等 1–2 分钟引导结束**再点——但当前版本引导结束后按钮仍不可用，**workaround 无效**。
3. 修复部署前：对单个 failed 服务点行内「启动」，或 `curl -X POST /api/start`（若未同样被 boot 标志拦截 — **会被拦截**，与 StartAll 相同根因）。

**结论：Inc 2 为阻塞性回归，应尽快合入。**
