# NFR 澄清: runAll 显式生命周期命令

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-31-runall-explicit-lifecycle-commands-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-31-runall-explicit-lifecycle-commands-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 单服务 stop/start 脚本 30s 内完成（不含 health 等待） |
| 可用性 | L2 | 组关闭 best-effort：一项失败不阻塞其余 stop |
| 可维护性 | L3 | 100% 服务 strict 必填 start/stop；命令可 CLI 复现 |
| 可观测性 | L2 | 生命周期命令 stderr 写入服务日志 |
| 数据一致性 | L0 | 无业务数据 |
| 安全性 | L1 | 本地开发编排，无多租户 |
| 可伸缩性 | L0 | 单机开发者 |

## 质量场景

### QS-01: UI 关闭 docker-kafka
| 要素 | 内容 |
|------|------|
| 刺激源 | 开发者 |
| 刺激 | POST /api/stop docker-kafka |
| 制品 | stop_command → docker compose down |
| 环境 | 正常 |
| 响应 | 18080 不可达；docker 无 kafka 容器 |
| 响应度量 | `docker ps` 无 `kafka-` 前缀容器；status stopped |

### QS-02: 关闭本组一项失败
| 要素 | 内容 |
|------|------|
| 刺激源 | 开发者 |
| 刺激 | stop-group，首服务 stop 脚本 exit 1 |
| 制品 | stopGroup best-effort |
| 环境 | 正常 |
| 响应 | 后续服务仍执行 stop_command |
| 响应度量 | Go 测试：B 为 stopped；API 返回聚合 error |

### QS-03: strict 配置拒绝启动
| 要素 | 内容 |
|------|------|
| 刺激源 | runAll 进程 |
| 刺激 | LoadConfig 缺 stop_command |
| 制品 | config validate |
| 环境 | 启动前 |
| 响应 | 进程 exit，错误含服务名 |
| 响应度量 | `LoadConfig` 返回 error |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| 可维护性 L3 | 生命周期命令为一等值对象 | `ServiceLifecycleCommands` VO |
| 可用性 L2 组关闭 | 组停止为领域服务，收集部分失败 | `GroupStopOrchestrationService` |
| 性能 L2 | 执行器与策略分离 | `LifecycleExecutor` 基础设施端口 |

## 权衡与边界

### 取舍
- restart 由 runner 组合 stop+start，减少 YAML 重复与双轨脚本。

### 明确不做什么
- 不为「启动本组」做 best-effort（仍 fail-fast）。
- 不实现远程/SSH 生命周期。

### 升级触发条件
- 若需 CI 无 UI 编排 → 增加 `runAll --stop-all` CLI（未来）。

## 跳过声明
- 合规/多区域：不适用（本地工具）。
