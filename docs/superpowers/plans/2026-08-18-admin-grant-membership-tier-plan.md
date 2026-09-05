# 实施计划 — 管理端赠送页修改 VIP 等级

- **日期**: 2026-08-18
- **设计 / NFR / DDD**: 见同前缀 specs/plans

不新增 Kafka（意图例外已书面）。

## Task 1 — DDL：admin_tier_locked

- 改：`dataMigrate/taskBill/046_membership_admin_tier_locked.sql`
- 验证：`python3 -m py_compile` 不适用；SQL 经 migrate 测试库应用

## Task 2 — 后端：设等级 + 锁定（Red→Green）

- 测：`taskBill/src/admin_grant_membership_test.go`
- 改：`membership.go`、`admin_grant.go`、`handlers_admin_grant.go`
- 验证：`go test ./src -count=1 -run 'AdminGrantMembership|SyncMembership.*Lock'`

## Task 3 — 自动升级尊重 lock

- 测：locked normal 不升；未 lock 仍升
- 改：`syncMembershipConsumption`

## Task 4 — 前端：VIP 区块

- 测：`taskFE/app/src/views/SystemAdminGrantPoints.membership.unit.test.js`
- 改：抽出 `GrantPointsMembershipSection.vue`；`SystemAdminGrantPoints.vue` ≤500
- 验证：`npx vitest run src/views/SystemAdminGrantPoints.membership.unit.test.js src/views/SystemAdminGrantPoints.region.unit.test.js`

## Task 5 — 意图与文档

- `docs/intents/INDEX.md` B-049c
- 价值流图补测试点

## 验证命令

```bash
cd /tmp/ram-work/taskBill && go test ./src -count=1 -run 'AdminGrantMembership|AdminGrantGitlab|AdminGrantTaskPost'
cd /tmp/ram-work/taskFE/app && npx vitest run src/views/SystemAdminGrantPoints.membership.unit.test.js src/views/SystemAdminGrantPoints.region.unit.test.js
```
