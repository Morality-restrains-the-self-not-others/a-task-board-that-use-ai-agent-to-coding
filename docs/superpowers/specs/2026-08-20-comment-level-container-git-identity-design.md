<!-- markdownlint-disable MD013 MD060 -->

# 设计文档：容器拉取评论级 Git 提交者身份

- **日期**: 2026-08-20
- **作者**: cursor
- **迭代**: comment-level-container-git-identity
- **状态**: accepted（/goal 自动采纳）
- **前置**: v83 `2026-08-16-comment-level-repo-identity-design.md`（已 accepted）
- **python_api_approval**: n/a（零新增 Python 接口）
- **架构**: 不写新 target（无组件增删；补齐 v83 D10 消费缺口）

---

## 0. 目标与成功标准

容器 `task-detail` / `repo-clone-credentials` / layer OAuth 拉取提交者身份时，**必须使用该评论** `task_comments.repo_identities_json` 选定的 `git_identity_id`（解析为 name/email）。禁止再用空的任务级表去「把作者全部身份摊到每个仓」。

| # | 标准 | 验收 |
|---|------|------|
| S1 | `FetchRepoIdentities(taskID, commentID)` 有评论 JSON 时返回评论选定身份 | SQLite 单测：任务级 gid-task vs 评论 gid-comment → 返回 comment |
| S2 | 评论 JSON 空/缺列时 advisory 回退 `task_repo_identities` | 单测 fallback |
| S3 | `repo_git_identities` 带 user_name / user_email | snapshot 或 credential 解析 `task_git_identities` |
| S4 | 评论已选身份时 **禁止** `FetchUserGitIdentities` 全量覆盖 | SelectIdentities 保留 per-repo 绑定 |
| S5 | clone credentials / layer oauth 传入 token.CommentID | 调用点带 commentID |
| S6 | 无新 Python 接口；无新 path | 扩展既有端口 |

## 1. 根因

v83 已让 `container-snapshot?comment_id=` 优先评论 JSON，但 **taskCredentialService 生产路径走直连 MySQL**：

- `SQLiteBusinessRepository.FetchRepoIdentities(taskID)` 只读 **`task_repo_identities`**（关联项目已停写）
- `SelectIdentitiesForCommentAuthor` 在任务级无匹配时，把作者 **全部** `task_git_identities` 摊到每个仓
- HTTP fallback 的 `FetchRepoIdentities` 恒返回 nil
- 容器 `task-detail` 的 `repo_git_identities` 因此不是评论 composer 所选身份

## 2. 方案（采纳）

| 决策 | 内容 |
|------|------|
| D1 | 端口改为 `FetchRepoIdentities(taskID, commentID)` |
| D2 | commentID 非空且 JSON 有项 → 用评论；否则任务级表 |
| D3 | 用 `task_git_identities` 解析 name/email（与现 lookup 相同） |
| D4 | snapshot JSON 增补 `user_name`/`user_email`（HTTP fallback 可直接用） |
| D5 | 已有 per-repo `git_identity_id` 时不再用作者全量身份覆盖 |
| D6 | 无新事件名；只读查询无 MQ |

拒绝：容器再调 cloud `layer-git-repo-identities/prepare` 当主路径（现网提交者身份来自 credential `task-detail`）。

## 3. 🕸️ Code Review Graph

| 项 | 内容 |
|----|------|
| skip 理由 | `unavailable` — empty graph；改用源码检索 |

## 4. 业务意图 → 事件

| 业务意图 | 事件 | 例外 |
|---------|------|------|
| 容器拉取提交者身份 | — | 纯查询；身份已在发评时随 `TASK_COMMENT_IMAGE_MENTIONED` 落库 |

## 5. 架构变更影响

无新文件。current 保持 v85。v83 消费缺口在应用层补齐。
