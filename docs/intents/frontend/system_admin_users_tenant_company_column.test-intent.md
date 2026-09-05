# 测试意图：系统管理员用户列表所属租户公司列

## 测试目标

表头与行单元格正确展示 `tenant_companies`。

## 测试分层

| 层 | 位置 |
|----|------|
| 行组件 | `taskFE/app/src/components/UserListRow.test.js` |
| 页面表头 | `taskFE/app/src/views/SystemAdminUsers.tenant-company-column.test.js` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| 一个公司 | 显示 name |
| 两个公司 | 顿号拼接 |
| 空数组 / 缺字段 | — |
| 表头 | 含「所属租户公司」，位于「邮箱」之后 |

## 通过标准

上述测例全绿。
