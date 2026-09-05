# 评论级仓库身份与授权（关联项目只读）

- **日期**: 2026-08-16
- **作者**: cursor
- **迭代**: comment-level-repo-identity-v83
- **状态**: accepted（goal-mode 自动采纳）
- **ADR**: ADR-0009
- **意图**: `docs/intents/frontend/task_detail/034_comment_level_repo_identity.intent.md`

## 1. 问题背景

任务详情「关联项目」面板仍是**任务级操作台**：OAuth 绑定、保存 GitHub 账号、Git 提交身份、克隆进度、重新克隆、把身份同步进容器。容器启动、CSC、令牌已经是**评论级**（一评论一容器）。结果是：

- 用户在任务级配身份，却以为对所有评论生效；并行评论会互相覆盖 `task_repo_identities` / `cloud_task_repo_github_bindings`。
- 评论级克隆进度出现在任务级面板，与「该次运行」脱节。
- 评论 composer 已有执行依赖 / 镜像 / 硬件，却没有本次运行要用的提交身份与授权身份。

用户要求：关联项目只用于**展示**任务绑了哪些项目/仓库/基准分支；评论级提交身份与授权身份在**添加评论时**确定。

## 2. 对当前架构的理解

根据 `docs/architecture/` current：

- 共有 2 个 current 视图：`enterprise-landscape` v81、`application-integration` v81
- 应用层：`taskFE` TaskDetail；`taskTaskService` 拥有 `task_projects` / `task_repo_identities` / `task_comments`；`taskCloudService` 拥有 `cloud_task_repo_github_bindings`（任务级 repo→github_user_id）
- 已交付评论级 CSC / 容器令牌（ADR-0005）；克隆进度条已规划在评论执行细节（`comment_clone_progress_under_trace`）
- v82 仍为 target（终态释放评论 CSC），与本次正交

📋 架构版本历史（节选）：

- v81 (2026-08-14) ✅ current — 工作空间任务帖人读序号
- v82 🎯 target — 终态释放对齐评论级 CSC（积压）
- 本次在 v81 current 上改 **身份生命周期从 Task → Comment**，批准后写 **v83**

## 3. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 关联项目非编辑态只展示项目名、仓库 URL、基准分支；无 OAuth/保存账号/Git 身份/克隆进度/重新克隆/拉身份进容器 | `task-linked-projects-panel` 无上述控件；文案不再写「启动前需完成两步」 |
| S2 | 编辑态仍可增删任务关联项目、改基准分支（任务定义） | 现有 EditMode 保留 |
| S3 | 「提交并运行」（有 `@镜像`）时 composer 按仓库选择 Git 提交身份 + OAuth/GitHub 账号 | `data-testid="comment-composer-repo-identity"` |
| S4 | 纯评论（无运行）不强制身份，不展示身份区 | 无 mention 时无该区块 |
| S5 | 运行评论 POST 可带 `repo_identities`；落库 `task_comments.repo_identities_json` | 缺项不拦截；克隆阶段缺凭证正确失败 |
| S6 | `TASK_COMMENT_IMAGE_MENTIONED` 带 `repo_identities`；container-snapshot 优先评论身份 | 并行评论互不覆盖 |
| S7 | 克隆进度只出现在该评论执行细节，不在关联项目 | 关联项目无进度条 |
| S8 | 旧评论无 JSON 时 snapshot 回退 `task_repo_identities` | Expand/Contract |

## 4. 方案决策

| # | 决策 | 理由 |
|---|------|------|
| D1 | 关联项目 ViewMode **只读展示** + 仓库地址失配同步（任务数据修正） | 失配是 `task_projects.repo_address` 任务级事实，不是评论运行态 |
| D2 | 项目级 `auto_clone_nested_repos` 开关挂在 composer 身份行（仍 PUT 项目配置）；**克隆状态/进度**在评论执行细节 | 关联项目只读化后开关无挂载点；配置≠进度，也不写入评论 JSON |
| D3 | 身份 UI 只在 composer、且仅 `showRunConfig`（@镜像）时出现 | 与硬件/智能体配置同生命周期 |
| D4 | 新列 `task_comments.repo_identities_json`（JSON 数组），不新建分表 | 随评论写入、随评论读取；年增量随评论走，评论表已是实体爆炸型，JSON 避免每评论×仓一行的热路径 JOIN |
| D5 | JSON 项：`{repo_url, git_identity_id, github_user_id?}` | 提交署名与 GitHub App 账号分离，与现 UI 分区一致 |
| D6 | OAuth **连接**仍是用户级（账号中心 / 现有 bind 流程）；composer 只选「本次用哪个已连接账号」并触发未绑定仓的 bind | 不把用户 OAuth 连接下沉到评论 |
| D7 | **禁止**运行评论双写覆盖 `task_repo_identities` / `cloud_task_repo_github_bindings` | 并行评论 last-write-wins |
| D8 | 旧路径：无 JSON 的评论 snapshot/clone **advisory 回退**任务级表 | Expand/Contract；新运行评论不写任务级身份 |
| D9 | composer 预填：最近一条带身份的评论 → 否则 `task_repo_identities` | 减少重复选择 |
| D10 | `GET /api/internal/tasks/{id}/container-snapshot?comment_id=` 优先评论 JSON | 容器引导克隆用对身份 |
| D11 | 不新增 Python 接口；扩展既有 Go comment POST / snapshot / 事件 payload | 元规则 20 |
| D12 | 重新克隆从关联项目移除；需要时走评论执行细节（已有 comment_id 的 compute 路径） | 操作对象是评论容器 |

### 拒绝的方案

| 方案 | 拒绝原因 |
|------|----------|
| 仅搬 UI、仍写入任务级表 | 并行评论互相覆盖，与评论级 CSC 矛盾 |
| 新表 `task_comment_repo_identities` | 本增量每评论仓库数少；JSON 与 `mentions_json` 同模式；分表可后续再拆 |
| 关联项目完全只读含编辑 | 任务仍需绑定项目/分支，那是任务定义不是运行身份 |
| 纯评论也强制选身份 | 无容器启动，身份无消费方 |
| 把 GitHub 连接本身存进评论 | 连接是用户级；评论只存本次选用的 `github_user_id` |

## 5. 数据模型

`dataMigrate/taskTaskService/011_comment_repo_identities.sql`：

```sql
ALTER TABLE task_comments
  ADD COLUMN repo_identities_json TEXT NULL;
```

JSON 示例：

```json
[
  {
    "repo_url": "https://github.com/acme/demo.git",
    "git_identity_id": "gid-1",
    "github_user_id": "1321779"
  }
]
```

- 非 GitHub 仓可省略 `github_user_id`。OAuth 绑定状态在 composer 提示，不作为发评门禁。
- `task_repo_identities`、`cloud_task_repo_github_bindings` **保留**作遗留回退与预填，UI 不再写入。

冷热：`task_comments` 已是实体爆炸型；本列随评论行走，不单独分区。查询始终带 `task_id`（路径已有）。

## 6. API 契约

不新增 path。扩展：

| 方法 | 路径 | 变更 |
|------|------|------|
| POST | `/api/tasks/{taskId}/comments/tenant_id/{tid}` | body 可选 `repo_identities`；有 `mentions` 时建议覆盖任务全部 git_repos，缺项仍 200 |
| GET | 评论列表/详情 | 响应增补 `repo_identities`（无则 `[]`） |
| GET | `/api/internal/tasks/{id}/container-snapshot` | query `comment_id`；有则用评论 JSON，否则任务级表 |
| 事件 | `TASK_COMMENT_IMAGE_MENTIONED` | payload 增补 `repo_identities`（可选字段，向后兼容） |

错误：`{ error: { code, message, details } }` 既有格式。`repo_identities` **格式无效** → 400 `repo_identities_invalid`。缺身份不再 400。

`github-credential-status` 仍按 task 拉用户已连接账号列表（只读）；composer **不再**调用 `github-credential-approve` 作为本次运行真源。

## 7. 前端边界

**关联项目（展示）**

- Toolbar：标题「关联项目」+ 一句「仓库与基准分支；本次运行身份在添加评论时选择。」
- ViewMode：项目链接、仓库 URL、基准分支、地址失配同步；去掉 OAuth/身份/进度/reclone/容器身份行。
- EditMode：不变。

**Composer（运行身份）**

- 新组件 `CommentComposerRepoIdentity.vue`（从 ViewMode 抽出身份/OAuth 分区）。
- `v-if="showRunConfig"`，按 `taskProjectsWithDetails` 列出仓库。
- `@镜像` 后展示 Git OAuth 绑定提示（检查中 / 已绑定 / 未绑定）；未绑定含真实 `a[href]` 按仓库指向 github/gitlab-start-from-gateway（带 `repo_url`）。未绑定或检查失败时拦截「提交并运行」；纯评论不拦截。
- 草稿存在 `commentRepoIdentityDraft.js`（与 dependency draft 同模式），提交后清空。

## 8. 领域概念清单

- Bounded Context: Task Collaboration（taskTaskService）、Cloud Compute（读身份启动克隆）、Git OAuth（用户级连接）
- Aggregate: **Comment** 增加值对象 `RepoIdentitySelection`；Task 仍拥有 `TaskProject`（关联）
- Domain Event: `TASK_COMMENT_IMAGE_MENTIONED` 增补（非新事件名）
- 业务意图 → 事件：见意图文档

## 9. 废弃规划（advisory）

| 旧行为 | 状态 | 替代 | 移除期限 |
|--------|------|------|----------|
| 关联项目写 `task_repo_identities` | advisory 停写（UI） | 评论 JSON | 无硬期限；回退读保留 |
| 关联项目 `github-credential-approve` | UI 停用 | 评论 JSON `github_user_id` | 同上 |
| 关联项目克隆进度 / reclone | 删除 UI | 评论执行细节 | 本增量 |

## 10. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 提交并运行且选定仓库身份 | TASK_COMMENT_IMAGE_MENTIONED（增补 repo_identities） | handleCreateComment | taskEvents start-vm | 不新事件名 |
| 只读展示关联项目 | — | — | — | 纯展示 |
| 纯评论无运行 | — | — | — | 无副作用 |

## 11. 价值流影响（输入给 Step 4）

影响既有「任务详情添加评论并启动容器」流：身份选择从关联项目挪到评论提交。不新增值流域，扩展 task-collaboration。

字段：`taskTaskService.task_comments.repo_identities_json`

## 12. 🐍 Python 新增接口

无。全部 Go `taskTaskService` + 前端。`python_api_approval: n/a`

## 13. 🕸️ Code Review Graph 分析

- 图存在：108 nodes / 17 files（JS/TS/Python/bash；Go/Vue SFC 覆盖弱）
- `code-review-graph search` 对 Vue 面板无稳定命中；影响面由 grep：`TaskDetailLinkedProjects*`、`TaskDetailCommentComposer`、`submitComment`、`container_snapshot.go`、`compute_github_credential.go`、`TASK_COMMENT_IMAGE_MENTIONED`
- CRG 工具无 `--brief` 子参，search 需无该 flag

## 14. 架构变更影响

须写 v83 `application-integration` + `enterprise-landscape` 四类伴生文件。

- 🟢 `task_comments.repo_identities_json`
- 🟡 taskFE composer / linked projects
- 🟡 taskTaskService comment create + snapshot
- 🟡 TASK_COMMENT_IMAGE_MENTIONED
- 🔴 关联项目任务级身份操作台（UI）

## 15. 测试要点

- 面板：无 OAuth 文案/按钮/进度；仍有项目名与分支
- composer：无 run config 无身份区；有 mention 有身份区；缺身份不能提交
- POST body 含 repo_identities
- snapshot `comment_id` 优先 JSON
- 无 JSON 回退任务级表
- 事件 payload 含 identities
