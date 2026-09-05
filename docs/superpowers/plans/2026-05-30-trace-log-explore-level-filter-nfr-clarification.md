# NFR 澄清: Trace Log Explore level 过滤

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-30-trace-log-explore-level-filter-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-30-trace-log-explore-level-filter-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | Grafana 变量查询 < 2s（本地 Loki） |
| 可维护性 | L2 | dashboard JSON 有自动化结构测试 |
| 可观测性 | L2 | 复用现有 level label，无新采集 |
| 其余 | L0 | 不适用（纯 Grafana 配置） |

## 质量场景

### QS-01: level 下拉加载
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L1 |
| 刺激源 | 开发者 |
| 刺激 | 打开 dashboard 并展开 level 变量 |
| 制品 | Grafana templating query |
| 环境 | 本地 ai-monitor |
| 响应 | 显示 Loki 中存在的 level 值 |
| 响应度量 | 手动或 provisioning 后 5s 内可见选项 |

## 领域模型影响

无 — 纯 Grafana JSON 配置，跳过 DDD 代码生成。

## 权衡与边界

- 不修改 runAll 深链 API
- 无 level label 的历史行仅在 All 时可见

## 跳过声明

- 安全性、一致性、可用性：L0，无运行时变更
