# NFR 澄清: 任务详情未登录跳转登录页

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-05-27-task-detail-unauthenticated-login-redirect-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-05-27-task-detail-unauthenticated-login-redirect-value-stream.md`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L2 | 受保护路由导航 profile 校验 P95 ≤ 300ms（内网） |
| 安全性 | L3 | 未认证不可访问 task-detail；回跳 URL 仅允许站内相对路径 |
| 可用性 | L2 | 会话失效时自动清理 stale Cookie 并引导登录 |
| 可维护性 | L2 | 认证逻辑单测 + Playwright 回归 |

## 质量场景

### QS-01: 无效 userId 访问受保护页
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 浏览器（仅 stale userId Cookie） |
| 刺激 | GET task-detail/?relayToTrae=true |
| 制品 | Vue router beforeEach |
| 环境 | 正常 |
| 响应 | 302/客户端导航至 /auth/login/ |
| 响应度量 | Playwright 断言 final URL 匹配 `/auth/login/` |

### QS-02: 登录后回跳
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 已写入 postLoginRedirect 的登录成功 |
| 刺激 | 用户完成登录 |
| 制品 | Login.vue |
| 环境 | 正常 |
| 响应 | location.href 为原 task-detail fullPath |
| 响应度量 | localStorage 值以 `/tenant/` 开头且含 relayToTrae |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全 L3 | 认证状态须服务端可验证 | AuthSessionGuard 领域服务，profile 为唯一真源 |
| 安全 L3 | 回跳防开放重定向 | PostLoginReturnUrl 值对象校验相对路径 |

## 权衡与边界

### 明确不做什么
- 不引入 accessCode 免登录 task-detail
- 不做 profile 结果长期缓存（V1）

### 升级触发条件
- 受保护路由导航 profile QPS 过高时再考虑短 TTL 内存缓存
