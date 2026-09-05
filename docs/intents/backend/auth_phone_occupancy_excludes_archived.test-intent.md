# 测试意图：手机/邮箱占用仅计活跃用户

## 测试目标

证明归档/孤儿登录方式不再挡住注册，且超管能按手机号搜到归档占用者；活跃占用者行为不变。

## 测试分层

- 单元：`findLoginMethodByPhone` / `ByEmail` JOIN 活跃用户；作废陈旧绑定。
- 接口：`POST /api/accounts/users/phone_register/`；`GET /api/system-admin/users/`。

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 占用者 `is_archived=1` 后手机注册 | 201；旧 phone 绑定已作废；新占用者为新用户 |
| T2 | 删除 `auth_user` 留下 login_method | 注册 201 |
| T3 | 活跃占用者注册 | 400「该手机号已被注册，请直接登录」 |
| T4 | `is_archived=false` 且 `phone=` 或 `q=` | 结果含已归档占用者 |
| T5 | 已归档邮箱占用 | `findLoginMethodByEmail` 返回 nil |
| T6 | 活跃页无 phone/q | 默认仍排除归档用户 |

## 数据与环境

`setupAuthTestDB`；`createUserWithPhoneLogin` / `createUserWithEmailLogin`；超管 `X-User-Id: bootstrap-admin`。

## 通过标准

上述用例全部通过；无手机号原文写入日志断言字符串。
