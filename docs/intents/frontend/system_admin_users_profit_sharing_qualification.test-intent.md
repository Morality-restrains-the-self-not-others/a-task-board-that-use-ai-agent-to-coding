# 测试意图：用户列表分账资格列

## 测试目标

表头位置与单元格三态。

## 测试分层

Vue 组件单测（vitest + vue-test-utils）。

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 挂载 SystemAdminUsers | 「是否获得分账资格」在「角色」之后、「操作」之前 |
| T2 | `has_profit_sharing_qualification: true` | 单元格「是」 |
| T3 | `false` | 「否」 |
| T4 | 缺省 / null | 「—」 |

## 通过标准

T1–T4 全绿。无新事件（只读）。
