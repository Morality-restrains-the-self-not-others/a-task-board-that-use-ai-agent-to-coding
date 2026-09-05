# NFR 澄清: taskAgentSupport Phase 2 路径化 Internal API

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-30-task-agent-support-phase2-path-scoped-internal-api-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-30-task-agent-support-phase2-path-scoped-internal-api-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | internal 转发 P95 ≤ 500ms（不含 gitOauth） |
| 可用性 | L2 | bootstrap 关键 path 404 率为 0 |
| 安全性 | L3 | Internal Secret + 容器 token 校验不变 |
| 可观测性 | L2 | 404 path 含 tenant/workspace/task |
| 可维护性 | L2 | 单条 scoped url 覆盖 12 action |
| 可伸缩性 | L0 | 不适用（内网单实例） |
| 合规 | L0 | 不适用 |

## 质量场景

### QS-01: bootstrap task-detail 非 404
| 要素 | 内容 |
|------|------|
| 刺激源 | onlineServiceJS bootstrap |
| 刺激 | POST task-detail 经 :8011 |
| 制品 | scoped internal dispatch |
| 环境 | 本地 runAll |
| 响应 | 200/401/403，非 404 |
| 响应度量 | pytest + 手工 relay 直启 |

### QS-02: Internal 404 含 scope
| 要素 | 内容 |
|------|------|
| 刺激源 | 故意错误 action |
| 刺激 | POST .../tenant/T/workspace/W/task/K/bad-action/ |
| 响应 | 404 path 含 T/W/K |
| 响应度量 | test 断言 path 字段 |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|---------|
| 安全 L3 | TaskScope 仍为路径一等公民 | 复用现有 TaskScope VO |
| 可维护 L2 | 无新聚合 | Adapter 层扩展 _VIEW_BY_ACTION |

## 权衡与边界

- 不提供 envelope-only 旧 internal URL 兼容
- onlineServiceJS 公网路径不变

## 跳过声明

- 可伸缩性/合规：内网 adapter，无专项要求
