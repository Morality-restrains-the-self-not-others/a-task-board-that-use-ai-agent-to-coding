# 价值流：管理员模拟用户登录

- **Date:** 2026-08-23
- **Design:** `docs/superpowers/specs/2026-08-23-admin-user-impersonation-design.md`

Mapping the approved design into a value stream.

## 既有流

`conf/value-stream.yaml` 的 `user-auth`（用户与认证）覆盖注册/登录/会话。本增量挂入该域，不新建域。

## 增量切片（按用户价值）

| 序 | 增量 | 用户可感知价值 | 测试 |
|----|------|----------------|------|
| 1 | 领域规则 + 会话表 + start API | 有权限管理员可签发模拟会话 | taskAuth Go 单测 T1–T8,T12 |
| 2 | stop + status + forward-auth | 可退出、横幅有状态、网关认模拟 token | T9–T11 + forward-auth 测 |
| 3 | 编辑页按钮 + 凭据落盘跳转 | 页面上能点、能进用户前台 | FE F1–F6 |
| 4 | Navbar 横幅退出 | 随时回到管理员 | FE F7–F8 |

切片 1 单独即可后端验收；3 是用户给出的页面元素交付点。

## YAML 步骤（写入 user-auth）

- `admin-impersonate-start`
- `admin-impersonate-stop`

字段：`task-auth.auth_impersonation_session.id` / `actor_user_id` / `target_user_id` / `token_key` / `ended_at`
