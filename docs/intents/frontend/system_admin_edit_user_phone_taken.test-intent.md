# 测试意图：编辑用户手机号占用内联错误

## 测试目标

保存已被占用的手机号时，编辑弹窗保持打开并在手机号字段下展示带 `data-traceId` 的错误。

## 测试分层

| 层 | 位置 |
|----|------|
| composable | `taskFE/app/src/composables/useSystemAdminUsers.edit-phone-taken.test.js` |
| 页面 | `taskFE/app/src/views/SystemAdminUsers.edit-phone-taken.test.js` |
| 页面 | `taskFE/app/src/views/SystemAdminUsers.add-phone-taken.test.js` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| PUT 409 phone_taken | 弹窗不关；`edit-phone-error` 含「该手机号已被其他用户使用」且 `data-traceId` 等于响应 trace |

## 通过标准

上述测例全绿。
