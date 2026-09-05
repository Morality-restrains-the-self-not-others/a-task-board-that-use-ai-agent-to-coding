# taskAuth JWT 迁移设计 — Access JWT + Refresh Token（硬切，零兼容）

**日期：** 2026-05-31  
**状态：** 待审批  
**延续：** `2026-05-28-taskauth-increment7-auth-table-cleanup-design.md`  
**Supersedes：**
- `2026-05-28-taskauth-split-design.md`（含 Token 兼容、CustomToken、Session 认证）
- `2026-05-31-task-container-gateway-auth-design.md` 中 validate-session 身份校验部分

---

## 0. 零兼容政策（Non-Negotiable）

本方案 **禁止** 仓库内存在任何 Legacy 用户认证代码路径。下列模式 **一律不得出现**（含注释掉的 dead code、feature flag 分支、`TODO 稍后删除`）：

| 禁止项 | 说明 |
|--------|------|
| `CustomToken` / `accounts_customtoken` | 模型、表、ORM、fixture、migration state |
| `Authorization: Token …` | DRF、前端、Go 测试、Playwright |
| `localStorage.authToken` / `data.token` 登录字段 | 前端仅存 `access_token` |
| `SessionAuthentication*` / `sessionid` API 认证 | 含 `django_auth_login()` |
| `validate-session` internal | **整 endpoint 删除**，非「只删身份半段」 |
| Django auth **fallback** | delegate 失败仅 503，**无**本地 LoginMethod/Token 写入 |
| `TASKAUTH_ENABLED=false` 降级 | 删除该开关；taskAuth **硬依赖** |
| `DualAuthentication` / 双接受 Bearer+Token | 不存在 |
| 登录响应 `"token"` 字段 | 仅 `access_token` |

**命名隔离：** 容器 `CloudServerConfig.container_access_token`、Git `oauthTokens` 等 **与用户 JWT 无关**，不得复用 `CustomToken` 或 `Token` header 模式。

**验收：** merge 前 CI 运行 §14 硬切 grep，**零匹配**（白名单仅 `docs/**` 历史 spec 与 `container_access_token` 字段名）。

---

## 1. 目标

将用户认证凭证 **一次性替换** 为 taskAuth 签发的 **Access JWT + Refresh Token**：

1. **taskAuth** 独占签发、刷新、吊销、JWKS。
2. **Go 服务** JWKS 本地验签；**零** Django 身份 internal。
3. **saas-backend** **仅** `Authorization: Bearer <jwt>`。
4. **前端** **仅** Bearer + refresh httpOnly Cookie。
5. **同一 merge** 完成 T1–T4；仓库内 **无** Legacy Token/Session 代码残留。

---

## 2. 成功标准（SMART）

| 标准 | 验收 |
|------|------|
| 签发 | 登录/注册仅 `access_token` + Set-Cookie `refresh_token` |
| 刷新 / 登出 | `/api/auth/refresh/`、`/api/auth/logout/` |
| Go | gateway / taskAIEndPoint **无** validate-session / Token 构造 |
| Django | **仅** `JWTAuthentication`；`authentication.py` 中无 Token/Session 类 |
| 前端 | **仅** Bearer；无 `authToken` / `Token ${` |
| 表 | auth.db + default **无** `accounts_customtoken`；**无** API 级 `django_session` 依赖 |
| 开关 | **无** `TASKAUTH_ENABLED`；runAll 始终编排 task-auth |
| grep | §14 CI 脚本零匹配 |
| 测试 | user-auth + frontend-auth-guard-redirect + gateway 价值流全绿 |

---

## 3. 方案选定

**3A：Access JWT + Refresh Token**，**硬切、零兼容**。

| 决策 | 选择 |
|------|------|
| Access TTL | 1 小时 |
| Refresh TTL | 30 天，httpOnly Cookie |
| 签名 | Ed25519 |
| Legacy 代码 | **全部删除，不保留分支** |
| 发布 | T1–T4 同 merge |

---

## 4. 架构

```
前端 ── Bearer access_jwt ──▶ saas-backend / taskContainerGW / taskAIEndPoint
     ── Cookie refresh ──────▶ taskAuth :8003
                                    │
                                    ├─ auth.db (refresh_tokens, revoked_jti, login_method)
                                    └─ GET /api/auth/jwks.json

Go 服务：JWKS 本地验签 → Django validate-scope（仅 tenant/task 授权）
```

---

## 5. JWT 契约

### 5.1 Access Claims

| Claim | 值 |
|-------|-----|
| `sub` | user_id |
| `iss` | `task-auth` |
| `aud` | `task-platform` |
| `exp` / `iat` | Unix 秒 |
| `jti` | UUID |

Header：`alg: EdDSA`，`kid`.

### 5.2 登录/注册响应（唯一格式）

```json
{
  "access_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "user": { "...": "..." },
  "redirect_url": "/projects/"
}
```

**禁止：** `"token"`、`sessionid` Cookie、`csrftoken` 作为 API 认证依赖。

### 5.3 Refresh

`POST /api/auth/refresh/` — Cookie `refresh_token` → 新 `access_token` + 轮换 refresh Cookie。401 → 前端跳登录。

### 5.4 Logout

`POST /api/auth/logout/` — 吊销 refresh；access `jti` 入 `revoked_jti`；Clear-Cookie。204。

### 5.5 JWKS

`GET /api/auth/jwks.json` — 公开；消费者缓存 5 分钟。

---

## 6. 删除清单（实施 grep 归零）

### 6.1 整文件删除或替换

| 路径 | 动作 |
|------|------|
| `accounts/models/token.py` | **删除** |
| `accounts/authentication.py` | **删除** → 新建 `accounts/jwt_authentication.py` |
| `cloud/task_container_gateway/internal_views.py` 中 `validate_session` | **删除函数**；urls 删路由 |
| `cloud/urls_task_container_gateway_internal.py` 中 `validate-session/` | **删除** |
| `taskAuth/src/*` 中 `getOrCreateToken` / `deleteTokenByKey` / customtoken SQL | **删除** |
| `scripts/auxiliary/get_token.py` | **删除** → `taskAuth/scripts/issue_dev_jwt.sh` |

### 6.2 saas-backend 代码删除（非「禁用」）

| 删除项 | 说明 |
|--------|------|
| `CustomTokenAuthentication` | 类与所有 import |
| `SessionAuthenticationWithoutCSRF` | 类与所有 `@authentication_classes` 引用 |
| `CustomToken` 模型 export | `accounts/models/__init__.py` |
| `TaskAuthDatabaseRouter` 中 `CustomToken`/`LoginMethod` token 路由 | CustomToken 路由删除；LoginMethod 保留（凭证哈希） |
| `_authenticate_from_body` | 随 validate-session 删除 |
| `enrich-login` 内 `django_auth_login()` | 删除 |
| `UserViewSet` 各 action **fallback 块** | delegate 失败 → `taskauth_unavailable_response()`，**无** else 分支 |
| `accounts/services.py` 注册/登录重复实现 | 删除 fallback 专用段 |
| `frontend_hashed_password_backend.py` | 改调 taskAuth internal `bootstrap-login-method` 或删除 |
| `settings_test.py` 中 `TASKAUTH_ENABLED = False` | 删除；测试始终 JWT |
| `REST_FRAMEWORK` 默认认证 | 仅 `JWTAuthentication` |

### 6.3 配置 / 开关删除

| 删除 | 说明 |
|------|------|
| `TASKAUTH_ENABLED` 环境变量 | 删除读取逻辑；行为恒为 enabled |
| `settings.py` 中 `_env_bool('TASKAUTH_ENABLED', …)` | 删除 |
| runAll / 文档中「TASKAUTH_ENABLED=false 降级」 | 删除 |

**保留：** `TASKAUTH_BASE_URL`、`TASKAUTH_INTERNAL_SECRET`（taskAuth 桥接与 internal 仍需要）。

### 6.4 前端删除

| 删除 | 文件示例 |
|------|----------|
| `localStorage.authToken` 读写 | Login.vue, AuthForms.vue, Navbar.vue, PrivacyReconsentGate.vue |
| `Authorization: Token ${…}` | apiUtils.js, submitAIComment.js |
| 默认 `credentials: 'include'` | apiUtils.js（API 改 `omit`） |
| `attachCsrfForUnsafeMethod` | apiUtils.js（用户 API 路径） |
| Playwright `localStorage.authToken` / `Token ${tok}` | `playwright/front_project/tests/*.js` |

### 6.5 Go 删除

| 删除 | 位置 |
|------|------|
| `djangoValidateSession` | taskContainerGateway |
| `/validate-session/` 调用 | django_client.go |
| 测试中 `Authorization: Token` | handlers_*_test.go |

### 6.6 数据库

```sql
-- auth.db（taskAuth migration 003）
DROP TABLE IF EXISTS accounts_customtoken;

-- default db：API 不再读 django_session；表可保留给 Django admin 迁移期，但 **无代码引用**
```

### 6.7 value-stream.yaml

**删除所有** `task-auth.accounts_customtoken.*` fields。

**新增：**

- `task-auth.runtime.access_jwt`
- `task-auth.refresh_tokens.token_hash`
- `task-auth.runtime.jwks_kid`

**步骤：**

- 删除 / 重命名 `gateway-validate-session` → `gateway-validate-scope`
- 新增 `jwt-refresh`、`jwt-revocation`

### 6.8 测试 fixture 替换

| 旧 | 新 |
|----|-----|
| `CustomToken.objects.create` + `HTTP_AUTHORIZATION=Token …` | `tests/jwt_helpers.py`：`issue_test_jwt(user_id)` + `Bearer` |
| 全仓库 pytest / Go / vitest / Playwright | 统一 helper，**禁止** inline CustomToken |

---

## 7. taskAuth 实现

### 7.1 新增

- `jwt/sign.go`, `jwt/jwks.go`, `auth_refresh.go`, `auth_logout.go`
- `migrations/002_refresh_tokens.sql`, `003_drop_customtoken.sql`

### 7.2 登录 handler

成功路径 **仅**：`signAccessJWT` + `issueRefreshToken` + Set-Cookie。

**禁止：** 写 customtoken、回调 Django 写 session、响应 `token` 字段。

### 7.3 配置

```json
{
  "jwtAccessTtlSeconds": 3600,
  "jwtRefreshTtlSeconds": 2592000,
  "jwtSigningKeyPath": "taskAuth/data/jwt_ed25519.pem",
  "jwtKeyId": "2026-05-v1"
}
```

---

## 8. saas-backend

### 8.1 `JWTAuthentication`（唯一认证类）

Bearer → JWKS 验签 → `User.objects.get(pk=sub)` → 可选 taskAuth internal 查 `revoked_jti`。

### 8.2 `validate-scope`（新，替代 validate-session）

`POST /api/internal/task-container-gateway/validate-scope/`

Body：`{user_id, tenant_id, workspace_id, task_id}` → `{scope_ok: true|false}`。

**不** 接收 cookie / authorization。

### 8.3 taskauth_bridge

**保留** HTTP delegate（注册/登录路由）；**删除** bridge 内一切 Django 本地 auth 实现与 fallback。

### 8.4 镜像市场 / admin

- `/logout/` → `POST /api/auth/logout/`
- Django admin：单独 slice 评估；**不** 以 Session 作为 SPA API 认证回退

---

## 9. Go 消费者

`pkg/jwtverify.VerifyBearer(r)` → gateway → `djangoValidateScope`.

taskAIEndPoint：用户身份 JWKS；proxy token 逻辑不变。

---

## 10. 前端

- `sessionStorage.access_token` + Bearer header
- refresh/logout：`credentials: 'include'`
- 401 → refresh → 重试 → 跳登录
- **删除** 所有 `authToken` / `data.token` 引用

---

## 11. 领域概念（→ /5-ddd）

| 保留 | 删除 |
|------|------|
| AccessToken (JWT VO) | **CustomToken** aggregate |
| RefreshToken entity | Session-based API auth |
| Revocation service | CredentialSession (Django session) |

---

## 12. 价值流影响

| 流 | 变更 |
|----|------|
| **user-auth** | 全部 login/logout/register fields 切 JWT；删 customtoken fields |
| **user-auth** | + `jwt-refresh`, `jwt-revocation` |
| **frontend-auth-guard-redirect** | Bearer + refresh 断言 |
| **task-container-gateway** | `gateway-validate-scope` 替代 validate-session |
| **auth-table-cleanup** | 并入本方案；customtoken 必 DROP |

---

## 13. 实施切片

| 切片 | 内容 |
|------|------|
| **T1** | taskAuth JWT + refresh + logout + JWKS；删 Go customtoken 代码；DROP 表 |
| **T2** | Django JWTAuthentication；**删** authentication.py / token.py / validate-session；validate-scope |
| **T3** | 前端 + Playwright 全量 Bearer；删 authToken/CSRF/session 默认 |
| **T4** | Go jwtverify；删 djangoValidateSession |
| **T5** | 删 TASKAUTH_ENABLED；value-stream；§14 CI grep 门禁；E2E |

**T1–T4 必须同 merge。** 任一 PR 不得单独引入「仅加 JWT 不删 Token」状态。

---

## 14. 测试与 CI 门禁

### 14.1 硬切 grep（merge 阻塞）

```bash
#!/usr/bin/env bash
# scripts/verify_no_legacy_user_auth.sh
set -euo pipefail
PATTERNS=(
  'CustomTokenAuthentication'
  'SessionAuthenticationWithoutCSRF'
  'CustomToken\.objects'
  'accounts_customtoken'
  'django_auth_login'
  'localStorage\.authToken'
  "Token \\\\\\\\$"
  'Authorization: Token'
  'validate-session'
  'djangoValidateSession'
  'TASKAUTH_ENABLED'
  'getOrCreateToken'
  'deleteTokenByKey'
)
for p in "${PATTERNS[@]}"; do
  if rg -l "$p" \
    --glob '!docs/**' \
    --glob '!**/*.md' \
    --glob '!scripts/verify_no_legacy_user_auth.sh' \
    --glob '!**/container_access_token*' \
    . ; then
    echo "legacy user auth pattern found: $p" >&2
    exit 1
  fi
done
```

### 14.2 测试要求

- 响应 JSON **无** `token` 键（login/register）
- pytest **无** CustomToken import
- E2E：login → profile → gateway forward → logout → 401

---

## 15. 风险

| 风险 | 缓解 |
|------|------|
| 改动面大 | 单 branch + runAll E2E 门禁 |
| refresh Cookie 跨域 | Vite 代理 `/api/auth` |
| 无 legacy 回滚 | git revert 整 merge |
| 测试量大 | 统一 `jwt_helpers` 降低迁移成本 |

---

## 16. 不在范围

- RBAC / 公司成员（Django）
- Git OAuth、容器 runtime token
- OAuth2/OIDC IdP
- **任何** Legacy Token/Session 兼容代码

---

## 17. 决策记录

| 问题 | 决策 |
|------|------|
| JWT | 3A Access + Refresh |
| Legacy 兼容 | **零 — 全部删除** |
| TASKAUTH_ENABLED | **删除开关** |
| validate-session | **整 endpoint 删除** |
| CustomToken 表 | **DROP，无读写** |
| 发布 | T1–T4 同 merge |
| CI | §14 grep 阻塞 merge |

---

## 18. 参考

- `task2app/front_project/app/src/utils/apiUtils.js`
- `task2app/Saas_project/accounts/authentication.py`（**将删除**）
- `taskContainerGateway/src/django_client.go`
