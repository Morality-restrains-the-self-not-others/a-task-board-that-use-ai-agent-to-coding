# NFR 澄清: AI Provider OIDC 登录

> 输入:
> - 设计文档: `docs/specs/ai-provider-oidc-login/design.md`
> - 价值流文档: `docs/specs/ai-provider-oidc-login/value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L3 | OIDC PKCE + state CSRF + JWKS 签名验证 + 审计日志 |
| 性能 | L2 | OIDC authorize 302 < 100ms, callback JWT 签发 < 500ms |
| 可用性 | L1 | 依赖 taskAuth OIDC Provider 可用性，AI Provider 本身无额外保证 |
| 数据一致性 | L2 | Vendor/Staff 匹配最终一致（通过 email），saas_user_id 绑定原子写入 |
| 可观测性 | L2 | OIDC 登录成功/失败结构化日志 + 链路 trace_id |
| 容错机制 | L2 | taskAuth token endpoint 调用超时 5s，失败返回明确错误 |
| 可维护性 | L1 | 配置驱动，OIDC Provider URL 可通过环境变量切换 |
| 可伸缩性 | L0 | 不适用 — 登录为低频操作，日活 < 100 用户 |

## 逐增量 NFR 分析

### Increment 1: OIDC RP 核心 — 后端 authorize + callback

#### NFR 类别: 安全性
- **等级**: L3 - 增强（auth 领域默认 L3）
- **量化目标**:
  - id_token 签名验证 100% 执行（RSA 公钥从 JWKS endpoint 获取）
  - PKCE code_challenge_method=S256 强制
  - state 参数长度 ≥ 32 字节随机数
  - callback 中 state 与 session 中存储值严格比对
- **质量场景**: QS-01, QS-02, QS-03

#### NFR 类别: 性能
- **等级**: L2 - 标准
- **量化目标**: authorize 端点 302 重定向 < 100ms（仅构建 URL，无 I/O）；callback 端点 JWT 签发 < 500ms（含 token endpoint 调用）
- **质量场景**: QS-04

#### NFR 类别: 数据一致性
- **等级**: L2 - 标准
- **量化目标**: `exchange_oidc_for_vendor` 中 saas_user_id 绑定在 `transaction.atomic()` 内执行
- **质量场景**: QS-05

#### NFR 类别: 可观测性
- **等级**: L2 - 标准
- **量化目标**: OIDC 登录成功/失败均记录结构化日志（包含 trace_id, role, email, outcome）
- **质量场景**: QS-06

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: taskAuth token endpoint HTTP 调用超时 5s，连接超时 2s；失败返回 HTTP 400（非 500），用户看到明确错误信息
- **质量场景**: QS-07

### Increment 2 & 3: taskAuth Client 注册 + 前端入口

- 无额外 NFR 要求。继承 Increment 1 的所有 NFR 等级。

### Increment 4: E2E 测试

- 跳过 NFR 分析（测试代码不产生运行时质量要求）。

## 质量场景

### QS-01: OIDC callback state CSRF 防护
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 恶意网站 |
| 刺激 | 诱导用户浏览器访问 `/api/auth/oidc/callback/?code=stolen&state=attacker_state` |
| 制品 | `GET /api/auth/oidc/callback/` |
| 环境 | 正常 |
| 响应 | 400 Bad Request，`{"detail": "state 不匹配"}` |
| 响应度量 | 100% 伪造 state 请求被拒绝，单元测试 `test_oidc_callback_state_mismatch` 验证 |

### QS-02: id_token 签名验证
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 攻击者 |
| 刺激 | 提交自签名的 JWT 作为 id_token（无 taskAuth 私钥签名） |
| 制品 | `validate_id_token()` |
| 环境 | 正常 |
| 响应 | 抛出 `InvalidTokenError`，callback 返回 400 |
| 响应度量 | 100% 非法签名被拒绝，单元测试 `test_oidc_id_token_bad_signature` 验证 |

### QS-03: role 参数注入防护
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 厂商用户 |
| 刺激 | 调用 `GET /api/auth/oidc/authorize/?role=admin` 企图获取 staff token |
| 制品 | `authorize` 端点 |
| 环境 | 正常 |
| 响应 | role=vendor 的 OIDC callback 后仅执行 `exchange_oidc_for_vendor`，不可签发 staff JWT |
| 响应度量 | 100% vendor 流程无法获取 staff JWT，单元测试验证 |

### QS-04: OIDC callback 响应时间
| 要素 | 内容 |
|------|------|
| 类别 | 性能 |
| 等级 | L2 |
| 刺激源 | 合法 OIDC 用户 |
| 刺激 | taskAuth 302 redirect 回 AI Provider callback（携带有效 code） |
| 制品 | `GET /api/auth/oidc/callback/` |
| 环境 | 正常负载（taskAuth 可达） |
| 响应 | 302 redirect 到前端带 `#token=...` |
| 响应度量 | 服务端处理时间 P95 ≤ 500ms（含 taskAuth token endpoint RTT） |

### QS-05: Vendor saas_user_id 原子绑定
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L2 |
| 刺激源 | 同一 Vendor 首次 OIDC 登录 |
| 刺激 | callback 处理中写入 `vendor.saas_user_id = sub` |
| 制品 | `exchange_oidc_for_vendor()` |
| 环境 | 正常 |
| 响应 | saas_user_id 写入在数据库事务内完成，并发 OIDC 登录仅一个生效 |
| 响应度量 | `select_for_update` 或 `transaction.atomic()` 包裹，单元测试验证并发安全 |

### QS-06: OIDC 登录审计日志
| 要素 | 内容 |
|------|------|
| 类别 | 可观测性 |
| 等级 | L2 |
| 刺激源 | 任何 OIDC 登录尝试 |
| 刺激 | callback 处理完成（成功或失败） |
| 制品 | `views_oidc.callback` |
| 环境 | 正常 |
| 响应 | 成功日志 `INFO` 含 trace_id/role/email/outcome=success；失败日志 `WARNING` 含 trace_id/role/outcome=failure/reason |
| 响应度量 | 每次 callback 调用产生一条结构化日志，可通过 `trace_id` 关联 |

### QS-07: taskAuth 不可达时的容错
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | taskAuth OIDC Provider |
| 刺激 | taskAuth token endpoint 超时（5s 无响应）或返回 5xx |
| 制品 | `exchange_code_for_tokens()` |
| 环境 | taskAuth 故障 |
| 响应 | HTTP 400，`{"detail": "身份认证服务暂时不可用，请稍后重试"}` |
| 响应度量 | 5s 超时后返回用户友好错误（非 500 堆栈），单元测试 `test_oidc_token_endpoint_timeout` 验证 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全性 L3（PKCE + state） | OIDC session 状态需建模 | 引入 `OidcAuthSession` 值对象（code_verifier, state, nonce），生命周期绑定 Django session |
| 数据一致性 L2（原子绑定） | saas_user_id 写入需事务保护 | `exchange_oidc_for_vendor` 使用 `transaction.atomic()` + 可能的 `select_for_update` |
| 可观测性 L2（审计日志） | 登录结果需记录为领域事件 | 新增 `VendorLoggedInViaOidc` / `StaffLoggedInViaOidc` 领域事件 |
| 容错 L2（超时） | OIDC Provider 调用需防腐层隔离 | `OidcRpClient` 基础设施服务封装 HTTP 调用 + 超时 + 错误翻译 |

## 权衡与边界

### 取舍
- 选择手写轻量 OIDC RP（PyJWT + cryptography）而非 mozilla-django-oidc，换取零新增依赖和完全控制权，代价是需自行维护 OIDC 协议细节
- 选择 email 优先匹配（非 sub 优先），支持同一 email 跨多个主站用户共用一个 Vendor 的灵活性

### 明确不做什么
- 不支持多个 OIDC Provider 同时启用（V1 仅 taskAuth）
- 不支持 OIDC 动态客户端注册（仅 bootstrap 静态配置）
- 不做 OIDC 登出（SLO / RP-Initiated Logout），登出仍走现有主站 `/logout/` 流程
- 不实现 OIDC session management（session_state / check_session_iframe）

### 升级触发条件
- 当需要接入企业 IdP（Azure AD / Keycloak）时 → 升级为多 Provider 支持，配置从静态 bootstrap 改为数据库驱动
- 当 Vendor 日活 > 1000 时 → 性能从 L2 升级到 L3，引入 token 缓存避免每次 callback 都调 taskAuth userinfo endpoint
- 当需要合规审计（SOC2）时 → 可观测性从 L2 升级到 L3，登录事件写入持久化审计日志表

## 跳过声明
- **可伸缩性**: 跳过。登录为低频操作，日均 < 10 次 OIDC 登录，单实例 Django 完全满足。
- **合规与隐私**: 跳过。无 GDPR/SOC2/HIPAA 要求。
- **可用性**: L1 基础保证。taskAuth 故障时 OIDC 不可用，用户可回退到 SSO bridge，无需专项高可用架构。
