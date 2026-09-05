# 下单施工留言测试意图

## 对应功能意图

`docs/intents/billing-order-buyer-note.intent.md`

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | gitlab_disk + 合法留言 | 订单落库 buyer_note，JSON 回传 |
| T2 | 仅 task_post + 非空留言 | 错误且无订单 |
| T3 | 留言 >2000 字 | 错误 |
| T4 | 空/空白留言 | 成功，buyer_note 为空串 |
| T5 | OrderCreate：选磁盘出现 textarea，POST 含 buyer_note | 组件单测 |
| T6 | OrderCreate：仅任务帖无留言框 | 组件单测 |
| T7 | OrderDetail / 超管展开展示 buyer_note | 组件单测 |

## 实现位置

- `taskBill/src/orders_buyer_note_test.go`
- `taskFE/app/src/views/OrderCreate.contract.test.js`
- `taskFE/app/src/views/OrderDetail.contract.test.js`
