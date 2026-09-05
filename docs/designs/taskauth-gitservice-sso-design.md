# taskAuth → gitService SSO 单点登录 设计文档

> 状态: **草稿** | 日期: 2026-06-24 | 作者: Claude (brainstorming)

## 1. 问题陈述

**用户需求**: 用户通过 taskAuth 注册后，能否通过 SSO（单点登录）直接访问 gitService（自托管 GitLab CE），无需在 GitLab 端再次注册/登录？

**当前状态**: taskAuth 管理用户身份（auth.db SQLite），gitOauth 管理 Git Provider 的 OAuth Token（用于 API 访问如 clone/MR），gitService 是独立的 GitLab CE 实例，用户需要单独注册 GitLab 账号才能登录其 Web UI。

**目标**: 用户只需在 taskAuth 注册一次，即可 SSO 登录 gitService 的 Web 界面。

---

## 2. 当前架构分析

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   taskAuth   │────▶│   task2app   │     │   gitOauth   │
│  (Go :8003)  │     │ (Django :8001)│    │ (Django :8002)│
│              │     │              │     │              │
│ auth.db      │     │ default DB   │     │ default DB   │
│ - users      │     │ - companies  │     │ - credentials│
│ - login_mtd  │     │ - projects   │     │ - audit      │
│ - tokens     │     │ - workspaces │     │              │
└──────────────┘     └──────────────┘     └──────────────┘
       │                                        │
       │  身份管理（独立）                         │  OAuth Token 管理
       │                                        │  (GitHub/GitLab API 调用)
       ▼                                        ▼
┌──────────────────────────────────────────────────────┐
│                   gitService                         │
│              (GitLab CE Docker :8012)                │
│                                                      │
│  - 自托管 Git 仓库                                    │
│  - Web UI 独立认证（内置用户表）                        │
│  - 已有 OAuth App 注册（供 gitOauth 使用）             │
└──────────────────────────────────────────────────────┘
```

**关键发现**:
- taskAuth 已经是平台的身份真源（Single Source of Truth）
- gitOauth 已与 gitService 建立 OAuth App 关系（`conf/auth/git-oauth/providers/http-localhost-8012.yaml`）
- 但 gitOauth 管理的是 *API 访问 Token*（用于 clone/push/MR），不是 Web 登录
- gitService 的 Web UI 登录目前与 taskAuth 完全独立

---

## 3. 方案选择

### 方案 A: taskAuth 作为 OIDC Provider（推荐 ✅）

**思路**: taskAuth 实现 OpenID Connect (OIDC) Provider 协议，gitService (GitLab CE) 通过 OmniAuth OIDC 信任 taskAuth。

```
用户注册(taskAuth) ──▶ 身份存在于 taskAuth
                            │
用户访问 gitService ──▶ GitLab OmniAuth OIDC ──▶ 重定向到 taskAuth 登录
                            │
taskAuth 签发 id_token ◀── 用户已在 taskAuth 登录态
                            │
gitService 验证 id_token ──▶ 自动创建/关联 GitLab 用户 ──▶ 登录成功
```

| 维度 | 评价 |
|------|------|
| 标准化 | OIDC 是行业标准，GitLab 原生支持 |
| 安全性 | 基于 JWT + 签名密钥，企业级 |
| 复杂度 | taskAuth 需新增 OIDC Provider 模块（中等） |
| 用户体验 | 真 SSO，一次登录访问所有服务 |
| 可扩展性 | 未来其他服务也可接入 taskAuth OIDC |

### 方案 B: 注册时自动预配 GitLab 用户

**思路**: taskAuth 注册成功后，通过 GitLab Users API 自动创建 GitLab 用户，密码通过某种方式传递给用户。

| 维度 | 评价 |
|------|------|
| 复杂度 | 简单，但密码管理混乱 |
| 安全性 | 差 — 密码存储/传输有风险 |
| 用户体验 | 用户仍需记住/管理两套凭证 |
| 结论 | ❌ 不推荐 |

### 方案 C: 注册时自动绑定 GitLab OAuth

**思路**: taskAuth 注册后自动触发 gitOauth 的 GitLab OAuth 流程，预绑定 API Token。

| 维度 | 评价 |
|------|------|
| 现有设施 | 复用 gitOauth，改动小 |
| 覆盖范围 | 仅解决 API 访问，不解决 Web UI 登录 |
| 结论 | ❌ 无法满足 SSO Web 登录需求 |

### 推荐结论: **方案 A — taskAuth OIDC Provider**

---

## 4. 详细设计

### 4.1 新增组件: taskAuth OIDC Provider

在 taskAuth (Go) 中新增 OIDC Provider 模块，实现以下端点：

```
GET  /.well-known/openid-configuration    # OIDC Discovery
GET  /api/oidc/authorize                   # 授权端点（浏览器重定向）
POST /api/oidc/token                       # Token 端点（code → token）
GET  /api/oidc/userinfo                    # UserInfo 端点（用户声明）
GET  /api/oidc/jwks                        # JWKS 端点（公钥）
```

### 4.2 新增数据模型

**OIDC Client 表** (`auth.db`):

```sql
CREATE TABLE oidc_client (
    id            TEXT PRIMARY KEY,   -- 雪花 ID
    client_id     TEXT UNIQUE NOT NULL,
    client_secret TEXT NOT NULL,      -- bcrypt 哈希
    name          TEXT NOT NULL,      -- e.g. "gitService"
    redirect_uris TEXT NOT NULL,      -- JSON array of allowed redirect URIs
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);
```

**OIDC 授权码表** (`auth.db`):

```sql
CREATE TABLE oidc_authorization (
    id            TEXT PRIMARY KEY,
    code          TEXT UNIQUE NOT NULL,   -- 一次性授权码
    client_id     TEXT NOT NULL,
    user_id       TEXT NOT NULL,          -- taskAuth user object_id
    redirect_uri  TEXT NOT NULL,
    scope         TEXT NOT NULL,
    nonce         TEXT,
    expires_at    TEXT NOT NULL,
    used          INTEGER DEFAULT 0,
    created_at    TEXT NOT NULL
);
```

### 4.3 新增配置

`conf/auth/task-auth/config.yaml` 新增:

```yaml
oidc:
  issuer: "https://127.0.0.1:8443"  # 或生产域名，通过 gateway 暴露
  accessTokenTTL: 3600               # 1 小时
  idTokenTTL: 3600
  signingKeyPath: ""                 # RSA 私钥路径，留空自动生成
```

### 4.4 OIDC 流程

```
1. 用户访问 gitService (http://127.0.0.1:8012)
2. GitLab 登录页显示 "使用 taskAuth 登录" 按钮
3. 用户点击 → 浏览器重定向到:
   GET /api/oidc/authorize?
       response_type=code&
       client_id=gitlab-git-service&
       redirect_uri=http://127.0.0.1:8012/users/auth/openid_connect/callback&
       scope=openid+profile+email&
       state=xxxxx&
       nonce=xxxxx

4. taskAuth 校验用户登录态:
   - 已登录 → 直接生成 authorization code → 重定向回 GitLab
   - 未登录 → 提示用户登录 taskAuth → 登录后继续

5. GitLab 后端用 code 调用:
   POST /api/oidc/token
       grant_type=authorization_code&
       code=xxxxx&
       client_id=gitlab-git-service&
       client_secret=xxxxx

6. taskAuth 返回:
   {
     "access_token": "...",
     "id_token": "...",       // JWT，含 sub, email, name 等
     "token_type": "Bearer",
     "expires_in": 3600
   }

7. GitLab 通过 GET /api/oidc/userinfo 获取用户信息
8. GitLab 自动创建/关联用户 → 登录成功
```

### 4.5 GitLab (gitService) 配置

GitLab CE 通过 `gitlab.rb` 或环境变量配置 OmniAuth OIDC:

```ruby
gitlab_rails['omniauth_enabled'] = true
gitlab_rails['omniauth_allow_single_sign_on'] = ['openid_connect']
gitlab_rails['omniauth_block_auto_created_users'] = false
gitlab_rails['omniauth_auto_link_user'] = ['openid_connect']

gitlab_rails['omniauth_providers'] = [
  {
    name: 'openid_connect',
    label: 'taskAuth SSO',
    args: {
      name: 'openid_connect',
      scope: ['openid', 'profile', 'email'],
      response_type: 'code',
      issuer: 'https://127.0.0.1:8443',
      discovery: true,
      client_auth_method: 'basic',
      uid_field: 'sub',
      client_options: {
        identifier: '${OIDC_CLIENT_ID}',
        secret: '${OIDC_CLIENT_SECRET}',
        redirect_uri: 'http://127.0.0.1:8012/users/auth/openid_connect/callback'
      }
    }
  }
]
```

### 4.6 注册时的 SSO 引导

当用户通过 taskAuth 注册后，在注册成功响应或前端引导中，提示用户可以 SSO 登录 gitService。这不需要在注册流程中增加特殊逻辑——因为 SSO 的本质就是用户已经在 taskAuth 有了登录态，访问 gitService 时可以自动完成认证。

**可选增强**: 注册成功后，前端提供 "前往 Git 服务" 按钮，点击直接跳转 gitService 并携带 SSO 参数，实现无缝跳转。

---

## 5. 领域概念清单 (Domain Concept Inventory)

| 类别 | 概念 | 说明 |
|------|------|------|
| **Bounded Context** | Auth & Identity | taskAuth 为身份管理边界，OIDC 是其对外暴露的认证协议 |
| **Bounded Context** | Git Service Integration | gitService 作为独立的 Git 托管上下文，通过 OIDC 消费身份 |
| **实体 (Entity)** | OidcClient | 注册的 OIDC 客户端（如 gitService），有 client_id/secret |
| **实体 (Entity)** | OidcAuthorization | 一次性授权码，绑定 user + client + redirect_uri |
| **实体 (Entity)** | User (现有) | taskAuth 中的用户，OIDC 的 subject |
| **聚合根** | OidcClient | 管理 OIDC 客户端注册信息 |
| **值对象** | IDToken | JWT，含标准 OIDC claims (sub, iss, aud, exp, iat, email, name) |
| **领域事件** | OidcClientRegistered | 新 OIDC 客户端注册（用于通知运维/审计） |
| **领域事件** | OidcUserAuthenticated | 用户通过 OIDC 登录了外部服务（审计日志） |

---

## 6. 价值流影响分析

基于 `conf/value-stream.yaml` 分析：

### 受影响的现有价值流

| 价值流 | 影响 |
|--------|------|
| `user-auth` | 新增 `oidc-provider` step — taskAuth 增加 OIDC Provider 能力 |
| `gitlab-oauth-scope-failfast-governance` | `gitlab-oauth-app-bootstrap` step 需扩展：除 OAuth App 外还需配置 OmniAuth OIDC |
| `gitoauth-binding-state-persistence` | 不受直接影响（gitOauth 管理 API token，OIDC 管理 Web 登录） |

### 建议新增价值流

**新价值流**: `git-service-sso`（领域: 用户与认证）

```yaml
- name: git-service-sso
  domain: 用户与认证
  description: taskAuth OIDC Provider → gitService SSO 单点登录
  steps:
    - name: oidc-provider-core
      status: planned
      fields:
        - name: task-auth.oidc_client.client_id
          description: OIDC 客户端注册标识
        - name: task-auth.oidc_client.redirect_uris
          description: 允许的回调 URI 列表
        - name: task-auth.accounts_user.id
          description: OIDC subject (user_id)
    - name: oidc-discovery-endpoint
      status: planned
      fields:
        - name: task-auth.runtime.oidc_discovery_url
          description: GET /.well-known/openid-configuration
    - name: gitservice-omniauth-config
      status: planned
      fields:
        - name: git-service.runtime.omniauth_oidc_enabled
          description: GitLab OmniAuth OIDC 配置生效
        - name: git-service.runtime.omniauth_provider
          description: openid_connect provider 注册
    - name: oidc-login-e2e
      status: planned
      fields:
        - name: task-auth.accounts_user.email
          description: OIDC id_token email claim 对应用户注册邮箱
        - name: git-service.runtime.external_user
          description: GitLab 外部用户由 OIDC 自动创建
```

### 字段影响

- **新增表**: `task-auth.oidc_client.*` — OIDC 客户端注册
- **新增表**: `task-auth.oidc_authorization.*` — 授权码管理
- **新增 runtime 字段**: `task-auth.runtime.oidc_provider_enabled`
- **修改**: `git-service.runtime.*` — OmniAuth 配置变更

### 测试影响

- **新增测试**: `tests/test_taskauth_oidc_provider.py` — OIDC Provider 单元测试
- **新增测试**: `tests/test_taskauth_oidc_gitservice_sso.py` — SSO 端到端测试
- **新增测试**: `../../taskAuth/src/oidc_provider_test.go` — Go 侧 OIDC 实现测试
- **可能影响**: `../../gitService/scripts/test_sync_oauth_app.sh` — 扩展覆盖 OmniAuth 配置同步

---

## 7. 实施策略

### Phase 1: OIDC Provider 核心 (taskAuth)

1. 在 taskAuth 中实现 OIDC Provider 协议：
   - OIDC Client 管理 API（注册/管理 GitLab 客户端）
   - /.well-known/openid-configuration 发现端点
   - /api/oidc/authorize 授权端点
   - /api/oidc/token Token 端点
   - /api/oidc/userinfo UserInfo 端点
   - /api/oidc/jwks JWKS 端点
   - RSA 密钥对生成与 JWT 签名

### Phase 2: GitLab OmniAuth 配置

2. 更新 `gitService/gitlab_home/config/gitlab.rb` 添加 OmniAuth OIDC 配置
3. 实现配置同步脚本（扩展现有 `conf/auth/git-oauth/sync.sh` 模式）
4. 在 docker-compose 中注入 OIDC_CLIENT_ID / OIDC_CLIENT_SECRET 环境变量

### Phase 3: 集成测试

5. 编写 OIDC Provider 单元测试（Go test）
6. 编写 SSO 端到端测试（Playwright/Python）
7. 验证 GitLab 用户自动创建与登录

### Phase 4: 用户体验优化（可选）

8. 注册成功页添加 "前往 Git 服务" 按钮
9. 前端处理 SSO 回调体验

---

## 8. 风险与注意事项

| 风险 | 缓解措施 |
|------|---------|
| GitLab CE OmniAuth OIDC 配置复杂 | 使用 discovery 模式，减少手动配置 |
| 用户 email 冲突（taskAuth 与 GitLab 内置用户） | OmniAuth `auto_link_user` 按 email 自动关联 |
| taskAuth 增加复杂度 | OIDC Provider 作为独立模块，不侵入现有注册/登录逻辑 |
| 密钥管理 | RSA 密钥对可配置路径，支持轮换 |
| Gateway 路由 | 需在 taskGateway routes.yaml 中暴露 OIDC 端点 |

---

## 9. 未决问题

1. **GitLab 内置 root 用户**: gitService 初始 root 用户是否保留？建议保留作为管理员备用入口
2. **用户属性映射**: taskAuth 的哪些字段映射到 GitLab 用户属性？（email → email, username → username）
3. **客户端注册方式**: OIDC Client 是通过 API 动态注册还是启动时配置文件预注册？建议配置文件预注册（与现有 conf 模式一致）
4. **是否需要同时支持 SAML**: 如果未来有更多服务需要 SSO，是否需要更通用的协议？
