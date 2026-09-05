# NFR 澄清: git-service 容器复用快速重启

> 输入:
> - 设计文档: `.claude/plans/git-service-fast-restart-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-30-git-service-fast-restart-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 | 理由 |
|------|------|-----------|------|
| **性能** | L2 | 重启 < 5s（容器存在时），首次 ≤ 4.5 min | 核心目标 |
| **可用性** | L1 | 不降级现有可用性 | 不改变健康检查逻辑 |
| **安全性** | L0 | 不适用 | 无新增认证/授权/加密 |
| **数据一致性** | L0 | 不适用 | 无数据写入/事务 |
| **容错机制** | L2 | 容器损坏时自动 fallback 重建 | 核心容错路径 |
| **可观测性** | L1 | runAll 现有日志/健康检查不变 | 无新增指标 |
| **可维护性** | L2 | 向后兼容现有 run.sh 接口 | start/stop/managed 模式不变 |
| **可伸缩性** | L0 | 不适用 | 单容器模式 |
| **合规与隐私** | L0 | 不适用 | 无数据变更 |

## 逐增量 NFR 分析

### Increment 1: 容器复用 — ensure_container 逻辑

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: 容器已存在时重启 ≤ 5s（从 runAll start 到 health check 通过）；首次冷启动保持不变（≤ 4.5 min）
- **质量场景**: QS-01

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: 容器状态异常（已停止但损坏）时，fallback 到 `down → up -d` 重建，增加不超过 5 min
- **质量场景**: QS-02

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: `run.sh start` / `run.sh stop` / `run.sh managed` 接口行为向后兼容；新增 `--clean` 标志
- **质量场景**: QS-03

### Increment 2: Bootstrap 标记跳过

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: Bootstrap 脚本在标记存在时 < 1s 跳过（vs 当前 2-3 min）
- **质量场景**: QS-04

## 质量场景

### QS-01: 容器复用秒级重启
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | runAll 编排器 |
| 刺激 | 发起 git-service restart（stop → start） |
| 制品 | `gitService/run.sh start` |
| 环境 | GitLab 容器已存在且健康 |
| 响应 | 检测到运行中容器，跳过重建和 bootstrap，直接返回 |
| 响应度量 | 从 `run.sh start` 调用到 `docker ps` 显示 healthy ≤ 5s |

### QS-02: 容器损坏自动重建
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | runAll 编排器 |
| 刺激 | 发起 git-service start，但已存在容器已停止且 `compose start` 失败 |
| 制品 | `gitService/run.sh start` |
| 环境 | 容器状态异常（exited with error） |
| 响应 | fallback 执行 `compose down` + `compose up -d` 重建 |
| 响应度量 | 重建后 runAll health check 在 15 min 内通过 |

### QS-03: 向后兼容
| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 开发者 / runAll |
| 刺激 | 调用 `bash gitService/run.sh start` / `stop` / `managed` |
| 制品 | `gitService/run.sh` |
| 环境 | 各种容器状态（无容器/已停止/运行中） |
| 响应 | 所有三种模式正常完成；EXIT CODE 0 |
| 响应度量 | `managed` 模式在 runAll 超时（900s）内健康检查通过 |

### QS-04: Bootstrap 跳过
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | `run.sh`（start/managed 分支） |
| 刺激 | Container 复用场景下执行 bootstrap 脚本 |
| 制品 | `run_bootstrap_if_needed()` 函数 |
| 环境 | Bootstrap 标记文件存在且有效 |
| 响应 | 三个脚本全部跳过，打印"已跳过"日志 |
| 响应度量 | Bootstrap 阶段总耗时 < 1s |

## 领域模型影响

本次为基础设施 Bash 脚本优化，**不引入新的领域概念或不改变现有领域模型**。Docker 容器生命周期管理属于基础设施层，不触及 `gitService/domain/` 中的 Go 领域模型。

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| （无） | （无） | 无需 DDD 建模 |

## 权衡与边界

### 取舍
- **选择保留容器而非彻底清理**：`docker compose stop` 保留容器（占用少量磁盘），换取秒级重启。若需完全清理，使用 `--clean`。
- **选择文件标记而非配置校验**：Bootstrap 标记文件简单有效，不实现配置 hash 校验（过度设计）。OAuth 配置变更时需要手动 `stop --clean && start`。

### 明确不做什么
- 不实现 GitLab 进程级热重启（`gitlab-ctl restart` 替代容器重启）—— 收益有限，复杂度高
- 不修改 runAll 健康检查配置 —— 超时/重试参数不变
- 不实现配置变更自动检测 —— 标记文件方案足够，变更后手动 `--clean`
- 不修改其他 Docker 服务（redis/kafka/portainer）—— 它们的启动够快，不需要优化

### 升级触发条件
- 当开发团队反馈 bootstrap 标记经常过期需要手动清理 → 引入配置 hash 校验
- 当 GitLab CE 升级导致 `compose start` 行为变化 → 增加容器版本检测

## 跳过声明

- **安全性**: 跳过。仅修改 Bash 启动脚本，无新增端点和数据流。
- **数据一致性**: 跳过。无数据写入操作，只改容器生命周期命令。
- **可伸缩性**: 跳过。git-service 始终单容器模式。
- **合规与隐私**: 跳过。无数据变更。
- **可用性**: L1 基础（隐含，无专项目标）。不改变健康检查和服务可用性保证。
