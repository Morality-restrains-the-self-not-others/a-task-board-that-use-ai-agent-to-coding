# NFR 澄清: runAll dockerInfra 拆分

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-31-runall-docker-infra-split-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-31-runall-docker-infra-split-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 镜像已齐时 Redis/Kafka 栈 30s 内 healthy |
| 可用性 | L1 | 本地开发工具，无 SLA |
| 可维护性 | L2 | 旧路径删除 + README 指向新目录 |
| 可观测性 | L1 | runAll UI 状态 + 现有 Prometheus HTTP 目标（Kafka UI） |
| 容错机制 | L2 | pull 分离 + detach compose 健康等待（沿用） |
| 安全性 | L0 | 本地 localhost 绑定，无变更 |
| 数据一致性 | L0 | 无业务数据 |
| 可伸缩性 | L0 | 单开发者单实例 |

## 逐增量 NFR 分析

### Increment 1: Redis 栈

#### 性能 — L2
- **量化目标:** 镜像已齐时 P95 启动就绪 ≤ 15s（TCP 6379）
- **质量场景:** QS-01

#### 容错机制 — L2
- 缺镜像时一次 pull；日常 `--pull never`

### Increment 2–4: Kafka + 迁移 + runner

#### 性能 — L2
- Kafka 栈（三容器）镜像已齐时 P95 ≤ 45s（HTTP 18080）

#### 可维护性 — L2
- 删除 `task2app/Saas_project/docker-compose.yml`；`dockerInfra/README.md` 说明迁移

## 质量场景

### QS-01: Redis 独立启动就绪
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 开发者 runAll UI |
| 刺激 | 仅启动 docker-redis，本地已有 redis:7.0-alpine |
| 制品 | dockerInfra/redis/run.sh + runAll TCP 探针 |
| 环境 | 正常（Docker Desktop 运行） |
| 响应 | 6379 可连接，runAll 状态 healthy |
| 响应度量 | `nc -z 127.0.0.1 6379` 成功；UI 30s 内 green |

### QS-02: Kafka 独立启动不 pull 重复镜像
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 开发者第二次启动 docker-kafka |
| 刺激 | 镜像已 inspect 成功 |
| 制品 | dockerInfra/kafka/run.sh |
| 环境 | 正常 |
| 响应 | 日志无 Pulling；18080 HTTP 200 |
| 响应度量 | 日志 grep 无 Pulling；curl 18080 成功 |

### QS-03: platform 链不依赖 Kafka
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 开发者启动 saas-backend 链 |
| 刺激 | 仅 docker-redis healthy，docker-kafka 未启 |
| 制品 | runAll depends_on |
| 环境 | domainEvents.transport=redis |
| 响应 | task-sse → saas-backend 级联成功 |
| 响应度量 | saas-backend `/api/health/` 200 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| TCP 探针 L2 | ReadinessProbe 需区分 HTTP/TCP | 值对象 `InfrastructureStackReadinessProbe` |
| 栈独立生命周期 L2 | RedisStack / KafkaStack 分离聚合 | 两个 Stack 实体，各自 PullPolicy |
| detach compose L2 | StackLifecycle 与 ManagedService 解耦 | 领域服务识别 managed script 模式 |

## 权衡与边界

### 取舍
- 默认不自动 depends_on docker-kafka，换取更短启动路径
- Prometheus 暂不为 Redis 增加 TCP/blackbox 目标

### 明确不做什么
- 不动 Kafka/Redis 端口与业务 transport 代码
- 不做 runAll 读 port_config 动态 depends_on（后续增量）

### 升级触发条件
- 若 default transport 回退 kafka → 文档或配置恢复 docker-kafka 硬依赖

## 跳过声明

- 安全性 L0：本地开发 localhost，无 prod 暴露
- 合规 L0：不涉及用户数据
