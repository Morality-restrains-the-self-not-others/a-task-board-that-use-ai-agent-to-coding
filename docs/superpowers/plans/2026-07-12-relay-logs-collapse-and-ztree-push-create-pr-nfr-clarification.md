# NFR 澄清：启动日志默认折叠 + 推送并创建PR

- **日期**: 2026-07-12
- **默认级别**: L2 Standard
- **状态**: approved (auto)

| 质量属性 | 级别 | 场景 | 对领域/实现影响 |
|----------|------|------|-----------------|
| 可用性 | L2 | 日志折叠不丢引导条 | UI 仅隐藏正文 |
| 性能 | L2 | wait_for_pr ≤ ~50s；超时仍返回 push 成功 | 轮询 get_external_job，不阻塞其他请求过久 |
| 安全性 | L3 | PR 创建仍需批准凭据 / GitHub 连接 | 不绕过 ensure_approval |
| 可靠性 | L2 | PR 失败不回滚已推送 | skipped + alert / compare_url |
| 可观测性 | L2 | 记录 wait_for_pr / job 终态日志 | LayerGitPush observation |

## 领域模型影响

无新聚合；应用服务增加可选同步等待语义。
