# 测试意图：测试角色后端标志与网关头

- **对应功能意图**: `auth_user_tester_flag.intent.md`
- **日期**: 2026-08-23

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| B1 | PATCH is_tester=true | 行 is_tester=1 is_tenant=1 |
| B2 | 过滤器 tester | 命中 is_tester 用户 |
| B3 | forward-auth | 头 X-User-Is-Tester=1 |
| B4 | 成功 PATCH | 发布 UserTesterFlagChanged |

## 通过标准

Go 单测全绿。
