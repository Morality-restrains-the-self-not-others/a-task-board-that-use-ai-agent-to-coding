# 测试意图：管理端赠送页可修改租户 VIP 等级

## 覆盖

| 场景 | 测试 | 期望 |
|------|------|------|
| 设为 vip1 并锁定 | `taskBill/src/admin_grant_membership_test.go` `TestAdminGrantMembershipSetsVip1AndLocks` | tier=vip1, locked=1 |
| 非法 tier | `TestAdminGrantMembershipRejectsInvalidTier` | error 含 invalid |
| 仅调级无资源 | `TestAdminGrantMembershipOnlyNoOrder` | 成功且无新 admin_grant 订单 |
| 不传 tier 不改会员 | `TestAdminGrantResourcesLeavesMembershipUnchanged` | 仍为 normal |
| locked 不自动升级 | `TestSyncMembershipConsumptionSkipsWhenAdminLocked` | 仍 normal |
| 未 lock 仍自动升级 | `TestSyncMembershipConsumptionUpgradesUnlockedNormal` | vip1 |
| 前端下拉与 POST | `SystemAdminGrantPoints.membership.unit.test.js` | 选 VIP1 后 body 含 membership_tier；不修改则不含 |
| 默认数量为空、仅 VIP 不赠帖 | `SystemAdminGrantPoints.explicit-quantity.unit.test.js` | 数量输入默认空；仅选 VIP1 时 POST `resources: []` 且含 Idempotency-Key |

## 命令

```bash
cd /tmp/ram-work/taskBill && go test ./src -count=1 -run 'AdminGrantMembership|SyncMembershipConsumption'
cd /tmp/ram-work/taskFE/app && npx vitest run src/views/SystemAdminGrantPoints.membership.unit.test.js
```
