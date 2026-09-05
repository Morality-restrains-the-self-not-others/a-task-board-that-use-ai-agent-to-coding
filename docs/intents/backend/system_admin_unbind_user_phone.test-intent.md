# 测试意图：系统管理编辑用户解绑手机号

## 测试目标

PUT 显式空 `phone` 必须作废该用户活 phone login method；省略 `phone` 不得解绑；前端解绑后保存发送空字符串。

## 测试分层

| 层 | 位置 |
|----|------|
| 后端 | `taskAuth/src/handlers_system_admin_user_write_phone_test.go` |
| composable | `taskFE/app/src/composables/useSystemAdminUsers.edit-phone-unbind.test.js` |
| 页面 | `taskFE/app/src/views/SystemAdminUsers.edit-phone-unbind.test.js` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| PUT `phone: ""` 且原有绑定 | 200，活 phone identifier 为空 |
| PUT 省略 `phone` | 原号码仍活 |
| PUT `phone: ""` 且本无绑定 | 200 |
| 点击「解绑」 | `#edit-phone` 为空 |
| 保存空手机号 | PUT body `phone === ""`，弹窗关闭 |

## 通过标准

上述测例全绿。
