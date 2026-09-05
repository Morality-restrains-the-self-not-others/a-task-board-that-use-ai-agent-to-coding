# Value Stream: 修复 git-service 启动 mkdir 权限拒绝

> Derived from design: `.claude/skills/1-brainstorming-设计文档/design.md`

## Value Summary

runAll 启动 git-service 时不再因 `mkdir: Permission denied` 而失败，服务可正常达到健康状态。

## Related Value Streams

- **gitlab-oauth-app-bootstrap-fix** (2026-06-22): 相关 — 该流修复了 bootstrap 脚本的 OAuth 同步逻辑，本流修复 bootstrap 标记目录的权限问题。两者触及同一 `run_bootstrap_if_needed()` 函数。
- **runall-startup-race-eaddrinuse-fix** (2026-06-22): 无关 — 同为 runAll 启动问题，但修复不同机制。

## End-to-End Flow

[runAll 触发 git-service DAG 启动] → [run.sh ensure_container 检测容器状态] → [NEED_BOOTSTRAP=false 提前返回] → [服务就绪]

（若 NEED_BOOTSTRAP=true）： → [在用户可写目录创建 .bootstrap_marks/] → [执行 bootstrap 脚本] → [写入标记文件] → [服务就绪]

## Value Increments

### Increment 1: 修复 mkdir 权限拒绝（唯一增量）

**Value to user:** 无论 gitlab_home/ 属主是否为 root，git-service 均可正常启动。

**Scope:**
- `gitService/run.sh:161` — `BOOTSTRAP_MARKS_DIR` 从 `./gitlab_home/bootstrap_marks` 改为 `./.bootstrap_marks`
- `gitService/run.sh:162` — `mkdir -p` 移至 `NEED_BOOTSTRAP` 守卫条件之后
- `gitService/run.sh:350` — `stop --clean` 路径同步更新

**Depends on:** nothing

## Fields Impact

无数据字段变更。纯宿主机脚本路径修复，不涉及数据库或 API。

## Test Impact

无自动化测试 — 手动验证（见设计文档第 9 节）。
