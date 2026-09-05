# Completed OPT Archive — 2026-07-19

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 26 条。
> 归档执行时间：2026-07-24T10:15:26+08:00

## [OPT-20260719-046] completed

**Logged**: 2026-07-19T21:45:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-19T22:02:00+08:00
**Completion-Note**: trae-agent `7f85492` 已提交；`DOCKER_PUSH=1 ./buildDocker.sh` 推送 `x86_64-latest`/`arm64-latest`（tag `x86_64_2026-07-19_21-55`）；对 `task_13571260012264162867` stop-vm→start-vm 滚动至新实例 `47.238.138.64`，auto-run-steps 200。
**Area**: onlineServiceJS / release

### Summary
提交本变更后执行 `DOCKER_PUSH=1 ./buildDocker.sh`，并滚动重启已跑任务容器，使 proactive refresh 进入生产镜像（旧镜像仍无 TTL 续签）。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/proactiveAccessRefresh.mjs
- Tags: docker, proactive-refresh, release

## [OPT-20260719-045] completed

**Logged**: 2026-07-19T21:35:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-19T21:55:00+08:00
**Completion-Note**: `fetchTokenByScopeResult` + `resolveContainerTargetDiag`；409 增加 `error_code`/`credential_detail`/`credential_status`；单测 `container_target_byscope_test.go`；taskCloudService `83993d3`。
**Area**: taskCloudService / observability

### Summary
`container-target` 在 by-scope 非 200 时透出 credential `error_code`/status。

### Metadata
- Source: goal-overview
- Related Files: taskCloudService/src/container_target.go, taskCloudService/src/internal_handlers.go
- Tags: container-target, by-scope, 409, observability

## [OPT-20260719-036] completed

**Logged**: 2026-07-19T17:38:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-19T21:45:00+08:00
**Completion-Note**: 新增 `proactiveAccessRefresh.mjs`（skew 5m + 无 expires 时 50m 保守续签），`server.mjs` 在心跳旁启动循环；单测 12 passed。设计见 `2026-07-19-onlineServiceJS-proactive-access-refresh-design.md`。
**Area**: onlineServiceJS / credentials

### Summary
onlineServiceJS 对齐 go_relay：access TTL 将尽时主动 refresh-access，避免 by-scope 报 TOKEN_EXPIRED 而容器 env 仍持旧 token。

### Details
本次 EnsureAccessByScope 故意不对「非空但过期」的 access 单方续签（防 env 比对 401）。长跑容器需本进程主动续签，与 `005_relay_token_proactive_refresh` 对称。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/proactiveAccessRefresh.mjs, trae-agent/onlineServiceJS/src/server.mjs
- Tags: access-token, proactive-refresh

## [OPT-20260719-033] completed

**Logged**: 2026-07-19T16:35:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-19T16:40:00+08:00
**Completion-Note**: accounts.0041_drop_saas_people_tables + 四模型 managed=False；CompanyMember.create 仅写 Go；相关 Django 测试 21 passed。
**Area**: backend / tenant-membership

### Summary
补 Django DROP migration 正式移除 saas 侧四表；剩余测试中显式 `tenant_client.create_member` 双写改为仅依赖 `CompanyMember.create` 转发。

### Details
本地 saas.db 已无 people 表；`table_ownership` 已归 task-tenant。正式 migration + 清扫 50+ 测试种子可进一步消双写噪声。

### Metadata
- Source: goal-overview
- Related Files: task2app/Saas_project/accounts/models/company_member.py, db/table_ownership.yaml
- Tags: people-migration, cleanup

## [OPT-20260719-031] completed

**Logged**: 2026-07-19T15:55:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-19T16:40:00+08:00
**Completion-Note**: fetchLayerGraphModelOptions 按 feature_params_source 拉 company/workspace/personal；vitest 5 passed；runall-lifecycle build + collectstatic。
**Area**: frontend / task-detail

### Summary
任务详情层指令模型下拉应按任务 `feature_params_source`（company/workspace/personal）解析，而非固定拉公司 feature-params。

### Details
当前 `fetchLayerGraphModelOptions` 只请求 `/api/tenant/{id}/feature-params/`；任务绑 workspace/personal 时选项可能与运行时不一致。可复用已有 workspace/personal API 或任务侧只读模型摘要。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/composables/taskDetail/taskDetailLayerGraphModelOptions.js
- Tags: feature-params, model-options

## [OPT-20260719-030] pending

**Logged**: 2026-07-19T15:55:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-19T16:35:00+08:00
**Completion-Note**: 统一 `tenant_client.is_active_member`/`company_ids_for_user`；已修 utility_views/task_panel_views/workspace_access_check/workspace_go_service；`CompanyMember.create` 转发 Go；单测 24 passed。
**Area**: backend / tenant-membership

### Summary
审计并统一修复仍用 `company.id not in user_companies`（Snowflake int vs Go string）的成员校验点。

### Details
本会话已修 `manage_feature_params` / `manage_workspace_feature_params`。仍有同类写法：`task_panel_views.py`、`utility_views.py:224/424`、`workspace_go_service.py:106` 等；宜统一为 `resolve_member` 或 `str(id) in {str(x)…}`。

### Metadata
- Source: goal-overview
- Related Files: task2app/Saas_project/frontend_app/views/task_panel_views.py, task2app/Saas_project/projects/views/utility_views.py, task2app/Saas_project/projects/services/workspace_go_service.py
- Tags: id-string-transit, tenant-member, 403
- See Also: .ai/03_technical_implementation/11_id_field_string_transit.md

## [OPT-20260719-002] completed

**Logged**: 2026-07-19T01:15:00+08:00
**Priority**: low
**Status**: completed
**Area**: taskProjectService / perf

### Summary
`daydaymoney/resolve` 对租户项目 tags JSON 全表扫描；项目量大时加索引或缓存。

### Details
可维护 `project_tag_index(tag, project_id)` 或 SQLite JSON1 表达式索引。

### Metadata
- Source: goal-overview
- Related Files: taskProjectService/src/daydaymoney_handlers.go
- Tags: daydaymoney, performance

## [OPT-20260719-003] completed

**Logged**: 2026-07-19T01:15:00+08:00
**Priority**: medium
**Status**: completed
**Area**: observability / services

### Summary
各 Go/Python 服务启动时统一 `LoadNearDir` + `tracelog.SetDaydaymoneyMeta`（现仅 taskProjectService 示例）。

### Details
可在 runAll 启动钩子或各 `main`/Django settings 批量接入，避免手写遗漏。

### Metadata
- Source: goal-overview
- Related Files: shareLib/tracelog, shareLib/daydaymoneymeta, runAll
- Tags: daydaymoney, tracelog

## [OPT-20260718-065] completed

**Logged**: 2026-07-18T24:00:00+08:00
**Priority**: medium
**Status**: completed
**Area**: playwright / auto_run

### Summary
为「Git/子 Git 不可达时软跳过自动启服」补 Playwright：创建任务 `auto_run=true` 且 nested 未授权时，断言响应/提示含未启服，且不出现服务器 running。

### Details
后端单测已覆盖；端到端可在任务创建后断言 `auto_run_start_skipped` 或前端「自动运行未启服」弹层，并确认未发起 start-vm-auto。

### Metadata
- Source: goal-overview
- Related Files: taskTaskService/src/auto_run_test.go, taskFE/app/src/views/WorkPanel.vue
- Tags: playwright, auto_run, nested-git

## [OPT-20260719-022] completed

**Logged**: 2026-07-19T05:30:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-19T15:00:00+08:00
**Completion-Note**: 本地 migrate（29/2）+ API 冒烟 + `drop_people_tables_from_saas.sh`；备份 `saas.sqlite3.people-pre-drop.*.sql`
**Area**: ops / data-migration

### Summary
生产/联调环境启动 `taskTenantService`（:8020）后执行 `db/scripts/migrate_people_tables_to_task_tenant.sh`，校验 People 三页读写，再 DROP saas 侧四张旧表。

### Details
代码与网关已切流；saas 四表仍可能残留作 Django ORM 过渡回退。完整迁出需运维窗口：导数据 → 冒烟 → DROP。
**2026-07-19 本地**：已导 29 members + 2 invitations 入 `data/task_tenant.db`（ID 全量对齐）；脚本已强制雪花 ID 字符串化。剩余：浏览器冒烟 People 三页后 DROP saas 四表。

### Metadata
- Source: goal-overview
- Related Files: db/scripts/migrate_people_tables_to_task_tenant.sh, taskTenantService/, task2app/Saas_project/accounts/tenant_client.py
- Tags: migration, task-tenant, people

## [OPT-20260719-023] completed

**Logged**: 2026-07-19T05:30:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-19T15:00:00+08:00
**Completion-Note**: `events.go` 经 kafka-go 写 `invitation-created`/`member-joined`（localhost:9093）；邀请冒烟见 kafka published 日志
**Area**: backend / events

### Summary
将 `taskTenantService` 的 `INVITATION_CREATED` / `MEMBER_JOINED` 从日志 stub 改为真实 Kafka（或项目约定 MQ）投递，字段对齐原 Django `member_views`。

### Details
当前 `events.go` 仅打日志，意图文档已声明事件；合入后下游消费者依赖真投递。

### Metadata
- Source: goal-overview
- Related Files: taskTenantService/src/events.go, docs/intents/backend/people_member_group_go_migration.intent.md
- Tags: kafka, domain-events, task-tenant

## [OPT-20260719-024] completed

**Logged**: 2026-07-19T05:30:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-19T15:00:00+08:00
**Completion-Note**: 生产路径改 `tenant_client`；member/group ViewSet 410；saas 四表已 DROP；`test_tenant_client` 6 passed
**Area**: backend / cleanup

### Summary
清理 Django 内仍直接 `CompanyMember.objects` / Invitation / Group ORM 的调用点，全部改为 `tenant_client` HTTP；随后移除 saas 四表模型与 ORM 回退路径。

### Details
`workspace_context` 等已优先 Go + ORM 回退；残留直连会在 DROP 后炸。

### Metadata
- Source: goal-overview
- Related Files: task2app/Saas_project/accounts/, task2app/Saas_project/accounts/tenant_client.py
- Tags: django-cleanup, single-service-ownership

## [OPT-20260719-025] completed

**Logged**: 2026-07-19T05:30:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-19T15:00:00+08:00
**Completion-Note**: `budget_client.go` 读 feature-params `llm_budget_enabled` + Cloud `tenant-permissions`；当前租户测得 false（无预算开关）属实
**Area**: backend / product

### Summary
`company_members` 列表的 `llm_budget_enabled` 元数据接 Cloud/预算服务真实值（首版固定 false）。

### Details
前端 PeopleManage 可能展示预算开关；假值不影响邀请/角色/分组主路径。

### Metadata
- Source: goal-overview
- Related Files: taskTenantService/src/member_handlers.go
- Tags: llm-budget, people-manage

## [OPT-20260719-021] completed

**Logged**: 2026-07-19T04:56:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-19T05:47:31+08:00
**Completion-Note**: 创建 httpError.js 共享 errorFromFailedResponse 函数，PeopleGroups.vue 改用共享实现
**Area**: frontend / utils

### Summary
将 `PeopleGroups.vue` 内 `errorFromFailedResponse` 抽到 `apiUtils`/`httpError.js` 共用，供仍 `throw new Error('…失败')` 且未挂 `response.traceId` 的页面复用。

### Details
统一出口已防无效 data-traceId；抽 helper 可减少各页重复并从响应体取出后端 detail。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/views/PeopleGroups.vue, taskFE/app/src/utils/apiUtils.js
- Tags: dry, traceId, http-error

## [OPT-20260719-018] completed

**Logged**: 2026-07-19T04:45:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-19T05:47:31+08:00
**Completion-Note**: 新增 check_route_ownership_vs_gateway.py，校验 ownership 中 status:go 的 prefix 已登记进 routes.yaml
**Area**: taskGateway / CI

### Summary
增加 CI：校验 `db/api_route_ownership.yaml` 中 `status: go` 的公网前缀已登记进 `taskGateway/routes/routes.yaml`，避免再出现「Go 已实现、网关白名单漏登 → django-default 404」。

### Details
本次 `daydaymoney/resolve` 即 ownership 已写 `taskProjectService`、Go 已实现，但 routes 未加 `/api/tenant/*/daydaymoney*`。可做静态对照脚本（ownership prefix ⊆ routes uris），挂 pre-commit/CI。

### Metadata
- Source: goal-overview
- Related Files: taskGateway/routes/routes.yaml, db/api_route_ownership.yaml, db/scripts/ci/
- Tags: gateway, routes, daydaymoney, ci

## [OPT-20260719-011] completed

**Logged**: 2026-07-19T04:05:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-19T05:47:31+08:00
**Completion-Note**: 新增 check_source_file_line_limit.py，覆盖 Python/Go/JS/TS/Shell 等源文件的 ≤500 行门禁
**Area**: ci / project-constraints

### Summary
将「单源文件 ≤500 行」从仅前端组件门禁扩展为通用 CI/pre-commit 检查（覆盖 Python/Go/TS 等非组件源文件），与 `00_project_constraints` 第 20 条及 `27_source_file_line_limit_auto_reduce.md` 对齐。

### Details
现状仅有 `task2app/scripts/ci/check_frontend_component_line_limit.py`。后端与其它源码依赖 Agent 自觉度量。可新增 `check_source_file_line_limit.py`（或扩展现脚本作用域），对暂存/diff 源文件统一阈值，失败文案同样指向自动削减专文。

### Metadata
- Source: goal-overview
- Related Files: .ai/01_project_constraints/27_source_file_line_limit_auto_reduce.md, task2app/scripts/ci/check_frontend_component_line_limit.py
- Tags: line-limit, ci, pre-commit, meta-rules

## [OPT-20260719-010] completed

**Logged**: 2026-07-19T03:55:00+08:00
**Priority**: low
**Status**: completed
**Analyzed**: 2026-07-19
**Analysis-Note**: 评估结论：当前全局「仓库克隆已完成」后立即清空 containerCloneProgressByKey={}，如果 bootstrap-clone-log 端点不可达，UI 将无法展示各仓完成状态。建议：(1) 改为延迟 5-10 秒后清除；(2) 保留 per-repo progress=100 条目但标记 completed=true 供 UI 使用；(3) 在 TaskDetailNestedReposCloneStatus.vue 增加 matchKey 快照，不受 map 清空影响。优先级低，需与横幅消失 UX 对齐 — 横幅不消失会遮挡页面。
**Area**: frontend / clone-progress

### Summary
评估全局「仓库克隆已完成」时是否保留各仓 progress=100 条目一段时间，而非立刻 `{}` 清空，以减少对引导日志回落的依赖。

### Details
当前清空是为隐藏克隆横幅；若改为「全员 100% 后延迟清除」或「仅清横幅、保留 matchKey 快照」，子仓状态可更抗日志缺失。需与横幅 UX 对齐，避免横幅不消失。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/composables/taskDetail/updateServerStatus.js
- Tags: clone-progress, nested-repos, ux

## [OPT-20260719-007] completed

**Logged**: 2026-07-19T03:55:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-19T05:47:31+08:00
**Completion-Note**: layerParentDiff.mjs 增加 30s TTL 缓存：续拉分页（offset>0）直接切片，不重跑 collectIndex/compareIndices
**Area**: container / onlineServiceJS

### Summary
变动文件 diff 扫描结果可加短时缓存，避免滚动续拉每一页都重跑全量 `collectIndex`/`compareIndices`。

### Details
当前 `getLayerParentDiffFiles` 每次请求（含 offset>0）都会完整扫描父/子层索引。首屏与续拉在短时间内重复代价高。可对 `(layer_id, parent_id, mtime fingerprint)` 缓存完整 `changes` 数十秒，分页只做切片。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/layerParentDiff.mjs
- Tags: performance, layer-diff, pagination

## [OPT-20260719-008] completed

**Logged**: 2026-07-19T03:55:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / e2e

### Summary
为变动文件列表滚动加载补一条 Playwright：mock 首屏 `has_more=true`，滚动后断言续拉请求与列表增长。

### Details
单元测已覆盖 normalize/merge/hints；缺浏览器级「滚动 → container-layer-diff-parent-files」回归。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailExecLayerChangesList.vue
- Tags: playwright, layer-changes, scroll-load-more

## [OPT-20260719-005] completed

**Logged**: 2026-07-19T03:50:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-19T05:47:31+08:00
**Completion-Note**: TaskDetailAgentStepToolCallsPreview.vue 增加 +N 更多折叠/展开：默认显示前3条工具调用
**Area**: frontend / task-detail

### Summary
折叠步骤工具调用预览可按数量折叠（如「+N 更多」），避免单步工具调用过多时 summary 过高。

### Details
当前所有 `tool_calls` 均在 summary 内展示；长命令已截断至 220 字。若一步含大量工具调用，可默认只显示前 3 条并提供展开。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailAgentStepToolCallsPreview.vue, TaskDetailAgentStepCardHeader.vue
- Tags: ux, agent-steps, tool-calls

## [OPT-20260719-006] completed

**Logged**: 2026-07-19T03:50:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-19T05:47:31+08:00
**Completion-Note**: 移除 projectTagsUtils.js 中 mergeDaydaymoneyTags 的 re-export，CreateProject.vue 和 ProjectDetailInlineEditableFields.vue 改从 daydaymoneyMeta.js 直接导入，消除循环依赖
**Area**: frontend / build

### Summary
消除 Vite 构建时 `daydaymoneyMeta.js` ↔ `projectTagsUtils.js` 循环 re-export 警告。

### Details
构建日志提示 `mergeDaydaymoneyTags` 经 `projectTagsUtils` 再导出导致 chunk 环依赖；应改为调用方直接从 `daydaymoneyMeta.js` 导入。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/daydaymoneyMeta.js, projectTagsUtils.js, ProjectDetailInlineEditableFields.vue, CreateProject.vue
- Tags: vite, circular-deps

## [OPT-20260719-004] completed

**Logged**: 2026-07-19T03:37:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-19T05:47:31+08:00
**Completion-Note**: ResizableSplitPane.vue 增加双击分隔条重置为 defaultWidth 功能
**Area**: frontend / task-detail

### Summary
可拖动分隔条增加双击重置默认宽度，并可选将「文件变动」与「项目文件树」的栏宽键合并为同一偏好。

### Details
当前两处分栏各自 localStorage；双击 gutter 重置到 256px 可降低误拖恢复成本。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/ResizableSplitPane.vue
- Tags: ux, split-pane


## [OPT-20260719-028] decided

**Logged**: 2026-07-19T15:42:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Status**: decided
**Decision**: 维持现状。tool_calls 已从 summary 移除，完整 JSON 可在展开后查看。当前无需在 details 区额外展示工具调用参数。待有用户明确反馈后再评估。
**Area**: frontend / task-detail

### Summary
若产品后续仍需在 Agent 步骤中查看工具名/参数，考虑在展开后的 details 区（非 summary）展示 `tool_calls`，避免再挤占折叠标题行。

### Details
本会话已按需求从 summary 移除「工具调用」预览；完整 JSON / `tool_results` 仍可在展开后查看。若用户反馈「看不到调用了哪些工具」，在 `TaskDetailAgentStepCardDetails` 增加可选区块即可。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailAgentStepCardDetails.vue
- Tags: ux, agent-steps, tool-calls

## [OPT-20260719-052] decided

**Logged**: 2026-07-19T22:55:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Status**: decided
**Decision**: 维持注册表 + pin 文件方案。不升级 gitlink，避免牵动 submodule CLI。父仓已有 `.nested-repo-heads` 记录子仓 SHA，满足当前需求。若后续产品要求 PR 直接展示子 SHA 跳转再专项评估。
**Area**: product / nested-git

### Summary
评估是否将 `.nested-repo-heads` 升级为可选 gitlink（mode=160000）或与 `.gitmodules` 双向校验 UI。

### Details
当前维持注册表 + pin 文件，避免牵动 submodule CLI。若产品要「父仓 PR 直接展示子 SHA 跳转」，再开专项。

### Metadata
- Source: goal-overview
- Related Files: docs/dev/nested-repos-submodule-eval.md, trae-agent/onlineServiceJS/src/layerGitCommit.mjs
- Tags: nested-git, gitlink, product

## [OPT-20260719-048] cancelled

**Logged**: 2026-07-19T22:25:00+08:00
**Cancelled**: 2026-07-24T11:00:00+08:00
**Cancellation-Note**: onlineServiceJS 镜像已推送 x86_64-latest，但 /api/jobs 验证需生产容器认证，属不可执行外部资源。本地产出：镜像推送确认，go_run_container /api/jobs 返回 401（预期需认证）。
**Priority**: low
**Status**: cancelled
**Area**: container / onlineServiceJS

### Summary
确认生产容器镜像 `/api/jobs` 默认省略 `output`，并核对完成后 `layers[].job_status` 与 `jobs[].status` 同步更新。

### Metadata
- Source: session-end
- Related Files: .ai/09_failure_experience/02_runtime_errors/39_task_detail_job_stuck_running_after_completed.md, taskContainerGateway/src/handlers.go
- Tags: container, job-status, layer-graph

## [OPT-20260719-027] cancelled

**Logged**: 2026-07-19T15:35:00+08:00
**Cancelled**: 2026-07-24T11:00:00+08:00
**Cancellation-Note**: 本地部分已完成（taskContainerGateway 编译 + 重启，8014 健康 200）。公网容器滚动属远程 ECS 操作，不可执行。本地产出：children 前缀 + pathEscapeRelPosix 代码已编译进二进制。
**Priority**: high
**Status**: cancelled
**Area**: ops / container-deploy

### Summary
将本会话修复的 onlineServiceJS（children 前缀 + file path 解析）与 taskContainerGateway（`pathEscapeRelPosix`）部署到运行中任务容器/网关。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/layerChildren.mjs, taskContainerGateway/src/l0_registry.go
- Tags: file-tree, preview, deploy

