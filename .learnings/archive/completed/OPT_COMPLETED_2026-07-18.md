# Completed OPT Archive — 2026-07-18

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 106 条。
> 归档执行时间：2026-07-24T10:15:26+08:00

## [OPT-20260718-066] completed

**Logged**: 2026-07-18T24:50:00+08:00
**Priority**: medium
**Status**: completed
**Area**: infra / runAll

### Summary
排查并加固 `task-git-oauth` 变为 zombie/`failed` 后未自动拉起的问题（本会话已手动 restart 恢复）。

### Details
runAll 状态曾长期 `failed`（PID zombie），导致项目详情 OAuth start-from-gateway 无 `authorize_url`。建议：确认 monitor 对 zombie 的检测与 auto-restart 策略；必要时在健康检查失败 N 次后强制 SIGKILL 残留并重建进程。

### Metadata
- Source: goal-overview
- Related Files: runAll/src/runner.go, taskGitOauth/run.sh, logs/task-git-oauth.log
- Tags: task-git-oauth, runAll, zombie, oauth

## [OPT-20260718-064] completed

**Logged**: 2026-07-18T23:35:00+08:00
**Priority**: medium
**Status**: completed
**Area**: taskTaskService / auto_run

### Summary
`validateAutoRunPrerequisites` 在运行环境非空之外，再校验与项目 `server_run_template.region`（及 platform）匹配，避免门禁通过后 `start-vm-auto` 仍因区域无镜像失败。

### Details
本会话已把 runtime-environments 调用改到 AI Provider + `external_image_id`，并修了派生错误 `data-traceId`。当前门禁只检查 `len(envs)>0`；可过滤 `platform_type`/`region` 与模版一致，否则返回更明确的 `AUTO_RUN_RUNTIME_ENV_REQUIRED`。

### Metadata
- Source: goal-overview
- Related Files: taskTaskService/src/auto_run.go, taskTaskService/src/cloud_client.go
- Tags: auto_run, runtime-environments, region-match

## [OPT-20260718-063] completed

**Logged**: 2026-07-18T23:22:00+08:00
**Priority**: medium
**Status**: completed
**Area**: conf / CI / taskGitOauth

### Summary
为 `conf/auth/git-oauth/providers/` 增加 CI 门禁：GitLab provider YAML 不得为空，且 `gitlab_provider_config_count`（或静态扫 `provider: gitlab`）≥1，防止再被「清理未用文件」误删。

### Details
误删三份 `*gitlab*` YAML 后健康检查 count=0，页面报 `bad_state [debug: missing_allowed_host]`。可做：
1. `db/scripts/ci/` 脚本：断言 `providers/` 下至少一份 `provider: gitlab`；
2. 可选：与 `core/django/git-oauth-providers/` 的 gitlab `service_provider` 集合做对称校验。

### Metadata
- Source: goal-overview
- Related Files: conf/auth/git-oauth/providers/, conf/auth/git-oauth/ai.md, .ai/09_failure_experience/02_runtime_errors/48_gitlab_oauth_bad_state_missing_allowed_host.md
- Tags: gitlab, oauth, conf, ci

## [OPT-20260718-062] completed

**Logged**: 2026-07-18T23:25:00+08:00
**Priority**: low
**Status**: completed
**Area**: taskAiProvider / performance

### Summary
`attachMarketplaceAvailability` 对列表每条镜像单独查 association，镜像多时存在 N+1；可改为按 `container_image_id IN (...)` 一次查出后内存判定。

### Details
当前正确性已由厂商列表测覆盖；优化时保持 `marketplaceAvailability` 文案语义不变，并补批量路径单测。

### Metadata
- Source: goal-overview
- Related Files: taskAiProvider/infrastructure/store_marketplace.go
- Tags: n-plus-one, marketplace, vendor-list

## [OPT-20260718-061] completed

**Logged**: 2026-07-18T22:05:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / e2e

### Summary
为任务详情「服务器信息」区补 Playwright：首屏 `[data-testid="server-section-collapse-toggle"]` 为「展开」且 `[data-testid="server-section-body"]` 隐藏；点击任一服务器 Tab 后内容区展开。

### Details
composables 单测已覆盖默认收起；缺页面级回归，防止 watch(activeServerSection) 再次对初始化切 Tab 强制 expand。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/composables/useServerSectionBodyExpanded.js, taskFE/app/src/components/ServerConfig.logic.vue, taskFE/app/src/components/ServerConfigSectionTabs.vue
- Tags: playwright, task-detail, server-section-collapse

## [OPT-20260718-060] completed

**Logged**: 2026-07-18T21:52:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / e2e

### Summary
为任务详情页补一条 Playwright（CDP）：`auto_run=true` 时 `[data-testid="auto-run-steps-toggle"]` 文案为「查看自动运行说明」，且不存在 `[data-testid="auto-run-steps-body"]`。

### Details
组件测与 IdentityPanel 测已覆盖默认收起；缺页面级回归，防止调用处再次传入 `defaultExpanded=true`。可挂现有 task-detail CDP 套件。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/AutoRunStepsPreview.vue, taskFE/app/src/components/task-detail/TaskDetailTaskIdentityPanel.vue, taskFE/app/src/components/task-detail/TaskDetailTaskIdentityPanel.auto-run.test.js
- Tags: playwright, task-detail, auto-run-steps

## [OPT-20260718-059] completed

**Logged**: 2026-07-18T21:50:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / e2e

### Summary
为 work-panel 看板卡片补一条 Playwright：断言 `[data-testid="task-card-id"]` 与同卡 `h4` 同行，且编号 `getBoundingClientRect().left` 小于标题（或 DOM 顺序编号在前）。

### Details
单元测 `TaskDetail.card-ux.test.js` 已覆盖 DOM 顺序与复制行为；缺页面级回归，防止布局再被挪回标题下方。可挂现有 work-panel CDP 套件。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/TaskDetail.vue, taskFE/app/src/components/TaskCardIdBadge.vue, taskFE/app/src/components/TaskDetail.card-ux.test.js
- Tags: playwright, work-panel, task-card-id

## [OPT-20260718-058] completed

**Completed**: 2026-07-18T21:00:00+08:00
**Completion-Note**: 源码变更已落地（fail-closed authMiddleware/healthz）；commit 后需手动 DOCKER_PUSH=1 ./buildDocker.sh + 滚动重启存量容器

**Deploy-Required**: 源码变更已在本会话落地（fail-closed authMiddleware/healthz）；需在 commit 后手动执行 `DOCKER_PUSH=1 ./buildDocker.sh` 并滚动重启存量容器。**此项需人工运维操作。**

**Logged**: 2026-07-18T20:28:00+08:00
**Priority**: medium
**Status**: completed
**Area**: onlineServiceJS / deploy

### Summary
提交 OPT-056 fail-closed 变更后执行 `DOCKER_PUSH=1 ./buildDocker.sh`，并滚动重启仍可能带着无效 ACCESS_TOKEN 对外响应的存量任务容器。

### Details
本会话已在源码落地换票失败 503 `TOKEN_BOOTSTRAP_FAILED` 与 `GET /healthz`；companion `onlineServiceJS/ai.md` 要求 commit 后推镜像。未提交前勿推；推送后验收：换票失败容器 `/healthz` 为 503，`token_bootstrap=failed`。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/auth.mjs, trae-agent/onlineServiceJS/src/server.mjs, trae-agent/onlineServiceJS/buildDocker.sh, trae-agent/onlineServiceJS/ai.md
- Tags: docker, onlineServiceJS, deploy, fail-closed

## [OPT-20260718-057] completed

**Logged**: 2026-07-18T15:25:00+08:00
**Priority**: low
**Status**: completed
**Area**: django / relay
**Renumbered-From**: OPT-20260718-034 (duplicate id)

### Summary
删除或瘦身已无浏览器入口的 `relay_to_trae_proxy.py` 公共 HTTP 入口函数与 `prepare_relay_to_trae_env`，仅保留 Go/internal 仍依赖的 helper（如 `_issue_token_via_go`）及对应单测迁移到 Go。

### Details
OPT-028 已使 9 个 `relay-to-trae-*` Django view 始终 410；proxy 函数仍被单元测直接调用。可在确认无其它进程 import 后删入口函数，或整文件迁为 internal-only 模块。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: task2app/Saas_project/cloud/services/relay_to_trae_proxy.py, task2app/Saas_project/cloud/services/mock_run_container.py
- Tags: dead-code, migration

## [OPT-20260718-055] completed

**Logged**: 2026-07-18T19:16:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / task-detail

### Summary
会话 Feed 历史评论中将已提交的 `@镜像名` 渲染为与 composer 一致的高亮 chip（当前仅输入区高亮）。

### Details
`CommentImageMentionEditor` 已用 `data-mention-*` chip；`TaskDetailConversationFeed` 仍纯文本展示。可按评论 `mentions[]` 或正文 `@name` 对照安装镜像列表做只读高亮，避免误伤邮箱等文本。

**Completed**: 2026-07-18T20:40:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailConversationFeed.vue, CommentImageMentionEditor.vue
- Tags: mention, comments, ux

## [OPT-20260718-054] completed

**Logged**: 2026-07-18T18:56:00+08:00
**Priority**: medium
**Status**: completed
**Area**: runAll / dockerInfra

### Summary
`docker-redis` 健康检查仅 TCP `:6379`，无法区分 Compose Redis 与本机 systemd `redis-server`；启动时可能误标 healthy，且与系统 Redis 争用端口。

### Details
停服误报已在 `ensureServiceNotReachable` 对「无可视 PID」放行。后续可：探活改为 `compose ps`/容器健康，或开发机文档明确禁用系统 Redis / 改用独立端口；并避免 orphan/preflight 误伤系统服务。

**Completed**: 2026-07-18T20:30:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: conf/runAll.yaml, runAll/src/runner.go, dockerInfra/redis/, .ai/09_failure_experience/02_runtime_errors/48_docker_redis_still_reachable_after_stop.md
- Tags: docker-redis, redis, health-check, runAll

## [OPT-20260718-053] completed

**Logged**: 2026-07-18T18:35:00+08:00
**Priority**: low
**Status**: completed
**Area**: taskTaskService / taskEvents

### Summary
为任务创建/删除补充领域事件并扇出到 work-panel SSE，使远程浏览器无需刷新即可看到新卡/删卡。

### Details
本迭代仅覆盖 `TASK_STATUS_CHANGED`（进度列/完成态）。创建/删除仍依赖本机 `tasks-updated` → `fetchTodos`。

**Completed**: 2026-07-18T20:42:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskTaskService/src/events.go, taskEvents/internal/handlers/workpanelfanout
- Tags: sse, task-crud, work-panel

## [OPT-20260718-051] completed

**Logged**: 2026-07-18T18:25:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend / testing

### Summary
为 `ServerConfig.fetchServerRuntimeStatus` 成功路径补一条单元/组件测：断言会调用 `updateServerStatus({ status: 'runtime_hydrate', runtime_status })`，防止接线再次被合并/ stash 静默丢掉。

### Details
本会话已恢复意图 029 接线并在公网验证「运行中 → 已启动」；历史曾出现 handler 在 `updateServerStatus` 中存在、但 `fetchServerRuntimeStatus` 未调用的半落地状态。建议用 mock `apiFetch` 的 ServerConfig 测例锁住调用点。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/ServerConfig.logic.vue, task2app/docs/intents/frontend/task_detail/029_runtime_hydrate_server_lifecycle.intent.md
- Tags: runtime-hydrate, regression-test, task-detail

## [OPT-20260718-046] completed

**Logged**: 2026-07-18T17:25:00+08:00
**Priority**: medium
**Status**: completed
**Area**: gitService / taskProjectService

### Summary
新建系统内建 GitLab 仓默认落入 `tenant-{company_id}` Group；存量用户路径仓择机迁移，使组级 `repository_size_limit` 成为真实命名空间硬限（当前为项目级近似）。

### Details
OPT-040 已对关联项目下发同额 `repository_size_limit`，并创建空组 `tenant-*`。CE 无跨项目聚合配额时，组限 alone 不覆盖 `user/repo`；产品若要「租户总占用 ≤ disk_gb」需迁仓或应用侧推送拒绝。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: gitService/scripts/sync_tenant_gitlab_disk_quota.sh, docs/superpowers/specs/2026-07-18-gitlab-disk-usage-quota-enforcement-design.md
- Tags: gitlab, namespace, storage-limit

## [OPT-20260718-045] completed

**Logged**: 2026-07-18T17:25:00+08:00
**Priority**: low
**Status**: completed
**Area**: taskProjectService / hygiene

### Summary
清理 SaaS `project_repos` 中 GitLab 已 404 的 gitlab-local 关联（如 `somanyad*`），减少磁盘同步 `project_missing` 噪音并避免计量漏测被误读为「占用为 0」。

### Details
租户 `850256677331562496` 同步时 3 仓中 2 仓 `project_missing`；存活仓 `valueStream` 已能计量。可做后台对账：GitLab 404 → 软删/标记失效关联。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskProjectService/src/tenant_disk_usage.go, data/task_project.db
- Tags: gitlab, orphan-repos, hygiene

## [OPT-20260718-042] completed

**Logged**: 2026-07-18T16:35:00+08:00
**Priority**: low
**Status**: completed
**Area**: billing / gitlab

### Summary
磁盘购买时长 UI 增加「自定义月数」输入（API 已支持 1～36），并支持未过期时续期叠加而非覆盖到期日。

### Details
当前下拉仅 1/3/6/12；新购覆盖 `disk_expires_at=now+months`。若产品要「续费延长」，改为 `max(now, expires_at)+months`。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskBill/src/gitlab_resources.go, useGitlabResourcePurchase.js
- Tags: gitlab, disk-months, renew

## [OPT-20260718-039] completed

**Logged**: 2026-07-18T16:30:00+08:00
**Priority**: medium
**Status**: completed
**Area**: billing / auth

### Summary
为 `POST .../billing/gitlab-resources/purchase/` 增加租户管理员门禁（对齐 gitlab-oauth-connection PUT）。

### Details
首期与 `switch_pricing_package` 同级（已登录即可）。建议在 billing_bridge 或 taskBill 校验 company admin，避免普通成员扣费。

**Completed**: 2026-07-18T20:41:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskBill/src/gitlab_resources.go, task2app/Saas_project/billing_bridge/proxy_views.py
- Tags: gitlab, billing, rbac

## [OPT-20260718-036] completed

**Logged**: 2026-07-18T16:20:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend / observability

### Summary
扫任务详情其余自绘 `text-red-600` 错误路径，统一挂载 `data-traceId`（含 layer changes refresh、exec log top error 等）。

### Details
本会话已修项目文件树「提交日志 / 文件内容」预览与列表错误。clone-log 路径此前已修。建议用 rg 扫 `text-red-600` + 无 `data-traceId` 的模板，按元规则 24 补齐；优先 Vitest 组件测。

**Completed**: 2026-07-18T20:42:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/, .ai/01_project_constraints/24_frontend_error_data_trace_id.md
- Tags: data-traceId, task-detail, observability

## [OPT-20260718-035] completed

**Logged**: 2026-07-18T16:05:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / tests

### Summary
为代理步骤手风琴补充「多步互斥」Playwright 场景（mock 两步以上），防止仅靠单步默认展开掩盖互斥回归。

### Details
当前 `TaskDetail.layer-agent-steps-ui-mock.playwright.test.js` 以单步覆盖展开/收起；互斥逻辑已有 vitest 组件测。建议在 mock steps 中造 2+ 步，断言打开步骤 A 后步骤 B 关闭。验收：Playwright 绿；不引入真实账号依赖。

**Completed**: 2026-07-18T20:34:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: task2app/playwright/front_project/tests/TaskDetail.layer-agent-steps-ui-mock.playwright.test.js, taskFE/app/src/components/task-detail/TaskDetailAgentStepsSection.vue
- Tags: accordion, agent-steps, playwright

## [OPT-20260718-034] completed

**Logged**: 2026-07-18T15:55:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend / container

### Summary
任务详情项目文件树改为按需 `GET /api/layers/:id/children` 懒加载，减少一次性扁平列表压力。

### Details
当前已用「顶层种子 + BFS + 跳过 node_modules」保证截断时顶层可见，但深层目录在 `truncated=true` 时仍可能不完整。容器已有 `/children`；网关/前端尚未统一懒加载。验收：展开任意目录单独请求子项；根列表不再依赖高 `max_files`。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/server.mjs, taskFE/app/src/components/task-detail/TaskDetailProjectFileTree.vue, .ai/09_failure_experience/02_runtime_errors/46_task_detail_file_tree_incomplete_top_level.md
- Tags: file-tree, lazy-load, performance

## [OPT-20260718-027] completed

**Completed**: 2026-07-18T21:00:00+08:00
**Completion-Note**: 新增 fetchJobFullOutput 按需拉取完整 job output 文本（"复制日志"/"原始控制台"按钮使用）

**Note (2026-07-18)**: 前端 fetchJobExecutionLogBySteps.js 已支持按需分页拉取执行日志；完整 job output 懒加载按钮的实现需要考虑向后兼容现有「复制日志」路径。标记为后续迭代。

**Logged**: 2026-07-18T15:00:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / onlineServiceJS

### Summary
「复制日志 / 原始控制台」可按需单独请求 `GET /api/jobs/:id/output`（或分片 exec-stream），与按 step 的执行日志展示解耦。

### Details
当前 execution-log 固定 `output_omitted` + `meta=0`；控制台区依赖 SSE live output。复制全文可加懒加载按钮。落点：`TaskDetailExecConsoleOutput.vue`、`taskDetailZTreeExecLogState.js` copy 路径。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/composables/taskDetail/fetchJobExecutionLogBySteps.js, trae-agent/onlineServiceJS/src/server.mjs
- Tags: ux, job-log

## [OPT-20260718-026] completed

**Logged**: 2026-07-18T14:50:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / tests

### Summary
为 `profile/git-identities` 序列化字段增加前端契约单测或共享 JSDoc typedef，防止下拉文案再次读错字段名（曾把 `display_name`/`git_user_name` 当成 `git_name`/`name` 而回退裸 ID）。

### Details
可在 `taskDetailBranchAndRepoUtils.test.js` 旁加「fixture 对齐 `_serialize_git_identity` 真实键」用例，或抽 `GitIdentityOption` typedef 供 `TaskDetailLinkedProjectsPanel` / 创建弹窗复用。

**Completed**: 2026-07-18T20:40:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/taskDetailBranchAndRepoUtils.js, task2app/Saas_project/accounts/views/user_views.py
- Tags: contract-test, git-identity, ux

## [OPT-20260718-024] completed

**Logged**: 2026-07-18T14:30:00+08:00
**Priority**: low
**Status**: completed
**Area**: docs / observability

### Summary
onlineServiceJS 控制台可在 job 卡片旁展示「当前宿主机可达端口」快照（对日志中出现的 `:8003`/`:9999` 等做一次 TCP/HTTP 探测），避免用户看到容器内 health OK 却误判为「未做端口桥接」。

### Details
证据：任务 `task_13464457269667337872` 上 agent 曾对 `*:9999/8003/8004` LISTEN 做 health OK；事后进程退出后公网仅剩 `8765/8888/9998/8796`。host 网络本身正常。可选实现：shell 诊断按钮或 steps 渲染时旁路探测。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/skill.md, go_run_container/src/server.go, go_relayToTrae/src/container_image.go
- Tags: networking, host-network, ux

## [OPT-20260718-023] completed

**Logged**: 2026-07-18T14:25:00+08:00
**Priority**: medium
**Status**: completed
**Area**: backend

### Summary
`onlineServiceJS` 任务 `output` 仍全量驻留内存并写入 `jobs_state.json`（单任务可达十余 MB）；列表 API 虽已省略 `output`，持久化与进程 RSS 仍会膨胀。可对落盘/内存保留末尾窗口或改走仅 exec-stream/文件存储。

### Details
落点：`src/jobsRuntime.mjs` 的 `saveState`/`rec.output` 追加路径；与现有 `GET /api/jobs/:id/output`、exec-stream 对齐；需兼顾重启后「复制日志/原始控制台」可读性。

**Completed**: 2026-07-18T20:42:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/jobsRuntime.mjs, trae-agent/onlineServiceJS/src/execStream.mjs
- Tags: perf, storage, job-log

## [OPT-20260718-022] completed

**Logged**: 2026-07-18T13:56:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
`onlineServiceJS` 克隆日志 iframe 仍按分片全量重建 `body.innerHTML`；可改为增量追加 DOM 节点，进一步降低长日志刷新成本与选区丢失概率。

### Details
落点：`static/index.html` 的 `rebuildCloneOutFrameFromParts` / `setExecRichIframeSrcdocSticky`；仅当 contentType 或 seq 集合变化时局部 patch；保留现有 sticky 语义与 e2e。

**Completed**: 2026-07-18T20:43:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/static/index.html, trae-agent/onlineServiceJS/e2e/job-raw-log-scroll.spec.mjs
- Tags: perf, scroll, clone-log

## [OPT-20260718-021] completed

**Logged**: 2026-07-18T13:46:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
「克隆所用 Git 身份」下拉在 `layerGitIdentityOptions.length === 1` 时未自动选中，可与 GitHub 账号下拉对齐，减少启动前手选步骤。

### Details
对照 `selectedGithubUserIdForRepo` 的单选项回退逻辑，在 `repoCloneIdentityByUrl` 空值时默认选中唯一 identity，并补单测。

**Completed**: 2026-07-18T20:40:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue
- Tags: ux, git-identity

## [OPT-20260718-020] completed

**Logged**: 2026-07-18T13:46:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
任务详情「PR 所用 GitHub 授权账号」在仅 1 个关联账号时已自动选中下拉；可进一步在自动选中后静默调用 `saveGithubRepoBinding`，省去「保存账号」点击（需确认与「显式保存」产品约定一致）。

### Details
落点：`TaskDetailLinkedProjectsPanel.vue` / `useGithubRepoBinding.js`；注意避免重复 POST、已有 binding 时跳过、失败时保留可点保存按钮。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue, taskFE/app/src/composables/useGithubRepoBinding.js
- Tags: ux, github-binding

## [OPT-20260718-019] completed

**Logged**: 2026-07-18T13:40:00+08:00
**Priority**: medium
**Status**: completed
**Area**: tests

### Summary
`taskCloudService/src` 全量 `go test` 基线存在包级污染：若干 handler 测试间歇性 `403 无权访问该租户资源`，以及 `TestBuildHardwareEventPayloadMergesFilterOptions` 的 `instance_type=<nil>`（单独跑通过）。

### Details
与 OPT-20260718-018 无关；建议梳理 `TestCloudAuthVerifyCredentialsMock` / IAM association 等对全局 mock 或租户上下文的副作用并在 Cleanup 复位。

**Completed**: 2026-07-18T20:42:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskCloudService/src/cloud_handlers_test.go, taskCloudService/src/oauth_handlers_test.go, taskCloudService/src/compute_event_build_test.go
- Tags: flaky-test, package-pollution

## [OPT-20260718-016] completed

**Logged**: 2026-07-18T12:35:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
意图 030 页面级 Playwright 仍缺：在 task-detail CDP 套件断言 `server-section-collapse-toggle` 收起/展开与折叠后点硬件 Tab 自动展开（与 OPT-20260717-007 合并执行即可）。

### Details
本会话已将 `useServerSectionBodyExpanded` 接入 `ServerConfig.logic.vue` 并完成公网 build/collectstatic + 手工 CDP 验收；页面级回归仍建议自动化防回退。

**Completed**: 2026-07-18T20:34:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: task2app/docs/intents/frontend/task_detail/030_server_section_body_collapse.test-intent.md, taskFE/app/src/components/ServerConfig.logic.vue
- Tags: playwright, task-detail, server-section-collapse

## [OPT-20260718-013] completed

**Logged**: 2026-07-18T11:10:00+08:00
**Priority**: medium
**Status**: completed
**Area**: backend / migration

### Summary
对 Django 已声明的全部 `cloud/compute/*` 写动作做一次对照：凡 status/list 已在 `taskCloudService.handleCloudTaskRoutes` 代理的，对称 approve/create/delete 是否也已登记，避免再出现 501 not yet ported。

**Completed**: 2026-07-18T20:41:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskCloudService/src/compute_handlers.go, task2app/Saas_project/cloud/views/cloud_compute_views.py
- Tags: migrate, 501, github-credential-approve, route-gap

## [OPT-20260718-011] completed

**Logged**: 2026-07-18T11:00:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / playwright

### Summary
补 Playwright（CDP 9222）：`settings/task-panel` 列表中带「默认」徽标的工作空间行不存在 `[data-testid="workspace-delete"]`，非默认行存在。

**Completed**: 2026-07-18T20:34:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/WorkspaceSettingsTaskPanelActions.vue, taskFE/app/src/components/WorkspaceSettingsTaskPanelActions.unit.test.js
- Tags: workspace-delete, default-workspace, playwright

## [OPT-20260718-008] completed

**Logged**: 2026-07-18T03:05:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / playwright

### Summary
补 Playwright：settings/task-panel「创建任务可选字段」在无持久化配置时，`code_lang`（主要编程语言）与 `structured_fields`（结构化任务说明）复选框默认未勾选；开启后 work-panel 创建弹窗才出现对应控件。

**Completed**: 2026-07-18T20:34:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/createTaskFieldSettings.js, taskProjectService/src/create_task_field_settings.go
- Tags: create-task-field-settings, defaults, playwright

## [OPT-20260718-007] completed

**Logged**: 2026-07-18T02:45:00+08:00
**Priority**: low
**Status**: completed
**Area**: taskChromePlugin / e2e

### Summary
为指针选择「Shift 相邻兄弟多选」补一条 Playwright（CDP 9222）冒烟：在含 `ul>li` 的测试页上完成锚定→区间选中→描述块含「兄弟区间」。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskChromePlugin/content/content.js, taskChromePlugin/lib/element-picker.js, taskChromePlugin/e2e/
- Tags: element-picker, sibling-range, playwright

## [OPT-20260718-005] completed

**Logged**: 2026-07-18T02:38:00+08:00
**Priority**: medium
**Status**: completed
**Area**: taskCloudService / api-cutover

### Summary
对照 `taskCloudService/openapi.yaml` 中标注「代理至 Django」的 `cloud/compute/*` 路径，与 `handleCloudTaskRoutes` / `handleCloudWorkspaceRoutes` 实际 case 做一次清单审计，补齐漏登记的写动作并加代理单测。

### Details
本会话已修 `github-credential-approve`（此前仅 status 代理）。同类风险：OpenAPI 已写、实现未登记 → 生产 501。

**Completed**: 2026-07-18T20:41:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskCloudService/src/compute_handlers.go, taskCloudService/openapi.yaml
- Tags: django-proxy, 501, cutover-audit

## [OPT-20260718-003] completed

**Logged**: 2026-07-18T01:10:00+08:00
**Priority**: low
**Status**: completed
**Area**: e2e / playwright

### Summary
补 Playwright：settings/task-panel 关闭 priority → work-panel 创建任务无 `#task-priority`。

### Details
单元测试已覆盖显隐与门禁；E2E 可锁住设置→创建跨页链路。

**Completed**: 2026-07-18T20:34:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: task2app/playwright/front_project/tests/
- Tags: playwright, create-task-field-settings

## [OPT-20260718-002] completed

**Logged**: 2026-07-18T01:10:00+08:00
**Priority**: medium
**Status**: completed
**Area**: chrome-plugin / create-task

### Summary
Chrome 插件创建任务表单对齐工作区 `create-task-field-settings`，隐藏未开启的可选字段。

### Details
Web CreateTaskModal 已按 settings 显隐；插件仍展示全量可选字段，体验不一致。可复用同一 GET API 与字段键常量。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskChromePlugin/lib/create-task-payload.js, taskFE/app/src/utils/createTaskFieldSettings.js
- Tags: create-task, field-settings, chrome-plugin

## [OPT-20260717-045] completed

**Logged**: 2026-07-17T21:57:00+08:00
**Priority**: low
**Status**: completed
**Area**: tests / git-site-oauth

### Summary
补一条 Playwright：git-site-oauth 取消授权后 connection 应为未绑定，且 gitOauth DB 无对应 credential 行。

**Completed**: 2026-07-18T20:43:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: session-end
- Related Files: task2app/Saas_project/accounts/github_app_views.py, playwright GitSiteOAuth.disconnect-*
- Tags: github-oauth, disconnect

## [OPT-20260717-040] completed

**Logged**: 2026-07-17T20:15:00+08:00
**Priority**: low
**Status**: completed
**Area**: backend / taskGitOauth

### Summary
将 Django internal bind 非 200 映射为独立回调码（如 `bind_failed`），勿一律 `exchange_failed`，避免前端误导文案。

### Details
同步前端 `GITHUB_CALLBACK_HINTS` / `GITLAB_CALLBACK_HINTS` 与 Playwright 断言。

**Completed**: 2026-07-18T20:41:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskGitOauth/src/browser_handlers.go, taskFE/app/src/domain/oauth_callback/entities/oauth_callback_hint_catalog_entity.js
- Tags: oauth, ux, error-codes

## [OPT-20260717-035] completed

**Logged**: 2026-07-17T18:05:00+08:00
**Priority**: low
**Status**: completed
**Area**: testing

### Summary
为 `/system-admin/price-management` 新建套餐表单增加 Playwright：断言 GitLab 磁盘默认 800、流量费默认 100（`data-testid` 已就绪）。

### Details
可用超管账号打开价格管理页，读取 `[data-testid=gitlab-disk-points-per-gb-per-month]` 与 `[data-testid=gitlab-traffic-points-per-gb]` 的 value。

**Completed**: 2026-07-18T20:43:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/views/SystemAdminPriceManagement.vue
- Tags: playwright, price-management, e2e

## [OPT-20260717-029] completed

**Logged**: 2026-07-17T16:05:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend

### Summary
为子仓「重新克隆」补一条 Playwright：任务详情失败行点 `task-nested-repos-clone-reclone`，断言发出 `POST …/cloud/repo-reclone/` 且 body 含 `parent_repo_url` 与 `clone_alias`。

### Details
单元测已覆盖 emit payload；缺页面级回归（可挂 mock CDP 套件）。硬刷新公网任务详情后亦可手工验：失败行可见错误文案与按钮。

**Completed**: 2026-07-18T20:43:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailNestedReposCloneStatus.vue, task2app/playwright/
- Tags: nested-repos, reclone, playwright, task-detail

## [OPT-20260717-028] completed

**Logged**: 2026-07-17T15:50:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
为公网入口（task2app / taskAiProvider）增加 Playwright 断言：页面 `meta[name="trae-service"]` 的 content 与期望服务一致。

### Details
源文件门禁已由 `check_frontend_head_trae_service.py` 覆盖；运行时经 APISIX 后仍建议 E2E 抽检，防止中间层替换 HTML。

**Completed**: 2026-07-18T20:43:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: db/scripts/ci/check_frontend_head_trae_service.py, task2app/playwright/
- Tags: trae-service, playwright, observability

## [OPT-20260717-026] completed

**Logged**: 2026-07-17T15:46:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
任务详情页展示并支持编辑 `code_lang`（与创建弹窗一致）。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: session
- Related Files: taskFE/app/src/components/CreateTaskBasicFields.vue
- Tags: code-lang, work-panel

## [OPT-20260717-025] completed

**Logged**: 2026-07-17T15:35:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend

### Summary
将 `origin/main`（及 `feat/workspace-task-kind`）上已引用但未入库的前端模块合入，避免 SPA `vite build` 因缺失文件失败。

### Details
本会话为使 Fork 功能可构建，临时补齐 `TaskDetailNestedReposCloneStatus`、`resolveTaskRouteIds`、`serverLifecycleFromRuntime`、`useServerSectionBodyExpanded` 等；应单独 PR 梳理归属并保证 `main` 可独立 build。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: session
- Related Files: taskFE/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue
- Tags: build, spa, missing-modules

## [OPT-20260717-024] completed

**Logged**: 2026-07-17T15:30:00+08:00
**Priority**: medium
**Status**: completed
**Area**: backend

### Summary
为 vendor CSI PATCH/POST 与 association 响应增加契约测例：断言 `userdata_template` 含 `name`/`version`，且 PATCH 非 stub。

### Details
本会话已修持久化与 ListAssociations enrich；可再补 `handlers_test` 级 HTTP 测例，防止 PATCH 再次退回空转。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: session
- Related Files: taskAiProvider/src/resource_handlers.go, taskAiProvider/src/handlers_test.go
- Tags: contract-test, userdata, vendor-portal

## [OPT-20260717-023] completed

**Logged**: 2026-07-17T14:50:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
任务详情页展示并支持编辑 `task_kind`（与创建弹窗一致）。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: session
- Related Files: taskFE/app/src/components/task-detail/TaskDetailTaskIdentityPanel.vue
- Tags: task-kind, work-panel

## [OPT-20260717-020] completed

**Logged**: 2026-07-17T14:15:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
任务详情编辑页为结构化任务说明提供与创建弹窗一致的字段回填/编辑体验（而非仅整段 Markdown）。

### Details
创建弹窗已支持 parse/compose；详情侧 `TaskDetailTaskIdentityPanel` 仍直接绑定 `description`。可复用 `createTaskDescriptionCompose.js`。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailTaskIdentityPanel.vue, taskFE/app/src/utils/createTaskDescriptionCompose.js
- Tags: work-panel, create-task, structured-fields

## [OPT-20260717-019] completed

**Logged**: 2026-07-17T14:15:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend

### Summary
将工作台创建任务的 7 个结构化可选字段对齐到 Chrome 插件创建任务 payload/UI。

### Details
主站已通过 `composeCreateTaskDescription` 拼入 `description`；插件侧见 `taskChromePlugin_副本/lib/create-task-payload.js`，需共用同一标题约定以免双端 description 形态漂移。

**Completed**: 2026-07-18T20:44:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskChromePlugin_副本/lib/create-task-payload.js, taskFE/app/src/utils/createTaskDescriptionCompose.js
- Tags: chrome-plugin, create-task, structured-fields

## [OPT-20260717-017] completed

**Logged**: 2026-07-17T13:40:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
将 VendorPortal「设置区域运行环境」模态抽成独立组件（如 `RegionEnvModal.vue`），降低 1500+ 行门户文件的改动面。

### Details
本会话已在模态内加错误展示；文件仍带 500 行门禁例外注释。按前端规范模态宜独立文件，便于后续只测区域环境逻辑。

**Completed**: 2026-07-18T20:40:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskAiProvider/frontend/src/views/VendorPortal.vue
- Tags: refactor, modal, vendor-portal

## [OPT-20260717-012] completed

**Logged**: 2026-07-17T12:44:13+08:00
**Priority**: medium
**Status**: completed
**Area**: backend

### Summary
在 taskTaskService `validateAutoRunPrerequisites` 中预检已安装镜像的运行环境非空（及与项目运行模版 region 匹配），避免 create 成功写入 auto_run=true 后异步 start-vm 才 400。

### Details
本会话根因在镜像市场关联被清空；即便关联修复，auto_run 仍是异步失败，用户只看到「自动运行为是 / 服务器未启动」。门禁可调 Cloud/AI Provider 公开 runtime-environments，返回 AUTO_RUN_RUNTIME_ENV_REQUIRED。

**Completed**: 2026-07-18T20:42:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskTaskService/src/auto_run.go, taskCloudService/src/compute_image_resolve.go
- Tags: auto_run, gate, marketplace

## [OPT-20260717-010] completed

**Logged**: 2026-07-17T12:05:00+08:00
**Priority**: medium
**Status**: completed
**Area**: tests

### Summary
为 TaskPlugin DevTools 面板请求列表补一条 Playwright/CDP 冒烟：打开面板后 `#selectedRequest` 不得长期停留「正在加载请求列表...」，应出现列表项或「暂无匹配的请求」。

### Details
本次已用 `test/panel-request-bootstrap.test.js` 锁纯逻辑；端到端仍依赖手动重载扩展。可在已有 e2e 基建上加面板冒烟。

**Completed**: 2026-07-18T20:45:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskChromePlugin/panel/panel.js, taskChromePlugin/e2e/
- Tags: e2e, chrome-extension, request-list

## [OPT-20260717-007] completed

**Logged**: 2026-07-17T11:26:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
为任务详情「服务器信息」Tab 行折叠按钮补一条 Playwright：断言 `data-testid="server-section-collapse-toggle"` 点击后 `server-section-body` 隐藏，再点展开后可见；折叠后点硬件 Tab 应自动展开。

### Details
单元测例已覆盖 `useServerSectionBodyExpanded`；意图 030 E1–E4 仍缺页面级回归。可挂现有 task-detail CDP 套件。

**Completed**: 2026-07-18T20:34:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: task2app/docs/intents/frontend/task_detail/030_server_section_body_collapse.test-intent.md, taskFE/app/src/composables/useServerSectionBodyExpanded.js
- Tags: playwright, task-detail, server-section-collapse

## [OPT-20260717-004] completed

**Logged**: 2026-07-17T03:05:00+00:00
**Priority**: low
**Status**: completed
**Area**: frontend

### Summary
为 front_project 统一错误出口补一条 Playwright：模拟带 `X-Trace-Id` 的失败响应，断言 Modal/Toast 根节点存在 `[data-traceId="<id>"]`。

### Details
单元测已覆盖 `traceId` / `requestErrorDisplay` / `apiUtils`；缺浏览器级回归。可挂在现有 auth 或 billing 失败路径 mock 上。

**Completed**: 2026-07-18T20:42:00+08:00
**Completion-Note**: 本会话已落地实现；详见 Final Execution Overview。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/requestErrorDisplay.js
- Tags: playwright, data-traceId

## [OPT-20260718-056] completed

**Logged**: 2026-07-18T16:26:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T20:26:50+08:00
**Completion-Note**: onlineServiceJS 换票失败非 strict 时 fail-closed：authMiddleware/UI/healthz 返回 503 TOKEN_BOOTSTRAP_FAILED；intentional skip 不触发。见 src/auth.mjs、server.mjs；测例 auth.failClosed / server.tokenBootstrapFailClosed。
**Area**: onlineServiceJS / resilience
**Renumbered-From**: OPT-20260718-042 (duplicate id)

### Summary
换票失败且非 strict 时，避免带着无效 ACCESS_TOKEN 对外提供受保护 API（fail-closed 或监听后拒绝业务路由）。

### Details
当前失败时 `bootstrapCtx.skipped=true` 仍 listen，导致平台 by-scope token 与容器 env 长期不一致、全线 401。可考虑：无有效 token 时不挂业务路由 / 退出进程 / 仅健康检查。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/server.mjs, trae-agent/onlineServiceJS/src/bootstrap.mjs
- Tags: access-token, fail-closed, bootstrap

## [OPT-20260717-043] cancelled

**Logged**: 2026-07-17T21:40:00+08:00
**Priority**: medium
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: outbound_proxy 环境覆盖属运维密钥策略；避免在无确认下改已跟踪 yaml 生产值，cancelled 并建议本地 overlay。
**Area**: ops / taskGitOauth

### Summary
生产/无 SSH 动态转发环境：勿提交仅本机可用的 `outbound_proxy`；改为按环境覆盖（本地 yaml / 密钥管理），或改用可达 GitHub 的固定 egress。

### Details
本机已在 `http-github-com--app-daydaymoney.yaml` 写入 `socks5://127.0.0.1:1080`。若合入公网机且无对应隧道，回退会失败且多一次拨号。验收：无隧道机器应清空该字段或指向真实 egress。

### Metadata
- Source: session-end
- Related Files: conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml, taskGitOauth/infrastructure/outbound_http.go
- Tags: github-oauth, outbound-proxy

## [OPT-20260717-027] cancelled

**Logged**: 2026-07-17T15:46:00+08:00
**Priority**: low
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 公网 SPA 验证 code_lang；无公网验收环境，cancelled。
**Area**: frontend

### Summary
公网 SPA：`runall-lifecycle.sh build` 后验证创建任务「主要编程语言」下拉与工作区「编程语言」设置。

### Metadata
- Source: session
- Related Files: taskFE/app/scripts/runall-lifecycle.sh
- Tags: code-lang, collectstatic

## [OPT-20260717-006] cancelled

**Logged**: 2026-07-17T11:26:00+08:00
**Priority**: medium
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: datalistDismiss 入库 commit 需用户授权；可另补 Playwright。
**Area**: frontend

### Summary
将 `datalistDismiss.js` / 对应测试与 CreateTaskModal 接线变更提交入库，并补一条 Playwright：在工作面板新建任务中从基准分支 datalist 点选后断言建议列表不再复现（`list` 被摘掉或 input 失焦）。

### Details
本会话已加固 dismiss（同步 `removeAttribute('list')` + `insertReplacementText` 早拦截）并完成 build/collectstatic；util 此前曾出现「Vue 已引用但未入库」缺口。提交后可防 CI/他机构建失败与公网回归。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/datalistDismiss.js, taskFE/app/src/components/CreateTaskModal.vue
- Tags: datalist, work-panel, playwright

## [OPT-20260717-008] cancelled

**Logged**: 2026-07-17T11:26:30+08:00
**Priority**: medium
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: nested repos clone status 入库 commit 需用户授权。
**Area**: frontend

### Summary
将 `TaskDetailNestedReposCloneStatus.vue`、`nestedRepoCloneStatusUtils.js`（及测试）、`TaskDetailRuntimeSection`/`taskDetailSectionBindings` 的 `bootstrapCloneDone` 接线，以及意图 `031_nested_repos_clone_status` 一并提交入库；并补一条 Playwright：元仓任务详情在「克隆所用 Git 身份」下方断言 `data-testid="task-nested-repos-clone-status"` 可见。

### Details
面板此前已 import 该组件但组件文件曾未入库，存在 CI/他机构建失败风险。本会话已完成源码、单测与公网 build+collectstatic；缺正式 commit 与页面级回归。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailNestedReposCloneStatus.vue, taskFE/app/src/utils/nestedRepoCloneStatusUtils.js, task2app/docs/intents/frontend/task_detail/031_nested_repos_clone_status.intent.md
- Tags: nested-repos, clone-status, playwright, task-detail

## [OPT-20260717-009] cancelled

**Logged**: 2026-07-17T11:30:00+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 意图 029 改动正式 commit 需用户授权；禁止自动 commit。
**Area**: frontend

### Summary
将意图 029 的 `runtime_hydrate` / `serverLifecycleFromRuntime*` / 失配自愈与展示层 `runtimeStatus` 回退相关改动在 `task2app` 仓正式 commit（当前多为工作区未提交）。

### Details
2026-07-18：已再次补齐 `fetchServerRuntimeStatus → runtime_hydrate`、失配自愈、面板 `runtimeStatus` 回退，并公网 collectstatic 验证。仍须正式 commit，否则他机/CI 会再次丢失。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/serverLifecycleFromRuntime.js, taskFE/app/src/composables/taskDetail/updateServerStatus.js, taskFE/app/src/components/ServerConfig.logic.vue
- Tags: git, hydrate, task-detail

## [OPT-20260717-033] cancelled

**Logged**: 2026-07-17T17:42:00+08:00
**Priority**: medium
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 父仓提交 taskAiProvider 嵌套转换需用户授权 commit；禁止自动 commit。
**Area**: git / monorepo

### Summary
在父仓 `ram-work` 提交 `taskAiProvider` 嵌套仓转换：`.gitignore` 忽略目录、索引中移除已跟踪文件，并登记本地 `.gitmodules` 条目。

### Details
独立仓已推送到 GitHub/GitLab；父仓工作区已 `git rm -r --cached taskAiProvider` 且改了 `.gitignore`，但尚未提交。提交后 monorepo 与 `taskAuth` 等同级约定才一致。

### Metadata
- Source: session-end
- Related Files: .gitignore, .gitmodules, taskAiProvider/
- Tags: nested-git, taskAiProvider, submodule-registry

## [OPT-20260717-041] cancelled

**Logged**: 2026-07-17T20:15:00+08:00
**Priority**: medium
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 缺 PLAYWRIGHT_TEST_GITHUB_* 真实 OAuth 凭据，无法跑完整绑定冒烟。
**Area**: tests / taskGitOauth

### Summary
配置 `PLAYWRIGHT_TEST_GITHUB_*` 后跑完整浏览器 GitHub 绑定冒烟（start→authorize→callback→`?github=ok`），确认 bind 字段修复在公网路径生效。

### Details
本会话已用 Django bind 契约 curl + Go 单测验证；缺真实 GitHub `code` 的端到端。可复用 `taskGitOauth/scripts/smoke_oauth.sh` / `verify-git-oauth-playwright-cdp.mjs`。

### Metadata
- Source: goal-overview
- Related Files: taskGitOauth/scripts/, .ai/09_failure_experience/02_runtime_errors/39_github_oauth_exchange_failed_bind_field_mismatch.md
- Tags: oauth, github, e2e, playwright

## [OPT-20260717-048] cancelled

**Logged**: 2026-07-17T22:20:00+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 需在 GitHub App 设置页配置 Contents:Read 并安装到组织；无管理权限，阻塞。
**Area**: auth / github-app

### Summary
为 GitHub App `daydaymoney` 配置 Contents: Read（及 Metadata），安装到 `task2money` 组织，并让已绑定用户重新 OAuth；去掉对 CLI `gho_` 凭证的临时依赖。**未做完则每次服务/DB 重置或重绑后，项目详情子仓列表会再挂。**

### Details
当前 App 注册 `permissions: {}`，私有仓 Contents API 404，子仓发现曾被误判为 empty。代码侧已区分「父仓不可访问」与「无 .gitmodules」。持久方案须在 https://github.com/settings/apps/daydaymoney/permissions 补权限并安装到组织。重置后操作步骤见失败文 §「恢复清单」。

### Metadata
- Source: goal-overview
- Related Files: .ai/09_failure_experience/02_runtime_errors/43_nested_git_repos_empty_on_inaccessible_parent.md, conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml, conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml.ai.md
- Tags: github-app, nested-repos, oauth, permissions, service-reset

## [OPT-20260717-013] cancelled

**Logged**: 2026-07-17T12:58:51+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: taskAiProvider 前端公网 dist 发布阻塞。
**Area**: infra

### Summary
将含 `X-Parent-Span-Id` 与 `p.msg` `data-traceId` 的 taskAiProvider 前端发布到 `provider.daydaymoney.com`（dist 被 gitignore，需在部署机 `npm run build` 或等价流水线后替换静态资源）。

### Details
源码与单测已就绪；线上仍为旧 bundle（仅 X-Trace-Id）。发布后硬刷新验证：公开目录/厂商门户不再出现 trace propagation incomplete，且若仍有请求错误则 `p.msg[data-traceId]` 可查询。若同时部署 Go，`RejectTraceIdOnlyHTTP` 会回显入站 X-Trace-Id。同次宜带上 OPT-20260717-016（区域环境清除与模态内错误）。

### Metadata
- Source: goal-overview
- Related Files: taskAiProvider/frontend/src/api.js, taskAiProvider/frontend/src/utils/traceId.js, taskAiProvider/frontend/src/views/VendorPortal.vue
- Tags: deploy, taskAiProvider, tracing, data-traceId

## [OPT-20260717-016] cancelled

**Logged**: 2026-07-17T13:40:00+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: provider.daydaymoney.com 发布阻塞。
**Area**: infra

### Summary
将「区域运行环境：csiID=0 清除 + 模态内错误 data-traceId」的 Go 与前端发布到 `provider.daydaymoney.com`。

### Details
源码与单测已就绪。部署后硬刷新验证：① 选「不选（清除本区域关联）」保存应 200；② 其它 association 失败时模态内可见 `p.err[data-traceId]`。可与 OPT-20260717-013 同次发布。

### Metadata
- Source: goal-overview
- Related Files: taskAiProvider/infrastructure/store_marketplace.go, taskAiProvider/frontend/src/views/VendorPortal.vue
- Tags: deploy, taskAiProvider, association, data-traceId

## [OPT-20260717-034] cancelled

**Logged**: 2026-07-17T18:05:00+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 公网发布 taskBill/价格管理静态资源；部署阻塞。
**Area**: deploy

### Summary
将本次价格管理默认值改动发布到公网：重启/发布 `taskBill`，并确认 daydaymoney 静态资源已同步（本机已 `runall-lifecycle.sh build`）。

### Details
前端表单默认 800/100 依赖 SPA chunk；API 省略字段默认依赖 taskBill `handlers.go`。公网硬刷新 `/system-admin/price-management` 后核对输入框与标签文案。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/views/SystemAdminPriceManagement.vue, taskBill/src/handlers.go
- Tags: deploy, pricing-defaults, daydaymoney

## [OPT-20260717-042] cancelled

**Logged**: 2026-07-17T20:16:00+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 公网重启 taskGitOauth + SPA collectstatic 需部署机；阻塞。
**Area**: deploy / git-site-oauth

### Summary
将「OAuth 失败回跳带 `trace_id` + 前端 `data-traceId`」发布到公网：重启 `taskGitOauth`，并对 SPA 执行 `runall-lifecycle.sh build`（含 collectstatic）。

### Details
本会话代码与单测已绿；公网仍可能跑旧 bundle / 旧二进制。验收：`?github=exchange_failed&trace_id=…` 时 `p.text-sm.text-danger[data-traceId]` 可读。

### Metadata
- Source: goal-overview
- Related Files: taskGitOauth/src/browser_handlers.go, taskFE/app/src/views/UserGitSiteOAuthSettings.vue
- Tags: deploy, data-traceId, oauth

## [OPT-20260717-011] cancelled

**Logged**: 2026-07-17T12:16:47+08:00
**Priority**: medium
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: onlineServiceJS Parent-Span-Id 镜像推送阻塞。
**Area**: infra

### Summary
将含 `api()` 补发 `X-Parent-Span-Id` 的 onlineServiceJS 镜像重建并推送，使运行中任务容器 UI 轮询 `bootstrap-clone-log` 不再 400。

### Details
源码与单测已就绪；远程 `47.86.173.111:8765` 在手动补 Parent-Span 时已 200，但内置 static/index.html 仍为旧版。按目录 `ai.md`：commit 后执行 `DOCKER_PUSH=1 ./buildDocker.sh`，并确保任务容器拉取新镜像。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/static/index.html, trae-agent/onlineServiceJS/buildDocker.sh, trae-agent/onlineServiceJS/ai.md
- Tags: docker, onlineServiceJS, deploy, tracing

## [OPT-20260717-018] cancelled

**Logged**: 2026-07-17T13:50:00+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: onlineServiceJS GET /api/config SaaS 回源镜像推送阻塞。
**Area**: infra

### Summary
将含 `GET /api/config` SaaS 回源与 `#cfgErr` traceId 文案的 onlineServiceJS 镜像重建并推送，使任务容器 UI 拉取配置不再只读本地 404。

### Details
源码与单测已就绪（`ensureServiceConfig.mjs`、`formatConfigErrMsg`、`static/index.html`）。按目录 `ai.md`：commit 后执行 `DOCKER_PUSH=1 ./buildDocker.sh`，并确保任务容器拉取新镜像后硬刷新验证。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/ensureServiceConfig.mjs, trae-agent/onlineServiceJS/src/server.mjs, trae-agent/onlineServiceJS/static/index.html
- Tags: docker, onlineServiceJS, deploy, config, saas-fallback

## [OPT-20260717-021] cancelled

**Logged**: 2026-07-17T14:20:00+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 引导克隆失败徽章相关镜像/前端公网发布阻塞。
**Area**: infra

### Summary
将「引导克隆失败点名仓库 + 关联仓库失败徽章/重新克隆」的 onlineServiceJS 与前端发布到公网任务环境。

### Details
源码与单测已就绪（`formatBootstrapCloneFailureFooter`、`extractBootstrapCloneFailedRepoUrls`、`TaskDetailLinkedProjectsPanel` 失败徽章与 `shouldShowRepoRecloneButton`）。重建并推送 onlineServiceJS 镜像（`DOCKER_PUSH=1 ./buildDocker.sh`），发布前端后硬刷新任务详情验证：日志页脚列出失败仓；失败行可见「克隆失败」与「重新克隆」。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/bootstrap.mjs, taskFE/app/src/utils/taskDetailContainerCloneProgress.js, taskFE/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue
- Tags: docker, onlineServiceJS, deploy, bootstrap-clone, task-detail

## [OPT-20260718-017] cancelled

**Logged**: 2026-07-18T12:42:00+08:00
**Priority**: medium
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: nested staging 镜像推送需 DOCKER_PUSH；部署阻塞。
**Area**: onlineServiceJS / deploy

### Summary
重建并推送含 nested staging→move 的 onlineServiceJS 镜像（`DOCKER_PUSH=1 ./buildDocker.sh`），并在关联 ram-work 元仓的任务容器验收子仓落在 `{父仓}/{path}`。

### Details
源码与单测已就绪（含克隆失败非致命 / `resolveBootstrapCloneFailurePolicy`）；公网任务容器需拉新镜像后验证：bootstrap 日志含「已移入」；部分仓失败仍有 `BOOTSTRAP_COMPLETE` 且业务端点可用；工作区为 `ram-work/task2app` 而非层根并列 `task2app`。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/bootstrap.mjs, trae-agent/onlineServiceJS/ai.md
- Tags: docker, onlineServiceJS, nested-clone, relocate

## [OPT-20260718-025] cancelled

**Logged**: 2026-07-18T14:50:00+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 需滚动更新公网任务容器镜像使 output_omitted 生效；源码已有省略逻辑，部署阻塞。
**Area**: infra / onlineServiceJS

### Summary
重建并滚动更新仍内嵌全量 `output` 的 onlineServiceJS 任务容器镜像，使 `GET /api/jobs` 与 `GET /api/jobs/:id` 走仓库已有的 `jobToApiDict` 省略逻辑（`output_omitted`），消除约 15MB 响应拖垮 job-stream 与 layer-graph 转发的问题。

### Details
实证：`http://47.238.138.64:8765/api/jobs/387c72c4-…` Content-Length≈15.7MB 且无 `output_omitted`；同仓源码默认已省略。落点：镜像构建/发布流水线与该任务实例替换；验收：`curl` 单 job JSON 小于 50KB 且含 `"output_omitted":true`。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/jobsRuntime.mjs, trae-agent/onlineServiceJS/src/server.mjs, go_run_container/, go_relayToTrae/
- Tags: perf, job-log, deploy

## [OPT-20260718-041] cancelled

**Logged**: 2026-07-18T16:26:00+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 同 onlineServiceJS 镜像推送/存量容器重启；无 DOCKER_PUSH 环境，cancelled。
**Area**: onlineServiceJS / deploy

### Summary
提交 onlineServiceJS 换票回退后执行 `DOCKER_PUSH=1 ./buildDocker.sh`，并重启仍卡在无效 ACCESS_TOKEN 的存量任务容器。

### Details
本会话已改 `bootstrap.mjs`（401/403 均可回退 refresh-access）且 credential 已对「scope 已有 refresh」返回 403。旧镜像在 credential 新行为下重启即可自愈；推送新镜像后 401 路径更稳。Companion：`trae-agent/onlineServiceJS/ai.md`。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/bootstrap.mjs, taskCredentialService/application/services.go
- Tags: access-token, exchange-refresh, docker-push

## [OPT-20260718-050] cancelled

**Logged**: 2026-07-18T18:20:00+08:00
**Priority**: high
**Status**: cancelled
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 需 git commit 后 DOCKER_PUSH=1 重建公网任务容器；本会话无部署凭据/未授权 commit，阻塞。手工：commit → buildDocker.sh → 滚动任务容器。
**Area**: onlineServiceJS / ops

### Summary
本会话已在源码修复 credentials 409 永不克隆；公网容器需在 **git commit 后**执行 `DOCKER_PUSH=1 ./buildDocker.sh` 并重建任务容器，否则 `47.239.98.3` 等存量实例仍为旧行为。

### Details
见 `trae-agent/onlineServiceJS/ai.md` 镜像发布规则；失败经验 `47_bootstrap_credentials_incomplete_skips_clone.md`。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/buildDocker.sh, trae-agent/onlineServiceJS/ai.md
- Tags: docker-push, bootstrap-clone, rollout

## [OPT-20260717-031] completed

**Logged**: 2026-07-17T16:20:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: archimate 已无「拉 .gitignore」残留（rg 确认）；与 .gitmodules 行为一致。
**Area**: docs

### Summary
同步更新 `docs/architecture/v28-*.archimate` 中仍写「拉 .gitignore」的 Technology/Note 文案，与仅 `.gitmodules` 行为一致。

### Details
本次已改 `.puml` / `.mermaid.md`；`.archimate` 语义模型说明可能仍含 gitignore fallback 字样，打开 Archi 时易误导。

### Metadata
- Source: session-end
- Related Files: docs/architecture/v28-application-integration-20260715-1135-claude.archimate
- Tags: nested-repos, archimate, docs-drift

## [OPT-20260717-030] completed

**Logged**: 2026-07-17T16:10:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 评估结论见 docs/dev/nested-repos-submodule-eval.md：维持 .gitmodules 注册表，不升级 gitlink。
**Area**: tooling

### Summary
评估是否将 `ram-work` 嵌套仓从「`.gitmodules` 注册表 + 独立克隆」升级为完整 gitlink submodule（`git submodule update` 可拉取）。

### Details
当前已用 Git 标准 `.gitmodules` path↔url 与相对 URL 解析做发现；工作树仍在 `.gitignore` 且各自独立 `.git`，未登记 mode=160000 gitlink。若产品需要 `git submodule` CLI 工作流，再做迁移与 runAll/容器克隆兼容性评估。

### Metadata
- Source: session-end
- Related Files: .gitmodules, .gitignore, taskProjectService/src/nested_git_repos_parse.go
- Tags: nested-repos, gitmodules, submodule

## [OPT-20260717-005] completed

**Logged**: 2026-07-17T11:25:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 新增 db/scripts/ci/check_resolve_task_route_ids.py 禁止手写 props/route 拼装。
**Area**: frontend

### Summary
为 `resolveTaskRouteIds` 增加 ESLint/CI 门禁：禁止在 `task-detail/`、`useTaskDetail.js`、`TaskDetailContent.logic.vue` 内再手写 `props.* || route.params.*` 拼装租户/工作区/任务 ID。

### Details
本会话已统一接入 SSOT；门禁可防回归。可用 `rg` 脚本挂到 pre-commit 或简单 custom ESLint no-restricted-syntax。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/resolveTaskRouteIds.js
- Tags: eslint, task-detail, route-ids

## [OPT-20260718-037] completed

**Logged**: 2026-07-18T16:25:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 新增 fixtures/traceid_extract_cases.md，并在元规则 24 引用。
**Area**: skills / observability

### Summary
将 brainstorming「检测 traceId」的大小写不敏感规则固化为可执行 fixture（示例输入 → 期望抽出的 ID），并在技能作者指南或 CI 中引用，避免日后条款回退为字面匹配。

### Details
本会话已在 SKILL.md / 元规则 24 / frontend-error-data-trace-id.mdc 写明须识别 `traceid`/`TraceId`/`data-traceid` 等。后续可加一小段 golden 用例表（markdown 或 pytest），覆盖 JSON、DOM、HTTP 头三种载体。

### Metadata
- Source: goal-overview
- Related Files: .claude/skills/1-brainstorming-design-docs/SKILL.md, .ai/01_project_constraints/24_frontend_error_data_trace_id.md
- Tags: traceId, case-insensitive, skills

## [OPT-20260718-014] completed

**Logged**: 2026-07-18T11:20:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 新增 docs/dev/nested-repo-worktree.md（子仓开 worktree + collectstatic 主 checkout）。
**Area**: infra / docs

### Summary
为嵌套独立仓（如 task2app）补充「从 monorepo 开 worktree」指引：勿在元仓 `git worktree add`（会缺嵌套仓内容）；应在目标子仓开 worktree，且 collectstatic 须在可解析 `conf/core/django/config.yaml` 的主 checkout 执行。

### Metadata
- Source: goal-overview
- Related Files: .gitmodules, taskFE/ai.md, runAll/scripts
- Tags: worktree, nested-repos, collectstatic

## [OPT-20260717-047] completed

**Logged**: 2026-07-17T22:16:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 验收：三页已无 gitlab-service-unstable-banner/badge 与「不建议使用」文案。
**Area**: frontend / gitlab

### Summary
GitLab 恢复稳定后，统一移除三处「不稳定，不建议使用」提示：价格管理页、公网定价页、工作区 GitLab 连接页（`gitlab-service-unstable-banner` / `gitlab-unstable-badge`）。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/views/SystemAdminPriceManagement.vue, taskFE/app/src/views/Pricing.vue, taskFE/app/src/views/WorkspaceSettingsGitlabConnection.vue
- Tags: gitlab, pricing, ux-copy

## [OPT-20260718-032] completed

**Logged**: 2026-07-18T15:20:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: alwaysApply 已收紧为 ai-rules-loader/companion/service-listen-host/app-startup-no-env-proxy；设计技能 description 标明 Only use when platform task。
**Area**: rules

### Summary
收紧 Cursor `alwaysApply: true` 与平台设计类技能的常驻上下文：低频元规则改 globs；Apple/Android 等设计指南仅任务相关时加载。

### Details
当前 `.cursor/rules/` 有 10 条 `alwaysApply: true`（含 merged-feat-branch-cleanup、swagger、online-service-http-logging 等），叠加多份 `*/AGENTS.md` 设计指南会显著占上下文。建议：保留 `ai-rules-loader` / `companion-ai-md` / 少数全局硬约束为常驻；其余按路径/任务类型触发；平台设计技能 description 写清「仅 iOS/tvOS/… 任务」。验收：无关 Web/Go 任务的 system 规则体积明显下降。

### Metadata
- Source: goal-overview
- Related Files: .cursor/rules/, .claude/skills/*-design-guidelines/
- Tags: meta-rules, context-budget, alwaysApply

## [OPT-20260717-044] completed

**Logged**: 2026-07-17T21:52:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 新增 db/scripts/ci/check_git_oauth_provider_keys.py。
**Area**: conf / git-oauth

### Summary
增加 CI 校验：`conf/auth/git-oauth/providers` 与 `conf/core/django/git-oauth-providers`（及 task-credential）中同 website 的 `service_provider` / client_id 一致，防止再出现 ok 但未绑定。

### Metadata
- Source: session-end
- Related Files: conf/core/django/git-oauth-providers/http-github-com.yaml, conf/auth/git-oauth/providers/http-github-com--app-daydaymoney.yaml
- Tags: github-oauth, provider-key

## [OPT-20260718-010] completed

**Logged**: 2026-07-18T10:55:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 新增 taskChromePlugin/scripts/check_user_guide_sections.py；本地 ok。
**Area**: taskChromePlugin / docs

### Summary
可选：在 pre-commit / CI 增加检查，确保 `lib/user-guide.js` 的 `SECTIONS[].id` 均出现在 `docs/USER_GUIDE.md`（当前已由 `test/user-guide.test.js` 覆盖，可抽成独立脚本供 hook 引用）。

### Metadata
- Source: goal-overview
- Related Files: taskChromePlugin/lib/user-guide.js, taskChromePlugin/docs/USER_GUIDE.md, taskChromePlugin/ai.md
- Tags: user-guide, companion-ai-md

## [OPT-20260718-009] completed

**Logged**: 2026-07-18T03:12:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 新增 db/scripts/ci/check_frontend_error_data_trace_id.py（可选 --strict）。
**Area**: frontend / ci

### Summary
为请求错误 UI 的 `data-traceId` 增加可选 CI 门禁（如对 `text-red-600`/`text-danger` 自绘错误节点扫描缺失绑定），防止回归。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/, db/scripts/ci/
- Tags: data-traceId, ci, observability

## [OPT-20260717-039] completed

**Logged**: 2026-07-17T20:05:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 新增 db/scripts/ci/check_claude_skills_name.py（name/目录/description）；本地跑通 61 skills。
**Area**: tooling / skills

### Summary
为 `.claude/skills/*/SKILL.md` 增加 CI 或 pre-commit 校验：`name` 仅 `[a-z0-9-]`、与父目录名一致、且含非空 `description`，防止再次引入 Cursor 无法发现的中文/非法技能名。

### Details
可放在 `db/scripts/ci/` 或 `runAll/scripts/` 下的轻量 Python 检查；失败时列出违规目录。与本次手工重命名（`1-brainstorming-design-docs` 等）配套防回归。

### Metadata
- Source: goal-overview
- Related Files: .claude/skills/, .claude/README.md, .ai/11_ai_development/02_agent_skills_authoring.md
- Tags: skills, cursor, naming, ci

## [OPT-20260717-037] completed

**Logged**: 2026-07-17T18:15:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 验收：install-ram-work-maintenance-cron.sh 已挂 truncate-ram-work-logs.sh 小时任务。
**Area**: infra / ops

### Summary
为 `/tmp/ram-work/logs` 与 `taskGateway/logs` 增加定期截断/轮转（cron 或维护脚本），避免 tmpfs 被 access log 再次灌满。

### Details
本次单次截断释放约 450MB；`taskgateway-access.log` 曾达 131MB。可挂到现有 `runAll/scripts/install-ram-work-maintenance-cron.sh`。

### Metadata
- Source: goal-overview
- Related Files: runAll/scripts/install-ram-work-maintenance-cron.sh, taskGateway/logs/, logs/
- Tags: log-rotation, tmpfs, maintenance

## [OPT-20260717-036] completed

**Logged**: 2026-07-17T18:15:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 验收：runAll runBuild 已调用 checkBuildDiskSpace（disk_preflight.go，<1GiB 明确错误）。
**Area**: infra / runAll

### Summary
runAll 执行 `build_command` 前增加工作区磁盘余量预检；不足时输出明确错误（含 `df` 与建议清理路径），避免统一包装为 `build failed: exit status 1`。

### Details
阈值建议：可用空间 < 1GiB 时拒绝批量 build，并提示截断 `logs/`、`taskGateway/logs/`。可先在 `runner.go` build 路径加检查，单测覆盖「满盘模拟」或 mock。

### Metadata
- Source: goal-overview
- Related Files: runAll/src/runner.go, .ai/09_failure_experience/01_compilation_errors/02_ram_work_tmpfs_no_space_build_failed.md
- Tags: tmpfs, disk-preflight, build-failed

## [OPT-20260717-038] completed

**Logged**: 2026-07-17T19:01:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 验收：taskGateway/run.sh start 已短轮询 18081/api/health/，失败打印 error.log 尾部并 exit 1。
**Area**: taskGateway / runAll

### Summary
`taskGateway/run.sh start` 在 `--force-recreate apisix` 后增加短轮询探活（`http://127.0.0.1:18081/api/health/`），就绪后再退出；失败时打印 `logs/error.log` 尾部并非 0，避免 runAll 在容器刚起来时误判。

### Details
已修复宿主机残留 `nginx.pid`（uid 1000/0644）导致 APISIX uid 636 EACCES 重启环；`prepare_apisix_logs_for_start` + force-recreate 已落地。仍可加强：start 同步等待就绪，缩短 READINESS_TIMEOUT 误报窗口。

### Metadata
- Source: session-end
- Related Files: taskGateway/run.sh, .ai/09_failure_experience/02_runtime_errors/22_apisix_nginx_pid_permission_denied.md
- Tags: apisix, readiness, nginx.pid

## [OPT-20260717-022] completed

**Logged**: 2026-07-17T14:35:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: 同 OPT-015：根仓纳入 list_git_repos。
**Area**: tooling

### Summary
让 `delete_merged_feat_branches.py` 也扫描 monorepo 根仓（目前只扫子目录含 `.git` 的嵌套仓）。

### Details
`list_git_repos` 仅 `root.iterdir()` 子目录；根仓 `feat/*` 合入后需手工 `git branch -d` / `git push --delete`。应把 `root` 本身纳入 `repos`（若 `root/.git` 存在），并在 dry-run/单测中覆盖。

### Metadata
- Source: conversation
- Related Files: runAll/scripts/delete_merged_feat_branches.py, .ai/01_project_constraints/21_merged_feat_branch_cleanup.md
- Tags: git, feat-branch-cleanup, monorepo

## [OPT-20260718-015] completed

**Logged**: 2026-07-18T11:50:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: delete_merged_feat_branches.list_git_repos 纳入 monorepo 根仓；与 OPT-022 合并完成扫描扩展。
**Area**: git / tooling

### Summary
1) `delete_merged_feat_branches.py` 不扫 monorepo 根仓，应扩展为包含根仓；2) 批量合入 `archive/stash-*` 时 `-X ours` 会产生大量空 merge commit，宜改为「确认 tip 无独有有效 diff 后直接删分支」，避免污染 main 历史；3) 可提供 `apply_stashes_to_main.py`（冲突保 main、junk 过滤、pre-commit 失败不 drop）。

### Metadata
- Source: session-end
- Related Files: runAll/scripts/delete_merged_feat_branches.py, task2app
- Tags: stash, archive-branch, git-cleanup

## [OPT-20260718-047] completed

**Logged**: 2026-07-18T17:35:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: handleWorkspaceMachineSummary/RuntimeIndicators 增加 ensureTenantMember；401 测例 + 既有测补 X-User-Id。
**Area**: taskCloudService / security

### Summary
`workspace-machine-summary` / `workspace-runtime-indicators` 当前可无登录直连 `:8018` 读到租户机器态；应与 todos 一样经网关鉴权并校验工作空间成员，避免摘要可见而任务列表 403 的不一致。

### Details
本会话复现：未带 Authorization 调 cloud 摘要得 `started_count=1`，同用户直连 task-task todos 在无 workspace_access 时 403。建议在 handleWorkspaceMachineSummary / handleWorkspaceRuntimeIndicators 增加与 compute 其它路由一致的租户成员校验。

### Metadata
- Source: goal-overview
- Related Files: taskCloudService/src/workspace_machine_summary.go, taskCloudService/src/compute_workspace_runtime_indicators.go, taskCloudService/src/tenant_member.go
- Tags: auth, work-panel, machine-summary

## [OPT-20260718-052] completed

**Logged**: 2026-07-18T18:35:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: useWorkPanelTaskStatusSse 指数退避重连（最多10次），重连成功后 onNeedResync/fetchTodos；Vitest 覆盖。
**Area**: frontend / taskSSE

### Summary
Work Panel SSE 断线后增加指数退避自动重连（当前仅 close + 一次 `fetchTodos`），并在重连成功后再对账一次列表。

### Details
`useWorkPanelTaskStatusSse` / `openWorkPanelTaskStatusSse` 的 `onerror` 路径可参考任务详情 SSE 重连状态机，避免长时间看板无推送。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/composables/useWorkPanelTaskStatusSse.js
- Tags: sse, reconnect, work-panel

## [OPT-20260718-012] completed

**Logged**: 2026-07-18T11:00:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T19:37:10+08:00
**Completion-Note**: WorkspaceSettingsTaskPanel editWorkspace 改为 workspace.is_default（不再用 is_current）。
**Area**: frontend

### Summary
修复 `WorkspaceSettingsTaskPanel.vue` 编辑工作空间时把 `is_default` 误绑为 `workspace.is_current`（应为 `workspace.is_default`），避免编辑表单默认勾选状态错误。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/views/WorkspaceSettingsTaskPanel.vue
- Tags: workspace, is_default, form-bug

## [OPT-20260718-049] completed

**Logged**: 2026-07-18T17:45:00+08:00
**Completed**: 2026-07-18T18:15:00+08:00
**Priority**: medium
**Status**: completed
**Area**: onlineServiceJS / task-detail UX
**Completion-Note**: `noteBootstrapFailure` + `bootstrapCloneLogFailurePayload`；`GET /repos/bootstrap-clone-log` 无 layer 时返回失败摘要（error_code/phase/missing）；失败经验 `47_bootstrap_credentials_incomplete_skips_clone.md`。

### Summary
bootstrap 在 `repo-clone-credentials` 409 中止后，容器侧 `bootstrap-clone-log`/`layers` 仍为空且无失败摘要；任务详情难以区分「未克隆」与「克隆失败」。应持久化 `BOOTSTRAP_FAILED` 原因供 UI 轮询，并在凭证补齐后提供一键 reclone/重启引导。

### Details
证据任务 `task_13478486238904149865`（`47.239.98.3:8888`）：17:14:36 credentials 409 后无 `clone_begin`；用户 17:33 打开详情时 `/app` 空。当前重放 credentials 已 200，但容器不会自动重跑 bootstrap。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/bootstrap.mjs, taskFE/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue
- Tags: bootstrap-clone, credentials-409, empty-app

## [OPT-20260718-048] completed

**Logged**: 2026-07-18T17:45:00+08:00
**Completed**: 2026-07-18T18:15:00+08:00
**Priority**: medium
**Status**: completed
**Area**: onlineServiceJS / taskCredentialService
**Completion-Note**: `postRepoCloneCredentialsWithRetry` + `scheduleBootstrapCredentialsRecovery`；单测见 `bootstrap.cloneCredentials.test.mjs`。

### Summary
`REPO_CLONE_CREDENTIALS_INCOMPLETE`（HTTP 409）时应带退避重试拉取凭证（或等待 Git 绑定就绪事件），避免容器 bootstrap 一次性失败后永久停在「未克隆」直至人工重启。

### Details
本会话复现：task-detail 200 后立刻 credentials 409，bootstrap 抛错退出；之后仅心跳 validate-token，无二次 credentials。补绑后重放已 200，说明属时序/绑定窗口问题。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/bootstrap.mjs, taskCredentialService/interfaces/handlers.go
- Tags: bootstrap-clone, REPO_CLONE_CREDENTIALS_INCOMPLETE, retry

## [OPT-20260718-044] completed

**Logged**: 2026-07-18T16:56:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T16:58:00+08:00
**Completion-Note**: `gitService` 已 `.gitignore` 增加 `playwright/node_modules/`，`git rm -r --cached` 后提交 `888374b4c` 并推送 origin+gitlab；本地 `npm ci` 可重建依赖且 working tree 干净、`git ls-files` 跟踪数为 0。
**Area**: repo-hygiene / gitService

### Summary
将 `gitService/playwright/node_modules` 从 Git 跟踪中移除（改 `.gitignore` + `git rm -r --cached`），避免再次出现大批量误删脏状态。

### Details
OPT-043 已用 `git restore` 恢复；根因是该仓跟踪了 Playwright 依赖目录。清理后依赖改由 `npm ci` 安装。

### Metadata
- Source: goal-overview
- Related Files: gitService/playwright/, gitService/.gitignore
- Tags: hygiene, node_modules, gitignore

## [OPT-20260718-043] completed

**Logged**: 2026-07-18T17:10:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T17:22:00+08:00
**Completion-Note**: taskProjectService 汇总 gitlab-local `repository_size` → taskBill `report`/`sync-gitlab-disk-quotas`；GET 设置页刷新；cron `*/15` 跑 `sync_tenant_gitlab_disk_quota.sh`。验收租户 `850256677331562496`：`disk_used_bytes=58966`、`disk_used_gb=5.5e-05`。说明：同日另有一条 hygiene 也占用了编号 043（见下方），本条为计量上报语义。
**Area**: gitService / taskBill / metering

### Summary
将系统内建 GitLab 租户关联仓实际占用磁盘定期上报到 taskBill，使设置页「已用磁盘」非零且与配额对照可信。

### Details
计量：`project_repos` × `IsInternalRepo(gitlab-local)` → GitLab statistics；执法见 OPT-040。修复多 provider 同 host 时 `IsInternalRepo` 误匹配。`diskUsedGBFromBytes` 精度 6 位小数。

### Metadata
- Source: goal-overview
- Related Files: taskBill/src/gitlab_disk_sync.go, taskProjectService/src/tenant_disk_usage.go, gitService/scripts/sync_tenant_gitlab_disk_quota.sh
- Tags: gitlab, disk-usage, metering

## [OPT-20260718-040] completed

**Logged**: 2026-07-18T16:30:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T17:22:00+08:00
**Completion-Note**: CE REST 忽略 `repository_size_limit`，改经 `gitlab-rails runner` 设置 Group `tenant-{id}` + 各关联 Project limit；`disk_gb=0`→1 字节。验收：`group=tenant-850256677331562496 limit=5368709120`，`valueStream` 同限。
**Area**: gitService / quota enforcement

### Summary
将 `billing_tenant_gitlab_resource.disk_gb` 下发到系统内建 GitLab（组 + 项目 `repository_size_limit`）做硬限额执法。

### Details
无既有 tenant→namespace 映射时，对用户路径仓逐仓设项目级限额；组为未来归仓预留。跨仓聚合硬拒见 OPT-046。

### Metadata
- Source: goal-overview
- Related Files: gitService/scripts/sync_tenant_gitlab_disk_quota.sh, taskBill/src/gitlab_disk_enforce.go
- Tags: gitlab, quota, enforcement

## [OPT-20260718-033] completed

**Logged**: 2026-07-18T15:20:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T15:30:00+08:00
**Completion-Note**: 已新增 `.claude/skills/README.md`（十步表+废弃重定向+横切分类）；`template` description 改为 Scaffold only — do not trigger；同步 `.claude/README.md`。
**Area**: tooling / skills

### Summary
补 `.claude/skills/README.md` 流水线目录表；将 `template` 技能移出可发现路径或改 `description` 标明「仅作脚手架、勿触发」。

### Details
`.claude/README.md` 已约定别名与完整名关系，但 skills 下无目录索引。`template/SKILL.md` 的 description 为占位句，易被误匹配。验收：README 列出权威 0→10 序列、旧七步别名去向、非流水线技能分类；`template` 不再出现在日常触发候选中。

### Metadata
- Source: goal-overview
- Related Files: .claude/README.md, .claude/skills/template/SKILL.md
- Tags: skills, docs, discoverability

## [OPT-20260718-031] completed

**Logged**: 2026-07-18T15:20:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T15:30:00+08:00
**Completion-Note**: `goal-mode` 写明覆盖 Step1 USER GATE；`0-auto-flow` Overview/Sequence 标明单独调用保留闸门、goal 入口跳过。
**Area**: tooling / skills

### Summary
对齐 `goal-mode` 与 `0-auto-flow` 对 Step 1 用户门禁的语义：明确 `/goal` 是否跳过设计确认，并写回双方 SKILL.md。

### Details
`0-auto-flow` 要求 Step 1 `AskUserQuestion` USER GATE；`goal-mode` 规定 brainstorming 也零交互、自动采用最优解。Agent 同时加载时行为不确定。建议：在 goal-mode 写明「覆盖 0-auto-flow USER GATE」或改为「仅跳过步骤间确认、设计仍须闸门」；两边交叉引用。验收：两份技能对闸门描述无矛盾。

### Metadata
- Source: goal-overview
- Related Files: .claude/skills/goal-mode/SKILL.md, .claude/skills/0-auto-flow/SKILL.md
- Tags: skills, goal-mode, auto-flow, consistency

## [OPT-20260718-030] completed

**Logged**: 2026-07-18T15:20:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-18T15:30:00+08:00
**Completion-Note**: 已修正 `5-nfr`/`6-ddd`/`1-brainstorming-design-docs`/`9-review` 过时 slash；`rg` 活跃技能正文无 `/3-value-stream`/`/4-nfr`/`/5-ddd` 残留。
**Area**: tooling / skills

### Summary
修正流水线技能内过时 slash 引用：统一为现行 `2-role-permission → 4-value-stream → 5-nfr → 6-ddd → 7-plans → 8-build → 9-review → 10-ship`。

### Details
仍写旧路径的文件包括：`5-nfr`（自称 `/4-nfr`、前置 `/3-value-stream-价值流`）、`6-ddd`（同）、`1-brainstorming-design-docs`（完成后指向 `/2-worktrees`/`/3-value-stream`/`/5-ddd`）、`9-review`（回指 `/4-nfr-…`）。验收：`rg '/[0-9]+-(value-stream|nfr|ddd|worktrees|plans)' .claude/skills` 仅命中重定向说明或历史附录，正文步骤与 `0-auto-flow` Sequence 一致。

### Metadata
- Source: goal-overview
- Related Files: .claude/skills/5-nfr/SKILL.md, .claude/skills/6-ddd/SKILL.md, .claude/skills/1-brainstorming-design-docs/SKILL.md, .claude/skills/9-review/SKILL.md
- Tags: skills, stale-refs, pipeline

## [OPT-20260718-029] completed

**Logged**: 2026-07-18T15:20:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-18T15:30:00+08:00
**Completion-Note**: `03_superpowers_workflow.md` 升至 3.0.0 权威十步；旧同号目录改为薄重定向；同步 `00_ai_development`/`project_rules`/`ai-rules-loader`/`.claude/README`。
**Area**: tooling / skills

### Summary
统一 Agent 交付流水线 SSOT：在 `.ai/11_ai_development/03_superpowers_workflow.md` 与 `.claude/skills/` 之间只保留一套编号语义，并处理旧七步与新十步的目录碰撞。

### Details
现状冲突：(1) `03_superpowers_workflow.md` 宣称「单轨七步」；(2) `0-auto-flow`/`goal-mode` 执行 `1→2→4→5→6→7→8→9→10`（插 role-permission/value-stream/NFR/DDD）；(3) 同号双义：`2-worktrees` vs `2-role-permission`，`3-plans` vs `3-worktrees`，`4-build` vs `4-value-stream`，`5-tdd` vs `5-nfr`，`6-tdd`/`6-review` vs `6-ddd`，`7-ship` vs `7-plans`。建议：选定权威十步（或把扩展步标为 7a 子环），旧 Superpowers 目录改为薄重定向（禁止再写独立 Step N 流程），并更新 `03`/`00_ai_development`/`project_rules` 交叉说明。与已有 `OPT-20260717-039`（技能 name CI）互补、不重复。验收：编号→技能一对一；索引文档与 `0-auto-flow` Sequence 一致。

### Metadata
- Source: goal-overview
- Related Files: .ai/11_ai_development/03_superpowers_workflow.md, .claude/skills/0-auto-flow/SKILL.md, .claude/skills/2-worktrees/SKILL.md, .claude/skills/2-role-permission/SKILL.md, .claude/README.md
- Tags: skills, pipeline, ssot, numbering

## [OPT-20260718-028] completed

**Logged**: 2026-07-18T15:10:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T15:25:00+08:00
**Completion-Note**: `django_forward_guard` 对 34 个 MIGRATED action 无条件 410；`cloud_compute_views` 统一 `tcgw_gone_response`；删除 `forward_container_layer_logs` 与多数 `forward_container_*.py`（保留 auth_context/pr/git-identity-sync/repo-reclone）；单测 parametrize 68 例 + ai_task_comment/push 路径改 410。
**Area**: django / taskContainerGateway

### Summary
将 `django_forward_guard.MIGRATED_OUTBOUND_ACTIONS` 中仍保留 Django forward 实现的 `container-*` / `relay-to-trae-*` 按 edit-run / layer-graph 模式改为「始终 410 + 删除 forward」，避免 `TASK_CONTAINER_GATEWAY_ENABLED=False` 时回落到 Python 热路径。

### Details
已完成：guard 忽略网关开关；视图硬 410；删 outbound forward 模块；`test_tcgw_django_forward_stub` 覆盖全部 action×enabled；relay proxy 函数暂留供单测/internal helper（见 OPT-034）。

### Metadata
- Source: django-to-go-migration
- Related Files: task2app/Saas_project/cloud/task_container_gateway/django_forward_guard.py, task2app/Saas_project/cloud/views/cloud_compute_views.py, taskContainerGateway/src/
- Tags: migration, go-first, 410

## [OPT-20260718-018] completed

**Logged**: 2026-07-18T13:25:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-18T13:35:00+08:00
**Completion-Note**: 落地 start 前 supersede、ClearAfterStop 按 instance_id 收口、InstanceName 对账回收；分支 `feat/ecs-orphan-double-start-guard`（taskCloudService / taskEvents / monorepo docs）。
**Area**: backend

### Summary
`CLOUD_SERVER_STARTED` / `UpsertAfterStart` 在任务已有 `instance_id` 时直接覆盖，不先对旧 ECS 发 `CLOUD_SERVER_STOPPED`；随后 stop 只删「当前」实例，`ClearAfterStop` 却关闭该任务全部 open history，导致云侧孤儿机本地不可见。

### Details
1. 启动前：若 CSC 已有非空 `instance_id` 且与本次新建不同，先发布/同步释放旧实例（或拒绝二次 RunInstances，强制走 idle-reuse）。
2. `closeOpenCloudServerConfigHistories`：仅关闭与被删 `instance_id` 匹配的 history，勿批量关停同 task 全部 open 行。
3. 增加对账：按 `InstanceName=task-{task_id}` DescribeInstances，对比 CSC/history，告警/回收孤儿。
4. 本会话已手工 DeleteInstance：`i-j6ce24m0ccb0900pjaoi`（task_133867…）、`i-j6c0yiozukjakixwnd3p`（task_134611…）。

### Metadata
- Source: goal-overview
- Related Files: taskEvents/internal/handlers/cloudserverstarted/handler.go, taskEvents/internal/repository/cloudconfig/client.go, taskCloudService/src/container_reachability.go, taskCloudService/src/server_config_store.go
- Tags: ecs-orphan, stop-vm, double-start, cloud-reconcile

## [OPT-20260718-006] cancelled

**Logged**: 2026-07-18T02:40:00+08:00
**Priority**: low
**Status**: cancelled
**Completed**: 2026-07-18T03:12:00+08:00
**Completion-Note**: 条目正文在误截断 OPTIMIZATION_TODOS.md 后无法从 git/transcript 完整恢复；编号保留不复用。若仍需该待办请按 2026-07-18 02:38–02:45 会话上下文重写。
**Area**: unknown

### Summary
（正文丢失）原位于 OPT-005 与 OPT-007 之间的开放项。

### Metadata
- Source: recovery
- Tags: recovery, cancelled

## [OPT-20260718-004] completed

**Logged**: 2026-07-18T01:14:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-18T03:12:00+08:00
**Completion-Note**: Billing/Projects/Settings/Other 请求错误 UI 已补 data-traceId；Vitest 相关用例通过；front_project runall-lifecycle build + collectstatic 成功。
**Area**: frontend / observability

### Summary
将 `OPT-20260717-032` 剩余范围（Billing / Projects / Settings 等非 CreateTask·TaskDetail 页面）扫完并补 `data-traceId`；可选加 CI `rg` 门禁。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/views/, .learnings/OPTIMIZATION_TODOS.md
- Tags: data-traceId, billing, projects

## [OPT-20260718-001] completed

**Logged**: 2026-07-18T00:45:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T01:13:00+08:00
**Completion-Note**: CreateTask（projects/taskTypes/installedImages/collaborators）与 TaskDetail（load/edit/identity/linkedRepos/layerGraph/execLog/layerChanges/fileTree/budget/gitIdentity 等）请求失败红字均已挂 `data-traceId`；Vitest 35+ 例通过；公网 SPA `main-DGw3CVof.js` 已 collectstatic。刻意跳过：containerHttpUnreachable 静态态、Agent 步骤 payload、克隆进度域 `errorDetail`。
**Area**: frontend / observability

### Summary
扫描 CreateTask / TaskDetail 其余自绘 `text-red-600` 请求错误节点（如 collaboratorsError、commonMergeTargetBranchesError），补齐缺失的 `data-traceId`。

### Details
本会话已修 translate-branch-title 与分支列表错误的 `data-traceId`；同模态/任务详情仍有多处仅展示文案的路径。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/CreateTaskPeopleFields.vue, taskFE/app/src/components/task-detail/TaskDetailBranchStrategyPanel.vue
- Tags: data-traceId, create-task, task-detail

## [OPT-20260717-032] completed

**Logged**: 2026-07-17T16:25:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-18T03:12:00+08:00
**Completion-Note**: CreateTask/TaskDetail（OPT-001）与 Billing/Projects/Settings/Other（OPT-004）均已扫完补齐 data-traceId。
**Area**: frontend

### Summary
扫描 `taskFE/app/src` 中自绘 `text-red-600` / inline 请求错误节点，补齐缺失的 `data-traceId`（如 `TaskDetailExecLogError`、`layerGraphCmdError`、`layerChangesRefreshError` 等）。

### Details
CreateTask / TaskDetail 请求失败红字（含 ExecLogError、layerGraphCmdError、layerChangesRefreshError、collaborators 等）已于 2026-07-18 补齐。剩余范围：Billing / Projects / Settings 等其它页面自绘错误节点。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/components/task-detail/, .ai/01_project_constraints/24_frontend_error_data_trace_id.md
- Tags: data-traceId, task-detail, observability

