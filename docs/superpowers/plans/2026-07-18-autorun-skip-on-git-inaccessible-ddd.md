# DDD 裁剪说明：Git 不可用时自动运行软跳过

**日期**: 2026-07-18  
**设计**: `docs/superpowers/specs/2026-07-18-autorun-skip-on-git-inaccessible-design.md`

## 结论

无新聚合根 / 仓储 / 领域事件。属既有 **Task.auto_run** 应用服务策略扩展：

- **领域规则**：`AutoRunMayPersistWithoutStart` — 当关联仓库 Git 可达性探测失败时，允许 `auto_run=true` 持久化，但禁止触发云启动用例。
- **端口**：复用 ProjectService `listNestedGitRepos`（internal HTTP 适配器）。
- **事件例外（书面）**：本切片不新增 MQ 业务事件；既有 `TASK_CREATED` 仍在创建成功时投递，与是否 start-vm 无关。

## 术语

| 术语 | 含义 |
|------|------|
| 软跳过 | 保存 auto_run，跳过 start-vm |
| Git 可达性探测 | 对关联仓调用 nested-git-repos，以 error 非空判定不可用 |
