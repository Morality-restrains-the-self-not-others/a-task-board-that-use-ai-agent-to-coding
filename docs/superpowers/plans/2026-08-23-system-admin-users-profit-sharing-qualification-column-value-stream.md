# 价值流 — 超管查看用户是否获得分账资格

- **日期**: 2026-08-23
- **增量**: 用户列表只读列

## Related Value Streams

- `2026-08-23-system-admin-users-tenant-company-column-value-stream.md`：同一列表的只读投影列（租户公司）。本增量同模式，数据源换成 taskReferral 资格。
- 推荐码申请 tab / 审批流：写路径，本增量不改。

## 当前价值流（增量）

1. 超管打开 `/system-admin/users/`。
2. 前端 `GET /api/system-admin/users/?limit=&offset=&is_archived=`。
3. taskAuth 查 `auth_user` 分页，批量补登录方式 / 推荐人 / 租户公司。
4. **新增**：同一批 `user_id` 调 taskReferral `qualification/active/batch`，投影当前活跃分账资格。
5. 表格「是否获得分账资格」渲染 是/否；下游失败显示 —。

## 测试点

| ID | 步骤 | 断言 |
|----|------|------|
| TP-PSQ-1 | 用户 `referral_code` approved 且未过期 | 单元格「是」 |
| TP-PSQ-2 | 用户无申请 / 已拒绝 / 已取消 / 已过期 | 单元格「否」 |
| TP-PSQ-3 | taskReferral 不可达 | 列表仍 200，该列为 — |
| TP-PSQ-4 | 非超管 GET 列表 | 403 |
| TP-PSQ-5 | 表头位置 | 「是否获得分账资格」在「角色」之后、「操作」之前 |

无新 MQ 事件（只读查询）。
