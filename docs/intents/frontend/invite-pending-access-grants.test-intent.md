# 测试意图：邀请时预授访问管理等页面/区域权限

- **对应意图:** `invite-pending-access-grants.intent.md`
- **日期:** 2026-08-11

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | POST invite 带 people.access grants | 201；`pending_grants` JSON 含对应 group_key |
| T2 | POST invite role=admin 且带 grants | 201；`pending_grants` 为空/NULL |
| T3 | join 带 pending_grants 的邀请 | 成员创建；member-role=自定义；apply-member-grants 被调用 |
| T4 | invite grants 非法 effect | 400 |
| T5 | FE 快捷「授予访问管理」 | payload.grants 含 people.access 或其 region |
| T6 | FE role=admin | 不提交 grants / 矩阵禁用 |

## 自动化落点

- Go: `taskTenantService/src/invite_handlers_test.go`、`taskAuth` apply-member-grants 单测
- FE: `PeopleInvite` / `InviteAccessGrants` unit test
