# 测试意图：系统管理用户「测试」角色

- **对应功能意图**: `system_admin_user_tester_role.intent.md`
- **日期**: 2026-08-23

## 测试目标

验证筛选、展示、勾选、JSON 与事件。

## 测试分层

- 前端 Vitest：筛选项、角色徽章、编辑表单勾选。
- 后端 Go：过滤器 SQL、PATCH `is_tester` 强制租户、forward-auth 头、事件名。

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| F1 | 角色下拉 | 含 value=tester 标签「测试」 |
| F2 | 选测试筛选 | GET 带 `role=tester` |
| F3 | is_tester 用户行 | 徽章「测试」 |
| F4 | 编辑勾选测试并保存 | PATCH body 含 is_tester true |
| B1 | role=tester | WHERE is_tester=1 |
| B2 | role=tenant | is_tenant=1 且 is_tester=0 |
| B3 | PATCH is_tester=true | 同时 is_tenant=1；发布 UserTesterFlagChanged |
| B4 | 测试用户 forward-auth | X-User-Is-Tester=1 且 X-User-Roles 不含 tester |
| B5 | 事件投递 | 成功路径调用 publish UserTesterFlagChanged |

## 通过标准

上述用例全绿。
