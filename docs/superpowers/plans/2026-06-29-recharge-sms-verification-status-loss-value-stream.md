# Value Stream: 充值 SMS 验证状态持久化修复

> Derived from design: `docs/superpowers/specs/2026-06-29-recharge-sms-verification-status-loss-design.md`

## Value Summary

用户在充值页完成 SMS 验证后，验证状态立即可见于状态轮询，无需等待 60 秒缓存过期。

## Related Value Streams

- **system-admin-phone-login-recharge-policy** (conf/value-stream.yaml L817): extension — 本修复保障该流 `recharge-policy-core` 步骤中 SMS 验证状态读取的运行时正确性。该流关注策略开关，本修复关注状态持久化。

## End-to-End Flow

```
[用户提交验证码] → [SMS 验证码校验] → [手机号绑定 taskAuth] → [Identity 缓存刷新] → [SMS 验证标记写入]
                                                                          ↓
[前端轮询状态] → [读取 Identity 缓存 (已刷新)] → [读取 SMS 验证缓存] → [返回 sms_verified=true]
```

## Value Increments

### Increment 1: Identity 缓存刷新 (Thin Slice)

**Value to user:** SMS 验证后立即显示"已验证"状态，无需等待或刷新页面。

**Scope:**
- `upsert_phone_login_method()` 成功后清除 `taskauth:identity:user:{uid}` 缓存
- `upsert_username_login_method()` 成功后同样清除（一致性）

**Depends on:** nothing

**Test:** `tests/test_billing_recharge_validation.py` — 新增 `test_upsert_phone_login_method_invalidates_identity_cache`

### Increment 2: RechargePhoneStatusView 防御性兜底

**Value to user:** 即使在 identity 缓存未及时刷新时，SMS 验证状态也不丢失。

**Scope:**
- `RechargePhoneStatusView.get()` 无论 `has_phone` 取值都调用 `is_recharge_sms_verified_for_user()`

**Depends on:** Increment 1

**Test:** `tests/test_billing_recharge_validation.py` — 新增 `test_recharge_phone_status_sms_verified_without_bound_phone`

## Fields Impact

| 字段 | 变更 |
|------|------|
| `task-auth.accounts_login_method.identifier` | 不变 — upsert 逻辑不变，仅增加 Django 侧缓存刷新 |
| `saas-backend.accounts_sms_verification_code.is_used` | 不变 — 验证码消费逻辑不变 |
