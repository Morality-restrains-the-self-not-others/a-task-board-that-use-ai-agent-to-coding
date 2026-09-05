# 价值流：登录页电话验证码登录可用

**日期**: 2026-07-15  
**设计**: `docs/superpowers/specs/2026-07-15-login-phone-otp-enable-design.md`

## 影响的既有流

| Stream | Step | 影响 |
|--------|------|------|
| user-auth | phone-otp-login | 前端入口依赖策略；迁移打开后可用 |
| system-admin-phone-login-recharge-policy | policy-thin-slice-phone-login | 迁移写入 True；管理员仍可改 |

## 增量切片（按价值）

### Increment 1 — 策略打开（运维可验收）

- Data migration：`enable_phone_login=True`
- 验收：`GET /api/public/system-feature-policy/` → true

### Increment 2 — 登录页 next + add_account（本 URL 闭环）

- `Login.vue`：同站 `next` 回跳 + `persistLoginAccountSlot`
- 验收：`?add_account=1&next=/tenant/.../billing/recharge/` 登录后落到 recharge；槽 upsert

### Increment 3 — 文档与回归

- intents + value-stream YAML
- Vitest / Django migration 测

## YAML 变更草案

```yaml
# user-auth 下新增（或激活）:
- name: login-phone-otp-ui-next-redirect
  status: active
  test_file: ../front_project/app/src/tests/views/login_phone_otp_redirect.test.js
  fields:
  - name: saas-backend.system_feature_policy.enable_phone_login
    description: 登录页展示手机验证码入口的门禁
```

## 排序

Increment 1 → 2 → 3（无阻塞依赖）
