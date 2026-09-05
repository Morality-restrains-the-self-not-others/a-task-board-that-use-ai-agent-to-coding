# 功能意图：特权角色一号最多绑定 5 个账号

## 背景与目标

内部超级用户、员工、测试需要用同一测试/工作手机号登录多个账号。当前 `phoneLoginMethodTaken` 对任意活跃占用一律 409，导致必须 reclaim 互抢。

目标：这三类角色在同一规范化手机号上最多 5 个活跃绑定；普通客户仍一号一户。

## 范围与边界

- 范围内：绑定占用策略、内部 phone-taken / upsert、手机密码登录消歧、手机重置消歧、资料页/登录错误码、解档冲突在「全员特权且 ≤5」时不阻断。
- 范围外：公开手机注册一号多户、登录账号选择器、邮箱/用户名共享、超管强制解绑 UI。

## 约束与风险

- 日志禁止完整手机号。
- 不得把普通客户号码静默共享给特权账号（须 reclaim）。
- 手机登录/重置不得 `LIMIT 1` 随机命中共享集合。

## 验收标准

1. 两名 `is_staff` 账号绑定同一号 → 第二次 200，两行 `binding_voided_at IS NULL`。
2. 第 6 个特权绑定 → 409 `phone_bind_limit`。
3. 普通客户绑已被员工占用的号 → 409 `phone_taken`。
4. 员工绑已被普通客户占用的号 → 409 `phone_taken`，reclaim 后客户绑定作废。
5. 已占用号走 `phone_register` → 仍 400 已注册。
6. 两账号不同密码、同一手机号登录 → 命中正确账号；相同密码 → 400 `phone_ambiguous`。
7. 共享号发重置验证码 → 400 `phone_ambiguous`。

## 业务意图 → 事件对照

| 意图 | 事件名 | 发布点 | 消费者 | MQ类型/契约 |
|------|--------|--------|--------|-------------|
| 特权账号共享绑定手机号 | 无新事件 | `handleBindPhone` / `upsertPhoneLoginMethod` | KYC `maybeEvaluateKycAfterPhoneVerified` | 例外：存量 bind 路径不发新领域事件；login_method 行即身份事实。证据豁免：`auth-privileged-phone-multi-bind-no-new-event` |
| 共享号密码登录成功 | USER_LOGGED_IN（既有） | `finalizeLogin` | 既有消费者 | 不改契约 |
