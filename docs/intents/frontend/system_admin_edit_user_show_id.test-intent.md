# 测试意图：编辑用户表单显示用户 ID

## 测试目标

打开编辑用户模态框后，只读字段展示该用户的字符串 ID。

## 测试分层

| 层 | 位置 |
|----|------|
| 页面 | `taskFE/app/src/views/SystemAdminUsers.edit-user-id.test.js` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| 点击编辑 | `#edit-user-id` 值为列表用户 `id` 字符串 |
| 只读 | `#edit-user-id` 为 `readonly` |

## 通过标准

上述测例全绿。
