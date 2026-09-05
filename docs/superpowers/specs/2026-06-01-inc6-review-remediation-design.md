# Inc-6 审查补救设计（Inc-6f～6h）

> 日期：2026-06-01  
> 状态：**已实施（Inc-6f～6h）**  
> 前置：[`2026-06-01-no-user-orm-http-only-identity-design.md`](./2026-06-01-no-user-orm-http-only-identity-design.md)（Inc-6a～6e 已实施）  
> 触发：`/8-review` 有条件通过 — P0 手机 OTP token 断裂、503 语义、Inc-6c 未完成、CI/value-stream/测试缺口

---

## 1. 背景与目标

Inc-6 主路径（HTTP IdentityClient、token/resolve、生产零 `User.objects`、CI grep）已落地。审查发现 **生产手机 OTP 登录 token 不在 auth.db**，与 value-stream `phone-otp-login` 字段声明（`task-auth.accounts_customtoken.key` 写入 auth.db）矛盾。

**本设计目标（SMART）**：

1. 所有登录路径（密码 / 手机 OTP / enrich-login）返回的 token **均可**经 `POST /api/internal/token/resolve/` 校验  
2. taskAuth 不可达时 API 返回 **503**，不得误报 404  
3. 完成 Inc-6c：saas 生产路径 **零** `LoginMethod.objects` / `CustomToken.objects` 写操作  
4. CI 门禁接入 `check_ddd_bdd_compliance.py`；value-stream 新增 step 并 `active`  
5. `test_login_phone_code.py` 及 OTP 集成测试绿

---

## 2. 已确认决策（审查驱动）

| # | 问题 | 决策 |
|---|------|------|
| R1 | OTP token 谁创建 | **taskAuth auth.db 唯一写入口**；saas `forward_login` **禁止** `CustomToken.objects` |
| R2 | OTP token 创建时机 | **方案 A（推荐）**：taskAuth `proxyToDjangoAuth` 在 saas 返回成功后调用 `getOrCreateToken` 并注入响应 |
| R3 | 503 传播 | `AuthenticationServiceUnavailable` **不得**在 `principal_loader` 吞掉；视图层统一映射 503 |
| R4 | AUTH 类名 | **保留** `AUTH_USER_MODEL = accounts.User`（migration 链）；`PrincipalAccount = User` 别名；设计文档同步 |
| R5 | LoginMethod ORM | **测试专用**（`settings_test` / `TASKAUTH_ENABLED=False`）；生产路径删除或 `taskauth_enabled()` 早退 |
| R6 | dead code | 删除 `accounts/db_router.py`；运维脚本改 HTTP 或 `user_id` 参数 |

---

## 3. P0 — 手机 OTP token 断裂

### 3.1 现状（根因）

```
taskAuth handleLogin (phone+code)
  → proxyToDjangoAuth
    → saas forward_login
      → CustomToken.objects.get_or_create  ❌ default DB
  → 返回 token 给客户端

CustomTokenAuthentication
  → resolve_token HTTP → auth.db  ❌ 找不到 token → 401
```

密码登录路径正确：taskAuth `getOrCreateToken` → enrich-login（仅 session，token 已在 auth.db）。

### 3.2 方案对比

| 方案 | 描述 | 优点 | 缺点 |
|------|------|------|------|
| **A（推荐）** | taskAuth 包装 forward-login：saas 只返回 `user_id` + user JSON；taskAuth 调 `getOrCreateToken` 写 auth.db | 与密码登录一致；token 写入口单一；saas 无需新 HTTP | taskAuth 需解析 saas 响应 JSON |
| B | saas `forward_login` 调 taskAuth 新 API `POST /api/internal/token/create/` | saas 显式委托 | 多一次 HTTP；仍有两处 orchestration |
| C | saas 继续写 token，resolve 也查 saas DB | 改动小 | **违反 Inc-6 D1**；双真源 |

**选定：方案 A**

### 3.3 目标数据流

```
taskAuth proxyToDjangoAuth
  → POST saas /api/internal/taskauth/forward-login/
  ← { user: { id }, redirect_url, ... }   # 无 token 字段
  → user_id = user.id
  → token = getOrCreateToken(user_id, content_type_id)
  ← { token, user, redirect_url }         # taskAuth 组装最终响应
```

### 3.4 saas 变更

**文件**：`accounts/taskauth_internal_views.py` — `forward_login`

- 删除 `CustomToken.objects.get_or_create(...)`
- 响应 JSON **不含** `token` 字段（或显式 `"token": null` 供 taskAuth 忽略）
- 保留 `django_auth_login` + `serialize_principal_for_login`（域内 session / 隐私协议）

### 3.5 taskAuth 变更

**文件**：`taskAuth/src/auth_login.go` — `proxyToDjangoAuth`

1. 解析 saas 响应 `user.id`（string）
2. `token, err := getOrCreateToken(userID, cfg.UserContentTypeID)`
3. 合并 `token` 进 outbound JSON
4. saas 4xx/5xx 原样转发；saas 502 映射 502

**测试**：

- Go：`TestProxyToDjangoAuthInjectsAuthDBToken`（mock django 响应 + 断言 auth.db 有 token row）
- Python：`tests/test_forward_login_token_in_auth_db.py` — mock taskAuth 包装层或 integration 测 resolve_token 成功

---

## 4. P1 — 503 错误语义

### 4.1 问题

`load_principal_from_user_id` / `load_principal_from_token` 捕获 `AuthenticationServiceUnavailable` 返回 `None` → 上层 404。

### 4.2 设计

**分层契约**：

| 层 | 行为 |
|----|------|
| `TaskAuthIdentityClient` | 网络/5xx → raise `AuthenticationServiceUnavailable` |
| `principal_loader` | **重新抛出**（不捕获）或返回 `Result`；Inc-6f 选 **重新抛出** |
| DRF 认证 | 已有：`AuthenticationFailed('Authentication service unavailable.')` → 401；改为 **503** 需自定义 exception handler 或 `APIException(status_code=503)` |
| 内部视图 `enrich_login` / `github_app_views` | 捕获 → `JsonResponse({'detail': '...'}, status=503)` |

**新增**（可选集中）：

```python
# accounts/taskauth_bridge/errors.py
class IdentityServiceUnavailable(APIException):
    status_code = 503
    default_detail = 'authentication service unavailable'
```

**禁止**：taskAuth 宕机时返回 `found: false` 或 `user not found`。

### 4.3 测试

- `tests/test_identity_service_unavailable_503.py`：mock `forward_to_taskauth` raise → assert enrich-login 503、token auth 503

---

## 5. P1 — 完成 Inc-6c（LoginMethod / CustomToken ORM）

### 5.1 生产路径清扫清单

| 区域 | 文件 | 动作 |
|------|------|------|
| forward-login | `taskauth_internal_views.forward_login` | 删 CustomToken 写（§3） |
| legacy login | `user_views.py` login/logout 等 | `taskauth_enabled()` 时仅 delegate；**删除** fallback 中 CustomToken 写 |
| phone 注册 | `phone_login_auto_register.py` | 生产 delegate taskAuth；本地路径仅测试 |
| init | `scripts/init/init_tenant.py` | 改调 taskAuth bootstrap / HTTP；或标记 dev-only |
| billing | `billing_bridge/utils.py` | 邮箱解析改 `identity_client.get_user` |
| utils | `accounts/views/utils.py` | 同上 |
| serializers | `login_serializer.py` | `TASKAUTH_ENABLED` 时走 delegate，不查 LoginMethod |
| sso | `sso_user_resolve.py` | HTTP get_user.login_methods |

### 5.2 ORM 模型处置

- **不删除** `accounts/models/login_method.py`、`token.py` migration state（历史 FK 依赖）
- **标记**：模块 docstring `TEST_ONLY when TASKAUTH_ENABLED=False`
- **删除** `accounts/db_router.py`（已无 `DATABASE_ROUTERS` 引用）

### 5.3 CustomTokenAuthentication

- 保持 HTTP-only；**禁止**任何 saas 侧 token 持久化

---

## 6. P2 — CI 与 value-stream

### 6.1 CI 接入

**文件**：`task2app/scripts/ci/check_ddd_bdd_compliance.py`（或根 `scripts/ci/check_ddd_bdd_compliance.py`）

在 DDD 检查通过后追加：

```bash
bash task2app/Saas_project/scripts/ci/check_no_user_objects.sh
bash task2app/Saas_project/scripts/ci/check_no_auth_db_in_saas.sh
```

失败即 non-zero exit。

### 6.2 value-stream.yaml 变更

**更新** `phone-otp-login` step description：token 由 taskAuth 在 proxy 层写入 auth.db（非 saas forward_login）。

**新增 step**（`status: active`）：

```yaml
- name: http-only-identity-client
  status: active
  test_file: tests/test_taskauth_identity_client_http.py
  fields:
    - name: task-auth.accounts_user.id
      description: GET /api/accounts/users/{id}/ internal secret
    - name: task-auth.accounts_user.is_active
      description: 会话与业务门禁

- name: no-user-objects-guard
  status: active
  test_file: scripts/ci/check_no_user_objects.sh
  fields:
    - name: saas-backend.runtime.lifecycle_status
      description: CI 禁止生产路径 User.objects

- name: forward-login-authdb-token
  status: active
  test_file: tests/test_forward_login_token_in_auth_db.py
  fields:
    - name: task-auth.accounts_customtoken.key
      description: OTP forward-login 后 token 仅写 auth.db
    - name: task-auth.accounts_user.id
      description: forward-login 返回 user_id 供 taskAuth 建 token
```

---

## 7. P2 — 测试与工厂

### 7.1 修复 `test_login_phone_code.py`

| 失败 | 修复 |
|------|------|
| `User.objects.count()` | 改用 `LoginMethod.objects.filter(...).count()` 或 mock `send_event` 断言 payload |
| pk 类型 str vs int | 断言 `str(user.pk)` |
| inactive user | `User()` stub + `LoginMethod` 绑定 `object_id`；修复 `lm.user` GenericFK 在无表 User 下异常 → 用 `user_id` 字符串断言 |

### 7.2 新增测试

| 文件 | 覆盖 |
|------|------|
| `tests/test_forward_login_token_in_auth_db.py` | OTP 路径 token resolve 绿 |
| `tests/test_identity_service_unavailable_503.py` | 503 语义 |

### 7.3 测试工厂（延续 Inc-6e）

- 推广 `tests/factories/principal.py` 的 `create_test_user` / `make_user_id`
- view_test 逐步迁移（非阻塞，独立 PR）

---

## 8. P3 — 运维与文档

| 项 | 动作 |
|----|------|
| `selfcheck_system_deliverable_systems.py` | 改 `--user-id` CLI 或 HTTP batch-resolve |
| Inc-6 主设计 §10 | 勾选已实现项；注明 AUTH 类名偏差 R4 |
| Inc-6 主设计 §5.4 | 补充 forward_login token 由 taskAuth 注入 |

---

## 9. 领域概念（供 `/5-ddd`）

| 限界上下文 | 变更 |
|------------|------|
| **Auth（taskAuth）** | **Token 聚合写入口** — 所有会话 token 仅 `getOrCreateToken` / `deleteTokenByKey` |
| **Identity（saas）** | `forward_login` 仅负责域内副作用（隐私协议、session）；**不拥有 Token** |
| **跨上下文** | saas 返回 `user_id` → taskAuth 创建 Token → 客户端持 token → saas 读 token 经 HTTP resolve |

**规则**：Token 生命周期不得跨越 auth.db 边界写入 saas default DB。

---

## 10. Increment 路线图

### Inc-6f — P0 OTP token（ship 阻塞）

- taskAuth `proxyToDjangoAuth` 注入 auth.db token
- saas `forward_login` 删除 CustomToken 写
- 测试：`test_forward_login_token_in_auth_db.py` + 修 `test_login_phone_code.py`

### Inc-6g — P1 503 + Inc-6c 生产清扫

- `principal_loader` 503 传播
- 生产路径去除 LoginMethod/CustomToken 写
- 删 `db_router.py`

### Inc-6h — P2 CI + value-stream + 文档

- CI 脚本接入 compliance
- value-stream 三 step active
- 更新 Inc-6 主设计 §10

---

## 11. 成功标准

- [x] 手机 OTP 登录后 `resolve_token` 返回正确 `user_id`
- [x] taskAuth 不可达时 enrich-login / API token auth 返回 **503**
- [x] 生产 `forward_login` 零 `CustomToken.objects`（legacy login fallback 仅 TASKAUTH_ENABLED=False 测试路径）
- [x] `check_no_user_objects.sh` + `check_no_auth_db_in_saas.sh` 在 CI compliance 中执行
- [x] value-stream 新增 step 并 active
- [x] `test_login_phone_code.py` 全绿

---

## 12. 风险与缓解

| 风险 | 缓解 |
|------|------|
| taskAuth 解析 saas JSON 字段变更 | 契约测试 + 固定 `user.id` 字段 |
| 删除 login fallback 影响本地 dev | `TASKAUTH_ENABLED=False` 保留测试路径；dev 文档要求启 taskAuth |
| 503 改变现有客户端行为 | 仅 taskAuth 宕机场景；优于 silent 404 |

---

## 13. 变更日志

- 2026-06-01：初稿 — 覆盖 `/8-review` 全部 Critical/Important/Minor 项
