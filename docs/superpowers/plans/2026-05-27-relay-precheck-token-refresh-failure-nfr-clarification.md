# NFR: relay 预检 token 换发失败分流

| 类别 | 等级 | 目标 |
|------|------|------|
| 可用性 | L2 | token 换发失败 100% 返回可执行提示（含 git host） |
| 可观测性 | L2 | 502 响应含 `error_code` + `trace_id` + `token_refresh_failures` |
| 性能 | L2 | 预检 P95 ≤ 800ms（不变） |
| 安全 | L2 | 不泄露 refresh token；detail 截断 |

**领域影响：** `TaskRepoCloneCredentialsGuardService` 需区分 coverage gap 与 upstream refresh failure。
