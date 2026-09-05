# 测试意图：超管用户列表最后活跃时间列

## 测试目标

表头与行单元格正确展示 `last_login`。

## 测试分层

| 层 | 位置 |
|----|------|
| 行组件 | `taskFE/app/src/components/UserListRow.test.js` |
| 页面表头 | `taskFE/app/src/views/SystemAdminUsers.last-active-column.test.js` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| 有 last_login | 单元格含本地化日期 |
| 空 / 缺字段 | — |
| 表头 | 「最后活跃时间」在「注册时间」之后 |

## 通过标准

上述测例全绿。
