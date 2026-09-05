# NFR 澄清: Fix Port Occupancy Race in runAll startAndCheck

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-port-occupancy-race-analysis.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-port-occupancy-race-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 容错机制 | L2 | 端口释放等待 30s 超时，500ms 轮询 |
| 可用性 | L2 | runAll 重启后所有托管服务 60s 内恢复 healthy |
| 可观测性 | L1 | 端口等待过程记录日志（等待中/超时/成功） |

## 逐增量 NFR 分析

### Increment 1: Port-Release Wait Loop

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: 旧进程收到 SIGTERM 后最多等待 30s 释放端口（500ms 轮询），超时后 force-kill + 启动新进程
- **质量场景**: QS-01

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: 平台重启后，全部 17 个 taskEvents 消费者在 60s 内恢复 healthy
- **质量场景**: QS-02

#### NFR 类别: 可观测性
- **等级**: L1 - 基础
- **量化目标**: 端口等待过程有日志（`waiting for port %s to free, attempt %d/%d`），超时时记录警告
- **质量场景**: 无需独立场景

### Increment 2: SO_REUSEADDR for taskEvents

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: 消费者进程意外退出后，新进程可在 1s 内重新绑定端口（无 TIME_WAIT 阻塞）
- **质量场景**: QS-03

## 质量场景

### QS-01: Port release wait on restart
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | runAll restart 命令 |
| 刺激 | 停止旧消费者进程，旧进程 3s 后才释放 socket |
| 制品 | runAll startAndCheck() |
| 环境 | 正常重启 |
| 响应 | 轮询等待端口释放，检测到空闲后启动新进程 |
| 响应度量 | 新进程在旧进程退出后 500ms 内获得端口；等待不超过 30s |

### QS-02: All consumers healthy after restart
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | runAll 平台重启 |
| 刺激 | 级联重启全部 platform 组服务 |
| 制品 | runAll 编排引擎 |
| 环境 | 正常 |
| 响应 | 所有依赖服务进入 healthy |
| 响应度量 | 17 个 taskEvents 消费者在重启后 60s 内全部 healthy |

### QS-03: SO_REUSEADDR quick rebind
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | taskEvents 消费者进程 crash |
| 刺激 | 进程异常退出（socket 进入 TIME_WAIT） |
| 制品 | eventbin HTTP listener |
| 环境 | 故障恢复 |
| 响应 | 新进程立即 bind 成功 |
| 响应度量 | 进程退出后 1s 内新进程可绑定同端口 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 无 | 纯基础设施修复，不改变领域模型 | 无需 DDD 建模 |

## 权衡与边界

### 取舍
- 30s 端口等待超时：足够覆盖正常清理（<5s），又不会无限阻塞启动流程

### 明确不做什么
- 不改变 runAll 的 stop 命令机制（仍使用现有 stop_command）
- 不在 Docker 容器场景测试（仅本地裸进程）
- 不为 taskEvents 消费者增加主动健康上报（已有 passive health check）

### 升级触发条件
- 如果 30s 超时频繁触发 → 调查具体消费者清理时间，可能调整超时或优化 stop 信号处理

## 跳过声明
- **性能**: 跳过。端口轮询（500ms sleep）不影响正常请求路径。
- **可伸缩性**: 跳过。无新增负载。
- **安全性**: 跳过。无安全面变更。
- **数据一致性**: 跳过。无数据流变更。
- **合规与隐私**: 跳过。无新数据处理。
