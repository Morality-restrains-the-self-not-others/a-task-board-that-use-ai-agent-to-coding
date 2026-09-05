# 测试意图：管理员模拟用户登录（后端）

- **对应功能意图**: `admin_user_impersonation.intent.md`
- **日期**: 2026-08-23

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 无平台权限 | 403，不写会话表 |
| T2 | super_admin 模拟普通活跃用户 | 200，token 解析为 target |
| T3 | 模拟自己 | 409 |
| T4 | 目标停用或归档 | 422 |
| T5 | 员工模拟超管 | 403 |
| T6 | 超管模拟超管 | 200 |
| T7 | 已在模拟中再次开始**另一用户** | 409 |
| T7b | 同一目标、不同 Idempotency-Key（含已持有模拟 token） | 200，同一 token，redirect 非系统管理页 |
| T8 | 相同 Idempotency-Key 重放 | 200，同一 token |
| T9 | stop 非模拟会话 | 409 |
| T10 | start 后 stop | actor token 恢复 |
| T11 | 过期会话 token | 鉴权失败 |
| T12 | 事件：start/stop 各投递一次 |
| T13 | 无 reason | 400 |
| T14 | 成功写信 | 收信箱含理由；同一 session 不重复 |

## 回归

- 原登录 / activate-session / system-admin 用户列表不受影响。
