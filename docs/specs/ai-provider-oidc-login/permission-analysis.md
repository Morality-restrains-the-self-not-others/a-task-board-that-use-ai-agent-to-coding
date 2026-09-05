# AI Provider OIDC 登录 — 角色权限分析

基于 `docs/specs/ai-provider-oidc-login/design.md` 的权限影响审核。

---

## 1. 权限影响矩阵

### 新增 API 端点

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| `GET /api/auth/oidc/authorize/` | Anonymous | System | 触发 OIDC 重定向 | `AllowAny`（预期） | ✅ 充分 | `role` 参数仅支持 `vendor`/`admin` 白名单校验，拒绝非法值 |
| `GET /api/auth/oidc/callback/` | Anonymous (OIDC 回调) | System | 处理 OIDC callback，签发 JWT | `AllowAny`（预期） | ✅ 充分 | 信任边界在 `id_token` 验证（签名/iss/aud/exp），不在 session auth |

### 新增内部函数/模块

| 改动点 | 调用者 | 敏感操作 | 权限边界 | 是否缺失 | 建议 |
|--------|--------|----------|----------|----------|------|
| `oidc_rp.build_authorize_url()` | `views_oidc.authorize` | 无 | N/A（URL 构建） | ✅ | — |
| `oidc_rp.exchange_code_for_tokens()` | `views_oidc.callback` | `POST` 到 OIDC Provider token endpoint | N/A（服务端→服务端） | ✅ | — |
| `oidc_rp.validate_id_token()` | `views_oidc.callback` | 签名验证 | N/A（密码学校验） | ✅ | 必须验证 `iss`/`aud`/`exp`/`nonce` |
| `oidc_rp.exchange_oidc_for_vendor()` | `views_oidc.callback` | 签发 `vendor` JWT | email 匹配 + `is_active` 过滤 | ✅ | 已检查 `is_active=True` |
| `oidc_rp.exchange_oidc_for_staff()` | `views_oidc.callback` | 签发 `staff` JWT | `is_superuser` claim + `is_active` 过滤 | ⚠️ | `exchange_oidc_for_staff` 需补充 `is_active` 检查 |

### 修改文件

| 文件 | 改动类型 | 权限影响 | 建议 |
|------|----------|----------|------|
| `urls.py` | 新增 2 条路由 | 无直接权限影响 | 确保路由不可被反向代理绕过 |
| `settings.py` | 新增 OIDC RP 配置 | 无直接权限影响 | `OIDC_RP_CLIENT_SECRET` 不可写入日志 |
| `VendorPortal.vue` | 前端 UI 新增按钮 | 无后端权限影响 | — |
| `AdminPortal.vue` | 前端 UI 新增按钮 | 无后端权限影响 | — |
| `conf/auth/task-auth/config.yaml` | 新增 bootstrap client | 新增 client credential | client secret 强度足够 |

---

## 2. 新增角色/权限建模

### 无需新增角色

OIDC 登录不引入新角色类型。Vendor 和 PlatformStaff 使用现有的角色模型：

```
OIDC 认证 (taskAuth)
  │
  ├─ email 匹配 → Vendor (现有角色, is_active=True)
  │     └─ 获得 vendor JWT (typ: vendor)
  │
  └─ saas_superadmin_id 匹配 → PlatformStaff (现有角色)
        └─ 获得 staff JWT (typ: staff)
```

### 权限检查路径

| 层级 | OIDC 相关 | 现有端点（不变） |
|------|-----------|-----------------|
| L1 Middleware | OIDC authorize/callback — `AllowAny` | 所有 vendor/admin API — Bearer auth |
| L2 Permission | N/A（OIDC 端点无资源操作） | `IsVendorUser` / `IsStaffUser` |
| L3 Decorator | N/A | 无新增 |
| L4 Service Guard | `exchange_oidc_for_vendor`: `is_active` 过滤 | 现有 `exchange_vendor_bridge` 逻辑 |

### 与 SSO Bridge 的权限差异

| 检查项 | SSO Bridge | OIDC |
|--------|-----------|------|
| 验证方式 | HMAC 共享密钥签名 | RSA id_token 签名 + PKCE |
| 信任锚点 | bridge JWT payload | OIDC id_token claims |
| Vendor 查找 | `saas_user_id` 优先 | `email` 优先 |
| Vendor 绑定冲突防护 | `saas_user_id != uid` → 报错 | 暂无（email 匹配后不验证 saas_user_id 冲突） |
| Staff 创建条件 | 首次 admin SSO 自动创建 | `is_superuser=true` 才自动创建 |

---

## 3. 安全审计检查

- [x] **IDOR 风险**: OIDC authorize/callback 无 `/<resource>_id/` 路径参数，无 IDOR 风险。✅
- [x] **权限提升**: `role=vendor` 不可获取 `staff` JWT（在 callback 中按 role 参数分发到不同 exchange 函数）。✅
- [x] **跨租户泄露**: OIDC 登录仅涉及 Vendor/Staff 匹配，不查询 tenant 数据。✅
- [x] **403 vs 404**: 匹配失败返回 "未找到匹配账号"（不泄露 Vendor 是否存在）。✅
- [x] **user_id 注入**: callback 不接收用户指定的 `user_id`，身份由 OIDC Provider 签发。✅
- [x] **敏感操作**: OIDC callback 签发 JWT 属敏感操作，需记录审计日志。⚠️ 设计未提及日志 — 建议在 callback 成功/失败时记录 `INFO`/`WARNING` 级别日志
- [x] **state 参数**: 设计已提及 state 防 CSRF。需确保 state 存储在 Django session 中且 callback 时校验一致。
- [x] **PKCE**: 设计已提及 PKCE S256。需确保 `code_verifier` 存储在 Django session 中且 callback 时使用。
- [x] **`redirect_uri` 校验**: 必须白名单校验（仅允许 AI Provider 自己的 callback URL），防止授权码泄露到第三方。
- [x] **id_token `nonce` 校验**: 防重放攻击，需在 authorize 时生成 nonce 存入 session，callback 时验证 id_token 中的 nonce 一致。
- [x] **session 轮换**: OIDC callback 成功后应 `request.session.cycle_key()` 防止 session fixation。

### ⚠️ 发现的问题

| # | 严重度 | 问题 | 影响 | 修复建议 |
|---|--------|------|------|----------|
| 1 | ⚠️ 中 | `exchange_oidc_for_staff` 未检查 `is_active` | 已禁用 Staff 仍可通过 OIDC 登录 | 添加 `is_active=True` 过滤 |
| 2 | ⚠️ 中 | `exchange_oidc_for_vendor` 未验证 `saas_user_id` 冲突 | 若 Vendor 已绑定其他用户，OIDC 登录会覆盖 `saas_user_id` | 添加：已在绑定且 `!= sub` 时报错 |
| 3 | ⚠️ 低 | 审计日志缺失 | OIDC 登录成功/失败无 trace | callback 中记录结构化日志 |
| 4 | ⚠️ 低 | `redirect_uri` 动态拼接 | 需明确白名单，不可接受外部参数 | 在 settings 中硬编码或白名单校验 |

---

## 4. 测试用例清单

### 权限相关测试

| 测试场景 | 角色 | 操作 | 预期 |
|----------|------|------|------|
| OIDC authorize 缺少 role 参数 | Anonymous | `GET /api/auth/oidc/authorize/` | 400 |
| OIDC authorize role=invalid | Anonymous | `GET /api/auth/oidc/authorize/?role=hacker` | 400 |
| OIDC authorize role=vendor | Anonymous | `GET /api/auth/oidc/authorize/?role=vendor` | 302 → taskAuth |
| OIDC authorize role=admin | Anonymous | `GET /api/auth/oidc/authorize/?role=admin` | 302 → taskAuth |
| OIDC callback 缺失 state | Anonymous | `GET /api/auth/oidc/callback/?code=xxx` | 400 |
| OIDC callback state 不匹配 | Anonymous | `GET /api/auth/oidc/callback/?code=xxx&state=bad` | 400 |
| OIDC callback 无效 code | Anonymous | `GET /api/auth/oidc/callback/?code=fake&state=ok` | 400（token 交换失败） |
| OIDC callback role=vendor + email 匹配 | Anonymous | callback → `exchange_oidc_for_vendor` | 302 + vendor JWT |
| OIDC callback role=vendor + email 不匹配 | Anonymous | callback → 无匹配 Vendor | 400 "未找到匹配厂商" |
| OIDC callback role=admin + superuser | Anonymous | callback → `exchange_oidc_for_staff` | 302 + staff JWT |
| OIDC callback role=admin + 非 superuser | Anonymous | callback → is_superuser=false | 400 "非超级管理员" |
| 已禁用 Vendor OIDC 登录 | Anonymous (disabled Vendor) | callback → `is_active=False` | 400 "未找到匹配厂商" |
| 已禁用 Staff OIDC 登录 | Anonymous (disabled Staff) | callback → `is_active=False` | 400 "非超级管理员" |
| OIDC callback role=vendor 不签发 staff token | Anonymous | callback → 获得 vendor JWT → 访问 admin API | 401（`VendorBearerAuthentication` 拒绝 vendor token 访问 admin API） |
| OIDC callback role=admin 不签发 vendor token | Anonymous | callback → 获得 staff JWT → 访问 vendor API | 401（`StaffBearerAuthentication` 拒绝 staff token 访问 vendor API） |
| id_token 签发方非 taskAuth | Anonymous | `exchange_code_for_tokens` → id_token `iss != OIDC_RP_ISSUER` | 400 "无效身份提供方" |
| id_token aud 不匹配 | Anonymous | validate → `aud != "ai-provider"` | 400 "token audience 不匹配" |
| id_token 过期 | Anonymous | validate → `exp < now` | 400 "id_token 已过期" |

---

## 5. 风险评级与缓解

| 级别 | 风险项 | 缓解措施 |
|------|--------|----------|
| 🔴 高 | — | — |
| 🟡 中 | `saas_user_id` 覆盖（问题 #2） | 在 `exchange_oidc_for_vendor` 中添加已绑定检查 |
| 🟡 中 | Staff `is_active` 未检查（问题 #1） | 添加 `is_active=True` 过滤 |
| 🟢 低 | 审计日志缺失 | callback 日志记录 `INFO` 级别 |
| 🟢 低 | session fixation | callback 后 `cycle_key()` |

---

## 6. 设计文档回写建议

在原设计文档 `docs/specs/ai-provider-oidc-login/design.md` 中 **`exchange_oidc_for_vendor` 函数需更新**：

```python
def exchange_oidc_for_vendor(id_token_claims: dict) -> str:
    email = (id_token_claims.get("email") or "").strip().lower()
    sub = int(id_token_claims.get("sub"))

    vendor = Vendor.objects.filter(email__iexact=email, is_active=True).first()
    if not vendor:
        vendor = Vendor.objects.filter(saas_user_id=sub, is_active=True).first()

    if not vendor:
        raise ValueError("未找到与 OIDC 账号匹配的厂商账号，请先在镜像市场完成厂商注册且邮箱一致")

    # 【新增】已绑定其他用户时拒绝
    if vendor.saas_user_id is not None and vendor.saas_user_id != sub:
        raise ValueError("该厂商账号已绑定其他主站用户")

    if vendor.email.lower() != email:
        raise ValueError("OIDC 邮箱与厂商注册邮箱不一致")

    if vendor.saas_user_id is None:
        vendor.saas_user_id = sub
        vendor.save(update_fields=["saas_user_id", "updated_at"])

    return issue_token(str(vendor.id), "vendor")
```

**`exchange_oidc_for_staff` 函数需更新**：

```python
def exchange_oidc_for_staff(id_token_claims: dict) -> str:
    sub = int(id_token_claims.get("sub"))

    # 【修改】添加 is_active 过滤
    staff = PlatformStaff.objects.filter(saas_superadmin_id=sub, is_active=True).first()
    if staff:
        return issue_token(str(staff.id), "staff")

    if not id_token_claims.get("is_superuser"):
        raise ValueError("非超级管理员，无法访问管理端")

    staff = PlatformStaff(
        username=f"saas_{sub}"[:64],
        display_name=id_token_claims.get("preferred_username", "")[:100],
        saas_superadmin_id=sub,
    )
    staff.set_password(secrets.token_urlsafe(48))
    staff.save()
    return issue_token(str(staff.id), "staff")
```

---

## 判定结论: 🟡 黄灯

- **无红灯问题** — 无 IDOR、无跨租户泄露、无权限提升路径
- **黄灯问题** — 2 个中等风险需在实施前修复（`saas_user_id` 覆盖防护 + `is_active` 检查）
- **所有低风险项** — 可在实施时一并处理

修复后即在设计层面达到安全基线，可进入后续步骤。

---

## 总结清单

- `exchange_oidc_for_vendor` saas_user_id 冲突防护: 添加绑定检查（与 SSO bridge 行为对齐）、不添加（OIDC 登录以 email 为主匹配，允许覆盖旧绑定）
- `exchange_oidc_for_staff` is_active 检查: 添加 `is_active=True` 过滤、不添加（禁用 staff 直接删记录即可）
- OIDC callback 审计日志: 记录结构化日志、仅记录失败日志、不记录日志
