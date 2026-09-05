# Code Review: AI Provider OIDC 登录

## 审查范围

| 文件 | 类型 | 行数 |
|------|------|------|
| `domain/value_objects/oidc_auth_session.py` | 新建 | ~60 |
| `domain/value_objects/oidc_id_token_claims.py` | 新建 | ~35 |
| `domain/events/oidc_events.py` | 新建 | ~60 |
| `domain/repositories/vendor_repository.py` | 新建 | ~20 |
| `domain/repositories/platform_staff_repository.py` | 新建 | ~20 |
| `domain/services/oidc_authentication_service.py` | 新建 | ~130 |
| `oidc_rp.py` | 新建 | ~320 |
| `repositories.py` | 新建 | ~110 |
| `views_oidc.py` | 新建 | ~120 |
| `urls.py` | 修改 | +3 |
| `settings.py` | 修改 | +6 |
| `VendorPortal.vue` | 修改 | +31 |
| `AdminPortal.vue` | 修改 | +30 |
| `conf/value-stream.yaml` | 修改 | +28 |
| `tests/test_oidc_domain.py` | 新建 | ~80 |
| `tests/test_oidc_authentication_service.py` | 新建 | ~200 |

---

## DDD 合规检查

✅ `check_ddd_bdd_compliance.py` 通过 — 领域层无基础设施导入
✅ Repository 使用 ABC 抽象
✅ 领域服务通过构造函数注入依赖
✅ 领域事件使用过去式命名
✅ 值对象不可变（frozen dataclass）

## 测试结果

✅ 27/27 unit tests pass (0.03s)
✅ 11 domain value object tests
✅ 16 authentication service tests (in-memory fakes, no DB)

## 逐文件审查

### ✅ `domain/value_objects/oidc_auth_session.py`
- PKCE S256 code_challenge 正确实现
- 随机参数强度足够（state 32B, code_verifier 64B, nonce 32B）
- role 白名单校验（vendor/admin only）

### ✅ `domain/value_objects/oidc_id_token_claims.py`
- email 格式基本校验（含 @ 检查）
- 必填字段（sub/email/issuer/audience）构造时校验

### ✅ `domain/events/oidc_events.py`
- 三种事件类型覆盖成功/失败场景
- 使用 `datetime.now(timezone.utc)` (非 deprecated `utcnow()`)

### ✅ `domain/repositories/`
- 接口抽象清晰，仅定义 OIDC 需要的方法
- 不暴露 ORM 细节

### ✅ `domain/services/oidc_authentication_service.py`
- 纯领域逻辑，所有外部依赖通过 Protocol 注入
- Vendor 匹配：email 优先 → saas_user_id 回退
- saas_user_id 已绑定检查 ✅（权限分析 Issue #2 已修复）
- is_active 过滤 ✅（权限分析 Issue #1 已修复）

### ✅ `oidc_rp.py`
- OIDC Authorization Code Flow 完整实现
- JWKS 缓存避免重复请求
- 错误信息中文化，用户友好
- 使用标准库 urllib（无额外依赖）

### ✅ `repositories.py`
- Django ORM 实现通过 ABC 接口隔离
- LoggingEventPublisher 结构化日志

### ✅ `views_oidc.py`
- state CSRF 防护 ✅
- session 单次使用后清除 ✅
- role 参数隔离（vendor callback 不签发 staff token）

### ⚠️ `views_oidc.py` — session fixation
- `oidc_callback` 成功后未调用 `request.session.cycle_key()`
- **风险**: 低。OIDC session key 在回调后清除，且 Django session cookie 随响应更新
- **建议**: 在 callback 成功后添加 `request.session.cycle_key()`

### ⚠️ `oidc_rp.py` — HTTP 重定向跟随
- `urllib.request.urlopen` 默认跟随重定向，需确认 token endpoint 不会被重定向到恶意地址
- **风险**: 极低。token endpoint URL 由 settings 配置，不可由用户控制
- **建议**: 生产环境确保 `OIDC_RP_TOKEN_ENDPOINT` 为 HTTPS

### ✅ 前端 — VendorPortal / AdminPortal
- OIDC 按钮与 SSO 按钮并存
- `#token=` hash 处理在 SSO bridge 之前执行
- hash 清除后从 URL 移除

---

## 安全清单

| 检查项 | 状态 |
|--------|------|
| PKCE S256 强制 | ✅ |
| state CSRF 防护 | ✅ |
| id_token 签名验证 (RS256) | ✅ |
| iss/aud/exp claims 验证 | ✅ |
| nonce 生成与验证 | ✅ |
| redirect_uri 固定（非用户输入） | ✅ |
| role 参数不会导致权限提升 | ✅ |
| saas_user_id 绑定冲突检查 | ✅ |
| is_active 过滤 | ✅ |
| 审计日志（登录成功/失败） | ✅ |
| session 轮换 (cycle_key) | ⚠️ 缺失 |

---

## 审查结论

🟡 **Conditional Pass** — 1 个低风险项：

1. ⚠️ `views_oidc.py`: callback 成功后缺少 `request.session.cycle_key()` — 可在实施时添加，不阻塞

**无阻塞性问题。** 代码质量良好，领域层设计清晰，测试覆盖核心场景。

## 总结清单

- DDD 合规: ✅ 通过
- 单元测试: ✅ 27/27 pass
- 安全审查: 🟡 1 个低风险（session fixation — 添加 cycle_key 即可）
- 建议: 可进入 ship 步骤，推送前补充 cycle_key
