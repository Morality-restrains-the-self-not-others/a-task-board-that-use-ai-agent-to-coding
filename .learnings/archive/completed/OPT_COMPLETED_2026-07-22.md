# Completed OPT Archive — 2026-07-22

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 32 条。
> 归档执行时间：2026-07-24T10:15:26+08:00

## [OPT-20260722-071] completed

**Logged**: 2026-07-22T23:16:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-22T23:52:00+08:00
**Completion-Note**: `ensure_gitlab_mirror_remotes.sh --create-missing` 新建约 30 个项目（含 taskTenantService）；35/35 仓 `git push -u gitlab main` 成功。
**Area**: gitlab / mirrors

### Summary
对其余嵌套仓执行 `bash db/scripts/ensure_gitlab_mirror_remotes.sh --create-missing` 并 `git push -u gitlab main`，补齐自建 GitLab 镜像。

### Metadata
- Source: /goal gitlab HTTPS credentials
- Related: `db/scripts/ensure_gitlab_mirror_remotes.sh`

## [OPT-20260722-070] completed

**Logged**: 2026-07-22T23:16:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T23:35:00+08:00
**Completion-Note**: 改用公网 :22（非 :2222）转发：HK `GatewayPorts clientspecified` + autossh `-R 0.0.0.0:22:127.0.0.1:2222`；运维 SSH 仍占 2222。验收 `ssh -T git@gitlab.daydaymoney.com` → Welcome to GitLab。
**Area**: gitlab / edge networking

### Summary
HK 边缘将公网 Git SSH 转发到本机 GitLab gitlab-shell，使外网 `git@gitlab.daydaymoney.com` 无需本机 `HostName 127.0.0.1` 特例。

### Metadata
- Source: /goal gitlab HTTPS credentials + HK :22 forward
- Related: `.ai/09_failure_experience/02_runtime_errors/79_gitlab_https_push_credentials_and_ssh_port_mismatch.md`
- Related: `~/scripts/enable_daydaymoney_tunnel.sh`
- Related: HK `/etc/ssh/sshd_config.d/99-gitlab-git-forward.conf`

## [OPT-20260722-064] completed

**Logged**: 2026-07-22T22:30:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-22T22:40:00+08:00
**Completion-Note**: taskTaskService/taskProjectService/taskCloudService/taskAIComment 启动路径改 `tracelog.Fatalf`；重建并重启对应服务；runAll `./build.sh` 后热替换，enrichment 单测通过。
**Area**: taskTaskService / taskProjectService / tracelog

### Summary
将 `taskTaskService` / `taskProjectService`（及同类 Go 服务）启动路径的 `log.Fatalf` 改为 `tracelog.Fatalf`，保证退出前输出 `"level":"error"`，便于 runAll/Promtail 识别；避免 fatal 文案被包装成 stdout `info`。

### Metadata
- Source: session-end /goal task-task-service malformed sqlite
- Related: shareLib/tracelog/fatal.go, taskTaskService/src/main.go, taskProjectService/src/main.go, taskCloudService/src/main.go, taskAIComment/src/main.go

## [OPT-20260722-063] completed

**Logged**: 2026-07-22T22:30:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T22:40:00+08:00
**Completion-Note**: 从 `sqlite3 .recover` 的 lost_and_found 重建：projects=288 workspaces=779 repos=228；tasks 全量行 13 + stub 74=87；脚本 `data/corrupt-backup/restore_from_lost_and_found.py`；credential live 测例 ok。评论等未进 lost_and_found 的表无法恢复。
**Area**: data / sqlite

### Summary
`data/task_task.db` 与 `data/task_project.db` 在 bulk URL rewrite 中被写空（0 字节）。需从备份/服务导出恢复业务库，并复跑 taskCredentialService live identity 测例。

### Metadata
- Source: session-end /goal gitlab-url-prefix
- Related: taskCredentialService/infrastructure/sqlite_business_live_test.go
- Related: `.ai/09_failure_experience/02_runtime_errors/78_task_sqlite_malformed_launch_exited.md`
- Related: `data/corrupt-backup/restore_from_lost_and_found.py`

## [OPT-20260722-058] completed

**Logged**: 2026-07-22T21:32:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T21:35:00+08:00
**Completion-Note**: 已写入 `.ai/01_project_constraints/29_service_own_conf_directory_only_via_sync.md` 并挂入索引/Cursor/companion；存量跨目录 ReadAppConfig 审计另见 OPT-20260722-059
**Area**: conf / confload

### Summary
将「各服务仅读自己 conf 目录；跨服务配置须 sync 到本目录片段」写入 `.ai` 项目约束，并审计 taskAuth 等对 `task-gateway`/`events/domain-events` 的跨目录 ReadAppConfig 是否也应改为 sync 片段。

### Metadata
- Source: session-end /goal sms-sync-only-own-conf
- Files: `shareLib/confload/load.go`, `conf/auth/task-auth/sync.manifest.yaml`

## 归档项（completed / cancelled）

## [OPT-20260722-039] completed

**Logged**: 2026-07-22T19:00:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T19:30:00+08:00
**Completion-Note**: 前端执行细节切换 wait_previous/independent；TTS/taskAIComment PATCH+create；Feed 接线。
**Area**: frontend / task-detail / comments

### Summary
评论「执行细节」支持用户显式切换 `wait_previous` / `independent`，并持久化到评论元数据（Go API），供后续调度器消费。

### Details
本期仅前端契约与 badge 展示。后续：composer/评论菜单切换依赖模式；写入 `execution_mode`；Feed 回放；与多容器编排联调。

### Metadata
- Source: session-end /goal comment-execution-details
- Related Files: taskFE/app/src/composables/taskDetail/useCommentExecutionContext.js, docs/intents/frontend/comment_execution_details.intent.md
- Tags: comment-execution, dependency-mode

## [OPT-20260722-038] completed

**Logged**: 2026-07-22T19:00:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-22T19:30:00+08:00
**Completion-Note**: taskCloudService comment_container_bindings + 调度；workspace compute 网关；前端 ensure/advance/状态展示；MVP 共享 CSC。
**Area**: cloud / container / comments

### Summary
实现真正「每评论一容器」：Comment→Container 绑定、独立心跳/可写层、以及 `wait_previous` 串行调度与 `independent` 并行启动。

### Details
MVP 已将 UI 迁入评论执行细节，但仍共享任务级容器。后续需 Go 服务持有 commentId↔containerRef，SSE 按评论分发 heartbeat，非 active 评论可挂载自身面板。

### Metadata
- Source: session-end /goal comment-execution-details
- Related Files: docs/superpowers/specs/2026-07-22-comment-execution-details-design.md, docs/architecture/v51-application-integration-20260722-1850-claude.puml
- Tags: multi-container, comment-execution

## [OPT-20260722-034] completed

**Logged**: 2026-07-22T17:20:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T17:20:00+08:00
**Completion-Note**: available-instances 支持 InstanceType/instance_type；有搜索时前端不传 Cores/Memory 等硬件筛选并重新拉取；Go 侧跳过硬件收窄；taskCloudService 已重建重启。
**Area**: frontend / taskCloudService / available-instances

### Summary
可用实例「实例编号搜索」升级为服务端过滤：支持精确查未加载进本地缓存的规格。

### Metadata
- Source: session-end /goal instance-type-search
- Related Files: taskFE/app/src/utils/availableInstancesQueryParams.js, taskCloudService/src/aliyun_sdk.go, available_instances_arch_filter.go
- Tags: hardware-panel, available-instances, search

## [OPT-20260722-010] completed

**Logged**: 2026-07-22T12:00:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T12:12:08+08:00
**Completion-Note**: Login.vue 拆至 345 行；抽出 LoginPhone*Fields/LoginAccessTokenFields + 6 个 composables/auth/*
**Area**: frontend / Login.vue

### Summary
继续拆分遗留超大 `Login.vue`（仍 ~1077 行），按登录方式/OIDC/验证码块抽取子组件，使绝对行数 ≤500。

### Details
本次邀请码改动已将注册入口抽为 `LoginRegisterInviteSection.vue` 并实现净减少，但文件仍远超阈值。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/views/Login.vue
- Tags: frontend, line-limit

## [OPT-20260722-011] completed

**Logged**: 2026-07-22T12:00:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-22T12:12:08+08:00
**Completion-Note**: taskEvents registration_invite/1_observability 订阅 3 事件，结构化日志观测；端口 18049；runAll 已登记
**Area**: taskAuth / observability

### Summary
为注册邀请码 ISSUED/REDEEMED/POLICY 事件补齐 taskEvents 消费侧（审计落库或指标），避免仅 publish 无消费。

### Details
当前 best-effort Kafka publish；无消费者时不影响主路径。

### Metadata
- Source: goal-overview
- Related Files: taskAuth/src/events.go, taskEvents/
- Tags: kafka, invite-code

## [OPT-20260722-007] completed

**Logged**: 2026-07-22T11:20:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T11:41:00+08:00
**Completion-Note**: `buildDisplayComments` 按 `parent_comment_id` 嵌套；Feed 渲染 children；`task2app@585dac20`。Overview 曾误标为 OPT-001（该号已用于 dockerInfra）。
**Area**: frontend / task-detail

### Summary
Feed 按 `parent_comment_id` 嵌套展示 container_agent。

### Metadata
- Source: goal-overview
- Related Files: buildDisplayComments.js, TaskDetailConversationFeed.vue
- Tags: feed, nest, auto_run

## [OPT-20260722-008] completed

**Logged**: 2026-07-22T11:20:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T11:41:00+08:00
**Completion-Note**: `mountedAgentCommentStream.mjs` + jobsRuntime 节流推 /stream；PR 回填追加 prior；`trae-agent@3824a33`。Overview 曾误标为 OPT-002（该号已用于 dockerInfra）。
**Area**: onlineServiceJS / auto_run

### Summary
首指令 streaming 同步写挂载 Agent 评论。

### Metadata
- Source: goal-overview
- Related Files: mountedAgentCommentStream.mjs, jobsRuntime.mjs, autoRunPrBackfill.mjs
- Tags: stream, container_agent, auto_run

## [OPT-20260722-005] completed

**Logged**: 2026-07-22T02:50:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T10:45:00+08:00
**Completion-Note**: trae-agent `6ba6cf4`；`DOCKER_PUSH=1 ./buildDocker.sh` 已推送 `x86_64_2026-07-22_10-04`/`x86_64-latest` 与 `arm64_2026-07-22_10-06`/`arm64-latest`（青岛 ACR）；镜像内 `layerGitOauthPushPr.mjs` 含 422 复用。新任务容器 stop→start 后生效。
**Area**: onlineServiceJS / docker

### Summary
发布含 GitHub PR 422 复用逻辑的 onlineServiceJS 镜像，使任务容器在「PR 已存在」时返回已有 html_url 而非 pr_error。

### Details
本会话已改 `layerGitOauthPushPr.mjs`；运行中容器仍用旧镜像时仅受 Cloud prepare 注入 `pr_base` 即可新建 PR。验收：容器 `DOCKER_PUSH=1 ./buildDocker.sh` 后新任务再推送，对已有 PR 不报 422。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/layerGitOauthPushPr.mjs
- Tags: onlineServiceJS, github-pr, docker

## [OPT-20260721-016] pending

**Logged**: 2026-07-21T23:30:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-22T00:05:00+08:00
**Completion-Note**: trae-agent d89add3；DOCKER_PUSH x86_64_2026-07-21_23-34 / x86_64-latest；任务 stop→start 至 i-j6cazl1uv2vgtt5xkoll / 47.76.236.180；Loki 见 AUTO_RUN_DELIVERY_BEGIN/FAILED（经 Cloud）。push 因 ruandao/somanyad 无 OAuth token 失败（见 OPT-20260721-018）。
**Area**: onlineServiceJS / docker / auto_run

### Summary
提交并推送含交付重试修复的 trae-agent onlineServiceJS 镜像，并对 `task_13759732724159256867`（及同类 completed 仍可推送任务）执行 stop-vm→start-vm，使启动补跑完成 push/PR。

### Details
工作区已实现：`shouldSkipAutoRunDelivery`、失败不写成功 done、`retryPendingAutoRunDeliveries`、嵌套 commit、分支解析对齐。验收：新容器 Loki 可见 `AUTO_RUN_DELIVERY_COMPLETE`，ztree ahead 归零或出现 PR 链接。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/autoRunOrchestration.mjs, trae-agent/onlineServiceJS/src/autoRunDeliveryHooks.mjs, .ai/09_failure_experience/02_runtime_errors/66_auto_run_delivery_done_locks_unpushed.md
- Tags: docker, auto_run, delivery, deploy

## [OPT-20260721-017] pending

**Logged**: 2026-07-21T23:30:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-22T00:05:00+08:00
**Completion-Note**: 拆为 jobsRuntimeState/Trae/Snapshot + jobsRuntime（148/140/169/474 行），单测通过。
**Area**: onlineServiceJS / line-limit

### Summary
拆分 `jobsRuntime.mjs`（当前 ~858 行）至 ≤500，满足源文件行数门禁。

### Details
交付钩子已抽到 `autoRunDeliveryHooks.mjs`；可继续抽出 job 队列 / spawn / layer mirror 等模块。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/jobsRuntime.mjs
- Tags: line-limit, refactor

## [OPT-20260720-048] pending

**Logged**: 2026-07-20T16:20:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T00:05:00+08:00
**Completion-Note**: 与 OPT-20260721-016 一并完成：镜像推送 + 任务滚动；交付补跑路径已在公网生效。
**Area**: onlineServiceJS / docker / auto_run

### Summary
将含 auto_run 交付重试 + nested commit 的 onlineServiceJS 镜像推送到公网，并对存量任务容器 stop-vm→start-vm 滚动，使「completed 仍可推送」可被启动补跑修复。

### Details
代码已合入本仓；容器内旧镜像仍会写失败 done。验收：新容器 Loki 可见 `AUTO_RUN_DELIVERY_*`（经 Cloud，非 Django 404），ztree ahead 在交付成功后归零。

### Metadata
- Source: session-end
- Related Files: trae-agent/onlineServiceJS/src/autoRunOrchestration.mjs, .ai/09_failure_experience/02_runtime_errors/66_auto_run_delivery_done_locks_unpushed.md
- Tags: docker, auto_run, delivery

## [OPT-20260720-049] pending

**Logged**: 2026-07-20T16:20:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T00:05:00+08:00
**Completion-Note**: taskAgentSupport 443ba01 提交并 rebuild/restart（pid 新）；Loki 可见 container_runtime_event 转发 AUTO_RUN_DELIVERY_*（非 Django 404）。
**Area**: taskAgentSupport / ops

### Summary
重启/发布 taskAgentSupport，使 `runtime-event` 转发到 taskCloudService 生效（当前生产仍 Django 404）。

### Details
`forwardsToCloudService` 已加入 `runtime-event`；未重启则 AUTO_RUN_* 仍不可进 Loki。

### Metadata
- Source: session-end
- Related Files: taskAgentSupport/src/handlers.go
- Tags: runtime-event, observability

## [OPT-20260722-029] completed

**Logged**: 2026-07-22T15:25:00+08:00
**Completed**: 2026-07-22T16:54:00+08:00
**Priority**: low
**Status**: completed
**Area**: task2app / referral
**Completion-Note**: 落地一级 5% 积分分成 + 365 日窗 + 15 日 settle；taskBill accrual/edge；Django stats；Vue 去二级。
**Analysis-Note**: 产品修订为固定积分方案（非运营费率时间表）。

### Summary
引荐一级：消费积分 5% 记待结算积分，满 15 日结算入账；绑定一年有效；无二级。

### Details
taskBill 持有 accrual 表与 settle job；`points_source_type=referral_commission`；Django/Vue 只展示 pending/settled；旧链接不变。

### Metadata
- Source: /goal OPT-20260722-029
- Related Files: taskBill/src/referral_commission.go, accounts/views/user_views.py, UserReferral.vue
- Tags: referral, commission-rate, settle

## [OPT-20260722-028] completed

**Logged**: 2026-07-22T15:10:00+08:00
**Completed**: 2026-07-22T15:10:00+08:00
**Priority**: medium
**Status**: completed
**Area**: task2app / frontend SPA
**Completion-Note**: `npm run build` 已串联 `scripts/collectstatic-after-vite.sh`；纯 Vite 为 `build:vite`；watch 改用 `build:vite -- --watch`。

### Summary
将 collectstatic（含 `.vite/manifest` 同步）集成进前端 `npm run build`，避免只跑 Vite 导致公网 SPA 404。

### Metadata
- Source: /goal collectstatic into frontend build
- Related Files: package.json, scripts/collectstatic-after-vite.sh, runall-lifecycle.sh, run.sh
- Tags: collectstatic, spa, npm-build

## [OPT-20260722-019] completed

**Logged**: 2026-07-22T14:05:00+08:00
**Completed**: 2026-07-22T14:15:00+08:00
**Priority**: medium
**Status**: completed
**Area**: front_project / SystemAdminUsers
**Completion-Note**: 已抽 `useSystemAdminUsers.js`，`SystemAdminUsers.vue` 降至 306 行。

### Summary
将 `SystemAdminUsers.vue`（~610 行）的添加/编辑/删除用户模态框拆到子组件，使主视图回落到 ≤500 行。

### Details
本会话仅薄接入 `SystemAdminUserRechargeDrawer`；三个用户 CRUD 模态仍内联。可抽 `SystemAdminUserFormModal.vue` 等，并更新行数例外注释。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/views/SystemAdminUsers.vue
- Tags: line-limit, system-admin, frontend


## [OPT-20260722-011-refund-provider] completed

**Logged**: 2026-07-22T12:40:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-22T12:55:00+08:00
**Completion-Note**: 原开放项编号 OPT-20260722-011（与当日更早归档项撞号）；已实现 PayPal Capture Refund + 微信 refunddomestic，入账写 provider_capture_id；TASKBILL_REFUND_PROVIDER=mock / 未配置渠道时回退 mock。
**Area**: taskBill / payments

### Summary
将退款审批中的 executeProviderRefund 升级为真实 PayPal/微信退款 API。

### Metadata
- Source: goal-overview
- Related Files: taskBill/src/refund_provider.go, paypal_refund.go, wechat_refund.go
- Tags: refund, paypal, wechat, payment

## [OPT-20260722-012-ledger-fifo] completed

**Logged**: 2026-07-22T12:40:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-22T12:55:00+08:00
**Completion-Note**: 原开放项编号 OPT-20260722-012（与当日更早归档项撞号）；recordConsumption* 调用 consumePaymentLedgerFIFO；审批仅退 remaining。
**Area**: taskBill / ledger

### Summary
消费扣费时 FIFO 扣减 billing_payment_ledger.remaining_points。

### Metadata
- Source: goal-overview
- Related Files: taskBill/src/payment_ledger_consume.go, charge.go
- Tags: refund, ledger, fifo

## [OPT-20260722-005-combobox] completed

**Logged**: 2026-07-22T02:40:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-22T02:47:30+08:00
**Completion-Note**: `CreateTaskParentDeliverableField` 已改为单框 combobox（输入过滤、↑↓/Enter、单击展开/双击收起）；单测 15 通过；已 SPA build+collectstatic。归档时因编号与镜像发布 OPT-20260722-005 冲突，后缀 -combobox 区分。
**Area**: frontend / work-panel

### Summary
上层交付物过滤若候选很多，可升级为单框 combobox（输入即过滤 + 键盘上下选择），替代「过滤框 + 原生 select」双控件。

### Details
当前已满足编号/名称过滤；双控件在移动端略占高。可复用 `TagMemberInput` / `NavbarTaskSearch` 模式，并遵守单击展开、双击收起。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/CreateTaskParentDeliverableField.vue, taskFE/app/src/utils/parentDeliverableFilter.js
- Tags: parent-deliverable, combobox, ux


## [OPT-20260721-018] cancelled

**Logged**: 2026-07-22T00:05:00+08:00
**Priority**: high
**Status**: cancelled
**Analyzed**: 2026-07-23
**Analysis-Note**: 需恢复指定任务 GitHub OAuth access_token（用户授权/凭据服务）；非纯代码可闭环。
**Area**: gitOauth / auto_run / delivery
**External-Resource**: deploy, remote-ops

### Summary
为任务 `task_13759732724159256867` 关联仓库 `ruandao/somanyad` 恢复可用 GitHub OAuth access_token，使 auto_run 交付 push/PR 成功、层 ahead 归零。

### Details
滚动新镜像后 Loki：`AUTO_RUN_DELIVERY_FAILED` detail=「该仓库未找到可用的 OAuth access_token」（仍 ahead=1）。交付重试逻辑已生效；阻塞在凭证绑定/换票。验收：同任务 `AUTO_RUN_DELIVERY_COMPLETE` 且 layer-graph ahead=0 或出现 PR。

### Metadata
- Source: goal-overview
- Related Files: taskCredentialService/application/layer_oauth.go, trae-agent/onlineServiceJS/src/layerGitOauthPush.mjs
- Tags: oauth, github, auto_run, delivery

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: 环境缺少 remote-ops（GitHub OAuth 用户授权恢复），无法恢复指定任务 access_token

## [OPT-20260722-002] cancelled

**Logged**: 2026-07-22T00:30:00+08:00
**Priority**: medium
**Status**: cancelled
**Analyzed**: 2026-07-24
**Analysis-Note**: 本地已全量 commit（`bf32c2a`），已推送 origin (GitHub)；gitlab remote 落后 1 个 commit（`bf32c2a`）但 gitlab.daydaymoney.com 502 不可达，待 GitLab 恢复后补推。
**Area**: dockerInfra
**External-Resource**: deploy, git-remote

### Summary
将本会话对 `dockerInfra`（嵌套仓）的 redis host-reuse 改动提交并推送到其远端，保证其它工作区/机器同步。

### Details
改动文件：`redis/run.sh`、`redis/health.sh`、`.gitignore`、`README.md`。父仓仅含失败经验文档；`dockerInfra` 为独立 git 仓。本地已 commit 并推送 GitHub origin；GitLab 端待 OPT-20260722-061 恢复后补推。

### Metadata
- Source: goal-overview
- Related Files: dockerInfra/redis/, dockerInfra/.gitignore, dockerInfra/README.md
- Tags: dockerInfra, git, sync

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: gitlab.daydaymoney.com 502 不可达，无法推送 GitLab remote

## [OPT-20260722-047] cancelled

**Logged**: 2026-07-22T21:08:00+08:00
**Priority**: high
**Status**: cancelled
**Analyzed**: 2026-07-23
**Analysis-Note**: 需查生产 ECS userdata/出站日志与 api.daydaymoney.com 连通性；本会话无该实例运维权限。
**Area**: cloud / userdata / reachability
**External-Resource**: deploy, remote-ops

### Summary
排查 `task_13837474438363563871`（实例 `i-j6ci4sjmjfq42yng97p4` / EIP `47.86.171.41`）：ECS Running 但无任何 `register-reachability`/`boot-progress` 回调；查 userdata 日志与出站到 `api.daydaymoney.com`。

### Details
两笔 start event（12:55、12:57）均 success，CSC `server_url` 仍空。前端已展示「等待容器登记」；需确认镜像初始化是否卡在 apt/docker pull。

### Metadata
- Source: session-end /goal container-ztree-idle-blank
- Related Files: taskCloudService cloud_server_configs, userdata init script
- Tags: register-reachability, userdata, ops

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: 环境缺少 remote-ops（ECS 实例运维权限），无法排查 userdata/出站连通性

## [OPT-20260722-060] cancelled

**Logged**: 2026-07-22T22:20:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Analysis-Note**: 2026-07-24 核查：7 个分支中 6 个仍存在且各有 1 个独有提交（见 Details），`feat/recharge-points-expiry-consent` 已清理。需人工审核后合入或删除。
**Status**: cancelled
**Area**: git / feat branches
**External-Resource**: cdp-e2e, git-remote

### Summary
以下本地 `feat/*` 仍有独有提交未进 main，需人工核对后合入或归档删除。

### Details
核查结果（2026-07-24）：
- 根 `feat/kyc-identity-tier-audit`: 1 commit (`9f55a77`) — KYC learnings
- 根 `feat/recharge-points-expiry-consent`: ✅ 已清理
- task2app `feat/autorun-skip-on-git-inaccessible`: 1 commit (`62d4ef6a`)
- task2app `feat/login-phone-otp-enable`: 1 commit (`67143aca`)
- task2app `feat/otp-email-and-cleanup-bridges`: 1 commit (`88387830`)
- task2app `feat/work-panel-progress-status-e2e-harden`: 1 commit (`d08734d9`)
- taskTaskService `feat/autorun-skip-on-git-inaccessible`: 1 commit (`7fe23ca`)

### Metadata
- Source: session-end / commit-all-to-main

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: gitlab.daydaymoney.com 502 不可达，无法操作远端 feat 分支

## [OPT-20260722-061] cancelled

**Logged**: 2026-07-22T22:20:00+08:00
**Priority**: medium
**Status**: cancelled
**Analyzed**: 2026-07-23
**Analysis-Note**: gitlab.daydaymoney.com HTTPS 502，需恢复 GitLab 服务后推送。
**Area**: git / remotes
**External-Resource**: deploy, remote-ops, git-remote

### Summary
各仓 `gitlab` remote 已改为 `https://gitlab.daydaymoney.com/example-user/...`；对该 HTTPS 端点 `ls-remote` 现返回 502，需恢复 GitLab 服务后补推落后的 main 并清理幽灵 feat/*，导致各仓 `main` 与已合入 `feat/*` 远端删除仅完成了 GitHub `origin`。恢复 SSH/VPN 后应对仍落后的 gitlab/main 做一次 FF 推送，并跑 `python3 runAll/scripts/delete_merged_feat_branches.py --apply` 清 gitlab 幽灵分支。

### Metadata
- Source: session-end / commit-all-to-main
- Related: runAll/scripts/delete_merged_feat_branches.py

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: gitlab.daydaymoney.com 502 不可达 + 缺少 remote-ops，无法恢复 GitLab 推送

## [OPT-20260722-062] cancelled

**Logged**: 2026-07-22T22:30:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 remote-ops, cdp-e2e, git-remote 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: cancelled
**Area**: task2app / playwright
**External-Resource**: remote-ops, cdp-e2e, git-remote

### Summary
Playwright 夹具 URL 已改为 `gitlab.daydaymoney.com`，但本机缺少 `@playwright/test` 导致无法随 unit 一并提交。安装依赖后提交：
`playwright/front_project/tests/{CreateProject.form-validation,ProjectDetail.ssh-git-url-branch-preview}.playwright.test.js`。

### Metadata
- Source: session-end /goal gitlab-url-prefix

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: gitlab.daydaymoney.com 502 不可达 + 缺少 @playwright/test node_modules

## [OPT-20260722-024] decided

**Logged**: 2026-07-22T14:25:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Status**: decided
**Decision**: 维持 MVP 设计：管理员 credit-recharge 走内部路径，不经过 KYC 门禁。已在 `handlers_internal_account.go` 中增加设计注释说明 bypass 是有意为之。若未来需施加 tier 约束，在该函数调用前增加 `checkKycRechargeGate` 即可。
**Area**: taskBill / KYC

### Summary
MVP 仅对用户侧 wechat/paypal create 执行 KYC 门禁；若产品要求管理员 `credit-recharge` 也受 tier 约束（或显式 bypass + 审计），补齐策略与单测。

### Details
当前 `handleInternalCreditRecharge` 未调用 `checkKycRechargeGate`（设计允许 MVP 仅用户侧）。确认产品后接入或文档化 bypass。

### Metadata
- Source: session-end / KYC B1-B3
- Related Files: taskBill/src/kyc_gate.go, taskBill/src/handlers.go
- Tags: kyc, taskBill, compliance

## [OPT-20260722-031] decided

**Logged**: 2026-07-22T16:55:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Status**: decided
**Decision**: 不做。已结算引荐分成退款扣回明确非 MVP 目标。`ReferralCommissionSettled` 后不退费是设计决策，符合结算语义（佣金已归属不可逆）。若后续业务需要，作为独立专项开二期。
**Area**: taskBill / referral

### Summary
已结算引荐分成在源消费退款后的积分扣回（二期）；MVP 仅支持结算前 void。

### Details
`ReferralCommissionSettled` 后若源 consumption 退款，需负向分录或扣余额；当前设计明确非目标。

### Metadata
- Source: goal-overview referral settle
- Related Files: taskBill/src/referral_commission.go, refund paths
- Tags: referral, refund, settle

## [OPT-20260722-042] cancelled

**Logged**: 2026-07-22T20:37:00+08:00
**Cancelled**: 2026-07-24T11:00:00+08:00
**Cancellation-Note**: 脚本已创建（backfill_image_invoker_user_id.sh）但 dev 环境无匹配事件数据（0 filled）。生产数据回填需在生产环境执行脚本，属不可执行的远程操作。本地产出：回填脚本就绪，含 dry-run 模式。
**Priority**: medium
**Status**: cancelled
**Area**: cloud / idle-reuse / migration

### Summary
为历史 CSC 回填 `image_invoker_user_id`（或一次性运维脚本），降低 fail-closed 导致的冷启动尖峰。

### Metadata
- Source: session-end /goal idle-reuse-image-invoker
- Related Files: taskCloudService/src/image_invoker.go, workspace_machine_idle_reuse.go
- Tags: migration, idle-reuse

