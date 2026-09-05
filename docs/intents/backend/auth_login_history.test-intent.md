# 测试意图：auth_login_history

## 可执行测试

- `taskAuth/domain/login_history_test.go`
- `taskAuth/src/login_history_store_test.go`
- `taskAuth/src/login_history_handlers_test.go`
- `taskAuth/src/auth_login_test.go`（扩展：客户/管理员入口写行）
- `taskAuth/src/handlers_impersonation_test.go`（扩展：不写目标历史）

## 用例

| ID | 断言 |
|----|------|
| T1 | 客户密码登录成功 → 一行 `entry=customer` 且 IP 为 XFF 链上最右侧公网地址（跳过 RFC1918/Docker NAT） |
| T2 | 管理员密码登录成功 → 一行 `entry=admin` |
| T3 | 登录失败（错密码）→ 0 行 |
| T4 | GET self 只返回自己的行，含 `entry_label` |
| T5 | 用户 A 带 B 的 id 仍只看到自己（路径无 user_id） |
| T6 | 超管可列出目标用户；普通用户调 admin API → 403 |
| T7 | 模拟登录不增加目标用户历史 |
| T8 | `limit>100` 被夹到 100 |
