# Completed OPT Archive — 2026-07-17

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 6 条。
> 归档执行时间：2026-07-24T10:15:26+08:00

## [OPT-20260717-046] completed

**Logged**: 2026-07-17T22:15:00+08:00
**Completed**: 2026-07-17T22:16:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / gitlab

### Summary
GitLab 恢复稳定后，移除价格管理页上的「不稳定，不建议使用」横幅与各字段旁徽章（`data-testid=gitlab-service-unstable-banner` / `gitlab-unstable-badge`）；必要时同步评估公网定价页、工作区 GitLab 连接页是否需同类提示。

### Completion-Note
用户确认公网定价页与工作区 GitLab 连接页需要同类提示；已在 `Pricing.vue` 与 `WorkspaceSettingsGitlabConnection.vue` 落地横幅/徽章，并 build+collectstatic。恢复稳定后的移除项见 OPT-20260717-047。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/views/SystemAdminPriceManagement.vue, taskFE/app/src/views/Pricing.vue, taskFE/app/src/views/WorkspaceSettingsGitlabConnection.vue
- Tags: gitlab, price-management, ux-copy

## [OPT-20260717-015] completed

**Logged**: 2026-07-17T13:02:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-17T13:40:00+08:00
**Completion-Note**: 基准分支 commit 校验增加 checking spinner、不存在弱提示（`base-branch-commit-missing`），有请求时挂 `data-traceId`；Vitest 已覆盖。
**Area**: frontend

### Summary
基准分支 commit 校验增加短暂 loading（如 label 旁小 spinner）与可选「不存在」弱提示，避免用户误以为输入无响应。

### Details
当前仅在 `exists=true` 显示打勾；校验中/不存在无视觉反馈（符合本次需求）。可迭代为 checking 态与 `exists=false` 的淡色提示，且错误节点带 `data-traceId`。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/composables/useBaseBranchCommitCheck.js, CreateTaskProjectBranchSection.vue
- Tags: ux, create-task, commit-check

## [OPT-20260717-014] completed

**Logged**: 2026-07-17T13:02:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-17T13:40:00+08:00
**Completion-Note**: CreateTaskModal.vue 降至 244 行；拆出 Basic/ProjectBranch/Meta/AutoRun/People 子组件与 5 个 composable，DOM id/testid 保持兼容，Vitest 22 例通过。
**Area**: frontend

### Summary
将 `CreateTaskModal.vue`（已远超 500 行）按前端行数门禁拆分为分支策略 / 项目选择 / commit 校验等子组件或 composable，降低后续改动回归面。

### Details
本会话已抽出 `useBaseBranchCommitCheck`；模态其余逻辑仍集中在单文件。建议按「项目选择 + 基准分支」「工作/目标分支」「自动运行」切分，并保持现有 Vitest 覆盖。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/CreateTaskModal.vue
- Tags: frontend, create-task, refactor

## [OPT-20260717-003] completed

**Logged**: 2026-07-17T01:52:00+00:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-17T11:30:00+08:00
**Completion-Note**: 已新增 `TaskDetail.runtime-hydrate-lifecycle.playwright.test.js`（mock Running → 断言 `server-lifecycle-status` 为「已启动」）；同步加固展示层 `runtimeStatus` 回退与失配自愈。
**Area**: frontend

### Summary
为任务详情「服务器启动状态」冷打开 hydrate 补一条 Playwright：机器 Running 时硬刷新后断言 `data-testid="server-lifecycle-status"` 为「已启动」。

### Details
单元测例已覆盖 `runtime_hydrate`；意图 029 E1 仍为手工。建议在 mock runtime-status=Running 下做页面级回归，防止 stash/合并再次丢 hydrate 接线。

### Metadata
- Source: goal-overview
- Related Files: task2app/docs/intents/frontend/task_detail/029_runtime_hydrate_server_lifecycle.test-intent.md, taskFE/app/src/utils/serverLifecycleFromRuntime.js
- Tags: playwright, task-detail, lifecycle-hydrate

## [OPT-20260717-002] completed

**Logged**: 2026-07-17T01:35:00+00:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-17T03:05:00+00:00
**Completion-Note**: 已在 front_project / taskAiProvider / Chrome 插件 / onlineServiceJS / DaydaymoneyGrafana / runAll 落地 HTTP trace + 错误 DOM data-traceId；存量请求失败 alert 已批量改 showRequestError。
**Area**: frontend

### Summary
在公网 SPA 统一请求/错误出口落地 `data-traceId`（解析 `X-Trace-Id` 并挂到错误 DOM）。

### Details
元规则 `.ai/01_project_constraints/24_frontend_error_data_trace_id.md` 已生效；存量错误 toast/inline 仍可能未挂属性。优先改 axios/fetch 封装与全局错误组件，并用 Playwright 断言 `[data-traceId]`。

### Metadata
- Source: goal-overview
- Related Files: .ai/01_project_constraints/24_frontend_error_data_trace_id.md, taskFE/app/
- Tags: observability, data-traceId, frontend-errors

## [OPT-20260717-001] completed

**Logged**: 2026-07-17T01:30:00+00:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-17T01:40:00+00:00
**Completion-Note**: `24_frontend_error_data_trace_id.md` 已存在；`00_project_constraints.md` 第 26 条重复已收敛为单条。
**Area**: rules

### Summary
为 `00_project_constraints.md` 中引用但缺失的 `24_frontend_error_data_trace_id.md` 补齐专文，或删除重复/悬空条目。

### Details
索引第 26 条曾重复且专文一度缺失；现已补齐专文并收敛重复条目。

### Metadata
- Source: conversation
- Related Files: .ai/01_project_constraints/00_project_constraints.md, .ai/01_project_constraints/24_frontend_error_data_trace_id.md
- Tags: meta-rules, docs-drift

