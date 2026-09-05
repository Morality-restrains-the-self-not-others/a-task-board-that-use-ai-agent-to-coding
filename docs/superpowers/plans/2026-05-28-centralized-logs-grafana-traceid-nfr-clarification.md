# NFR 澄清: 多服务日志集中收集与 Grafana traceId 查询

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-28-centralized-logs-grafana-traceid-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-28-centralized-logs-grafana-traceid-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | Promtail/Loki 不阻塞业务 stdout；查询 dev 量级 <3s |
| 可伸缩性 | L0 | 本地单副本，不做水平扩展 |
| 可用性 | L2 | Loki 宕机不影响业务服务启动与 runAll tee |
| 安全性 | L2 | access_token 等不做 Loki label |
| 数据一致性 | L0 | 日志最终一致即可，无事务 |
| 容错 | L2 | Promtail push 失败丢弃/重试，不反压 kill 子进程 |
| 可观测性 | L3 | 本特性即增强可观测性；Increment 1 覆盖 traceId 日志检索 |
| 合规 | L0 | 本地 dev，无跨境/GDPR 专项 |
| 可维护性 | L2 | 配置集中在 AiMonitor/；runAll 仅 tee 路径 |

## 逐增量 NFR 分析

### Increment 1: Loki MVP

#### 可用性 L2
- Loki/Promtail 未启动时 runAll 与各业务服务正常启动。
- 文件 tee 写失败仅 log.Printf，不终止子进程。

#### 安全性 L2
- Promtail 仅提取 trace_id/level/service 为 label。
- 日志正文可含敏感信息——仅限本地 dev Grafana（admin 默认凭据）。

#### 可观测性 L3
- QS-01：Grafana 按 traceId 返回 saas-backend 日志（见下）。

### Increment 2: 跨服务 JSON

#### 性能 L1
- JSON 行日志额外开销可忽略（dev）。

#### 安全性 L2
- Go/Python 中间件传播 X-Trace-Id，不记录 token 到结构化字段。

## 质量场景

### QS-01: Grafana traceId 检索 saas-backend 日志
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L3 |
| 刺激源 | 开发者 |
| 刺激 | 对 saas-backend 发 HTTP 请求，携带 `X-Trace-Id: test-trace-abc12345` |
| 制品 | Loki + Grafana Explore |
| 环境 | 本地 runAll + AiMonitor 运行 |
| 响应 | LogQL `{trace_id="test-trace-abc12345"}` 至少 1 条 saas-backend 日志 |
| 响应度量 | 手动或集成脚本查询 Loki API `/loki/api/v1/query_range` 返回非空 |

### QS-02: Loki 不可用不影响 saas-backend 启动
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 运维/开发者 |
| 刺激 | 停止 Loki 容器，runAll 启动 saas-backend |
| 制品 | runAll runner + 文件 tee |
| 环境 | 本地 |
| 响应 | saas-backend health 200；tee 文件仍有写入 |
| 响应度量 | `curl health` 成功；`RUNALL_LOG_ROOT/saas-backend.log` 非空 |

### QS-03: Promtail push 失败不阻塞 runAll
| 要素 | 内容 |
|------|------|
| 类别 | 容错 |
| 等级 | L2 |
| 刺激源 | 网络/容器 |
| 刺激 | Loki 不可达时 Promtail 持续 tail |
| 制品 | Promtail |
| 环境 | Loki 停止 |
| 响应 | Promtail 重试/丢弃，runAll 子进程不受影响 |
| 响应度量 | runAll 服务 status healthy |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| 可用性 L2 — 日志基础设施可选 | 日志采集与编排生命周期解耦 | `ServiceLogFileSink` 端口；失败不传播到 ManagedService |
| 安全 L2 — 限制 label | TraceId 值对象校验格式 | `TraceId` VO 复用 http_trace 规则 |
| 最终一致 L0 | 无跨聚合事务 | 小聚合：LogEntry + FileSink |

## 权衡与边界

### 取舍
- 日志进 Loki 而非 Prometheus，换取 traceId 全文检索。
- Increment 1 仅 regex 兼容旧 Django 文本，JSON 统一延后到 Increment 2。

### 明确不做什么
- V1 不做 Tempo/Jaeger span 树。
- 不做生产多租户 Loki 集群。

### 升级触发条件
- 生产部署 → 可伸缩性升至 L2，外部 Loki + 认证。
- 合规要求 → 安全升至 L3，日志脱敏 pipeline。

## 跳过声明
- 可伸缩性 L0：本地 dev 单实例。
- 数据一致性 L0：日志无强一致要求。
- 合规 L0：本地开发环境。
