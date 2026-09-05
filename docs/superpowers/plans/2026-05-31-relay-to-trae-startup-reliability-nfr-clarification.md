# NFR Clarification: relayToTrae 启动可靠性（统一版）

> 设计：`docs/superpowers/specs/2026-05-31-relay-to-trae-startup-reliability-design.md`

| 类别 | 级别 | 说明 |
|------|------|------|
| 可用性 | L2 | post-listen 写配置失败在 strict 下 exit；默认非 strict 保持现状 |
| 可靠性 | L2 | register-reachability 503 退避已落地；本迭代仅补配置路径单测 |
| 可观测性 | L2 | 成功/失败日志含 `bootstrap: wrote`；`bootstrap_stage` 为可选 follow-up |
| 安全 | L3 | 不换票语义、reachability 不回退 127.0.0.1 |
| 性能 | L2 | 重试总延迟 ≤~1.7s 可接受 |
