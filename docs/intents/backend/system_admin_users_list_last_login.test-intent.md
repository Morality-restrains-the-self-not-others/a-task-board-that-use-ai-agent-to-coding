# 测试意图：超管用户列表 last_login 回填与登录 touch

## 测试目标

列表契约与登录成功写入 `auth_user.last_login`。

## 测试分层

| 层 | 位置 |
|----|------|
| touch 纯函数/SQL | `taskAuth/src/auth_last_login_test.go` |
| 列表 | `taskAuth/src/handlers_system_admin_users_test.go` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| 库中有 last_login | 列表 JSON 带回该时间 |
| 库中 NULL | last_login 为空串 |
| 密码登录成功 | last_login 被写入 |
| 空 userID touch | 不 panic |

## 通过标准

上述测例全绿。
