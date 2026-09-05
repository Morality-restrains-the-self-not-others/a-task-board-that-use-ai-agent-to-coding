# 测试意图：邮件邀请退订

- **对应意图:** `email-invite-unsubscribe.intent.md`
- **日期:** 2026-08-29

## 测例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | HMAC token 规范化邮箱大小写 | `A@B.C` 与 `a@b.c` 同一签名邮箱 |
| T2 | 无效/篡改 token | 4xx，不写入表 |
| T3 | 重复退订同一邮箱 | 幂等成功，一行 |
| T4 | 已退订超管邀请 | 201 + `email_skipped`，不 SMTP |
| T5 | 已退订租户邮箱邀请 | 201 + `email_skipped` + `invitation_url` |
| T6 | 未退订租户邮箱邀请 | 行为与改造前一致（仍排队发信） |
| T7 | taskEvents 邀请投递遇退订 | Success + `skipped_unsubscribed`，不 SMTP |
| T8 | 验证码邮件 | 无退订链接、不被列表拦截 |
| T9 | 前端确认页 | `/auth/unsubscribe/?ok=1` 展示已退订 |
| T10 | 前端邀请人 | `email_skipped` 时展示复制链接，不 toast「已发送」 |
