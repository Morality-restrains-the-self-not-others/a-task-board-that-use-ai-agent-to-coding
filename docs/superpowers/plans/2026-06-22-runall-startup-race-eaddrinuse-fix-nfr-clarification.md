# NFR 澄清: runAll 启动竞态 EADDRINUSE 修复

> 输入:
> - 设计文档: `.claude/plans/01-brainstorming-设计文档.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-runall-startup-race-eaddrinuse-fix-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可用性 | L2 | runAll 启动时服务端口冲突率降为零 |
| 容错机制 | L2 | 并发启动自动互斥，lsof 错误 fail-fast |
| 性能 | L0 | 不适用 — bug 修复不改变性能特征 |
| 可伸缩性 | L0 | 不适用 — 单机开发环境 |
| 安全性 | L0 | 不适用 — 无新增安全相关变更 |
| 数据一致性 | L0 | 不适用 — 无新增数据流 |
| 可观测性 | L0 | 不适用 — 现有日志已覆盖 |
| 合规与隐私 | L0 | 不适用 |
| 可维护性 | L0 | 不适用 — 无新增 API 或配置契约 |

## 逐增量 NFR 分析

### Increment 1: startAndCheck CAS 并发守卫

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: DAG 引导与 API start-all 并发执行时，同一服务 0 次重复启动
- **质量场景**: QS-01

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: 不再出现因 runAll 自身竞态导致的 `Address already in use` 错误
- **质量场景**: QS-02

### Increment 2: killPreviousRunAllProcess 等待托管服务释放

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: 新旧 runAll 替换后，0 个孤儿进程占用托管服务端口
- **质量场景**: QS-03

### Increment 3: gitOauth SO_REUSEADDR 修复

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: gitOauth 重启时不再因 TIME_WAIT 导致端口绑定失败
- **质量场景**: QS-04

### Increment 4: listenerPIDs 错误不静默

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: lsof 检测失败时 100% 中止启动并输出明确错误
- **质量场景**: QS-05

## 质量场景

### QS-01: 并发启动互斥
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | DAG 引导 goroutine + API /start-all goroutine |
| 刺激 | 两个 goroutine 同时尝试启动同一服务（如 git-oauth） |
| 制品 | `startAndCheck()` 函数 |
| 环境 | runAll 正常启动中 |
| 响应 | 第一个 goroutine 获得 CAS 锁并执行 launch；第二个 CAS 失败 → 等待或返回错误 |
| 响应度量 | 同一服务 `cmd.Start()` 调用次数 = 1；无 EADDRINUSE |

### QS-02: 启动成功率
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 运维/开发者 |
| 刺激 | 启动 runAll（含 DAG 引导）或点击「全部启动」 |
| 制品 | runAll Runner |
| 环境 | 正常启动（无预存端口冲突） |
| 响应 | 所有服务进入 healthy 状态 |
| 响应度量 | git-oauth 启动成功率 100%（10 次启动 0 次 EADDRINUSE） |

### QS-03: 孤儿进程清理
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 新旧 runAll 实例替换（如重启 runAll） |
| 刺激 | 新 runAll 启动时旧 runAll 的托管服务仍在运行 |
| 制品 | `killPreviousRunAllProcess()` + `preflightService()` |
| 环境 | 旧 runAll 异常退出（未完成 Shutdown） |
| 响应 | 新 runAll 等待旧服务端口释放（最长 15s），超时则 SIGKILL 清理 |
| 响应度量 | 启动后无孤儿进程占用托管端口 |

### QS-04: SO_REUSEADDR 端口复用
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | runAll 重启 git-oauth 服务 |
| 刺激 | git-oauth 刚停止（端口处于 TIME_WAIT），立即重新启动 |
| 制品 | `NoReverseDNSWSGIServer.server_bind()` |
| 环境 | git-oauth 频繁重启 |
| 响应 | `bind()` 成功，即使在 TIME_WAIT 窗口内 |
| 响应度量 | 连续 5 次重启 git-oauth 全部成功 |

### QS-05: lsof 错误 fail-fast
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | `listenerPIDs` 调用失败（如 lsof 不可用） |
| 刺激 | `startAndCheck` 执行端口检测时 lsof 返回非退出码1的错误 |
| 制品 | `startAndCheck()` 端口检测逻辑 |
| 环境 | lsof 异常（权限不足/命令缺失） |
| 响应 | 服务标记 Failed，输出明确错误信息，不执行 `cmd.Start()` |
| 响应度量 | lsof 错误时 0 次执行 `cmd.Start()` |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 容错 L2: 启动互斥 | 服务状态机需支持「另一个 goroutine 正在启动」的中间态检测 | `ManagedService` 实体 `CanStart()` 保持现有逻辑；新增 `startAndCheck` 调用前的 CAS 守卫 |
| 可用性 L2: SO_REUSEADDR | 无领域层影响 — 基础设施层修复 | 无需 DDD 变更 |

> 本次修复为基础设施层变更，不引入新的领域概念。DDD 步骤主要是验证现有领域模型是否足以支撑 CAS 守卫逻辑。

## 权衡与边界

### 取舍
- 选择在 `startAndCheck` 入口增加 CAS（同步互斥），而非重构整个启动流程为单一路径 — 改动最小、风险最低
- `waitForServiceStart` 使用轮询而非 channel 通知 — 避免引入新的并发原语复杂度

### 明确不做什么
- 不重构 DAG 引导与 API 启动为统一调用链（风险过大）
- 不在 V1 引入分布式锁（单机 runAll 无需）
- 不修改其他 Python WSGI 服务的 `run.sh`（仅 gitOauth 确认有此问题，其他服务待排查后单独处理）

### 升级触发条件
- 如果未来 runAll 支持多实例协同（分布式部署），CAS 需升级为分布式锁（Redis/etcd）
- 如果 gitOauth 迁移到生产级 WSGI 服务器（gunicorn/uwsgi），SO_REUSEADDR 问题自动消除

## 跳过声明
- **性能**: 跳过。Bug 修复不改变服务端处理时间。
- **可伸缩性**: 跳过。单机开发环境，无扩展需求。
- **安全性**: 跳过。无新增认证/授权/加密变更。
- **数据一致性**: 跳过。无新增数据写入路径。
- **可观测性**: 跳过。现有日志已覆盖新增错误路径。
- **合规与隐私**: 跳过。无用户数据处理变更。
- **可维护性**: 跳过。无新增 API 或配置契约。
