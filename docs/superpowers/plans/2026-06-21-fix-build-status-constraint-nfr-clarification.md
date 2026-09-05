# NFR 澄清: 修复编译(build)受运行状态限制

> 输入:
> - 设计文档: `docs/design/fix-build-status-constraint.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-21-fix-build-status-constraint-value-stream.md`
>
> 输出使用者: `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 数据一致性 | L1 | CAS 原子操作防并发编译，行为不变 |
| 可维护性 | L1 | 测试覆盖所有 9 种状态的编译行为 |
| 性能 | L0 | 不适用 — 无新增 API/数据流 |
| 可伸缩性 | L0 | 不适用 — 无新增数据/连接 |
| 可用性 | L0 | 不适用 — 无变更可用性模型 |
| 安全性 | L0 | 不适用 — 无变更认证/授权 |
| 容错机制 | L0 | 不适用 — 编译失败处理逻辑不变 |
| 可观测性 | L0 | 不适用 — 已有日志覆盖 |
| 合规与隐私 | L0 | 不适用 |

## 逐增量 NFR 分析

### Increment 1: 移除编译状态限制 (Thin Slice)

#### NFR 类别: 数据一致性

- **等级**: L1 — 基础
- **量化目标**: CAS（Compare-And-Swap）原子操作正确性：同一服务同一时刻最多一个 build 执行。`status=building` 时第二个 build 请求被拒绝。
- **说明**: 此 fix 不改变并发编译防护（该机制已存在且正确）；仅放宽「允许编译的前置状态」集合。CAS 原子性由 `sync.Mutex` 保护，与变更前完全相同。

#### NFR 类别: 可维护性

- **等级**: L1 — 基础
- **量化目标**: 全部 9 种服务状态（pending/starting/retrying/healthy/failed/skipped/restarting/building/stopped）的编译行为有测试覆盖。
- **说明**: 测试需更新以反映新行为——8 种非 building 状态均允许编译，仅 building 拒绝。

## 质量场景

### QS-01: 非 building 状态均可编译

| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L1 |
| 刺激源 | 开发者点击 runAll UI 中的「编译」按钮 |
| 刺激 | 对处于 pending/starting/retrying/healthy/failed/skipped/restarting/stopped 状态的服务发起编译 |
| 制品 | `Runner.BuildService()` |
| 环境 | 正常 |
| 响应 | 状态转为 building → 执行构建命令 → 恢复为编译前状态 |
| 响应度量 | 单元测试覆盖全部 8 种状态，编译成功且不触发"can only build"错误 |

### QS-02: 并发编译被拒

| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L1 |
| 刺激源 | 开发者对同一服务连续快速点击两次「编译」 |
| 刺激 | 第二次编译请求在第一次编译进行中到达 |
| 制品 | `Runner.BuildService()` |
| 环境 | 正常 |
| 响应 | 第二次请求返回 "already building" 错误 |
| 响应度量 | `TestBuildService_ConcurrentBuildRejected` 通过 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| CAS 原子性保持 (L1) | 无模型变更 — `StatusBuilding` 仍为并发编译互斥信号 | 无需 DDD 变更 |
| 状态门放宽 (L1) | `IsTerminalBuildStatus` 语义从白名单变为排除 `building` | 已记录在 design 文档 |

> 此 fix 不引入新领域概念，无需 DDD 建模。建议跳过 Step 5 (DDD)，直接进入 Step 6 (实施计划)。

## 权衡与边界

### 取舍
- 选择放宽编译前状态门，接受「编译与运行并发」的可能（如服务 starting 时编译）。`go build` 写入是原子性 rename，不会导致运行中进程读取半成品二进制。

### 明确不做什么
- 不改变并发编译防护（已有 CAS 互斥）
- 不改变 `restartService` 的编译逻辑（已通过 `r.store.Update` + `r.runBuild` 直调，不经过 `BuildService`）
- 不改变前端 UI 行为

### 升级触发条件
- 无。此 fix 是简单的一次性修复，无后续升级路径。

## 跳过声明

| 类别 | 跳过理由 |
|------|---------|
| 性能 | 无新增 API 或数据流；编译执行时间不受此 fix 影响 |
| 可伸缩性 | 单用户操作，无横向扩展需求 |
| 可用性 | 编译是手动触发操作，无可用性 SLA |
| 安全性 | 无认证/授权变更 |
| 容错机制 | 编译失败处理逻辑不变（恢复为编译前状态） |
| 可观测性 | 已有 log.Printf 覆盖 build 生命周期 |
| 合规与隐私 | 内部开发工具，无合规要求 |
