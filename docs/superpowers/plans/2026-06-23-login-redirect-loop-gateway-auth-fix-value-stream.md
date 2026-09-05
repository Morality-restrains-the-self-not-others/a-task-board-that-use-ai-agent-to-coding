# Value Stream: 登录跳转循环修复 — 网关统一认证

> Derived from design: `docs/design/login-redirect-loop-fix.md`

## Value Summary

用户使用邮箱+密码登录后，token 通过网关 forward-auth 统一解析为 user_id 并注入请求头，后端服务信任网关头不再自行解析 token，消除登录后跳转回登录页的死循环。

## Related Value Streams

- **user-auth** (`conf/value-stream.yaml`): **modification** — 本修复修改现有 `login` 步骤的认证链路：从 "delegate 丢弃 Set-Cookie + 网关 forward-auth secret 不匹配" 修复为 "网关 forward-auth 正确解析 token → 后端信任网关头"。

## End-to-End Flow

```
用户点击登录 → POST /api/auth/ → delegate → taskAuth login → 200 + token
                                                          → delegate 传播响应头(新)
                                                          → 本地持久化 token(新 fallback)
→ window.location 跳转 → SPA 初始化 → router guard → GET /api/accounts/users/profile/
→ APISIX forward-auth → taskAuth resolve token → 200 + X-User-Id(修复后)
→ Django CustomTokenAuth → 信任网关头 → load_principal_from_user_id → ✅
→ 用户进入 /system-admin/
```

## Value Increments

### Increment 1: 网关统一认证修复 (Thin Slice)

**Value to user:** 登录成功后正常进入系统，不再跳回登录页。

**Scope:**
1. 配置：确保 `TASKAUTH_INTERNAL_SECRET` 与 APISIX forward-auth 配置一致
2. 配置：确保 `TASK_GATEWAY_INTERNAL_SECRET` 与 APISIX transformer 配置一致
3. 代码：`delegate.py` — 传播 taskAuth 响应头（保留 Set-Cookie）+ 登录时本地写回 token
4. 代码：`token_registry.py` — 新增 `upsert_local_token` / `resolve_local_token`
5. 代码：`principal_loader.py` — `load_principal_from_token` 增加本地 DB 回退

**Depends on:** nothing (bug fix on existing `user-auth` stream)
