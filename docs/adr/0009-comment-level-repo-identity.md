# ADR-0009: 仓库 Git 身份与授权随评论持久化

- **Status:** accepted
- **Date:** 2026-08-16
- **Author:** cursor
- **Deciders:** goal-mode（用户指令：关联项目只展示；评论级身份在添加评论时确定）

---

## Context

任务仍关联一组项目/仓库（`task_projects`），但运行时已是一评论一容器（评论级 CSC、ADR-0005 评论级令牌）。Git 提交身份却写在 `task_repo_identities`，GitHub 选用账号写在 `cloud_task_repo_github_bindings`，UI 挂在「关联项目」。并行评论 last-write-wins，且克隆进度被误放在任务级面板。

## Decision

We will treat **run-time repo identity** as part of the Comment aggregate:

1. `task_comments.repo_identities_json` 保存本次运行每仓的 `git_identity_id` 与可选 `github_user_id`。
2. 仅「提交并运行」（有 image mention）**可**带 `repo_identities`；缺身份/未绑 OAuth **不**拦截发评。composer 提示绑定状态；私有仓缺凭证在克隆阶段正确失败。
3. 关联项目 UI 只展示/编辑任务绑定（项目、仓库、基准分支），不操作身份、授权账号、克隆进度。
4. OAuth **连接**保持用户级；评论只记录选用哪个已连接账号。
5. `TASK_COMMENT_IMAGE_MENTIONED` 与 `container-snapshot?comment_id=` 以评论 JSON 为准；无 JSON 的旧评论 advisory 回退任务级表。
6. 新运行评论 **不** 双写任务级 identity/binding 表。

DDL 只放 `dataMigrate/taskTaskService/`。

## Alternatives Considered

### Alternative 1: 只搬 UI，仍写任务级表

- **Pros:** 改动小，clone 路径不用动。
- **Cons:** 并行评论覆盖；与评论级容器矛盾。
- **Why rejected:** 用户要求身份在添加评论时确定，且启动已是评论级。

### Alternative 2: 独立表 `task_comment_repo_identities`

- **Pros:** 可索引、易 JOIN。
- **Cons:** 本增量每评论仓库数少；与 `mentions_json` 模式不一致。
- **Why rejected:** 先 JSON；行数上来再分表（冷热元规则允许后续拆）。

### Alternative 3: 新事件名 COMMENT_REPO_IDENTITIES_SELECTED

- **Pros:** 意图更纯。
- **Cons:** 身份与「提交并运行」同一事务，拆事件增加消费者时序。
- **Why rejected:** 增补既有 `TASK_COMMENT_IMAGE_MENTIONED`。

## Consequences

### Positive

- 每条运行评论的署名与授权与其容器一致。
- 关联项目恢复为任务结构展示，认知负担下降。

### Negative / Trade-offs

- snapshot / start-vm 必须带 `comment_id` 才能拿到正确身份。
- 任务级表变成遗留回退，两套读取逻辑短期并存。

### Mitigations

- Expand/Contract：读路径 comment JSON → 任务级表。
- UI 停写任务级表，避免继续污染。
- 预填用最近评论 JSON，降低重复选择。

## References

- 设计：`docs/superpowers/specs/2026-08-16-comment-level-repo-identity-design.md`
- ADR-0005 评论级容器令牌
- 意图：`docs/intents/frontend/task_detail/034_comment_level_repo_identity.intent.md`
