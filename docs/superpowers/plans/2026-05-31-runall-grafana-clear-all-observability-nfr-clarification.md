# NFR 澄清: runAll 一键清空 Grafana 可观测数据

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-31-runall-grafana-clear-all-logs-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-31-runall-grafana-clear-all-observability-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 全栈 reset P95 ≤ 30s（本地 docker） |
| 可用性 | L1 | 清空期间 Loki/Tempo/Prometheus 短暂不可用可接受 |
| 安全性 | L1 | localhost dev-only，无鉴权（与 runAll 其他 API 一致） |
| 可观测性 | L2 | API 返回逐步 reset 状态 |
| 容错 | L2 | partial failure 返回明细，本地段仍尽力完成 |
| 可伸缩性 | L0 | 不适用（单开发者本地栈） |
| 数据一致性 | L1 | 最终一致；volume 删后无旧数据 |
| 合规 | L0 | 不适用 |

## 质量场景

### QS-01: 一键清空耗时
| 要素 | 内容 |
|------|------|
| 刺激源 | 开发者点击按钮 |
| 刺激 | POST clear-all |
| 制品 | runAll API + reset 脚本 |
| 环境 | ai-monitor 已启动，本地 docker |
| 响应 | 30s 内返回 ok/partial |
| 响应度量 | 端到端 wall time ≤ 30s |

### QS-02: partial failure 可诊断
| 要素 | 内容 |
|------|------|
| 刺激源 | docker 不可用 |
| 刺激 | clear-all |
| 响应 | status=partial，loki_reset 含 error 字符串 |
| 响应度量 | 内存/files_truncated 仍 > 0 |

## 领域模型影响

| NFR 决策 | 模型影响 |
|----------|---------|
| partial failure L2 | `ObservabilityStackResetResult` 逐步状态字段 |
| 无鉴权 L1 | 领域服务无 actor 参数 |

## 权衡与边界

- 不做生产多租户 delete API
- 不 stop 业务服务
- 保留 grafana_data

## 跳过声明

- 可伸缩性/合规：本地 dev 工具，跳过
