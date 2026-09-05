# NFR 澄清：任务子树状态与终态门禁

| 类别 | 级别 | 说明 |
|------|------|------|
| 正确性 | L3 | 门禁不可绕过（服务端强制） |
| 性能 | L2 | subtree max_depth 默认 2；门禁 BFS 同 workspace |
| 安全 | L2 | 复用 workspace 读写检查；无新密钥 |
| 可观测 | L2 | 门禁拒绝打 info 日志含 task_id、open_count |
| 可用性 | L2 | 列名映射失败时用 completed 降级，不 5xx |
