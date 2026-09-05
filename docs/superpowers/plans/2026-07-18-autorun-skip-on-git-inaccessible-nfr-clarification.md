# NFR 澄清：Git 不可用时自动运行软跳过

**日期**: 2026-07-18  
**设计**: `docs/superpowers/specs/2026-07-18-autorun-skip-on-git-inaccessible-design.md`  
**价值流**: `docs/superpowers/plans/2026-07-18-autorun-skip-on-git-inaccessible-value-stream.md`

| 类别 | 级别 | 量化 |
|------|------|------|
| 安全 | L2 | 探测失败 fail-closed；不泄露凭证 |
| 可用性 | L2 | 任务保存成功；响应含中文 skip_reason |
| 性能 | L2 | 每关联仓 ≤1 次 internal GET；超时 ≤15s |
| 可观测 | L2 | 日志含 task_id + skip reason（无 token） |
| 一致性 | L2 | create / update / force_auto_run 同策略 |

## 质量场景

| ID | 场景 | 期望 |
|----|------|------|
| QS-01 | nested error 含「未检测到可用授权」 | 不 schedule |
| QS-02 | nested error 含「无法访问父仓库」 | 不 schedule |
| QS-03 | nested error 空、repos 空 | schedule |
| QS-04 | project 服务 5xx / 超时 | 不 schedule |
