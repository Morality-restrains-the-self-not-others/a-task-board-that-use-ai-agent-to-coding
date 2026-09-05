# Inc-6 技术债清扫设计（Inc-7a～7c）

> 日期：2026-06-01  
> 状态：**已实施（Inc-7a～7c）**  
> 前置：[`2026-06-01-no-user-orm-http-only-identity-design.md`](./2026-06-01-no-user-orm-http-only-identity-design.md)（Inc-6 主体已实施）  
> 触发：Inc-6 `/goal` 完成后仍有三类残留 — legacy `CustomToken`、测试 `User.objects.create*`、生产 `CompanyMember.filter(user=)`

---

## 1. 背景与目标

Inc-6 已将 **生产路径** 切换为 taskAuth HTTP-only 身份。以下三类债务仍阻碍「架构终态」与 CI 门禁完整性：

| # | 残留 | 规模 | 风险 |
|---|------|------|------|
| D1 | `user_views.py` legacy 分支仍写 `CustomToken.objects` | 3 处 + 大量 `LoginMethod.objects` | CI 对 `user_views` 白名单；legacy token **无法**经 `resolve_token` 校验 → 测试已红（如 `test_gateway_validate_session`） |
| D2 | 测试仍用 `User.objects.create*` | **74 个文件**（含 view_test / migrations 各 1） | 与 Inc-6e 工厂计划未完成；测试语义仍假设 saas 本地 User 表 |
| D3 | `CompanyMember.filter(user=obj)` 依赖 `UserIdQuerySetMixin` | **12 个生产文件、~25 处** | Inc-6 §5.5 已禁止；Mixin 为过渡层，应显式 `user_id=` |

**本设计 SMART 目标**：

1. saas **零** `CustomToken.objects` / legacy 认证写路径（含 `user_views.py`）  
2. 测试统一经 `tests/factories/` 创建 principal + 注册 test token；**无**裸 `User.objects.create*`（migrations 除外）  
3. 生产代码 **零** `filter(user=)` / `members__user=`；CI 门禁无白名单  
4. value-stream `user-auth` 新增清扫 step；相关 pytest 全绿  

---

## 2. 已确认决策（待用户批准）

| # | 问题 | 决策 |
|---|------|------|
| C1 | legacy `user_views` 块如何处理 | **整段删除**；视图仅保留 `delegate_*` + `taskauth_unavailable_response()` 早退 |
| C2 | `settings_test.TASKAUTH_ENABLED` | 保持 `False`（单元测试不启真实 taskAuth）；**不**恢复 saas 侧 LoginMethod/CustomToken 写路径 |
| C3 | 测试如何拿 token | 新增 **`register_test_token(user_id) → key`**，由 `load_principal_from_token` 在 `TASKAUTH_ENABLED=False` 时读内存表 |
| C4 | view_test 如何测 login/register | 改为 **mock `forward_to_taskauth` / `delegate_*`** 或走 `forward_login` + `register_test_token`；不再依赖 legacy saas 注册 |
| C5 | `UserIdQuerySetMixin` | Inc-7b 完成后 **保留但标记 deprecated**（一版后再删，避免第三方/遗漏调用） |
| C6 | 测试迁移节奏 | **分 3 波**（view_test → CustomToken 测试 → 其余）；每波独立可合并 PR |
| C7 | `CustomToken` Django model | **保留 model/migration state**（历史兼容）；禁止一切 `.objects` 调用 |

---

## 3. 现状量化

### 3.1 CustomToken（生产）

```
accounts/views/user_views.py
  phone_register legacy  → CustomToken.objects.get_or_create   (L236)
  login legacy           → CustomToken.objects.get_or_create   (L470)
  logout legacy          → CustomToken.objects.filter.delete   (L491)
```

上述分支仅在 `delegate_* is None` 且 `TASKAUTH_ENABLED=False` 时执行。生产 `TASKAUTH_ENABLED=True` 不触发，但：

- `check_no_custom_token_in_saas.sh` 对 `user_views.py` **白名单排除**  
- `CustomTokenAuthentication` 已 **只** 走 HTTP `resolve_token`（或测试内存 token）→ legacy 创建的 token **无效**

### 3.2 User.objects（测试）

| 类别 | 文件数 | 说明 |
|------|--------|------|
| `tests/` | 58 | 业务/集成测试 |
| `accounts/view_test/` | 8 | value-stream user-auth 步骤测试 |
| `projects/view_test/` | 5 | |
| `cloud/view_test/` | 1 | |
| `tests/factories/principal.py` | 1 | 工厂本身仍调用 `User.objects.create` |
| migrations | 1 | **不修改** |

另有 **14 个测试文件** 使用 `CustomToken.objects.create`（与 D1 同源问题）。

### 3.3 CompanyMember.filter(user=)

| 文件 | 处数 |
|------|------|
| `projects/views/workspace_access_views.py` | 6 |
| `accounts/views/member_views.py` | 4 |
| `accounts/serializers/user_serializer.py` | 4 |
| `cloud/views/oauth_token_views.py` | 3 |
| `frontend_app/views/auth_views.py` | 4（含 2× `members__user=`） |
| 其余 7 文件 | 各 1 |

---

## 4. 方案设计

### 4.1 Inc-7a — 删除 legacy 认证 + 测试 token 注册表

#### 4.1.1 删除 legacy 块

**文件**：`accounts/views/user_views.py`

对每个 auth action（`phone_register`, `email_register`, `confirm_activation`, `login`, `logout`, 以及 reset/resend 等）：

```python
# To-Be 模式（每个 action 顶部）
delegated = delegate_login(request)
if delegated is not None:
    return delegated
if taskauth_enabled():
    return taskauth_unavailable_response()
return Response({'detail': 'authentication requires taskAuth'}, status=503)
```

- **删除** legacy 内所有 `LoginMethod.objects` / `CustomToken.objects` / `User()` 手工构造  
- reset/resend 等同理：delegate 失败 + taskAuth 关闭 → 503（与生产语义一致）

#### 4.1.2 测试 token 注册表

**新文件**：`accounts/models/test_token_registry.py`（或 `accounts/taskauth_bridge/test_token_registry.py`）

```python
_memory_tokens: dict[str, str] = {}  # token_key -> user_id

def register_test_token(user_id: str, key: str | None = None) -> str: ...
def clear_test_tokens_for_tests() -> None: ...
def resolve_test_token(key: str) -> str | None: ...
```

**修改**：`accounts/taskauth_bridge/principal_loader.py` — `load_principal_from_token`

```python
if not taskauth_enabled():
    uid = resolve_test_token(key)
    if uid:
        return load_principal_from_user_id(uid)  # 走内存 User
    return None
```

**工厂**：`tests/factories/auth_token.py`

```python
def issue_api_token(principal: User) -> str:
    return register_test_token(str(principal.pk))
```

**conftest**：autouse 增加 `clear_test_tokens_for_tests()`（与现有 `clear_memory_users_for_tests` 并列）。

#### 4.1.3 view_test 迁移策略

| 原测试 | 新策略 |
|--------|--------|
| `UserViewSet_login_test` | mock `delegate_login` 返回 token + user JSON；或 POST `forward_login` internal + `register_test_token` |
| `UserViewSet_email_register_test` | mock `delegate_email_register` |
| `UserViewSet_activate_test` | mock `delegate_confirm_activation` |
| 其余 view_test | 同上模式 |

**参考**：`tests/test_forward_login_token_in_auth_db.py`、`tests/test_identity_service_unavailable_503.py`。

#### 4.1.4 CI

- `check_no_custom_token_in_saas.sh`：**移除** `user_views.py` 白名单  
- 可选扩展：grep `LoginMethod\.objects` 于生产路径（`user_views` 删除后应零命中）

---

### 4.2 Inc-7b — CompanyMember 显式 user_id=

#### 4.2.1 替换规则

| 原写法 | 新写法 |
|--------|--------|
| `.filter(user=current_user)` | `.filter(user_id=str(current_user.pk))` |
| `.filter(user=obj, company=co)` | `.filter(user_id=str(obj.pk), company=co)` |
| `Company.objects.filter(members__user=user)` | `Company.objects.filter(members__user_id=str(user.pk))` |
| `CompanyMember.objects.create(..., user=u)` | `CompanyMember.objects.create(..., user_id=str(u.pk))` |

**辅助**（可选，减少重复）：

```python
# accounts/user_id_compat.py
def user_id_of(principal) -> str:
    return str(getattr(principal, 'pk', principal))
```

#### 4.2.2 文件清单（生产，按模块）

1. `accounts/serializers/user_serializer.py`（4）  
2. `accounts/views/member_views.py`（4）  
3. `projects/views/workspace_access_views.py`（6）  
4. `cloud/views/oauth_token_views.py`（3）  
5. `frontend_app/views/auth_views.py`（4）  
6. `frontend_app/decorators.py`、`frontend_app/views/project_views.py`  
7. `accounts/views/company_views.py`  
8. `cloud/views/cloud_platform_authorization_views.py`  
9. `cloud/utils/tenant_utils.py`  

`user_views.py` legacy 删除后，其内 2 处自然消失。

#### 4.2.3 CI 门禁

**新脚本**：`scripts/ci/check_no_filter_user_kwarg.sh`

```bash
# 禁止生产路径 filter(user= / members__user=
# exclude: tests/, view_test/, migrations/, user_id_compat.py
```

并入 `check_ddd_bdd_compliance.py`。

---

### 4.3 Inc-7c — 测试工厂统一（User.objects → factory）

#### 4.3.1 工厂 API（扩展 `tests/factories/principal.py`）

```python
def make_principal(**kwargs) -> User: ...       # 已有，内存实例，无 DB
def create_test_user(**kwargs) -> User: ...     # 改：内部 make_principal + register_memory_user
def create_test_superuser(**kwargs) -> User: ...
def user_id_of(user: User) -> str: ...
```

- **`create_test_user` 不再调用 `User.objects.create`**，改为 `make_principal` + `register_memory_user`  
- 需要 LoginMethod 的测试：保留 `LoginMethod.objects.create(object_id=str(user.pk), ...)`（saas 表，测试专用）或 mock identity client

#### 4.3.2 迁移波次

| 波次 | 范围 | 文件数 | 额外工作 |
|------|------|--------|----------|
| **W1** | `accounts/view_test/` | 8 | 与 Inc-7a delegate mock 同 PR |
| **W2** | 使用 `CustomToken.objects.create` 的 14 文件 | 14 | 改 `issue_api_token(principal)` |
| **W3** | 其余 `tests/` + `projects/view_test/` + `cloud/view_test/` | ~51 | 机械替换 import + factory |

#### 4.3.3 可选 codemod

`scripts/auxiliary/migrate_test_user_factory.py`：

- 将 `User.objects.create(` → `create_test_user(`  
- 将 `User.objects.create_user(` → `create_test_user(`  
- 自动插入 `from tests.factories.principal import create_test_user`  
- **人工 review** `create_superuser` / `filter(pk=)` 等边缘 case

#### 4.3.4 CI（软门禁 → 硬门禁）

**阶段 1**（W1+W2 后）：`check_no_user_objects.sh` 扩展 exclude 去掉 `tests/` 中对 `User.objects` 的新增（仅 warn）

**阶段 2**（W3 完成后）：`check_no_user_objects.sh` **包含 tests/**（仅 exclude `factories/`、`migrations/`）

---

## 5. 领域概念清单（供 `/5-ddd`）

| 限界上下文 | 概念 | 说明 |
|------------|------|------|
| **Auth（taskAuth）** | Token, LoginMethod | 唯一写入口；saas 不再 orchestrate |
| **Identity（saas 应用层）** | PrincipalAccount, TestTokenRegistry | 生产 HTTP；测试内存 registry |
| **Tenant（saas）** | CompanyMember | 外键已移除；**仅 `user_id: str` 引用** |

**不变量（Inc-7 后）**：

- saas 永不写 auth.db token  
- `CompanyMember` 查询参数只用 `user_id`，不用 `user=`  
- 测试 principal 与 test token 生命周期 = pytest fixture autouse 清理  

---

## 6. 价值流影响

### 6.1 受影响 stream

**`user-auth`**（主影响）：

| 现有 step | 影响 |
|-----------|------|
| login, phone-register, email-register, activate, logout, reset-password, resend-activation | view_test 改 mock delegate；**不再**测 saas legacy ORM |
| phone-otp-login, forward-login-authdb-token | 不变（已 HTTP） |
| no-user-objects-guard | 扩展 scope 至 tests（Inc-7c 阶段 2） |

### 6.2 建议新增 step（`/3-value-stream` 落地）

```yaml
- name: no-custom-token-guard
  status: planned
  test_file: scripts/ci/check_no_custom_token_in_saas.sh
  fields:
    - name: saas-backend.runtime.lifecycle_status
      description: 全路径禁止 CustomToken.objects（无 user_views 白名单）

- name: no-filter-user-kwarg-guard
  status: planned
  test_file: scripts/ci/check_no_filter_user_kwarg.sh
  fields:
    - name: saas-backend.accounts_companymember.user_id
      description: 生产查询显式 user_id，不依赖 UserIdQuerySetMixin

- name: test-principal-factory
  status: planned
  test_file: tests/factories/principal.py
  fields:
    - name: task-auth.accounts_user.id
      description: 测试经 make_principal/create_test_user 构造 user_id
```

### 6.3 测试影响摘要

| 测试集 | 变更 |
|--------|------|
| `accounts/view_test/UserViewSet_*` | 重写为 delegate/forward mock |
| `tests/test_gateway_validate_session.py` | `issue_api_token(user)` 替代 CustomToken ORM |
| ~60 个业务测试 | import factory；`CompanyMember.create(user_id=...)` |
| value-stream runner | W1 完成后 user-auth view_test 仍绿 |

---

## 7. Increment 路线图

```
Inc-7a (P0)  删 user_views legacy + test token registry + view_test 迁移 + CI 去白名单
    ↓
Inc-7b (P1)  CompanyMember user_id= 全生产替换 + check_no_filter_user_kwarg.sh
    ↓
Inc-7c (P2)  测试工厂 W2→W3 + CI tests 纳入 User.objects 门禁
```

| Increment | 交付物 | 验收 |
|-----------|--------|------|
| **Inc-7a** | 无 legacy auth；`register_test_token`；view_test 绿 | `rg CustomToken\.objects accounts/` 零命中；`test_gateway_validate_session` 绿 |
| **Inc-7b** | 12 文件 `user_id=`；新 CI 脚本 | `rg 'filter\(user=' production` 零命中 |
| **Inc-7c** | 74→0 测试文件裸 `User.objects` | extended CI 绿；全量 pytest 绿 |

---

## 8. 风险与缓解

| 风险 | 缓解 |
|------|------|
| view_test 大改导致 value-stream 红 | W1 与 Inc-7a 同 PR；逐步 mock 模板 |
| 测试 token 与生产 resolve 语义分叉 | registry **仅** `settings_test` + `TASKAUTH_ENABLED=False`；生产路径不可达 |
| W3 机械替换破坏边缘 case | codemod + 分目录 PR；每波跑 pytest |
| 删除 legacy 后本地无 taskAuth 无法手测登录 | 文档注明 dev 必须 `TASKAUTH_ENABLED=True` + runAll |

---

## 9. 成功标准

- [x] saas 全仓库（含 `user_views.py`）**零** `CustomToken.objects`  
- [x] saas 生产路径 **零** `filter(user=)` / `members__user=`  
- [x] 测试 **零** 裸 `User.objects.create*`（migrations / factories 除外）  
- [x] `check_ddd_bdd_compliance.py` + 新增门禁全绿  
- [x] value-stream `user-auth` 相关 pytest 全绿  
- [x] Inc-6 主设计 §5.5 禁止项无白名单例外  

---

## 10. 变更日志

- 2026-06-01：初稿 — Inc-6 三类技术债清扫（CustomToken legacy / 测试 User.objects / CompanyMember user=）
- 2026-06-01：**/goal 全部实施** — legacy auth 删除；`test_token_registry` + `issue_api_token`；view_test delegate mock；CompanyMember `user_id=`；CI 扩展 tests 范围
- 2026-06-01：**/8-review remediate** — `UserSerializer`/`CompanySerializer` 改 HTTP `login_methods_resolver`；`local_auth_delegate` 统一 view_test（删 `delegate_mocks.py`）；profile 昵称、503 重抛、`send_verification_code` delegate-only；taskAuth 新增 profile internal API；value-stream 新增 `no-custom-token-guard` / `no-filter-user-kwarg-guard` / `test-principal-factory`；`accounts/view_test/` 34 项 + `test_identity_service_unavailable_503` 全绿
