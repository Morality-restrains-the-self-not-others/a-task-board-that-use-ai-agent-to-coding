# 功能意图：资料页手机绑定真实写入且支持占用转移

## 背景与目标

用户 `contact@daydaymoney.com` 在资料页绑定 `18959264502` 后仍显示未绑定，并与登录后手机门禁互相跳转。根因有两层：

1. 前端 POST `/api/accounts/users/profile/bind-phone/` 被 Go ServeMux 尾斜杠子树匹配到 `POST /api/accounts/users/profile/`（`handlePublicUpsertProfile`），返回 200 `{ok:true}` 且不写 `auth_login_method`。
2. 该号码已绑定在另一活跃账号（仅手机注册、无邮箱），即使路由正确也会 409；此前假成功掩盖了占用。

目标：资料页绑定/换绑必须走 `handleBindPhone`；占用时明确 409；用户完成短信验证后可确认将号码转移到本账号。

## 范围与边界

- 范围内：`handleBindPhone` 路由别名、409 `phone_taken` + `reclaim`、资料页错误/转移按钮、仅 `bound:true` 才回跳。
- 范围外：账号合并、无短信的单独解绑、超管强制解绑 UI。

## 约束与风险

- 日志禁止输出完整手机号；可记 `user_id` / `from_user_id` / `reclaim`。
- 409 不得消耗短信验证码，以便确认转移时复用同一验证码。
- 转移只在短信校验通过后作废对方活跃绑定。

## 验收标准

1. `POST /api/accounts/users/profile/bind-phone/` 与 `replace-phone/` 匹配专用 pattern，不再落入 profile upsert。
2. 号码被其他活跃账号占用且未 `reclaim` → 409 `code=phone_taken` `reclaim_available=true`，验证码仍可用。
3. 同一验证码 `reclaim=true` → 200 `bound=true`，原账号 `binding_voided_at` 非空，当前账号持有该号。
4. 前端假成功 `{ok:true}` 不回跳；409 展示「确认将号码转移到本账号」。

## 业务意图 → 事件对照

| 意图 | 事件名 | 发布点 | 消费者 | MQ类型/契约 |
|------|--------|--------|--------|-------------|
| 资料页绑定/转移手机号 | 无新事件 | `handleBindPhone` 写 `auth_login_method` | KYC `maybeEvaluateKycAfterPhoneVerified` | 例外：存量 bind 路径本就不发新领域事件；身份事实已由 login_method 行表达。证据豁免：`auth-profile-phone-bind-no-new-event` |
