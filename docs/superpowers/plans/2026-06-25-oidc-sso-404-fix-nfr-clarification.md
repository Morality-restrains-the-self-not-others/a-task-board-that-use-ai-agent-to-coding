# NFR 澄清: taskAuth SSO 登录 404 修复

> 输入:
> - 设计文档: `docs/specs/oidc-sso-404-fix-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-25-oidc-sso-404-fix-value-stream.md`
>
> 输出使用者: `/5-ddd-领域设计驱动`, `/6-plans-实施计划`, `/7-build-构建`

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L3 | OIDC 重定向必须防 open-redirect；issuer 变更不破坏 token 签名验证 |
| 可用性 | L2 | issuer URL 变更后 GitLab 容器内 OIDC discovery 仍可达（30s 内恢复） |
| 可维护性 | L2 | SSL fix 容器重建后自动恢复；配置变更向后兼容 |
| 可观测性 | L1 | 无新增指标，复用现有 OIDC 端点日志 |
| 性能 | L0 | 不适用——此变更为重定向 URL 修正，不改变请求路径长度或延迟 |
| 可伸缩性 | L0 | 不适用——单实例部署，无伸缩需求 |
| 数据一致性 | L0 | 不适用——不引入新数据流或事务 |
| 容错机制 | L0 | 不适用——无新外部依赖 |
| 合规与隐私 | L0 | 不适用——不涉及数据本地化或合规标准变更 |

## 逐增量 NFR 分析

### Increment 1: oidc-authorize-login-redirect-fix

**核心变更**: `handleOidcAuthorize` 未认证重定向从 `/login/` → `/auth/login/`；网关新增路由。

#### NFR 类别: 安全性
- **等级**: L3 - 增强（auth 域默认 L3）
- **量化目标**: 
  - OIDC authorize 重定向的 `next` 参数必须经过 URL 校验，防止 open-redirect 攻击
  - redirect_uri 白名单校验不变（已有实现，回归验证）
  - 新增网关路由 `/auth/login` 必须仅允许 GET，auth_mode: none 仅此路由
- **质量场景**: QS-01

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: 网关路由变更后 taskAuth 健康检查正常；GitLab 容器内 OIDC discovery 在 reconfigure 后 30s 内恢复
- **质量场景**: QS-02

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: 配置变更（issuer URL）通过 `conf/auth/task-auth/config.yaml` 集中管理；`gitService/run.sh` 动态推导，无需手动同步
- **质量场景**: QS-03

### Increment 2: oidc-issuer-unify-gateway

**核心变更**: issuer 从 `:8003` → `:18081`（网关入口统一）。

#### NFR 类别: 安全性
- **等级**: L3 - 增强
- **量化目标**: 
  - 网关 OIDC 路由（oidc-authorize, oidc-token, oidc-userinfo, oidc-jwks）的 auth_mode: none 保留不变
  - `/api/oidc/token` 仍需 client_secret 认证（POST 端点），issuer 变更不绕过 token 端点鉴权
  - JWKS 公钥不变（仅 issuer URL 变化，签名密钥不变）
- **质量场景**: QS-04

#### NFR 类别: 可用性
- **等级**: L2 - 标准
- **量化目标**: OIDC discovery 通过网关端口 18081 可达；GitLab 容器内 `curl http://183.250.1.132:18081/.well-known/openid-configuration` → 200
- **质量场景**: QS-05

### Increment 3: oidc-ssl-fix-persistence + playwright-e2e

**核心变更**: SSL fix entrypoint 持久化 + Playwright 端到端测试。

#### NFR 类别: 可维护性
- **等级**: L2 - 标准
- **量化目标**: GitLab 容器重建后（`docker compose down && up`），SSL fix 在 5 分钟内自动生效，无需手动执行脚本
- **质量场景**: QS-06

## 质量场景

### QS-01: OIDC Redirect 防 Open-Redirect

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 恶意用户 |
| 刺激 | 构造 OIDC authorize 请求，将 redirect_uri 指向外部恶意站点（如 `http://evil.com`） |
| 制品 | `handleOidcAuthorize` (taskAuth/src/oidc_handlers.go) |
| 环境 | 正常 |
| 响应 | 返回 400 `redirect_uri not allowed`，不执行 302 重定向 |
| 响应度量 | redirect_uri 不在白名单时 HTTP 400；`next` 参数（登录重定向）必须 restrict to 同站 URL |

### QS-02: 未认证用户重定向到登录页（非 404）

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 未登录的浏览器用户 |
| 刺激 | 点击 GitLab「taskAuth SSO」按钮 |
| 制品 | OIDC authorize → 网关 `/auth/login` 路由 → 前端登录页 |
| 环境 | 正常 |
| 响应 | 浏览器最终显示登录页面（`/auth/login/?next=...`），状态码 200 |
| 响应度量 | 任何中间重定向 URL 不返回 404；最终页面是登录表单 |

### QS-03: Issuer 配置变更不需要手动同步

| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 开发者修改 `conf/auth/task-auth/config.yaml` 的 `oidc.issuer` |
| 刺激 | 重启 taskAuth + GitLab |
| 制品 | taskAuth issuerURL() + gitService run.sh GITLAB_OIDC_ISSUER |
| 环境 | 部署环境 |
| 响应 | GitLab 自动使用新 issuer 进行 OIDC discovery；OIDC 流程正常 |
| 响应度量 | 无需手动修改 `gitService/run.sh` 或 docker-compose 环境变量 |

### QS-04: Token 端点安全不受 Issuer 变更影响

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | GitLab OmniAuth（合法 OIDC client） |
| 刺激 | POST `/api/oidc/token` 携带 client_id + client_secret（basic auth）|
| 制品 | `handleOidcToken` (taskAuth) |
| 环境 | 正常 |
| 响应 | 返回 access_token + id_token（200），token 中的 `iss` claim 为新网关地址 |
| 响应度量 | id_token 的 `iss` = `http://183.250.1.132:18081`；GitLab 验证 token 通过；JWKS 公钥不变 |

### QS-05: GitLab 容器内 OIDC Discovery 通过网关可达

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | GitLab 容器（Docker 网络内） |
| 刺激 | `curl http://183.250.1.132:18081/.well-known/openid-configuration`（在 gitlab 容器内执行）|
| 制品 | 网关 OIDC discovery 路由 → taskAuth |
| 环境 | 正常 |
| 响应 | HTTP 200，JSON 包含 issuer, authorization_endpoint 等字段 |
| 响应度量 | 响应时间 < 5s；`authorization_endpoint` = `http://183.250.1.132:18081/api/oidc/authorize` |

### QS-06: SSL Fix 容器重建后自动恢复

| 要素 | 内容 |
|------|------|
| 类别 | 可维护性 |
| 等级 | L2 |
| 刺激源 | 运维操作 `docker compose down && docker compose up -d` |
| 刺激 | GitLab 容器重建 |
| 制品 | `SWD.url_builder = URI::HTTP` initializer |
| 环境 | 重建后 5 分钟内 |
| 响应 | OIDC discovery 使用 HTTP（非 HTTPS）成功 |
| 响应度量 | `docker exec gitlab gitlab-rails runner "require 'swd'; puts SWD.url_builder"` 输出 `URI::HTTP` |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| 安全性 L3: OIDC redirect_uri 白名单校验 | OidcClient 聚合必须保护 redirect_uris 集合不变性 | 聚合根提供 `isRedirectUriAllowed(uri)` 方法，封装白名单逻辑 |
| 安全性 L3: issuer 变更不影响 token 签名 | 签名密钥独立于 issuer URL，是 OidcProvider 的值对象 | JwksKey 作为值对象不变；issuer 仅是配置属性 |
| 可用性 L2: OIDC 流经网关 | 网关路由是基础设施关注点，不影响领域模型 | DDD 步骤仅建模核心 OIDC 实体（OidcClient, AuthorizationCode），不建模网关路由 |

## 权衡与边界

### 取舍
- 选择将 issuer 统一到网关端口（18081）而非保持直连端口（8003），增加一跳网络延迟但统一了入口和安全策略
- SSL fix 持久化选择 entrypoint 注入（而非修改 GitLab 源码），以运维复杂度换取不 fork GitLab CE

### 明确不做什么
- 不修改 GitLab CE 源码（不 fork），所有修复在容器初始化层完成
- 不引入 OIDC 动态客户端注册（Dynamic Client Registration），保持静态 bootstrap client
- 不在 V1 支持 HTTPS issuer（仅 HTTP），生产环境需通过反向代理提供 TLS
- 不做 `/auth/login` 路由的完整前端 SSR；复用前端 Vue SPA 的客户端渲染登录页

### 升级触发条件
- 当需要 HTTPS issuer 时 → 从 L2 可用性升级到 L3（需证书管理 + 网关 TLS 配置）
- 当支持多 GitLab 实例时 → OidcClient 从单例扩展为多租户模型
- 当 OIDC SSO 成为核心卖点时 → Playwright E2E 测试从回归测试升级为 CI 门禁

## 跳过声明
- **性能**: 跳过。此变更为重定向 URL 修正，不改变请求路径长度、处理逻辑或延迟特征。
- **可伸缩性**: 跳过。单实例部署，无伸缩需求。
- **数据一致性**: 跳过。不引入新数据流或事务边界。
- **容错机制**: 跳过。无新外部依赖；已有 OIDC 重试逻辑不变。
- **合规与隐私**: 跳过。不涉及数据本地化或合规标准变更。
