# 价值流：租户退款申请

## 端到端流

```
租户管理员打开账单页 → 查看可用/冻结积分 → 点击申请退款
  → taskBill 冻结 balance→frozen，创建 pending 申请，发 Submitted 事件
  → 系统管理员打开审批页 → 通过或驳回
     → 驳回：解冻 + Rejected 事件
     → 通过：FIFO 台账原路退（PayPal/微信）+ 核销冻结 + refund 流水 + Completed 事件
  → 租户在流水中看到 refund；可用积分为 0（若全额）
```

## 测试点（同步 value-stream-test-integration）

| ID | 步骤 | 断言 |
|----|------|------|
| VS-R1 | 申请 | pending + frozen |
| VS-R2 | 消费尝试 | 余额不足（已冻结） |
| VS-R3 | 驳回 | balance 恢复 |
| VS-R4 | 批准 | ledger remaining↓；支付退款 refs；frozen=0 |

## CRG 社区对齐

计费充值社区（taskBill + billing_bridge + Billing* Vue）；无独立新社区。
