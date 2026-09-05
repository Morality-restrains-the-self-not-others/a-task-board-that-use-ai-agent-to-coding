# task2app 零 User ORM + HTTP-only 身份读模型（Inc-6）

> 日期：2026-06-01  
> 状态：**Inc-6a～6e 已实施；Inc-6f～6h 审查补救（见 [`2026-06-01-inc6-review-remediation-design.md`](./2026-06-01-inc6-review-remediation-design.md)）**  
> 前置：[`2026-06-01-saas-user-delegation-to-taskauth-design.md`](./2026-06-01-saas-user-delegation-to-taskauth-design.md) Inc-1～Inc-5  
> 触发：Inc-5 仍保留 `User` stub、`principal_loader` 直连 auth.db、`User.objects.*` 残留；用户要求 **userId 只来自 taskAuth API**，saas 不再承载 User ORM。

---

## 1. 已确认决策

| # | 问题 | 决策 |
|---|------|------|
| D1 | saas 读身份 | **HTTP only** — saas **禁止** `connections['taskauth']` / `LoginMethod.objects.using(taskauth)` / `principal_loader` 直连 auth.db |
| D2 | User ORM | **不保留** `accounts.User`（含 stub、`UserManager`、`resolve_user_by_id` 返回 User 实例） |

**推论**：

- taskAuth auth.db 是身份**唯一**持久化边界；saas 仅持有 `user_id: string` 与域内 `user_profile`。
- saas 进程内身份解析统一经 **`TaskAuthIdentityClient`（HTTP）**；允许进程内 TTL 缓存，不允许 SQL 读 auth 表。
- Django 仍需满足 `AUTH_USER_MODEL` 约束 → 引入 **`PrincipalAccount`**（见 §5.2），**不是** `User`，且无 saas 表、无 `User.objects` 业务用法。

---

## 2. 问题陈述

### 2.1 Inc-5 与目标差距

| Inc-5 现状 | Inc-6 目标 |
|------------|------------|
| `load_principal_from_user_id` 直连 auth.db | `GET /api/accounts/users/{id}/` + internal API |
| `load_principal_from_token` 查 auth.db `customtoken` | `POST /api/internal/token/resolve/`（新增）或 taskAuth delegate |
| `LoginMethod` ORM + `TaskAuthDatabaseRouter` | saas **删除** auth 表 ORM 路由；凭证读写仅 taskAuth HTTP |
| `accounts.User` `managed=False` stub | **删除**；`AUTH_USER_MODEL` → `PrincipalAccount` |
| `User.objects.*` 生产残留 ~8 文件 | **0**；CI grep 门禁 |
| `resolve_user_by_id` → User stub | **删除**；展示用 `batch-resolve` |

### 2.2 目标 SMART

1. 生产代码（`accounts/` `projects/` `cloud/` `billing_bridge/`）**零** `User.objects` / `get_user_model()` 用于身份解析  
2. **零** saas 侧 SQL 访问 auth.db（含 raw cursor、`using('taskauth')`）  
3. `request.user` 类型为 **`PrincipalAccount` / `AuthPrincipal`**，由 HTTP 填充  
4. 列表/展示名 **强制** `POST /api/internal/users/batch-resolve/`（已有）  
5. CI：`check_no_user_objects.sh` + `check_no_auth_db_in_saas.sh` 纳入合规流水线  

---

## 3. 目标架构

```
┌──────────────────── taskAuth (:8003) ────────────────────┐
│ auth.db: accounts_user, login_method, token, super_admin │
│ HTTP: GET  /api/accounts/users/{id}/                     │
│       POST /api/internal/users/batch-resolve/            │
│       POST /api/internal/token/resolve/        [新增]    │
│       GET  /api/internal/users/{id}/super-admin/         │
└────────────────────────▲─────────────────────────────────┘
                         │ HTTP only (trust_env=False)
┌────────────────────────┴─────────────────────────────────┐
│ saas default (:8001)                                       │
│  PrincipalAccount (AUTH_USER_MODEL, 无 DB 表)              │
│  user_profile(user_id)  CompanyMember(user_id)  …          │
│  TaskAuthIdentityClient + 60s TTL cache                    │
│  禁止: User ORM / auth.db router / principal_loader SQL    │
└────────────────────────────────────────────────────────────┘
```

---

## 4. 方案对比（D1/D2 已定）

Inc-5 过渡方案（auth.db 桥接 + User stub）**废弃为终点**；Inc-6 严格执行原设计文档 **方案 A 的 HTTP 面**。

| 维度 | Inc-5 过渡 | Inc-6 终点 |
|------|------------|------------|
| 读用户 | SQL + 部分 HTTP | **仅 HTTP** |
| AUTH 模型 | `accounts.User` stub | **`PrincipalAccount`** |
| Token 校验 | SQL customtoken | **HTTP token/resolve** |
| 性能 | 最低延迟 | 缓存 + batch；taskAuth 同机部署可接受 |
| 多实例 | saas 绑 auth.db 路径 | saas 无 auth 文件依赖 ✅ |

---

## 5. 目标架构详设

### 5.1 taskAuth API（saas 唯一身份读入口）

**已有（Inc-1）**：

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/accounts/users/{user_id}/` | 单用户：id, is_active, is_superuser, email, login_methods |
| POST | `/api/internal/users/batch-resolve/` | `{user_ids:[]}` → `{results:[{user_id, found, display_label}]}` |
| GET | `/api/internal/users/{user_id}/super-admin/` | 超管校验 |

**Inc-6 新增（建议）**：

| 方法 | 路径 | 用途 |
|------|------|------|
| POST | `/api/internal/token/resolve/` | `{token}` → `{user_id, is_active, is_superuser, is_staff}`；供 saas `CustomTokenAuthentication` |
| GET | `/api/internal/users/{user_id}/login-methods/` | 可选：profile 页专用，减少公网 GET 暴露 |

**错误语义**（与 Inc-5 §7 一致）：

- taskAuth 不可达 → 503 `{detail: "authentication service unavailable"}`
- 404 user → batch `found:false`；写业务前必须 resolve 成功

### 5.2 替换 `accounts.User` — `PrincipalAccount`

Django 要求 `AUTH_USER_MODEL` 为 `AbstractBaseUser` 子类。Inc-6 **删除 `User`**，新增：

```python
# accounts/models/principal_account.py（示意）
class PrincipalAccount(AbstractBaseUser):
    id = models.CharField(max_length=36, primary_key=True)
    is_active = models.BooleanField(default=True)
    is_staff = models.BooleanField(default=False)
    is_superuser = models.BooleanField(default=False)

    class Meta:
        managed = False
        app_label = 'accounts'
        # 无 db_table — 永不 migrate 到 saas

    USERNAME_FIELD = 'id'
    objects = PrincipalAccountManager()  # 仅 get_by_id via HTTP，禁止 create_all
```

- **`AuthPrincipal`** 保留为轻量运行时对象，或合并进 `PrincipalAccount`（二选一，Inc-6a 定稿）。
- **删除**：`User`、`UserManager`、`UserModel`、`SuperAdmin = User`、`resolve_user_by_id`。
- **`settings.AUTH_USER_MODEL = 'accounts.PrincipalAccount'`**。

### 5.3 saas 身份客户端 — `TaskAuthIdentityClient`

统一封装（替代 `principal_loader.py` SQL 路径）：

| 方法 | HTTP | 缓存 |
|------|------|------|
| `get_user(user_id)` | GET `/api/accounts/users/{id}/` + internal secret | 60s key=user_id |
| `batch_resolve(user_ids)` | POST batch-resolve | 单次请求内 dedupe |
| `resolve_token(token)` | POST token/resolve | 否（或极短 TTL） |
| `is_super_admin(user_id)` | GET super-admin | 60s |

实现位置：`accounts/taskauth_bridge/identity_client.py`（新），复用 `client.py` 的 `forward_to_taskauth` / Session。

**删除/废弃**：

- `principal_loader.load_principal_from_user_id` 的 SQL 分支  
- `TaskAuthDatabaseRouter` 对 `LoginMethod` / `CustomToken` 的路由（saas 不再持有这两张表的 ORM）  
- saas `accounts/models/login_method.py`、`token.py` 在 **Inc-6c** 从 saas DB 迁移 state 中移除（仅 state；物理表仅在 auth.db）

### 5.4 认证链 To-Be

| 环节 | Inc-6 行为 |
|------|------------|
| 注册/登录/激活 | 已 delegate taskAuth；不变 |
| `CustomTokenAuthentication` | `resolve_token` HTTP → `PrincipalAccount` |
| `TaskAuthAdminBackend` | `get_user(pk)` → `identity_client.get_user(pk)` |
| `FrontendHashedPasswordBackend` | **删除**或仅测试；生产登录不走 saas password SQL |
| `enrich_login` / `forward_login` | 已 Principal；token 创建在 taskAuth 侧完成 |
| Django Admin | session `taskauth_token` + HTTP 校验 |

### 5.5 业务代码契约

**禁止**：

```python
User.objects.*
get_user_model().objects.*
LoginMethod.objects.*   # saas 侧
CustomToken.objects.*   # saas 侧
connections['taskauth']
load_principal_from_user_id  # SQL 版
resolve_user_by_id
CompanyMember.objects.filter(user=obj)  # 用 user_id=
```

**允许**：

```python
user_id = str(request.user.pk)
identity_client.batch_resolve([uid1, uid2])
CompanyMember.objects.filter(user_id=user_id)
UserProfile.objects.filter(user_id=user_id)
```

---

## 6. 领域概念清单（供 `/5-ddd`）

| 限界上下文 | 实体 | 聚合根 | 事件 |
|------------|------|--------|------|
| **Auth（taskAuth）** | User, LoginMethod, Token, SuperAdmin | User | USER_CREATED, USER_ACTIVATED |
| **Identity（saas 应用层）** | PrincipalAccount（会话）, TaskAuthIdentityClient | — | — |
| **Tenant（saas）** | Company, CompanyMember, UserProfile | Company | COMPANY_CREATED |
| **Workspace / Project / Cloud** | 各业务实体 | 各聚合 | —（仅 `*_user_id` 引用） |

**Identity 规则（Inc-6）**：

- saas **不拥有** User 聚合；`user_id` 为跨上下文引用值对象  
- 校验用户存在 = taskAuth 应用服务（HTTP），非仓储  
- `PrincipalAccount` 是会话 DTO，**非** saas 仓储实体  

---

## 7. 价值流影响

| 价值流 | 影响 step | 变更 |
|--------|-----------|------|
| **user-auth** | login, enrich-login, phone-otp-login, no-saas-accounts-user-table | 新增 `http-only-identity-client`；废弃 saas `LoginMethod` 字段引用 |
| **user-auth** | taskauth-bridge-regression | 测试改 mock HTTP，非 auth.db fixture |
| **company-management** | member-crud | 仅 `user_id`；batch-resolve 展示名 |
| **gitoauth / cloud-integration** | 凡 `task-auth.accounts_user.id` | 读路径改 HTTP batch |
| **dev-platform-reset** | dev-bootstrap-admin | init 不 import `User` |

**新增 value-stream step（`/3-value-stream` 落地）**：

```yaml
- name: http-only-identity-client
  status: planned
  test_file: tests/test_taskauth_identity_client_http.py
  fields:
    - name: task-auth.accounts_user.id
      description: GET /api/accounts/users/{id}/ 真源
    - name: task-auth.accounts_user.is_active
      description: 会话与业务门禁

- name: no-user-objects-guard
  status: planned
  test_file: scripts/ci/check_no_user_objects.sh
  fields:
    - name: saas-backend.runtime.lifecycle_status
      description: CI 禁止生产路径 User.objects
```

---

## 8. Increment 路线图

### Inc-6a — HTTP IdentityClient + token/resolve API

- taskAuth：实现 `POST /api/internal/token/resolve/`
- saas：`TaskAuthIdentityClient`；`CustomTokenAuthentication` / backends 改 HTTP
- 测试：mock taskAuth HTTP；删 SQL `principal_loader` 路径

### Inc-6b — 删除 `accounts.User`，引入 `PrincipalAccount`

- `AUTH_USER_MODEL` 切换；migration state 移除 User
- 删 `resolve_user_by_id`、`UserManager`
- 全库替换 `from accounts.models import User` → `PrincipalAccount` 或 `user_id: str`

### Inc-6c — 移除 saas 侧 LoginMethod / CustomToken ORM

- 删除 `TaskAuthDatabaseRouter`；LoginSerializer 等仅 delegate 路径
- 清理 `ContentType.objects.get_for_model(User)` → 固定 `user_content_type_id` 配置或 taskAuth 返回

### Inc-6d — 生产代码清扫 + CI

- 清 `User.objects` 残留（§9 清单）
- `CompanyMember.filter(user=)` → `user_id=`
- CI：`check_no_user_objects.sh` + `check_no_auth_db_in_saas.sh`

### Inc-6e — 测试工厂（可并行）

- `tests/factories/principal.py`：`make_user_id()` + mock identity client
- 逐步替换 ~70 个测试文件的 `User.objects.create*`

---

## 9. 生产代码清扫清单（Inc-6d 输入）

| 文件 | 动作 |
|------|------|
| `accounts/views/user_views.py` | 去 ModelViewSet / `User.objects.all()` |
| `accounts/github_app_views.py` | HTTP batch-resolve |
| `accounts/phone_login_auto_register.py` | 事件 payload 用 user_id，删 `User.objects.get` |
| `accounts/ensure_default_company.py` | **删除**（taskEvents 已负责） |
| `accounts/management/commands/cleanup_gitoauth_orphan_credentials.py` | batch-resolve exists |
| `accounts/models/super_admin_manager.py` | **删除** |
| `cloud/services/github_pull_request_after_push.py` | HTTP / user_id only |
| `accounts/taskauth_bridge/principal_loader.py` | 重写为 HTTP 薄封装或删除 |
| `accounts/serializers/user_serializer.py` | 改 `PrincipalAccount` + HTTP 字段 |
| `projects/views/workspace_access_views.py` 等 | `user_id=str(request.user.pk)` |

---

## 10. 成功标准

- [x] saas **无** auth.db `DATABASES['taskauth']` 配置（或仅 taskAuth 进程使用）
- [x] saas **无** saas 表 User ORM 业务用法；`AUTH_USER_MODEL=accounts.User`（migration 兼容别名 `PrincipalAccount`）
- [x] 身份读 **100%** 经 taskAuth HTTP（允许内存缓存；测试 fallback 除外）
- [x] 生产代码 **零** `User.objects` / `using('taskauth')` / `principal_loader` SQL
- [x] CI 门禁绿 + user-auth pytest + taskEvents integration 绿
- [x] value-stream 新增 `http-only-identity-client` step
- [x] forward_login token 由 taskAuth `proxyToDjangoAuth` 写入 auth.db（Inc-6f）

---

## 11. 风险与缓解

| 风险 | 缓解 |
|------|------|
| HTTP 延迟 | 60s TTL + batch-resolve；同机 loopback |
| taskAuth 宕机 | 503 统一错误；读降级为 user_id 前缀（Inc-5 §7） |
| Django AUTH 改造面大 | Inc-6a→6d 分步；每步可回滚 |
| 测试改写量大 | Inc-6e 独立 PR；mock HTTP client |

---

## 12. 与 Inc-5 设计文档关系

- Inc-5 **删 saas User 表** 仍然有效  
- Inc-5 **auth.db 桥接**（`principal_loader`、`TaskAuthDatabaseRouter`）标记为 **过渡实现，Inc-6 删除**  
- Inc-5 设计文档 §10 成功标准在 Inc-6 完成后需追加 HTTP-only / 无 User ORM 勾选  

---

## 13. 变更日志

- 2026-06-01：初稿 — 用户确认 D1=HTTP only、D2=不保留 User ORM
- 2026-06-01：**/goal 收尾** — `selfcheck_system_deliverable_systems.py` 改 `SYSTEM_ADMIN_USER_ID` + `load_principal_from_user_id`；新增 `check_no_custom_token_in_saas.sh` 并入 DDD 合规；`conftest.py` autouse 清理 `_memory_users`
