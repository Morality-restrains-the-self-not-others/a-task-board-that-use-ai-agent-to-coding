# 实施计划: AI Provider OIDC 登录

> 输入:
> - 设计文档: `docs/specs/ai-provider-oidc-login/design.md`
> - 价值流: `docs/specs/ai-provider-oidc-login/value-stream.md`
> - NFR 澄清: `docs/specs/ai-provider-oidc-login/nfr-clarification.md`
> - 领域模型: `task2app/Saas_Ai_Provider/apps/marketplace/domain/`

---

## Phase 0: 领域层（已由 DDD 步骤产出）

- [x] `domain/value_objects/oidc_auth_session.py` — OidcAuthSession（PKCE state + code_verifier + nonce）
- [x] `domain/value_objects/oidc_id_token_claims.py` — OidcIdTokenClaims（已验证的 id_token claims）
- [x] `domain/events/oidc_events.py` — VendorLoggedInViaOidc / StaffLoggedInViaOidc / OidcLoginFailed
- [x] `domain/repositories/vendor_repository.py` — VendorRepository ABC
- [x] `domain/repositories/platform_staff_repository.py` — PlatformStaffRepository ABC
- [x] `domain/services/oidc_authentication_service.py` — OidcAuthenticationService

---

## Phase 1: 基础设施层 — OIDC RP Client + 仓储实现

### Task 1.1: OIDC RP 基础设施客户端
**文件**: `apps/marketplace/oidc_rp.py` (新建)
**依赖**: domain 层 value_objects
**内容**:
- `OidcRpClient` 类：实现 `OidcTokenExchange` 协议
  - `build_authorize_url(session: OidcAuthSession, redirect_uri: str) -> str`
  - `exchange_code_for_claims(code, code_verifier, redirect_uri) -> OidcIdTokenClaims`
  - `_fetch_jwks()` — 从 taskAuth JWKS endpoint 获取 RSA 公钥
  - `_validate_id_token(raw_id_token)` — 签名验证 + claims 校验
- 从 `django.conf.settings` 读取 `OIDC_RP_*` 配置
- HTTP 调用使用 `urllib.request`（无外部依赖）或项目已有 `HTTPClient`
- 验证项：`iss` / `aud` / `exp` / `nonce` / 签名

### Task 1.2: Vendor 仓储实现
**文件**: `apps/marketplace/repositories.py` (新建，或扩展已有)
**依赖**: domain/repositories/vendor_repository.py
**内容**:
- `DjangoVendorRepository(VendorRepository)`
  - `find_active_by_email(email)` → `Vendor.objects.filter(email__iexact=email, is_active=True).first()`
  - `find_active_by_saas_user_id(sid)` → `Vendor.objects.filter(saas_user_id=sid, is_active=True).first()`
  - `bind_saas_user_id(vendor, sid)` → `vendor.saas_user_id = sid; vendor.save(update_fields=[...])`

### Task 1.3: PlatformStaff 仓储实现
**文件**: `apps/marketplace/repositories.py` (同上)
**依赖**: domain/repositories/platform_staff_repository.py
**内容**:
- `DjangoPlatformStaffRepository(PlatformStaffRepository)`
  - `find_active_by_saas_superadmin_id(sid)` → `PlatformStaff.objects.filter(saas_superadmin_id=sid, is_active=True).first()`
  - `create_from_oidc_claims(...)` → 创建 PlatformStaff + set_password

### Task 1.4: 事件发布器实现
**文件**: `apps/marketplace/repositories.py` (同上)
**依赖**: domain/services/oidc_authentication_service.py
**内容**:
- `LoggingEventPublisher(OidcEventPublisher)` — 日志记录实现
  - `publish(event)` → `logger.info(...)` / `logger.warning(...)`
  - 记录 trace_id, timestamp, event_type

### Task 1.5: 验证
```bash
cd task2app/Saas_Ai_Provider && python -c "
from apps.marketplace.domain.services.oidc_authentication_service import OidcAuthenticationService
print('Domain service import OK')
"
```

---

## Phase 2: 后端 API 端点

### Task 2.1: OIDC 视图
**文件**: `apps/marketplace/views_oidc.py` (新建)
**依赖**: Phase 1 (oidc_rp + repositories)
**内容**:
- `@api_view(["GET"]) @permission_classes([AllowAny]) def oidc_authorize(request):`
  - 读取 `?role=vendor|admin`
  - 生成 `OidcAuthSession.generate(role)`
  - 存入 `request.session["oidc_auth"]`
  - 构建 authorize URL → 302 redirect
- `@api_view(["GET"]) @permission_classes([AllowAny]) def oidc_callback(request):`
  - 读取 `?code=...&state=...`
  - 从 session 加载 `OidcAuthSession`
  - 校验 state → 创建 `OidcAuthenticationService` → 调用 `authenticate_vendor` / `authenticate_staff`
  - 签发 JWT (vendor/staff) → 302 redirect 到 `/vendor#token=...` 或 `/admin#token=...`
  - 异常处理：`expired` / `invalid` / `ValueError` → 302 到前端带 `?error=...`

### Task 2.2: URL 路由
**文件**: `apps/marketplace/urls.py` (修改)
**依赖**: Task 2.1
**内容**: 添加两条路由
```python
path("auth/oidc/authorize/", oidc_authorize, name="oidc-authorize"),
path("auth/oidc/callback/", oidc_callback, name="oidc-callback"),
```

### Task 2.3: Django Settings 配置
**文件**: `provider/settings.py` (修改)
**依赖**: Phase 1
**内容**: 添加 OIDC RP 配置
```python
OIDC_RP_CLIENT_ID = "ai-provider"
OIDC_RP_CLIENT_SECRET = os.environ.get("OIDC_RP_CLIENT_SECRET", "aip-oidc-dev-secret")
OIDC_RP_ISSUER = os.environ.get("OIDC_RP_ISSUER", "http://183.250.1.132:18081")
OIDC_RP_SCOPES = ["openid", "email", "profile"]
```

### Task 2.4: 验证
```bash
cd task2app/Saas_Ai_Provider && python manage.py check
curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:8010/api/auth/oidc/authorize/?role=vendor"
# 应返回 302
```

---

## Phase 3: taskAuth OIDC Client 注册

### Task 3.1: 配置更新
**文件**: `conf/auth/task-auth/config.yaml` (修改)
**内容**: 在 `oidc.bootstrapClients` 中添加 `ai-provider` 条目
```yaml
- clientId: "ai-provider"
  clientSecret: "aip-oidc-dev-secret"
  redirectUri: "http://183.250.1.132:8010/api/auth/oidc/callback/"
```

### Task 3.2: 验证
```bash
# 重启 taskAuth 使配置生效
# 验证 OIDC discovery endpoint
curl -s http://183.250.1.132:18081/.well-known/openid-configuration | python3 -m json.tool | head -10
```

---

## Phase 4: 前端 OIDC 登录入口

### Task 4.1: VendorPortal OIDC 按钮
**文件**: `frontend/src/views/VendorPortal.vue` (修改)
**内容**:
- 在 SSO 链接下方增加 OIDC 登录按钮
- 按钮点击 → `window.location.href = apiBase + '/api/auth/oidc/authorize/?role=vendor'`
- `onMounted` 中新增 OIDC callback 处理：检测 `#token=...` hash

### Task 4.2: AdminPortal OIDC 按钮
**文件**: `frontend/src/views/AdminPortal.vue` (修改)
**内容**:
- 同上，role=admin

### Task 4.3: 前端构建
```bash
cd task2app/Saas_Ai_Provider/frontend && npm run build
```

### Task 4.4: 验证
```bash
# 重启 AI Provider
# 访问 http://183.250.1.132:8010/
# 确认看到「通过 OIDC 账号登录」按钮
```

---

## Phase 5: 单元测试

### Task 5.1: 领域层测试
**文件**: `apps/marketplace/tests/test_oidc_rp.py` (新建)
**内容**:
- `test_oidc_auth_session_generate_creates_valid_params` — state/code_verifier/nonce 长度
- `test_oidc_auth_session_rejects_invalid_role` — role=invalid → ValueError
- `test_oidc_id_token_claims_rejects_empty_sub` — 无 sub → ValueError
- `test_oidc_auth_session_code_challenge_is_s256` — code_challenge 格式

### Task 5.2: 领域服务测试
**文件**: `apps/marketplace/tests/test_oidc_authentication_service.py` (新建)
**内容**:
- `test_authenticate_vendor_email_match` — 匹配成功 → returns vendor_id
- `test_authenticate_vendor_not_found` — 无匹配 → ValueError
- `test_authenticate_vendor_already_bound` — saas_user_id 冲突 → ValueError
- `test_authenticate_vendor_email_mismatch` — 邮箱不匹配 → ValueError
- `test_authenticate_staff_superuser_auto_create` — 首次 admin → 自动创建
- `test_authenticate_staff_not_superuser` — 非超管 → ValueError
- `test_authenticate_staff_inactive` — 已禁用 Staff → ValueError

### Task 5.3: 视图测试
**文件**: `apps/marketplace/tests/test_oidc_views.py` (新建)
**内容**:
- `test_authorize_redirects_with_valid_role` — role=vendor → 302
- `test_authorize_rejects_invalid_role` — role=hacker → 400
- `test_callback_state_mismatch` — state 不匹配 → 400
- `test_callback_missing_code` — 无 code → 400

### Task 5.4: 运行测试
```bash
cd task2app/Saas_Ai_Provider && python -m pytest apps/marketplace/tests/test_oidc*.py -v
```

---

## Phase 6: E2E 测试

### Task 6.1: Playwright E2E
**文件**: `playwright/saas_ai_provider/tests/django8010-oidc-login.playwright.test.js` (新建)
**内容**:
- `test_oidc_login_button_visible_on_vendor_portal`
- `test_oidc_login_button_redirects_to_taskauth`
- `test_oidc_callback_token_stored_in_localstorage`
- `test_oidc_login_failure_shows_error_message`

### Task 6.2: 运行 E2E
```bash
npx playwright test playwright/saas_ai_provider/tests/django8010-oidc-login.playwright.test.js
```

---

## 任务依赖图

```
Phase 0 (DDD 领域层) ✅ 已完成
  ├→ Phase 1.1 (OIDC RP Client)
  │     ├→ Phase 1.2-1.4 (仓储 + 事件发布器)
  │     │     └→ Phase 2.1 (视图) → 2.2 (路由) → 2.3 (settings)
  │     │           └→ Phase 5.1-5.3 (单元测试)
  │     └→ Phase 3.1 (taskAuth client 注册)
  └→ Phase 3.2 (验证) → Phase 2.4 (验证)
        └→ Phase 4.1-4.2 (前端) → 4.3 (构建) → 4.4 (验证)
              └→ Phase 6.1 (E2E) → 6.2 (运行)
```

## 预计影响文件

| 文件 | 操作 | 阶段 |
|------|------|------|
| `apps/marketplace/domain/value_objects/oidc_auth_session.py` | ✅ 已创建 | Phase 0 |
| `apps/marketplace/domain/value_objects/oidc_id_token_claims.py` | ✅ 已创建 | Phase 0 |
| `apps/marketplace/domain/events/oidc_events.py` | ✅ 已创建 | Phase 0 |
| `apps/marketplace/domain/repositories/vendor_repository.py` | ✅ 已创建 | Phase 0 |
| `apps/marketplace/domain/repositories/platform_staff_repository.py` | ✅ 已创建 | Phase 0 |
| `apps/marketplace/domain/services/oidc_authentication_service.py` | ✅ 已创建 | Phase 0 |
| `apps/marketplace/oidc_rp.py` | 新建 | Phase 1 |
| `apps/marketplace/repositories.py` | 新建 | Phase 1 |
| `apps/marketplace/views_oidc.py` | 新建 | Phase 2 |
| `apps/marketplace/urls.py` | 修改 | Phase 2 |
| `provider/settings.py` | 修改 | Phase 2 |
| `conf/auth/task-auth/config.yaml` | 修改 | Phase 3 |
| `frontend/src/views/VendorPortal.vue` | 修改 | Phase 4 |
| `frontend/src/views/AdminPortal.vue` | 修改 | Phase 4 |
| `apps/marketplace/tests/test_oidc_rp.py` | 新建 | Phase 5 |
| `apps/marketplace/tests/test_oidc_authentication_service.py` | 新建 | Phase 5 |
| `apps/marketplace/tests/test_oidc_views.py` | 新建 | Phase 5 |
| `playwright/saas_ai_provider/tests/django8010-oidc-login.playwright.test.js` | 新建 | Phase 6 |
