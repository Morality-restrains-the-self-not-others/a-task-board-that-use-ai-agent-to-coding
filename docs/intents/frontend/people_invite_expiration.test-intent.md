# 测试意图 — 租户邀请链接有效期加长（前端）

- **对应意图:** `people_invite_expiration.intent.md`

## 测试目标

确认邀请页有效期选项覆盖季度/半年/一年，默认与重发均为 90 天。

## 测试分层

- 单元：`peopleInviteExpiration.test.js`、`InviteLinkMethodPanel.test.js`、`PendingInvitations.click-guard.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 选项常量 | 含 90、180、365，最大 365，默认 90 |
| T2 | 有效期 select | 可见「90天」「180天」「365天」 |
| T3 | 重发邀请 | POST body `expiration_days` 为 90 |

## 通过标准

上述单测全绿。
