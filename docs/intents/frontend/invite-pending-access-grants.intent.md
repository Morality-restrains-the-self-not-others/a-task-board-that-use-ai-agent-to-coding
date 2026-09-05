# 意图：邀请时预授访问管理等页面/区域权限

- **状态:** active
- **日期:** 2026-08-11
- **设计:** `docs/superpowers/specs/2026-08-11-invite-pending-access-grants-design.md`

## 用户故事

作为拥有成员管理权限的租户管理员，我在「邀请人」页面发送邀请时，希望能同时为受邀人勾选页面组/UI 区域权限（含「访问管理」），以便对方加入公司后即可使用对应能力，无需再去「访问管理」单独配置。

## 验收标准

1. 邀请表单展示与访问管理同源的 page/region 勾选（view/operate）。
2. 提供「授予访问管理」快捷勾选 `people.access` 整页。
3. 邀请创建时 `pending_grants` 持久化；成员角色为 admin 时不持久化预授。
4. 受邀人 join 成功后，自定义访问角色已绑定且 PDP 含对应 page/region 码。
5. 未勾选任何区域时行为与改造前一致。

## 业务事件

| 意图路径 | 事件 | Topic |
|----------|------|-------|
| 创建带 grants 的邀请 | INVITATION_CREATED（payload 含 grants） | invitation-created |
| 接受邀请并落权 | MEMBER_JOINED（payload 可含 grants） | member-joined |
