# NFR 澄清: OIDC SLO 登出同步

> 输入:
> - 设计文档: `docs/specs/oidc-slo-logout-sync-设计文档.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-25-oidc-slo-logout-sync-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 性能 | L1 | 登出重定向链 ≤ 5 秒完成（3 跳），无严格延迟目标 |
| 可伸缩性 | L0 | 不适用 — 登出是低频操作 |
| 可用性 | L1 | taskAuth 不可达时主应用登出仍须成功（降级） |
| 安全性 | L2 | 服务端销毁 session + 清除 cookie；POST 登出需 CSRF 保护 |
| 数据一致性 | L1 | 尽力而为 (best-effort) — 主应用 session 优先销毁，GitLab 登出允许失败 |
| 容错机制 | L2 | GitLab 不可达时 SLO 降级但不阻断主应用登出 |
| 可观测性 | L1 | 标准日志 — 登出动作记录到现有日志流 |
| 合规与隐私 | L0 | 不适用 |
| 可维护性 | L1 | 新增端点不超过 50 行 Go 代码；前端改动不超过 30 行 JS |

## 逐增量 NFR 分析

### Increment 1: taskAuth EndSession 端点

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**: userId cookie 必须服务端清除（Set-Cookie: Max-Age=0）；customtoken 行必须物理删除
- **质量场景**: QS-SEC-01

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: EndSession 端点即使无法连接 GitLab（不验证 GitLab 可达性），也必须 302 重定向
- **质量场景**: QS-FT-01

### Increment 2: 主应用服务端登出

#### NFR 类别: 安全性
- **等级**: L2 - 标准
- **量化目标**: Django session 服务端 `auth_logout()` 调用 + `sessionid` cookie 删除；userId cookie 删除
- **质量场景**: QS-SEC-01

#### NFR 类别: 可用性
- **等级**: L1 - 基础
- **量化目标**: taskAuth 不可达时，主应用仍执行本地 session 销毁并返回成功
- **质量场景**: QS-AV-01

### Increment 3: 前端 SLO 重定向

#### NFR 类别: 性能
- **等级**: L1 - 基础
- **量化目标**: 用户感知的登出时间（点击 → 回到登录页）≤ 5 秒
- **质量场景**: QS-PF-01

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: API 调用失败时仍清除本地状态并跳转登录页（不回退到已登录状态）
- **质量场景**: QS-FT-02

### Increment 4: Playwright E2E 测试

无独立 NFR 要求。测试本身的通过标准覆盖上述所有质量场景。

## 质量场景

### QS-SEC-01: 登出必须服务端销毁所有会话令牌
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L2 |
| 刺激源 | 已认证用户点击「退出」按钮 |
| 刺激 | POST /api/accounts/users/logout/ + 浏览器重定向至 taskAuth EndSession |
| 制品 | Django session 中间件 + taskAuth EndSession handler |
| 环境 | 正常 |
| 响应 | Django session 从数据库删除；taskAuth customtoken 行删除；userId cookie Max-Age=0；sessionid cookie Max-Age=0 |
| 响应度量 | 登出后使用原 sessionid 访问 /api/accounts/profile/ 返回 401；使用原 token 访问 taskAuth 返回 401 |

### QS-AV-01: taskAuth 不可达时主应用登出降级成功
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L1 |
| 刺激源 | 已认证用户点击「退出」按钮（此时 taskAuth 宕机） |
| 刺激 | POST /api/accounts/users/logout/ |
| 制品 | Django logout API |
| 环境 | taskAuth 不可达 |
| 响应 | Django 本地 session 销毁成功；返回 200 + 无 slo_redirect_url（降级） |
| 响应度量 | 前端收到 200，清除本地状态，跳转 /auth/login/；GitLab 可能仍登录（可接受降级） |

### QS-FT-01: GitLab 不可达时 EndSession 仍完成本地清理
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 浏览器重定向至 taskAuth GET /api/oidc/endsession |
| 刺激 | taskAuth 收到 EndSession 请求（GitLab :8012 不可达） |
| 制品 | taskAuth EndSession handler |
| 环境 | GitLab 宕机 |
| 响应 | taskAuth 清除 userId cookie + 销毁 token；302 重定向至 GitLab sign_out（不验证可达性） |
| 响应度量 | 浏览器收到 302 Location = GitLab sign_out URL；即使 GitLab 返回连接错误，用户刷新后 taskAuth 已登出 |

### QS-FT-02: 登出 API 调用失败时前端仍清除本地状态
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 用户点击「退出」按钮（网络异常或服务端错误） |
| 刺激 | POST /api/accounts/users/logout/ 返回 4xx/5xx 或网络错误 |
| 制品 | Navbar.logic.vue handleLogout() |
| 环境 | 网络异常或服务端错误 |
| 响应 | catch 块清除 localStorage、userId cookie、window.currentUser；跳转 /auth/login/ |
| 响应度量 | 用户最终在登录页；localStorage 无 authToken；cookie 无 userId |

### QS-PF-01: 登出端到端延迟
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L1 |
| 刺激源 | 用户点击「退出」按钮 |
| 刺激 | 完整的登出重定向链（3 跳） |
| 制品 | 主应用 → taskAuth → GitLab → 登录页 |
| 环境 | 所有服务正常，局域网延迟 < 5ms |
| 响应 | 浏览器最终渲染 /auth/login/ 页面 |
| 响应度量 | 端到端耗时 ≤ 5 秒（由 Playwright E2E 测试测量 page.waitForURL 超时） |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全性 L2 — 服务端 session 销毁 | Django logout 视图需要显式调用 `auth_logout()` | auth_views.py 补全函数体 |
| 容错 L2 — GitLab 不可达降级 | EndSession 不验证 GitLab 可达性，直接 302 | taskAuth handler 无外部 HTTP 调用，纯重定向 |
| 可用性 L1 — taskAuth 不可达降级 | Django logout API 中 taskAuth 转发失败不抛异常 | try/except 包裹 forward_to_taskauth 调用 |
| 一致性 L1 — 尽力而为 (best-effort) | 不需要 Saga 或补偿事务；不需要 Outbox | 无领域事件；无需跨服务协调 |

## 权衡与边界

### 取舍
- **选择 best-effort SLO 而非强一致登出**: 主应用登出优先保证成功；GitLab 登出允许失败。不做两阶段提交或补偿事务。
- **选择浏览器重定向链而非 Back-Channel Logout**: 简单直接，不需要 taskAuth 维护 RP 列表。代价是用户看到短暂的中间跳转。

### 明确不做什么
- **不做 JWT token 撤销（黑名单）**: Access token 在 TTL 内仍然有效（最多 1 小时）。浏览器会话层的保护已足够。
- **不做 OIDC Session Management** (RP iframe + OP iframe): 复杂度高，浏览器支持差。
- **不做 Back-Channel Logout**: 需要 taskAuth 维护 RP 注册表并逐一下发 HTTP 通知。
- **不做 GitLab OmniAuth `end_session_endpoint` 配置**: 直接跳转 `/users/sign_out` 已足够。
- **不做 OIDC `sid` (session_id) claim**: 无 session 管理需求，不需要在 ID token 中追踪会话。

### 升级触发条件
- **当接入第三个 OIDC RP 时**: 考虑实现 Back-Channel Logout，而非继续扩展重定向链。
- **当 JWT token 生命周期需要动态撤销时**: 引入 Redis 黑名单机制（`jti` 检查）。
- **当安全审计要求全量登出日志时**: 可观测性从 L1 升级到 L2，登出事件进入审计日志。

## 跳过声明
- **可伸缩性 (L0)**: 登出是低频操作（用户会话结束时触发），不需要扩展性设计。
- **合规与隐私 (L0)**: 不涉及用户数据跨境、GDPR 遗忘权等要求。
- **数据一致性 (L1)**: 已分析但仅需 L1 best-effort，不需要分布式事务或 Saga。已在「权衡与边界」中说明。
