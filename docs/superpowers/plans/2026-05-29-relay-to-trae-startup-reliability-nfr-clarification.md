# NFR 澄清: relayToTrae 启动可靠性

> 输入: design + value-stream  
> 输出使用者: DDD / plans / build

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 容错 | L2 | register-reachability 503 自动重试 ≤4 次 / ~1.7s |
| 可观测性 | L2 | internal error 带 error_code；Grafana trace 可关联 |
| 数据一致性 | L2 | reachability 仍强一致写 CloudServerConfig |
| 性能 | L1 | 启动允许额外 1～2s 重试窗口 |
| 可伸缩性 | L0 | 不适用（本地 dev 链路） |

## 质量场景

### QS-01: SQLite busy 可恢复
| 要素 | 内容 |
|------|------|
| 刺激 | register-reachability 遇 database locked |
| 响应 | HTTP 503 + RELAY_DOWNSTREAM_BUSY |
| 度量 | onlineServiceJS 在 4 次内成功或明确失败 |

### QS-02: status-push 可达
| 要素 | 内容 |
|------|------|
| 刺激 | go_relay POST .../relay-to-trae/status-push/ |
| 响应 | 非 404，ACK 或业务 4xx |
| 度量 | taskAgentSupport 单测路径解析通过 |

## 领域模型影响

- 容错 L2 → reachability 应用层重试，领域服务不变
- 可观测性 L2 → internal_dispatch 结构化 error_code

## 权衡

- 不做 Phase 2 batch API
- reachability 失败仍 exit(1)（安全策略不变）
