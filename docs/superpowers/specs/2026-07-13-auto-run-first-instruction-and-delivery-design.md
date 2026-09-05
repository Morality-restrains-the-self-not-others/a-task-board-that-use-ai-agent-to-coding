# auto_run 首指令自动执行 + 完成后 Git 身份 / 提交 / 推送 / PR

- **日期**: 2026-07-13 14:27
- **作者**: goal-mode / 0-auto-flow（自动采纳最优方案）
- **状态**: approved（goal-mode 跳过用户确认门）
- **相关页面**: `https://…/tenant/…/workspace/…/task-detail/task_…/`（含 `?relayToTrae=true`）
- **python_api_approval**: n/a（**不新增** Python/Django HTTP 接口；仅扩展 Go taskCredentialService 契约 + onlineServiceJS 编排）

## 1. 问题 / 意图

今日 `auto_run=true` 仅表示：**后端异步 start-vm → 容器 bootstrap（换票、克隆、工作分支、feature-params）**。

缺口：

1. 克隆完成后**不会**把任务 `title` + `description` 作为第一条 Agent 指令自动执行。
2. Agent 完成后**不会**自动拉取各仓 Git 身份、提交到工作分支、推送远端并创建 PR。

目标：在容器内形成闭环——bootstrap 成功且 `auto_run` 时自动首指令；该指令成功结束后自动交付。

## 2. 成功标准（SMART）

| # | 标准 | 可验证方式 |
|---|------|------------|
| S1 | 容器 task-detail 响应含 `task.auto_run`（bool） | Go 单测 + machine_container.md §4.4 |
| S2 | task-detail 含各仓已解析身份 `repo_git_identities[]`（`repo_url`/`user_name`/`user_email`，缺绑定时可空数组） | Go 单测 |
| S3 | `auto_run=true` 且 bootstrap 完成（有克隆层）→ 自动 `POST /api/jobs`，`command` = title + `\n\n` + description，`command_kind=trae`，`repo_layer_id`=引导克隆层 | Node 单测 |
| S4 | `auto_run=false` 或不存在 → **不**自动建 job | Node 单测 |
| S5 | 自动 job `completed`（exit 0）→ 按身份 sync → commit（message=title）→ `runLayerOauthRefreshPush`（push + PR） | Node 单测（mock） |
| S6 | 自动 job `failed`/`interrupted` → **不**自动交付 | Node 单测 |
| S7 | 同一容器生命周期仅触发一次首指令与一次交付（幂等标志文件） | Node 单测 |
| S8 | 无仓库 / bootstrap 失败 → 不触发首指令 | 既有 BOOTSTRAP_FAILED 路径 + 单测 |
| S9 | 意图金字塔 + 价值流测试点更新 | intent / value-stream |

## 3. 方案对比与采纳

| 方案 | 描述 | 结论 |
|------|------|------|
| A. 容器内闭环 | bootstrap 末尾建 job；job close 钩子交付；复用 oauth-refresh-push | **采纳**：不依赖浏览器常开；与云主机/relay 一致 |
| B. 前端 watch | 详情页 SSE 侦测 completed 后调 compute API | 拒：离开页面即失效 |
| C. SaaS 编排服务 | 新 watcher 轮询层图 | 拒：本期过重 |

**不新增 Python 接口**；身份解析落在 **taskCredentialService**（已能读 `task_repo_identities` + SaaS 身份表）。

## 4. 详细设计

### 4.1 契约扩展（machine_container.md §4.4）

`task` 增加：

```json
"auto_run": true
```

顶层（或与 `task` 并列）增加：

```json
"repo_git_identities": [
  {
    "repo_url": "https://github.com/org/repo.git",
    "user_name": "Alice",
    "user_email": "alice@example.com",
    "identity_id": "…"
  }
]
```

- 无绑定：该项省略或数组为空；交付阶段若无有效身份则 **跳过 sync 并记日志**，仍尝试 commit（可能用已有 local config）→ push；push/PR 失败记 `AUTO_RUN_DELIVERY_FAILED`，不阻断容器服务。
- **禁止**日志输出 token / 完整邮箱以外的密钥。

### 4.2 Go：taskCredentialService

1. `TaskSnapshot` 增加 `AutoRun bool \`json:"auto_run"\``。
2. `FetchTaskSnapshot` SELECT `COALESCE(auto_run, 0)`。
3. `GitIdentitySnapshot` 增加 `UserName`/`UserEmail`（从 `accounts_user_company_git_identity` 读取 `git_user_name`/`git_user_email`）。
4. `ContainerTaskDetail` 增加 `RepoGitIdentities []RepoGitIdentityDTO`。
5. `FetchTaskDetail` 组装身份列表（与 `FetchRepoIdentities` 同源）。

### 4.3 Node：首指令（bootstrap → jobs）

新模块 `autoRunOrchestration.mjs`（或挂在 bootstrap/jobsRuntime）：

```
composeAutoRunCommand(title, description) →
  `${title.trim()}\n\n${description.trim()}`.trim()
  // 若两者皆空 → 不建 job，打 WARN
```

触发点：`server.mjs` 在 `BOOTSTRAP_COMPLETE` + `registerBootstrapCloneJob` **之后**：

```
if (detail.task.auto_run && bootstrapCloneLayerId && command) {
  createJob({ command, command_kind: 'trae', repo_layer_id: bootstrapCloneLayerId })
  // 标记 rec.auto_run_first = true
}
```

`bootstrap.mjs` 须把最后一次 task-detail 结果暴露给 server（返回值或模块级 getter）。

### 4.4 Node：完成后交付

在 `jobsRuntime` `proc.on('close')`：当 `rec.auto_run_first && status===completed'`：

1. **幂等**：`runtime/auto_run_delivery.done` 已存在则跳过。
2. **身份 sync**：用 bootstrap 缓存的 `repo_git_identities` 调内部 sync（抽 `syncRepoIdentities(layerId, repos)`，不经 HTTP）。
3. **commit**：对 `rec.layer_id` 工作区 `git add -A` + `git commit -m <title>`；无变更则跳过 commit 仍尝试 push（若 ahead）。
4. **push + PR**：`runLayerOauthRefreshPush({ layerId: rec.layer_id, targetBranch: work_branch })`。
5. 写标志文件；日志：`AUTO_RUN_DELIVERY_COMPLETE` / `AUTO_RUN_DELIVERY_FAILED`。

工作分支：优先 `task.target_branch` / `branch_strategy.work_branch_name`（oauth-refresh-push 已有回退拉 task-detail）。

### 4.5 幂等与并发

| 标志 | 路径 | 含义 |
|------|------|------|
| 首指令已发 | `runtime/auto_run_first_job.json` | 含 job_id；重启后不重复发 |
| 交付已完成 | `runtime/auto_run_delivery.done` | 交付成功或明确跳过（无 diff 且无 ahead）后写入 |

容器重建（新 VM）标志清空 → 允许新一轮（与 force_auto_run 重启语义一致）。

### 4.6 不做（本期）

- 不改前端手动 zTree 流程。
- 不在 `auto_run=false` 时做任何自动 job。
- 不自动 merge 到 merge_target。
- 不新增 Django/Flask 路由。

## 5. Domain 概念（轻量）

| 概念 | 类型 | 说明 |
|------|------|------|
| AutoRunTask | Entity 属性 | Task.auto_run |
| AutoRunFirstInstruction | Domain Service | 组装并下发首条 trae 指令 |
| AutoRunDelivery | Application Service | identities → commit → push → PR |
| RepoGitIdentity | Value Object | repo_url + name + email |

## 6. 价值流影响

- 扩展 `create-task-auto-run` 流：新增步骤 `auto-run-first-instruction`、`auto-run-delivery`。
- 测试：Go + Node 单测；可选 relay E2E 冒烟。

## 7. 🏛️ 架构变更影响

- **迭代版本**: v18 🎯 target
- **迭代名称**: auto_run 首指令与自动交付
- **变更明细**:
  - 🟡 [MODIFIED] taskCredentialService — task-detail 增 `auto_run` + `repo_git_identities`
  - 🟡 [MODIFIED] onlineServiceJS — bootstrap 后自动 job；job 完成后交付编排
  - 🟡 [MODIFIED] machine_container.md §4.4
- **新增文件**:
  - `docs/architecture/v18-application-integration-20260713-1427-claude.{puml,archimate,mermaid.md}`

### .archimate 架构变迁要点

| 元素 | 内容 |
|------|------|
| Plateau v13 | Container exec hot-path current |
| Plateau v18 | auto_run 容器闭环交付 |
| Gap | bootstrap 后无首指令；完成后无自动 commit/push/PR |
| WorkPackage | 契约扩展 + Node 编排 |

## 8. 风险

| 风险 | 缓解 |
|------|------|
| Agent 长时间运行 | 交付仅挂 close 钩子，不阻塞 HTTP |
| 身份未绑定 | 跳过 sync，日志明确；push 仍走 oauth |
| 无代码变更 | commit 失败「nothing to commit」视为可继续；若无 ahead 则跳过 push，仍写 done |
| 重复触发 | 标志文件幂等 |

## 9. 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-07-13 | 初版：容器闭环首指令 + 交付 | goal 需求 |
