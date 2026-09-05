# NFR 澄清: 任务详情 OAuth session userId 解析

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-28-task-detail-oauth-session-user-id-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-28-task-detail-oauth-session-user-id-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | profile 回退单次请求，6s connection 超时不变 |
| 安全性 | L3 | 仅 session 用户可读 profile；user-scoped API 仍靠后端 IsAuthenticated |
| 可用性 | L2 | profile 失败时显式错误，不 silent 降级为匿名 |
| 数据一致性 | L0 | 无持久化变更 |
| 可维护性 | L2 | 单工具函数复用 Navbar 语义 |

## 逐增量 NFR 分析

### Increment 1: session-user-id-resolver-thin-slice

#### 性能 — L2

- profile 仅在 cookie 缺失时调用；inflight 去重避免并发双请求

#### 安全性 — L3

- credentials: include；不暴露他人 user_id；connection API 仍 user-scoped

## 质量场景

### QS-01: 无 cookie 时 OAuth 绑定可继续

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 已登录用户 |
| 刺激 | 清除 userId cookie 后点击 OAuth 绑定 |
| 制品 | TaskDetailLinkedProjectsPanel + profile API |
| 环境 | 正常 |
| 响应 | 发出 connection GET，无「缺少 userId」弹窗 |
| 响应度量 | Vitest + 手工 Network 可见 profile（可选）与 connection 200/401 |

### QS-02: profile 与 cookie 并发去重

| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 页面加载多仓库 OAuth 检查 |
| 刺激 | 同 tick 内 3 次 resolveAuthenticatedUserId |
| 制品 | sessionUserIdUtils |
| 环境 | cookie 缺失 |
| 响应 | profile API 仅 1 次 |
| 响应度量 | 单元测试 mock call count ≤ 1 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全 L3 | 前端不持久化 userId，只读解析 | 轻量 SessionUserIdResolver 文档化，无后端聚合 |

## 权衡与边界

### 明确不做什么

- 不批量迁移其它 `getCookie('userId')` 调用点
- 不在 profile 成功时 backfill cookie

### 跳过声明

- 可伸缩性、合规、容错机制：不适用（纯前端读路径）
