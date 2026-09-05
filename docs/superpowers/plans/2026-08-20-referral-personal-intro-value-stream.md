# 价值流 — 推荐资格个人介绍

- **Date:** 2026-08-20
- **Design:** `docs/superpowers/specs/2026-08-20-referral-personal-intro-design.md`

## Related Value Streams

Greenfield 于「申请材料」增量。依赖既有推荐资格申请/审批流（`docs/intents/backend/task-referral.intent.md`），不撤销 accessCode 与资格解耦。

## 价值增量

| # | 增量 | 用户可见结果 | 验证 |
|---|------|--------------|------|
| 1 | 申请表强制个人介绍 | 未填/少于 20 字无法提交；名额有限说明可见 | UserReferral.applyIntro.test.js |
| 2 | 申请写入 `personal_intro` | pending 状态与超管列表看到原文 | applyReferralCode 单测 + 列表 Scan |
| 3 | 非法介绍 400 | 空/短/超长 → `invalid_intro` | normalizePersonalIntro / handler 单测 |

触发用户：已登录普通用户申请资格；平台超管审批。
