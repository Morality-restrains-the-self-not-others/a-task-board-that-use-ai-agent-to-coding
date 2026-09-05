# 功能意图：创建/配置项目是否自动克隆子仓库

- **日期**: 2026-08-12
- **状态**: 已实施
- **设计文档**: `docs/superpowers/specs/2026-08-12-project-auto-clone-nested-repos-design.md`

## 背景与目标

容器 bootstrap 无条件 enrich `.gitmodules` 子仓并克隆。用户需要在**创建项目**时选择是否自动克隆子仓库，并在项目详情可改。

## 验收标准

1. 创建页有勾选 `create-project-auto-clone-nested-repos`（有 Git URL 时可见，默认勾选）。
2. 字段 `auto_clone_nested_repos` 持久化；默认 true。
3. false 时 credential enrich 不合并子仓；发现 API 仍可用。
4. 详情可切换；空态区分「未发现」与「已关闭自动克隆」。
5. 任务详情：@镜像后 composer 身份行展示 `task-nested-repos-auto-clone-toggle`；false 时展示 `task-nested-repos-auto-clone-off-hint`，不展示 `task-nested-repos-clone-status` 进度面板（进度在评论执行细节）。
6. **生产直连路径**（taskCredentialService `SQLiteBusinessRepository`）必须从 `project_entries.auto_clone_nested_repos` 读取开关，不得写死 true；容器 `collectRepoCloneJobs` 对 `auto_clone_nested_repos=false` 的项目跳过带 `parent_repo_url` 的子仓。
7. **auto_clone=true 时即使任务未绑定 git identity（userID=0）也必须 enrich 子仓**，不得因此只克隆元仓。GitHub 公开仓走匿名 Contents API。
8. 任务级「子仓库克隆状态」不得仅因引导结束（bootstrapCloneDone）把未出现在克隆任务中的子仓标成已完成。
9. **私有父仓 nested 发现与克隆凭证必须使用创建该评论的用户身份**（`task_comments.created_by_id`）。禁止回退 `task_tasks.owner_id`，禁止用其他用户的 `task_repo_identities` 顶替。
10. **关闭自动克隆子仓库时，子仓 OAuth 异常 / 子仓列表获取失败不得阻断项目「是否允许自动运行」**。父仓已授权即可启用；子仓发现列表仍可展示（不影响门禁）。父仓 `token_error` 仍阻断。开启自动克隆时保持原门禁（子仓 `token_error` 或 nestedError → 无法启动）。

## 业务意图 → 事件对照

| 业务意图 | 事件 | 例外理由 |
|----------|------|----------|
| 配置自动克隆子仓 | — | 项目配置 CRUD，无跨服务最终一致需求 |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-08-12 | 初版 |
| 2026-08-13 | 补任务详情关闭自动克隆时的 UI 验收 |
| 2026-08-15 | 生产 MySQL 直连路径漏读开关；镜像 collectRepoCloneJobs 增加 false 门控 |
| 2026-08-15 | 任务无 git identity 时仍 enrich 子仓（匿名发现）；任务级状态不再用 bootstrapCloneDone 假完成 |
| 2026-08-16 | 私有仓发现/克隆改用评论创建者身份；禁止 owner_id 回退 |
| 2026-08-16 | 任务详情开关迁到评论 composer 身份行 | 关联项目只读化误卸 NestedReposCloneStatus |
| 2026-08-18 | 关闭自动克隆时子仓授权异常不再阻断项目默认自动运行 |
