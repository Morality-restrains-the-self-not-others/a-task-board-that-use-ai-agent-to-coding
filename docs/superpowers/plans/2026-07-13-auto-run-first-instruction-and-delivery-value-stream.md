# 价值流：auto_run 首指令与自动交付

- **日期**: 2026-07-13
- **设计文档**: `docs/superpowers/specs/2026-07-13-auto-run-first-instruction-and-delivery-design.md`
- **功能意图**: `task2app/docs/intents/engineering/cloud/006_auto_run_first_instruction_and_delivery.intent.md`
- **测试意图**: `task2app/docs/intents/engineering/cloud/006_auto_run_first_instruction_and_delivery.test-intent.md`

## 端到端价值流

```text
[任务创建者] 创建任务并设置 auto_run=true、绑定仓库 Git 身份
    → [平台] 异步 start-vm，下发 container access token
    → [容器] bootstrap：换票 → 克隆 → 工作分支 → feature-params
    → [容器] 读 task-detail（auto_run + repo_git_identities）
    → [容器] 组装 title + description，自动 POST /api/jobs（trae 首指令）
    → [Agent] 在工作层执行 trae 指令直至 exit
    → [容器] 交付：identities sync → git commit → oauth-refresh-push（push + PR）
    → [租户成员] 在远端/Git 平台查看 PR（手动 merge 不在本期）
```

## 价值流步骤与测试点

| 步骤 | 描述 | 用户价值 | 测试点 | 验收层级 |
|------|------|----------|--------|----------|
| **1. create auto_run** | 创建者在工作区创建任务，`auto_run=true`，配置 title/description 与仓库身份绑定 | 一次配置即可触发全自动闭环 | —（前置数据） | SaaS 既有流程 |
| **2. start-vm** | 平台异步启动 VM/relay 容器，注入 token 与 bootstrap 环境 | 无需用户守在前端等待 | —（既有 start-vm 路径） | 集成/冒烟 |
| **3. bootstrap** | 容器完成换票、克隆、工作分支、feature-params；暴露 bootstrap 克隆层 ID | 工作区就绪后可自动跑 Agent | **T1** task-detail 含 `task.auto_run`；**T2** 含已解析 `repo_git_identities` | Go 单测 + `machine_container.md` §4.4 |
| **4. first instruction** | `auto_run=true` 且有克隆层时，`composeAutoRunCommand` 后 `createJob`（`command_kind=trae`）；写 `runtime/auto_run_first_job.json` | 克隆完成即自动执行首条 Agent 指令 | **T3** 命令组装 title + 空行 + description；**T4** auto_run 真 + 有层 → createJob 一次；**T5** auto_run 假 → 不建 job；**T8**（首指令侧）标志已存在 → 不重复发 job | Node 单测 |
| **5. agent run** | trae job 在工作层执行；`proc.on('close')` 产生 completed / failed / interrupted | Agent 产出代码变更 | （T6/T7 前置）job 生命周期 | Node 单测 / 可选 E2E |
| **6. delivery** | completed + `auto_run_first`：sync 身份 → `git add -A` + commit(message=title) → `runLayerOauthRefreshPush`；写 `runtime/auto_run_delivery.done` | 完成后自动 push 并建 PR，无需手动 zTree | **T6** completed → sync→commit→push/PR；**T7** failed/interrupted → 不交付；**T8**（交付侧）delivery.done 已存在 → 跳过 | Node 单测（mock） |

## 测试点明细（T1–T8）

| ID | 场景 | 期望 | 对应价值流步骤 |
|----|------|------|----------------|
| T1 | FetchTaskSnapshot / task-detail 含 `auto_run=1` | JSON `task.auto_run=true` | 3. bootstrap |
| T2 | 任务已绑定仓库身份 | `repo_git_identities[]` 含 `repo_url`/`user_name`/`user_email` | 3. bootstrap |
| T3 | `composeAutoRunCommand(title, description)` | `title.trim() + "\n\n" + description.trim()`；皆空则不建 job | 4. first instruction |
| T4 | `auto_run=true` 且 bootstrap 有克隆层 | `createJob` 恰好调用一次 | 4. first instruction |
| T5 | `auto_run=false` 或缺失 | 不调用 `createJob` | 4. first instruction |
| T6 | 首指令 job `completed`（exit 0）且 `auto_run_first` | sync 身份 → commit → oauth-refresh-push | 6. delivery |
| T7 | 首指令 job `failed` / `interrupted` | 不触发交付编排 | 5→6 边界 |
| T8 | `auto_run_first_job.json` 或 `auto_run_delivery.done` 已存在 | 不重复首指令 / 不重复交付 | 4 / 6 |

## 异常与跳过路径

| 条件 | 行为 | 测试覆盖 |
|------|------|----------|
| 无仓库 / bootstrap 失败 | 不触发首指令（既有 BOOTSTRAP_FAILED） | 既有 bootstrap 单测 + T4 否定 |
| title 与 description 皆空 | WARN，不建 job | T3 扩展 |
| 无有效 Git 身份绑定 | 跳过 sync，仍尝试 commit/push | T6 mock 分支 |
| 无代码变更（nothing to commit） | 跳过 commit；若无 ahead 则跳过 push，仍写 done | T6 mock |
| 容器重建（新 VM） | 运行时标志清空，允许新一轮 | 文档/集成（非本期单测必项） |

## 架构版本

- **v17 target**：`docs/architecture/v17-application-integration-20260713-1427-claude.{puml,archimate,mermaid.md}`
