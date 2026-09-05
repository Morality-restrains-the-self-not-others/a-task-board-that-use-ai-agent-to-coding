# DDD 领域模型: taskAuth SSO 登录 404 修复

> 输入:
> - 设计文档: `docs/specs/oidc-sso-404-fix-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-25-oidc-sso-404-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-25-oidc-sso-404-fix-nfr-clarification.md`
>
> 结论: **无新增领域模型** — 本变更为纯配置/基础设施修复
>
> **审查修正 (2026-06-25):** `PlaywrightE2ETest` 从 `domain/entities/` 移至 `playwright/test_result.go`（测试结果容器非领域实体）

## 变更特征判定

| 变更项 | 类型 | 影响层 |
|--------|------|--------|
| OIDC authorize 重定向 URL `/login/` → `/auth/login/` | 基础设施修复 | handler（应用层） |
| OIDC issuer `:8003` → `:18081` | 配置变更 | config.yaml |
| 网关新增 `/auth/login` 路由 | 基础设施 | APISIX routes |
| SSL fix 容器 entrypoint 持久化 | 基础设施 | Docker / shell |
| Playwright E2E 测试 | 测试 | tests/ |

所有变更均在**应用层（handler）、基础设施层（gateway、docker）或配置层**，不触碰领域逻辑。

## 现有领域模型（不变）

本次修复依赖的现有领域模型如下，**无变更**：

### 限界上下文: OIDC Provider (taskAuth)

#### 聚合: OidcClient
- **聚合根**: `OidcClient` — OIDC 客户端注册信息
- **属性**: `client_id`, `client_secret_hash`, `name`, `redirect_uris` (JSON array)
- **不变量**: `redirect_uris` 必须至少包含一个合法 URI；`client_id` 全局唯一
- **行为**: `isRedirectUriAllowed(uri) -> bool`

#### 实体: AuthorizationCode
- **标识**: `code` (随机 hex 字符串)
- **属性**: `client_id`, `user_id`, `redirect_uri`, `scope`, `nonce`, `expires_at`
- **生命周期**: 创建 → 使用 (consumed) / 过期
- **行为**: `consume()`, `isExpired() -> bool`

#### 值对象: JwksKey
- **不可变**: RSA 公私钥对
- **行为**: `sign(claims) -> JWT`, `verify(token) -> claims`

#### 领域事件
- **OidcAuthorizationCodeIssued**: `{ code, client_id, user_id, redirect_uri }` — 授权码生成
- **OidcTokenIssued**: `{ user_id, client_id, scopes }` — Token 签发成功

#### 仓储接口

```go
// OidcClientRepository (已在 taskAuth/src/oidc_db.go 实现)
type OidcClientRepository interface {
    FindByClientID(clientID string) (*OidcClient, error)
    Save(client *OidcClient) error
}

// AuthorizationCodeRepository (已在 taskAuth/src/oidc_db.go 实现)
type AuthorizationCodeRepository interface {
    Store(code *AuthorizationCode) error
    Consume(code, clientID, redirectURI string) (*AuthorizationCode, error)
}
```

### 限界上下文: GitLab OmniAuth Consumer (gitService)

GitLab 作为 OIDC Relying Party，通过 OmniAuth `openid_connect` 策略消费 taskAuth 的 OIDC Provider。
不在本项目领域模型中——GitLab CE 是外部系统。

## NFR 驱动的领域模型影响（确认无需变更）

| NFR 决策 | 预期模型影响 | 实际判定 |
|----------|-------------|---------|
| 安全性 L3: redirect_uri 白名单 | OidcClient.isRedirectUriAllowed() | ✅ 已存在，不变 |
| 安全性 L3: issuer 变更不影响签名 | JwksKey 独立于 issuer | ✅ 已设计为此，不变 |
| 可用性 L2: 网关路由 | 基础设施关注点 | ✅ 不影响领域模型 |

## 自检

- [x] 无新增领域模型文件（变更不涉及领域概念）
- [x] 现有领域层无基础设施依赖（已通过之前的代码审查验证）
- [x] 现有聚合边界不变
- [x] 现有仓储接口不变
- [x] 本次变更限定在应用层/基础设施层/配置层

## 结论

**DDD 步骤无需产生新文件。** 直接进入实施计划（`/6-plans-实施计划`）。
