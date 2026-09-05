# Ship notes — rbac-logical-resource-group-v72 P1

- **Date:** 2026-08-11
- **Status:** code + migrate ready；待精准编译重启与提交/PR

## Pre-launch

1. ✅ `dataMigrate/taskAuth/032_logical_resource_groups.sql` 已应用到 `task_auth`
2. ⏳ http://10.2.150.68:9999/ 「精准编译重启」（已登记 task-auth / taskFE）
3. ⏳ 公网验收 → OPT-20260811-030

## Rollback

- 回滚 taskAuth/taskFE 二进制至上一版；表可保留（只增不破坏）
- 侧栏仍支持粗码回退，旧会话不致全盲

## Feature flag

- 无独立开关；未重启前旧 PDP 不读新表，行为与 v63 一致

## Commit / PR

- 本会话未自动 git commit（需用户明确要求后再提交）
