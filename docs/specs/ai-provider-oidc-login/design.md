# AI Provider OIDC 登录 — 设计文档

## 背景与动机

### 现状

AI Provider (saas-ai-provider, port 8010) 目前仅支持一种登录方式：

```
主站(4000) 登录 → 点击「厂商门户(SSO)」 → JWT bridge → AI Provider(8010) 换票
```

- `POST /api/vendor/auth/login/` → **403** (`code: sso_only`)
- `POST /api/admin/auth/login/` → **403** (`code: sso_only`)
- 本地密码登录已完全关闭，厂商注册入口也已关闭

### 问题

1. 用户**必须**先从主站登录才能进入 AI Provider，无法独立登录
2. 没有主站账号的外部合作厂商无法使用（虽然当前设计意图如此，但限制了灵活性）
3. 与标准 OIDC 协议不兼容，无法接入企业已有的 Identity Provider

### 目标

为 AI Provider 的 **Vendor（厂商）** 和 **Admin（PlatformStaff）** 账号增加 OIDC 登录支持，与现有 SSO bridge 并存。

## 现有基础设施

### taskAuth OIDC Provider

taskAuth (port 8003) 已经是一个功能完备的 OIDC Provider：

```yaml
# conf/auth/task-auth/config.yaml
oidc:
  signingKeyPath: ""                    # 留空自动生成临时 RSA 密钥
  accessTokenTTL: 3600
  idTokenTTL: 3600
  issuer: "${subdomains.gateway}"       # 当前: http://183.250.1.132:18081
  bootstrapClientId: "gitlab-git-service"
  bootstrapClientSecret: "gsoidc-dev-secret-do-not-use-in-prod"
  bootstrapRedirectUri: "http://${subdomains.gitlab}/users/auth/openid_connect/callback"
```

taskAuth 已为 GitLab 提供 OIDC SSO，可以直接新增一个 bootstrap client 给 AI Provider。

### AI Provider 认证体系

| 组件 | 路径 | 角色 |
|------|------|------|
| `authentication.py` | `VendorBearerAuthentication`, `StaffBearerAuthentication` | 从 Bearer JWT 中解析 Vendor/Staff |
| `jwt_utils.py` | `issue_token()`, `decode_token()` | JWT 签发/校验（`typ: vendor/staff`） |
| `sso_bridge_logic.py` | `exchange_vendor_bridge()`, `exchange_staff_bridge()` | SSO bridge 换票 + Vendor/Staff 查找或创建 |
| `views_sso.py` | `POST /api/auth/sso/exchange/` | SSO 换票 API |
| `urls.py` | 路由注册 | |

## 设计方案

### 整体架构

```
┌──────────────────────────────────────────────────────────┐
│                    OIDC 登录流程                           │
│                                                            │
│  Browser ─→ GET /api/auth/oidc/authorize/?role=vendor     │
│              │                                             │
│              ↓ 302 redirect                                │
│  Browser ─→ taskAuth /oauth2/authorize?...                │
│              │                                             │
│              ↓ 用户认证（taskAuth 内部逻辑）                  │
│  Browser ←─ 302 redirect with ?code=...                   │
│              │                                             │
│              ↓                                             │
│  Browser ─→ GET /api/auth/oidc/callback/?code=...&state=..│
│              │                                             │
│              ↓ AI Provider 后端：                            │
│              │  1. POST /oauth2/token (code → id_token)    │
│              │  2. 验证 id_token (iss/aud/exp/signature)    │
│              │  3. 提取 email claim                         │
│              │  4. 匹配 Vendor/PlatformStaff by email       │
│              │  5. 签发 AI Provider JWT (vendor/staff)      │
│              │                                             │
│  Browser ←─ 302 redirect with #token=<ai_provider_jwt>    │
└──────────────────────────────────────────────────────────┘
```

### 与 SSO Bridge 的对比

| 维度 | SSO Bridge (现有) | OIDC (新增) |
|------|-------------------|-------------|
| 触发方式 | 主站签发短期 JWT → URL fragment | 标准 OIDC Authorization Code Flow |
| 认证方 | 主站 Django Session | taskAuth OIDC Provider (可扩展) |
| 身份传递 | 自定义 bridge JWT | 标准 id_token (JWT) |
| 匹配字段 | `sub` (user_id) + `email` | `email` claim |
| Vendor 查找 | `saas_user_id` 优先 → `email` 回退 | `email` 优先 → `saas_user_id` 回退 |
| 安全性 | HMAC 共享密钥 | OIDC 标准 (PKCE + state) |

### Vendor/Staff 匹配逻辑

OIDC 登录的匹配逻辑与 SSO bridge 类似但更简单：

```python
def exchange_oidc_for_vendor(id_token_claims: dict) -> str:
    email = (id_token_claims.get("email") or "").strip().lower()
    sub = id_token_claims.get("sub")  # taskAuth user_id

    # Step 1: 按 email 查找 Vendor
    vendor = Vendor.objects.filter(email__iexact=email, is_active=True).first()

    if not vendor:
        # Step 2: 按 saas_user_id 回退（OIDC sub = main site user id）
        vendor = Vendor.objects.filter(saas_user_id=int(sub), is_active=True).first()

    if not vendor:
        raise ValueError("未找到与 OIDC 账号匹配的厂商账号")

    # 首次 OIDC 登录：绑定 saas_user_id
    if vendor.saas_user_id is None:
        vendor.saas_user_id = int(sub)
        vendor.save(update_fields=["saas_user_id", "updated_at"])

    return issue_token(str(vendor.id), "vendor")
```

**注意**：OIDC 登录**不会**自动创建 Vendor（与 Staff 的 `exchange_staff_bridge` 不同）。Vendor 仍需由运营在 Django Admin 预创建。

### Admin (PlatformStaff) 匹配逻辑

```python
def exchange_oidc_for_staff(id_token_claims: dict) -> str:
    sub = int(id_token_claims.get("sub"))

    # 按 saas_superadmin_id 查找或创建
    staff = PlatformStaff.objects.filter(saas_superadmin_id=sub).first()
    if staff:
        return issue_token(str(staff.id), "staff")

    # 仅限 taskAuth 中标记为 superuser 的用户自动创建 Staff
    if not id_token_claims.get("is_superuser"):
        raise ValueError("非超级管理员，无法访问管理端")

    # 自动创建 Staff 记录
    staff = PlatformStaff(
        username=f"saas_{sub}"[:64],
        display_name=id_token_claims.get("preferred_username", "")[:100],
        saas_superadmin_id=sub,
    )
    staff.set_password(secrets.token_urlsafe(48))
    staff.save()
    return issue_token(str(staff.id), "staff")
```

### 配置扩展

新增 AI Provider OIDC 客户端配置：

```yaml
# conf/auth/task-auth/config.yaml 新增
oidc:
  # ... 现有配置保持不变 ...
  bootstrapClients:
    - clientId: "gitlab-git-service"
      clientSecret: "gsoidc-dev-secret-do-not-use-in-prod"
      redirectUri: "http://${subdomains.gitlab}/users/auth/openid_connect/callback"
    - clientId: "ai-provider"                          # 新增
      clientSecret: "aip-oidc-dev-secret"              # 新增
      redirectUri: "http://${subdomains.provider}/api/auth/oidc/callback/"  # 新增
```

AI Provider 新增配置：

```python
# provider/settings.py 新增
OIDC_RP_CLIENT_ID = "ai-provider"
OIDC_RP_CLIENT_SECRET = os.environ.get("OIDC_RP_CLIENT_SECRET", "aip-oidc-dev-secret")
OIDC_RP_ISSUER = "http://183.250.1.132:18081"        # taskAuth OIDC issuer
OIDC_RP_AUTHORIZE_ENDPOINT = f"{OIDC_RP_ISSUER}/oauth2/authorize"
OIDC_RP_TOKEN_ENDPOINT = f"{OIDC_RP_ISSUER}/oauth2/token"
OIDC_RP_USERINFO_ENDPOINT = f"{OIDC_RP_ISSUER}/oauth2/userinfo"
OIDC_RP_SCOPES = ["openid", "email", "profile"]
```

## 实施计划

### Increment 1: 后端 OIDC RP 核心 (MVP)

**文件变更**:

| 文件 | 操作 | 说明 |
|------|------|------|
| `apps/marketplace/oidc_rp.py` | **新建** | OIDC RP 核心逻辑：authorize URL 构建、token 交换、id_token 验证 |
| `apps/marketplace/views_oidc.py` | **新建** | OIDC 端点：authorize 重定向、callback 处理 |
| `apps/marketplace/urls.py` | 修改 | 添加 OIDC 路由 |
| `provider/settings.py` | 修改 | 添加 OIDC RP 配置 |

**API 端点**:

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/auth/oidc/authorize/` | GET | `?role=vendor\|admin` → 302 到 taskAuth OIDC |
| `/api/auth/oidc/callback/` | GET | OIDC callback → 换票 → 302 回前端带 token |

**关键实现**:
- 使用 PyJWT + `cryptography` 验证 id_token RSA 签名（从 taskAuth 获取 JWKS）
- PKCE (S256) 防授权码拦截
- `state` 参数防 CSRF（Django session 中暂存）

### Increment 2: 前端 OIDC 登录入口

**文件变更**:

| 文件 | 操作 | 说明 |
|------|------|------|
| `frontend/src/views/VendorPortal.vue` | 修改 | 添加「OIDC 登录」按钮 + callback 处理 |
| `frontend/src/views/AdminPortal.vue` | 修改 | 添加「OIDC 登录」按钮 + callback 处理 |

**UI 变化**:
- 在 SSO 链接下方增加「或通过 OIDC 账号登录」按钮
- 登录区域从单一 SSO 链接变为双选项

### Increment 3: taskAuth OIDC Client 注册 + 测试

**文件变更**:

| 文件 | 操作 | 说明 |
|------|------|------|
| `conf/auth/task-auth/config.yaml` | 修改 | 添加 ai-provider bootstrap client |
| `playwright/saas_ai_provider/tests/django8010-oidc-login.playwright.test.js` | **新建** | E2E 测试 |

### Increment 4 (可选): 通用化 OIDC Provider 配置

- 支持通过环境变量配置任意 OIDC Provider（不限于 taskAuth）
- 支持多个 OIDC Provider 同时启用（如企业 Azure AD / Keycloak）

## Domain Concept Inventory

- **Bounded Contexts**: marketplace (AI Provider auth), task-auth (OIDC Provider)
- **Key Entities**: `Vendor`, `PlatformStaff`, OIDC `id_token` claims
- **Domain Events**: `VendorBoundToOidcIdentity`（首次 OIDC 绑定时 saas_user_id 写入）
- **Cross-context**: taskAuth OIDC Provider ↔ AI Provider OIDC RP（标准协议，松耦合）

## 安全考量

| 风险 | 缓解措施 |
|------|----------|
| 授权码拦截 | PKCE S256 |
| CSRF | OIDC `state` 参数 + Django session 校验 |
| id_token 伪造 | 验证 RSA 签名 + `iss`/`aud`/`exp` |
| 重定向到恶意站点 | 白名单 redirect_uri 校验 |
| email 不可信（自注册 IdP） | 仅 taskAuth（内部可控）签发时信任 email claim |
| session 固定 | OIDC callback 后轮换 Django session key |

## 设计决策

| 决策 | 选择 | 原因 |
|------|------|------|
| OIDC Provider | taskAuth | 已有 OIDC Provider 基础设施，用户身份在 taskAuth 中 |
| OIDC 库 | 手写轻量 RP（PyJWT + cryptography） | 避免引入重量级依赖（mozilla-django-oidc），当前规模不需要 |
| 是否保留 SSO bridge | ✅ 保留，双轨并行 | SSO bridge 与 OIDC 互补，不互相替代 |
| Vendor 自动创建 | ❌ 不自动创建 | 安全：厂商账号需运营预创建，与现有设计一致 |
| Staff 自动创建 | ✅ 自动创建（仅限 superuser） | 与现有 exchange_staff_bridge 行为一致 |
| PKCE | ✅ 强制 | 单页应用无 client_secret 安全存储，PKCE 是标准做法 |

## 总结清单

- **OIDC Provider**: 使用现有 taskAuth OIDC Provider，新增 `ai-provider` client
- **后端**: 新增 `oidc_rp.py` + `views_oidc.py`，约 200 行
- **前端**: VendorPortal/AdminPortal 各增加 OIDC 登录按钮 + callback fragment 处理
- **匹配逻辑**: email 优先 → saas_user_id 回退，Vendor 不自动创建
- **SSO bridge**: 保留不变，与 OIDC 并存
- **安全**: PKCE + state + JWKS 签名验证
