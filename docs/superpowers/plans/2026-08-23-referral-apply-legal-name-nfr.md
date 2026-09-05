# NFR：申请推荐资格个人名称

## 路径分片键强制审视

| 路径 | 分片 ID | 级别 | 说明 |
|------|---------|------|------|
| `POST /api/accounts/users/referral-codes/apply/` | user_id（会话） | L0 | 既有用户中心写接口，按用户一行申请 |
| `GET /profile/referral/` | 无 | L0 | 既有页 |

## 幂等性强制审视

| 路径 | 副作用 | 级别 | 键 |
|------|--------|------|----|
| 申请 POST | 插入申请行 | L2 | 既有 Idempotency-Key；业务上 pending 不可重复申请 |
| 添加微信接收方 | 微信侧登记 | L3 | referrer_user_id 行幂等 |

资金路径：分账仍走既有 L3 订单级 out_no。名称填错由微信拒绝，不另造补偿事件。
