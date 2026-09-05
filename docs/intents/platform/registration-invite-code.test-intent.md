# 平台注册邀请码 — 测试意图

对应：`registration-invite-code.intent.md`

## 单元 / 集成（taskAuth）

| ID | 场景 | 期望 |
|---|---|---|
| T1 | 关闭策略时 email_register 无 invite_code | 201 成功 |
| T2 | 开启时无 invite_code | 400 `invite_code_required` |
| T3 | 开启时无效码 | 400 `invite_code_invalid` |
| T4 | 开启时已使用码 | 400 `invite_code_used` |
| T5 | 有效码注册 | 201 且码 status=used、redeemed_by=新用户 |
| T6 | 每日配额耗尽后 apply | 429/400 `daily_quota_exhausted` |
| T7 | 跨日配额重置（mock 时钟或改 issued_day） | 可再申请 |
| T8 | 非 superuser 改策略 | 403 |
| T9 | 并发 redeem 同码 | 仅一成功 |
| T10 | 事件 ISSUED/REDEEMED/POLICY_UPDATED | 成功路径 publish（可 mock kafka） |

## 前端 / Playwright（可选冒烟）

| ID | 场景 | 期望 |
|---|---|---|
| F1 | system-admin 开关+配额保存 | 刷新后仍在 |
| F2 | profile 申请并列表展示 | 新码出现 unused |
| F3 | register 开启时必填邀请码 | 空提交拦截 |
| F4 | login add_account 字段写入 sessionStorage | register 预填 |
