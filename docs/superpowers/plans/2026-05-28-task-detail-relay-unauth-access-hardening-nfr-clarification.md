# NFR 澄清: 任务详情 relayToTrae 未登录访问加固

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-28-task-detail-relay-unauth-access-hardening-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-28-task-detail-relay-unauth-access-hardening-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 受保护路由 profile 校验 P95 ≤ 300ms（内网） |
| 安全性 | L3 | 未认证不可访问 task-detail；失败时清除全部前端凭据 |
| 可用性 | L2 | 会话失效自动清理 stale 凭据并引导登录 |
| 可维护性 | L2 | Vitest + Playwright 回归 |

## 质量场景

### QS-01: 无效凭据访问 task-detail
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激 | GET task-detail/?relayToTrae=true，无有效 session/token |
| 响应 | 导航至 /auth/login/ |
| 响应度量 | Playwright 断言 final URL；userId 与 authToken 均被清除 |

### QS-02: Navbar 与守卫一致
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激 | profile 200 的有效会话进入 task-detail |
| 响应 | Navbar 显示已登录 |
| 响应度量 | isUserAuthenticated === true |

## 领域模型影响

| NFR 决策 | 模型影响 | DDD 动作 |
|----------|---------|----------|
| 安全 L3 | 认证状态须服务端可验证 | 扩展 AuthSessionGuard 清除 authToken |
| 安全 L3 | UI 与守卫同源 | Navbar 复用 profile 判定 |

## 权衡与边界

### 明确不做什么
- 不引入 accessCode 免登录
- 不做 profile 缓存
