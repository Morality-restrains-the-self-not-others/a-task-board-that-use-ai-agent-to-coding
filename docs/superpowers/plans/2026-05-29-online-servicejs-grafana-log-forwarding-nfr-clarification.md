# NFR 澄清: onlineServiceJS Grafana 日志转发

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-29-online-servicejs-grafana-log-forwarding-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-29-online-servicejs-grafana-log-forwarding-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 可观测性 | L2 | Loki 含 `onlineServiceJS`；堆栈单条展示 |
| 性能 | L1 | 日志转发不阻塞子进程启动 |
| 可用性 | L0 | 日志失败不影响 relay 拉起 |
| 可维护性 | L2 | 单元测试覆盖透传与合并规则 |
| 安全性 | L0 | 无新数据面 |

## 逐增量 NFR 分析

### Increment 1–2

#### 可观测性 — L2
- Loki `service` label 含 `onlineServiceJS`
- 多行 stack 合并为单条 `msg`

#### 性能 — L1
- 逐行 scanner，无额外落盘

## 质量场景

### QS-01: Loki 服务标签
| 要素 | 内容 |
|------|------|
| 刺激源 | relay 拉起 onlineServiceJS |
| 刺激 | 子进程输出 `logJson` 行 |
| 制品 | go-relay → Promtail → Loki |
| 响应 | `label/service/values` 含 `onlineServiceJS` |
| 响应度量 | `docker exec aimonitor-loki wget -qO- .../label/service/values` |

### QS-02: 堆栈单条展示
| 要素 | 内容 |
|------|------|
| 刺激源 | onlineServiceJS `console.error('...', err)` |
| 刺激 | 4+ 行 stack |
| 制品 | subprocessLogGrouper |
| 响应 | 单条 JSON `msg` 含 `\n` 连接的全栈 |
| 响应度量 | `subprocess_log_grouper_test` 通过 |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| 可观测性 L2 | 子进程日志为独立值对象流 | `SubprocessLogForwarder` 领域服务 |

## 权衡与边界

### 明确不做什么
- 不为 onlineServiceJS 单独 tee 文件（Increment 3+ 可选）
- 不修改 Promtail/Grafana 仪表盘查询

### 跳过声明
- 可伸缩性、合规、安全性：不适用（本地 dev 日志管道）
