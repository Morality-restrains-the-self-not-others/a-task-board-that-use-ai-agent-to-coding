# 价值流：登录历史

- **Date:** 2026-08-25
- **Design:** `docs/superpowers/specs/2026-08-25-login-history-design.md`

Mapping the approved design into a value stream.

## 既有流

`conf/value-stream.yaml` 的 `user-auth`。本增量挂入该域。

## 增量切片

| 序 | 增量 | 用户可感知价值 | 测试 |
|----|------|----------------|------|
| 1 | 表 + 领域校验 + 登录写入 | 用户/管理员入口登录后有 IP 记录 | T1–T3, T7 |
| 2 | 用户 GET + 侧边栏页 | 账号中心能看自己的历史 | T4–T5, F1–F4 |
| 3 | 超管 GET + 用户行链接 | 超管能看指定用户 | T6, T8, F5 |

## YAML 步骤（写入 user-auth）

- `login-history-record`
- `login-history-self-view`

字段：`task-auth.auth_login_history.id` / `user_id` / `client_ip` / `entry`
