# NFR 澄清: SSO 401 Fix — Forward-Auth Cookie Bridge

> 输入:
> - 设计文档: `docs/design/sso-401-fix-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-29-sso-401-forward-auth-cookie-bridge-value-stream.md`
>
> 输出使用者: `/6-ddd-领域设计驱动`, `/7-plans-实施计划`, `/8-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L3 | userId cookie 验证用户存在且活跃；不削弱现有 Token auth |
| 可用性 | L2 | SSO 跳转可用，cookie 回退不增加延迟 |
| 性能 | L1 | 无额外开销——复用已有函数，仅增加一次 cookie 读取 |
| 数据一致性 | L0 | 不适用——无数据写入 |
| 可伸缩性 | L0 | 不适用——无新增资源消耗 |
| 可观测性 | L1 | 基础——现有日志/链路追踪覆盖 |
| 容错机制 | L2 | Token 不可用时回退至 cookie，不新增故障模式 |
| 合规与隐私 | L0 | 不适用 |
| 可维护性 | L1 | 代码行数减少（2行→1行），复用已有函数 |

## 逐增量 NFR 分析

### Increment 1: Forward-Auth userId Cookie Fallback

#### NFR 类别: 安全性
- **等级**: L3 - 增强（认证域）
- **量化目标**: 
  - userId cookie 验证用户存在且 is_active=true 后才通过认证
  - 不创建新 token，不绕过现有权限检查
  - `loadUserAuthFlags` 仍校验 is_superuser/is_staff
- **质量场景**: QS-01

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: SSO 跳转成功率 = Token auth 成功率（cookie 回退仅当 Token 不可用时触发）
- **质量场景**: QS-02

#### NFR 类别: 容错机制
- **等级**: L2 - 标准
- **量化目标**: Token header 路径不受影响；cookie 路径仅作为 fallback，失败不阻塞其他认证方式
- **质量场景**: QS-03

## 质量场景

### QS-01: userId Cookie 认证安全验证
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 已登录用户浏览器（仅携带 userId cookie，无 Token header） |
| 刺激 | GET /accounts/sso/ai-provider/admin/ |
| 制品 | taskAuth forward-auth handler |
| 环境 | 正常 |
| 响应 | 验证 userId 对应用户存在且 is_active=true → 200 + X-User-Id；用户不存在/未激活 → 401 |
| 响应度量 | 单元测试覆盖：valid userId→200, invalid userId→401, inactive user→401, empty cookie→401 |

### QS-02: Token Auth 优先级不受影响
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | API 客户端（携带 Authorization: Token xxx） |
| 刺激 | 任意 auth_mode: token 的 API 请求 |
| 制品 | taskAuth forward-auth handler |
| 环境 | 正常 |
| 响应 | Token header 优先认证，行为与前一致 |
| 响应度量 | 现有 Token auth 测试全部通过，无回归 |

### QS-03: 多层回退容错
| 要素 | 内容 |
|------|------|
| 类别 | 容错机制 |
| 等级 | L2 |
| 刺激源 | 浏览器（无 Token header，userId cookie 指向不存在用户） |
| 刺激 | GET /accounts/sso/ai-provider/admin/ |
| 制品 | taskAuth forward-auth handler |
| 环境 | 正常 |
| 响应 | 依次尝试 4 层认证，全部失败返回 401 |
| 响应度量 | 单元测试：all methods fail→401，任一成功→200 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全性 L3: userId cookie 需验证用户活跃性 | forward-auth 领域服务增加 `resolveTokenUserIDFromRequest` 作为统一入口 | 无需新增实体/值对象——复用已有认证链 |
| 容错 L2: 多层回退不可互相阻塞 | 认证方法链为责任链模式，每个方法独立失败 | 已在 `resolveTokenUserIDFromRequest` 中实现，本次仅改调用方 |

## 权衡与边界

### 取舍
- **选择复用 `resolveTokenUserIDFromRequest`** 而非在 forward-auth 中重写 cookie 逻辑——接受该函数中的 Bearer JWT 验证开销（当前 forward-auth 不触发，因 Bearer tokens 由 OIDC 路径单独处理）

### 明确不做什么
- 不在 forward-auth 中新增独立的 cookie 验证逻辑
- 不修改 APISIX 路由配置（保持 `django-default` 的 `auth_mode: token`）
- 不新增数据库表或配置项

### 升级触发条件
- 当 userId cookie 被滥用（如跨站伪造）→ 升级至 L4，增加 cookie signing/CSRF token
- 当 cookie 回退路径延迟 > 5ms → 性能升级至 L2，分析 cookie 读取开销

## 跳过声明
- **性能**: L1 即可。改动为函数调用替换（2 行→1 行），cpu profile 无差异。
- **可伸缩性**: 不适用。单文件修改，无新增资源。
- **数据一致性**: 不适用。纯读操作，无写入。
- **合规与隐私**: 不适用。userId 已在 cookie 中传输，本修复不改变其传输方式。
