# 测试意图：管理端赠送须显式填写数量

## 测试目标

锁住「只改 VIP 不会因默认 100 帖误建赠送订单」。

## 分层

- 前端单元：`taskFE/app/src/views/SystemAdminGrantPoints.explicit-quantity.unit.test.js`
- 回归：既有 membership / region 单测（region 路径须显式填数量后才 POST）

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| TQ-1 | 默认数量 | `#grant-quantity-input` 值为空 |
| TQ-2 | 仅 VIP1、不填数量 | POST `resources: []`，`membership_tier=vip1` |
| TQ-3 | 无 VIP、不填数量 | 不 POST，提示须填数量或改 VIP |
| TQ-4 | 预览 | 仅 VIP 时文案含「不会生成赠送订单」 |
| TQ-5 | 防重放 | POST headers 含 `Idempotency-Key`，body 含同值 `idempotency_key` |

## 命令

```bash
cd /tmp/ram-work/taskFE/app && npx vitest run src/views/SystemAdminGrantPoints.explicit-quantity.unit.test.js src/views/SystemAdminGrantPoints.membership.unit.test.js src/views/SystemAdminGrantPoints.region.unit.test.js
```
