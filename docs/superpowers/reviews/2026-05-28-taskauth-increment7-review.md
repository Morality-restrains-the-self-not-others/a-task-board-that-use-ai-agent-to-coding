# taskAuth Increment 7 Review

> 计划: `docs/superpowers/plans/2026-05-28-taskauth-increment7-plan.md`

## 结论

**可合并** — 生产路径已收口至 taskAuth API；共享库 auth 表可安全删除。

## 验证

| 项 | 结果 |
|----|------|
| `go test ./...` (taskAuth/src) | ✅ |
| taskAuth bridge pytest | ✅ 17 pass |
| `verify_auth_tables_dropped.sh` | ✅ |

## 要点

1. `send_verification_code` 已 delegate；验证码仍 Django internal 发送
2. `taskauth_enabled()` 时 delegate 失败 → **503**，禁止 fallback 写 default auth 表
3. fallback 代码保留供 `TASKAUTH_ENABLED=false` 测试环境
4. `drop_shared_auth_tables.sh` 自动备份后 DROP 两表

## 遗留（非阻塞）

- init 脚本仍 ORM 写 LoginMethod（运维场景；router 写 auth.db）
- 可后续删除 fallback 死代码（测试改 mock delegate 后）
