# NFR 澄清: taskContainerGateway

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-31-task-container-gateway-auth-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-31-task-container-gateway-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | git-commit P95 ≤ 5s（含 validate + forward） |
| 安全性 | L3 | Session/Token 经 Django 真源校验；internal secret |
| 容错 | L2 | 出站 timeout；无无限 pending |
| 可观测性 | L2 | 每跳 JSON log + trace_id |
| 可用性 | L1 | 本地 runAll 单实例 |
| 可伸缩性 | L0 | 本地开发，不专项 |
| 合规 | L0 | 不适用 |

## Increment 1 NFR

### 性能 — L2
- P95 ≤ 5s（gateway 入口到响应完成）
- QS-01: zTree 提交，job-stream 活跃时仍 ≤5s

### 安全性 — L3
- 未认证 → 401，无 scope → 403
- Go 不读 django_session 表
- QS-02: 无 cookie 请求被拒绝

### 容错 — L2
- OSJS 不可达 → 502 ≤15s
- QS-03: 容器 down 时 502 非 hang

### 可观测性 — L2
- 同一 trace_id ≥2 条（gateway + onlineServiceJS）

## 领域模型影响

| NFR | 模型影响 |
|-----|---------|
| 安全 L3 | GatewaySessionValidation 聚合；TaskScopeAuthorization 值对象 |
| 容错 L2 | ContainerForwardSession 含 timeout；防腐层独立 http.Client |
| 可观测 L2 | forward_stage 枚举值对象 |

## 权衡

- 不做 validate 结果缓存（P1）— 正确性优先
- 不做 nginx 边缘（本地）— P4 生产可选
- relay-to-trae 仍 Django — 范围 A

## 跳过

- 可伸缩性 L0：runAll 单用户开发
- 合规 L0：无新数据驻留
