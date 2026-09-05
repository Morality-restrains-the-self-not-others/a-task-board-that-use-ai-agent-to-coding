# 测试意图：管理端分账展示隔离

| ID | 场景 | 期望 |
|----|------|------|
| F1 | `profitSharing=null` | 无「分账」标题 |
| F2 | `profitSharing=[]` | 「本订单无分账记录」 |
| F3 | 有一行 | 可见 receiver id 与金额 |
| F4 | `useAdminOrderRowExpand` | 请求 `/api/system-admin/orders/{id}/` 而非租户 GET |

映射：`OrderExpandDetail.profitSharing.test.js`、`useAdminOrderRowExpand.test.js`
