# DDD：auto_run 首指令与自动交付

- **日期**: 2026-07-13
- **设计文档**: `docs/superpowers/specs/2026-07-13-auto-run-first-instruction-and-delivery-design.md`

## 限界上下文

| 上下文 | 职责 | 服务 / 模块 |
|--------|------|-------------|
| **Container Credential** | task 快照、仓库身份解析、container token 门 | `taskCredentialService`（Go） |
| **Container Runtime** | bootstrap、job 执行、Git 层操作、OAuth push/PR | `onlineServiceJS`（Node） |
| **Identity（SaaS）** | 用户/公司 Git 身份绑定 SSOT | Django `accounts_user_company_git_identity`（只读，经 Go） |
| **Task（SaaS）** | `auto_run` 任务属性、title/description | SaaS DB（经 Go 快照） |

容器内编排不跨越 Credential 上下文写 Identity；仅消费已解析 DTO。

## 领域概念

| 概念 | 类型 | 说明 |
|------|------|------|
| **AutoRunTask** | Entity 属性（Task 快照） | `Task.auto_run bool`；为 true 时 bootstrap 后可触发首指令 |
| **AutoRunFirstInstruction** | Domain Service | 组装 `title + "\n\n" + description` 并下发首条 `trae` job；守卫：有克隆层、命令非空、幂等标志 |
| **AutoRunDelivery** | Application Service | 首指令 job `completed` 后：identities sync → commit → oauth-refresh-push；守卫：exit 0、delivery 幂等 |
| **RepoGitIdentity** | Value Object | `repo_url` + `user_name` + `user_email` + `identity_id`（可选）；由 Go 解析，容器只读消费 |

## 聚合与不变量

### Task 快照（Credential 上下文）

- **不变量**：container token 有效时，`FetchTaskDetail` 返回的 `task.auto_run` 与 SaaS 库一致。
- **不变量**：`repo_git_identities` 与 `FetchRepoIdentities` 同源；无绑定时为空数组。

### AutoRunFirstInstruction

- **前置**：`auto_run == true`；bootstrap 克隆层 ID 存在；`composeAutoRunCommand` 非空。
- **后置**：至多一次 `createJob`；写入 `runtime/auto_run_first_job.json`。
- **禁止**：`auto_run == false` 时调用。

### AutoRunDelivery

- **前置**：关联 job 标记 `auto_run_first`；status === `completed`（exit 0）。
- **后置**：至多一次成功交付或明确跳过；写入 `runtime/auto_run_delivery.done`。
- **禁止**：`failed` / `interrupted` 时调用；`delivery.done` 已存在时重复 push。

## 领域事件

| 事件 | 触发时机 | 载荷要点 | 消费者 |
|------|----------|----------|--------|
| **AutoRunFirstInstructionStarted** | bootstrap 后成功 `createJob` | `task_id`, `job_id`, `layer_id`, `command_kind=trae` | 日志 / 可观测（本期无 Kafka） |
| **AutoRunDeliveryCompleted** | sync→commit→push/PR 成功或明确跳过（无 diff 且无 ahead） | `task_id`, `job_id`, `layer_id`, `pr_url?` | 日志 `AUTO_RUN_DELIVERY_COMPLETE` |
| **AutoRunDeliveryFailed** | push/PR 失败或其它交付步骤不可恢复（仍不阻断服务） | `task_id`, `job_id`, `reason` | 日志 `AUTO_RUN_DELIVERY_FAILED` |

> 本期事件以**结构化日志 + 本地标志文件**实现，不新增持久化事件表。

## 应用服务协作

```text
BootstrapComplete:
  detail = TaskCredential.FetchTaskDetail(token)
  if detail.task.auto_run && cloneLayerId && !firstJobFlagExists:
    cmd = AutoRunFirstInstruction.compose(title, description)
    job = Jobs.createJob({ command: cmd, command_kind: 'trae', repo_layer_id: cloneLayerId })
    persist auto_run_first_job.json
    emit AutoRunFirstInstructionStarted

JobClosed(auto_run_first, status):
  if status != completed: return  // failed → 不 emit Delivery*
  if delivery.done exists: return
  AutoRunDelivery.run({
    identities: cached repo_git_identities,
    layerId: job.layer_id,
    commitMessage: title,
    targetBranch: work_branch
  })
  emit AutoRunDeliveryCompleted | AutoRunDeliveryFailed
```

## 防腐层

| 外部 | 边界 |
|------|------|
| Django HTTP | **不新增**容器可调写身份 API |
| task-detail JSON | 扩展字段；Node 仅解析 DTO |
| oauth-refresh-push | 复用既有 `runLayerOauthRefreshPush`，不 duplication PR 逻辑 |

## 架构变更影响

- **Plateau v17**：bootstrap 后容器闭环（首指令 + 交付）。
- **Gap**：v13 热路径零 Django 后，auto_run 仍只 bootstrap 不跑 Agent、不 PR。
- **交付物**：`docs/architecture/v17-application-integration-20260713-1427-claude.{puml,archimate,mermaid.md}`
