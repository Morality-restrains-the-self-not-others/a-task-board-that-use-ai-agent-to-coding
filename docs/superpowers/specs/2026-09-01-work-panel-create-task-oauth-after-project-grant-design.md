# 2026-09-01 项目详情已 OAuth，工作面板创建任务仍要授权

- **Status:** approved（2026-09-01 总体批准：inherit_project_l2；v126 target + ADR-0055）
- **Date:** 2026-09-01
- **Pages:**
  - 项目详情：https://www.daydaymoney.com/tenant/882297276515512320/projects/proj_882614094526443520/?accessCode=nLrLGsmG7p
  - 工作面板：https://www.daydaymoney.com/tenant/882297276515512320/work-panel
- **python_api_approval:** not_applicable（无新增 Python HTTP 接口；落点均为现有 Go 服务）
- **拟 ADR:** ADR-0055（修订 ADR-0049「项目 L2 永不种到自动运行评论」条款）

## 当前架构理解

根据 `docs/architecture/` **v125 ✅ current**（2026-09-01 ImageMarket 申请入口）与 `VERSION_HISTORY.md`：

- 共有 2 个 current 视图：`enterprise-landscape`、`application-integration`（v125 主题是厂商申请，**不是** Git OAuth）。
- 业务层（本题相关，沿用 v118 / ADR-0049）：租户成员在项目上使用 Git 预览；在任务上启用自动运行克隆。
- 应用层：`taskFE`、`taskGateway`、`taskGitOauth`、`taskProjectService`、`taskTaskService`、`taskCredentialService`、`taskCloudService`。
- 技术层：L1 凭据仅 `taskGitOauth` 直连；项目 L2 表 `project_git_oauth_grant` 属 `taskProjectService`；评论 L2 在 `task_comments.repo_identities_json`。
- 上次更新的架构版本是 **v125**。Git OAuth 资源标记本身已在 **v118** 交付。

📋 架构版本历史（与本题相关）：

| 版本 | 状态 | 摘要 |
|------|------|------|
| v29 | 已交付 | `taskGitOauth` 持有用户级凭据 |
| v118 / ADR-0049 | 已交付 | L1 连接 vs 项目/评论 L2 使用标记；创建自动运行只认 pending `grant_ticket` |
| v125 | ✅ current | ImageMarket 厂商申请（正交） |

本次需求将在 v125 上修订 **自动运行如何获得评论 L2**：允许同用户、同项目、同 gitsite 的 **项目 L2 种下** 本条【自动运行】评论 L2。批准后写 **v126** 四类伴生文件。

## 🔍 Trace 日志分析

用户未粘贴 `data-traceId`。本问题是两页授权语义不一致，不是单次请求失败。跳过 Loki 主查询。

替代证据：源码对照 ADR-0049 与现网 URL（项目详情 `grant_kind=project`，创建任务 `grant_kind=pending`）。

## 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | 根图 `.code-review-graph/graph.db` 存在（2026-09-01T14:14:54，114 nodes / 18 files / main） |
| 关键发现 | `code-review-graph search "useCreateTaskRepoOAuth"` / `hasSessionGrantForRepo` → 0 nodes |
| 决策影响 | 图未覆盖 `taskFE` 子仓与 Go 服务符号；爆炸半径以源码检索为准 |
| skip 理由 | `unavailable`：索引面未包含本次改动的 taskFE / taskGitOauth / taskProjectService / taskTaskService |

源码爆炸半径（手工）：

- FE：`useCreateTaskRepoOAuth.js`、`CreateTaskRepoOAuthSection.vue`、Fork 自动运行 OAuth 门禁、`create_task_oauth_bind` 意图
- Go：`taskGitOauth/src/resource_grant.go`（project 回调不签发 ticket）、`taskTaskService` `applyGrantTicketToIdentities` / `ensureAutoRunAtComment`、`taskProjectService` `project_git_oauth_grant`

## 问题分析

### 现象

同一用户在项目详情完成 OAuth 后，徽章为「已授权」。到工作面板创建任务（项目 `default_auto_run` 常使自动运行默认开启）仍出现「OAuth 绑定」/「开启自动运行前须完成授权」。

### 根因（按设计，不是绑定丢失）

两页打的是 **不同 L2 资源**，创建门禁 **故意不看** 项目行。

| 入口 | OAuth `grant_kind` | 回调写入 | 前端「已授权」判定 |
|------|-------------------|----------|-------------------|
| 项目详情 | `project` + `grant_id=projectId` | `POST /api/internal/projects/git-oauth-grant/`（项目 L2）；**不**签发 `grant_ticket` | `validate-git-repos` + `project_id` → `token_available`（L1 ∧ 项目 L2 ∧ probe） |
| 创建/Fork 自动运行 | `pending`（默认） | 一次性 `grant_ticket` 进回调 URL / `sessionStorage` | **仅** `hasSessionGrantForRepo`（sessionStorage） |

`finishResourceGrant` 在 `grant_kind=project` 时 `return outPath, ""`，回流 URL **没有** `grant_ticket`。用户从项目页再进工作面板，session 里没有 ticket，门禁视为未绑定。

这与 [ADR-0049](../../adr/0049-git-oauth-resource-grant-marker.md) 原文一致：

> Project L2 is never copied onto auto-run comments  
> Auto-run cannot ride on project-page OAuth

用户本次明确要求：**项目页已经授过权，创建该项目下的自动运行任务不应再点一次 OAuth。**

`?accessCode=` 只影响回流 path，不是根因。

## 产品决策（已选）

**同项目 × 同操作者 × 同 gitsite 的项目 L2，可以种下本次【自动运行】评论 L2。** 不再为「已经在该项目授过权的人」二次拦截。

仍保持：

1. L1 仍是全平台一份凭据（不按仓 slug 再存 refresh）。
2. **别人的**项目 L2 不能给当前操作者用。
3. **其它项目**的 L2 不能给本项目任务用。
4. 同事后来手动「提交并运行」仍是 **另一条评论**，自己打评论 L2（不改）。
5. 换票仍认【自动运行】评论作者 + **该评论** L2（种下之后评论自带标记）。
6. 无项目 L2 时，仍走 pending `grant_ticket`（现网路径保留）。

## 方案

### 前端门禁

`useCreateTaskRepoOAuth` / Fork 等价逻辑改为 **OR**：

1. 本会话 `grant_ticket`（现网）；或
2. 该仓所属 **所选项目** 对当前用户该 gitsite 已有 L2，且 L1 可换票（`token_available`）

实现：对每个已选 `projectId` 调现有 `POST /api/projects/validate-git-repos/tenant_id/{tid}/`，`urls` = 该项目 GitHub/GitLab 仓，`project_id` = 该项目，`probe_access: false`（创建弹窗不要打远端 Git API；与 auto_run parent probe 一致）。`token_available` → 该 URL bound。

文案：

- 项目 L2 已齐：`已绑定 Git OAuth`（可沿用 `create-task-repo-oauth-bound`）
- 未齐：绑定链接仍 `grant_kind=pending`（任务/评论尚未存在）
- 拦截文案改为：「开启自动运行前，请为未授权的仓库完成 Git OAuth（可在项目详情授权，或点下方绑定）」

### 后端种下（硬门禁，禁止只信前端）

`ensureAutoRunAtComment` 在 `applyGrantTicketToIdentities` **之后**，对仍缺 `oauth_gitsite` 的仓：

1. 解析仓所属 `project_id`（创建/Fork payload 已有项目选择；复用评论则用任务已关联项目）。
2. `GET` `taskProjectService` 内部查询：`(project_id, user_id=操作者, gitsite)` → `{ has_grant, remote_user_id }`。
3. 有则 `stampCommentOAuthGrant`，发布 `COMMENT_GIT_OAUTH_GRANTED`（`via=project_l2_seed`），幂等键 `grant:comment:{commentID}:{userID}:{gitsite}`。
4. 无 ticket 且无项目 L2 → 行为与现网一致（评论无 L2；clone/push 仍会失败并提示授权）。前端已拦截为主。

**禁止**把项目 L2 行复制成评论行之外的共享状态。只在创建/复用【自动运行】评论时 **写入该评论 JSON**。

新内部接口（Go，`taskProjectService`）：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/internal/projects/git-oauth-grant/?project_id=&user_id=&gitsite=` | `X-Internal-Secret`；返回 `has_grant` + `remote_user_id`；只读，无新事件 |

`db/api_route_ownership.yaml` / Swagger 同步该 internal 路径。不新增公网 API。不新增 Python 接口。

### 优先级

1. 有效 `grant_ticket` → 消费 ticket（覆盖 remote_user_id）。
2. 否则项目 L2 seed。
3. 否则保持未标记。

### Fork

与创建任务同一套：Fork 源任务关联项目若已有操作者项目 L2，确认「自动运行并派生」不强制再走 pending。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 项目页 OAuth 回调打项目 L2 | PROJECT_GIT_OAUTH_GRANTED | taskGitOauth `markProjectGitOAuthGrant` | 审计 | 已有 |
| 创建/Fork 自动运行评论种下 L2 | COMMENT_GIT_OAUTH_GRANTED | taskTaskService `ensureAutoRunAtComment`（ticket 或 project seed） | 审计；clone 认评论 L2 | — |
| 查询项目 L2 | — | GET internal | — | 纯查询 |
| 创建弹窗检查绑定 | — | FE 调既有 validate-git-repos | — | 纯查询 |

## Domain Concept Inventory

| 概念 | 说明 |
|------|------|
| Bounded Contexts | Git OAuth（L1）、Project（项目 L2）、Task（评论 L2 / 自动运行） |
| Key Entities | ProjectGitOAuthGrant、AutoRunComment、GrantTicket |
| Candidate Aggregates | 项目 grant 一行；评论 `repo_identities_json` |
| Domain Events | `PROJECT_GIT_OAUTH_GRANTED`、`COMMENT_GIT_OAUTH_GRANTED` |

## 价值流影响

仓库根无 `value-stream.yaml`。受影响流（逻辑）：

- **项目 Git 授权**（项目详情徽章）— 不改写入语义
- **工作面板创建任务 / Fork 自动运行** — 门禁与【自动运行】评论 L2 来源增加项目 seed
- 测试意图：`docs/intents/frontend/create_task_oauth_bind.test-intent.md` 须改 T2/T3：L1 connected 且 **项目已 L2** → 可提交；仅 L1、无项目 L2、无 ticket → 仍拦截

## 🐍 Python 新增接口清单与 Go 替代评估

不触发。无新 Python endpoint。

## ADR-0049 修订要点（ADR-0055）

**We will** 允许：启用自动运行的操作者，若对任务所选项目的对应 gitsite **已有项目 L2**，则创建【自动运行】评论时用该行的 `remote_user_id` 写入评论 L2。

ADR-0049 中「Project L2 is never copied onto auto-run comments / Auto-run cannot ride on project-page OAuth」由 ADR-0055 **局部取代**。两层模型、评论级换票、同事另评须另授，仍然有效。

拒绝：

- 只用 L1 `connected` 放行（换仓徽章问题复现）
- 跨项目/跨用户继承
- 仅改前端不种评论 L2（clone 仍缺标记）

## 🏛️ 架构变更影响

- **迭代版本**: v126 🎯 target
- **迭代名称**: 项目 L2 种下自动运行评论
- **作者**: cursor
- **设计日期**: 2026-09-01 22:44
- **新增文件**（每个视图四类伴生格式）:
  - 🆕 `docs/architecture/v126-enterprise-landscape-20260901-2244-cursor.puml`
  - 🆕 `docs/architecture/v126-application-integration-20260901-2244-cursor.puml`
  - 🆕 `docs/architecture/v126-enterprise-landscape-20260901-2244-cursor.diff.archimate`（增量变迁）
  - 🆕 `docs/architecture/v126-application-integration-20260901-2244-cursor.diff.archimate`
  - 🆕 `docs/architecture/v126-enterprise-landscape-20260901-2244-cursor.full.archimate`（全量拓扑）
  - 🆕 `docs/architecture/v126-application-integration-20260901-2244-cursor.full.archimate`
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**:
  - `docs/architecture/v125-*-20260901-1405-cursor.puml` (current)
- **变更明细**: 🟢 GET internal grant + seed 流 / 🟡 CreateTask 门禁与 ensureAutoRunAtComment / 🔴 无

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | 增量 — Plateau v125 → Gap（创建任务只认 ticket）→ WP → Plateau v126；目标拓扑：CreateTask → Gateway → Project/Task → grant 表 |
| **`.full.archimate`** | 全量 — 项目页 OAuth、pending ticket、validate-git-repos、internal GET、评论 L2 seed |

> 老文件未被修改。目标架构将在 `/10-ship` 执行时切换为 current。

## 验收（实现阶段）

1. 项目详情对该仓「已授权」后，同用户在工作面板对该项目 `auto_run=true` 创建：出现 `create-task-repo-oauth-bound`，无 bind 链接，可提交（其它门禁除外）。
2. 仅 L1、项目无 L2、无 session ticket：仍拦截 + bind 链接。
3. 创建后【自动运行】评论 JSON 含 `oauth_gitsite` / `oauth_remote_user_id`，Kafka `COMMENT_GIT_OAUTH_GRANTED` 且 `via=project_l2_seed`（无 ticket 时）。
4. 另一项目未授权仓仍拦截。
5. `auto_run=false` 仍不展示 OAuth 行。
6. Fork 自动运行与创建一致。
7. 无新增 Python 路由；internal GET 有单测 + Swagger。

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-09-01 | 初版 | 项目页授权与创建任务门禁分裂；用户要求同项目不再二次 OAuth |
