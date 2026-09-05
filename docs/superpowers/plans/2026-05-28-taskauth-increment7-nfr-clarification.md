# taskAuth Increment 7 — NFR

| 属性 | 级别 | 场景 |
|------|------|------|
| 可用性 | L3 | taskAuth 宕机 → 503，禁止 silent fallback 写 default |
| 一致性 | L2 | auth 表仅 auth.db；User sync-user 最终一致 |
| 安全 | L3 | drop 脚本需备份；internal secret 校验 |
| 可维护性 | L2 | fallback 仅 TASKAUTH_ENABLED=false（测试） |
