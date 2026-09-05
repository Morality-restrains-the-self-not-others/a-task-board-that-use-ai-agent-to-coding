# 测试意图：系统管理编辑用户手机号/邮箱占用

## 测试目标

PUT 占用中的手机号必须 409；空闲号码必须写入；前端编辑弹窗展示带 traceId 的内联错误。

## 测试分层

| 层 | 位置 |
|----|------|
| 后端 | `taskAuth/src/handlers_system_admin_user_write_phone_test.go` |
| 页面 | `taskFE/app/src/composables/useSystemAdminUsers.edit-phone-taken.test.js` |
| 页面 | `taskFE/app/src/views/SystemAdminUsers.edit-phone-taken.test.js` |
| 页面 | `taskFE/app/src/views/SystemAdminUsers.add-phone-taken.test.js` |
| 基础设施 | `db/load/testdb_clone_test.go`（生成列克隆，保证 taskAuth 测例能建库） |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| PUT 他人已绑手机号 | 409 `phone_taken`，目标原号码不变 |
| PUT 空闲手机号 | 200，login method 为新号码 |
| PUT 空 `phone` 解绑 | 见 `system_admin_unbind_user_phone.test-intent.md` |
| PUT 他人邮箱 | 409 `email_taken` |
| 编辑弹窗保存占用号码 | 弹窗不关，`[data-testid=edit-phone-error]` 含「已被其他用户使用」，带 `data-traceId` |

## 通过标准

上述测例全绿。
