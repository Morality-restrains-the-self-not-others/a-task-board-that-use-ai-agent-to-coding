# 功能意图：容器 task-detail 下发最新子 Git 仓库并并发克隆

- **日期**: 2026-07-15
- **状态**: 待实施
- **设计文档**: `docs/superpowers/specs/2026-07-15-container-nested-git-repos-clone-design.md`

## 背景与目标

项目详情已能发现父仓嵌套子 Git 仓库（`project-nested-git-repos`），但容器 bootstrap 仍只克隆任务关联的**父仓**。元仓（如 `ram-work`）下数十独立子仓不会在容器工作区出现。

**目标**：容器 `POST …/task-detail/` 返回的 `project_repos[].git_repos` / `git_repo_entries` 含**最新**发现的子仓（`clone_alias`=子路径名）；`repo-clone-credentials` 为子仓 URL 返回可用 OAuth（继承父仓/任务身份）；容器**并发**克隆且限流；**nested 发现失败不阻断父仓克隆**。

## 范围与边界

### 在范围内

- `taskProjectService`：`GET /api/internal/nested-git-repos/`（服务间，复用 `listNestedGitRepos`）
- `taskCredentialService`：`FetchTaskDetail` / `BuildRepoCloneCredentials` 共用 enrich + 凭证 inherit
- `onlineServiceJS`：`BOOTSTRAP_CLONE_CONCURRENCY`（默认 8）、`cloneReposIntoSharedLayer` 池化并发
- 契约文档：`machine_container.md` §4.4

### 非目标

- 不持久化子仓到 `project_repos` 表
- 不改 Django 公网路由
- 不改 go_run_container / mock_run_container 启动链路
- 不新增 MQ 业务事件（见下表书面例外）

## 约束与风险

| 约束 | 说明 |
|------|------|
| 只读 enrich | merge 仅影响 task-detail 响应，不写 DB |
| 任务身份 | internal nested 调用使用任务 `FetchRepoIdentities` 中首个 `UserID>0` |
| 凭证继承 | 子仓无独立 identity 时复制 UserID/GitIdentityID，仅换 RepoURL |
| 失败隔离 | nested API / 单子仓 clone 失败不得 cancel 父仓 |
| 并发上限 | 默认 8，防打爆 Git 主机 |

## 验收标准

1. 关联 ram-work 类元仓的任务启动容器后，子仓出现在**父仓工作树内**对应 path（如 `ram-work/task2app/`、`ram-work/docs/`），而非仅层根并列目录。
2. `task-detail` 响应 `git_repo_entries` 含子仓 URL、`clone_alias` 为 path 名，且 nested 项含 `parent_repo_url`。
3. `repo-clone-credentials` 对子仓 URL 返回与父仓同用户的 OAuth（或 409 若任务本身缺凭证）。
4. bootstrap 日志含「并行克隆 N 仓，并发上限 C」；子仓经 staging 克隆后有「已移入 …」记录。
5. **克隆层锁定时机**：须在「并行克隆 → 子仓移入父仓 → 工作分支切换」全部结束后才密封（`bootstrapReposLayoutReady`）；此前 `task-gate.clone_done` 为 false，且引导上下文下禁止建任务叠层。
5. mock internal nested 失败时，父仓仍出现在响应且可克隆。
6. 无 nested 的普通任务行为与改前一致。
7. 子仓先落 `.bootstrap-staging/`，成功后再 move 到最终 path（覆盖空占位）。
8. 任一仓库克隆失败不抛出引导级错误：日志标明部分失败并继续 feature-params / `BOOTSTRAP_COMPLETE`；不得导致前端「容器业务端点尚未就绪」。

## 业务意图 → 事件对照

> 对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：只读 enrich + 容器侧克隆，无平台业务状态变更。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 容器 task-detail 下发最新子 Git 仓库并并发克隆 | — | — | — | — | 只读 enrich + 容器侧克隆，无平台业务状态变更 |

## 实施计划

见 `docs/superpowers/plans/2026-07-15-container-nested-git-repos-clone-plan.md`（T1–T7）。

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-15 | 初版意图 |
| 2026-07-18 | 补充：子仓 staging→移入父仓 path；`parent_repo_url` 字段（见 `2026-07-18-nested-repo-clone-relocate-design.md`） |
