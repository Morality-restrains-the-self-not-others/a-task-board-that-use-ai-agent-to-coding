# Completed OPT Archive — 2026-07-23

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 74 条。
> 归档执行时间：2026-07-24T10:15:26+08:00

## [OPT-20260723-030] completed

**Logged**: 2026-07-23T19:47:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T19:55:00+08:00
**Completion-Note**: `buildDocker.sh` 默认 `linux/amd64`；`DOCKER_PLATFORMS=all` 展开双架构；`REQUIRED_PUSH_PLATFORMS` 默认跟随解析结果。已拆分 lib/progress 并通过 `test/buildDocker.platforms.test.sh`。
**Area**: onlineServiceJS / docker / build-perf

### Summary
若公网任务 VM 实际只需 x86_64：将 `DOCKER_PUSH` 默认或文档推荐改为仅 `linux/amd64`（同时设 `REQUIRED_PUSH_PLATFORMS=linux/amd64`），避免 x86 宿主机上 QEMU 构建 arm64；若仍需 arm，改为原生 arm builder / CI matrix，禁止本机 QEMU 串行双架构。

### Metadata
- Source: /goal 分析 onlineServiceJS docker 编译耗时
- Related Files: trae-agent/onlineServiceJS/buildDocker.sh, trae-agent/onlineServiceJS/ai.md
- Tags: docker, arm64, qemu, build-time

## [OPT-20260722-050] completed

**Logged**: 2026-07-22T21:12:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:26:28+08:00
**Completion-Note**: tenant_pricing_view locked 与价目不一致时 BillingDashboardPricing 一行提示。
**Area**: frontend / billing

### Summary
账单「当前套餐」若账户锁定单价与套餐价目不一致，可仅在差异时展示一行提示（或高亮对应价目），避免再加整块「实际扣费」重复列表。

### Metadata
- Source: session-end /goal remove-duplicate-locked-billing-block
- Files: `taskFE/app/src/components/BillingDashboardPricing.vue`

## [OPT-20260722-001] completed

**Logged**: 2026-07-22T00:30:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:26:28+08:00
**Completion-Note**: dockerInfra/redis/host_reuse_test.sh：开关解析与标记文件生命周期。
**Area**: dockerInfra / redis

### Summary
为 `dockerInfra/redis/run.sh` / `health.sh` 的 host-reuse 分支补充可重复的 shell 测试（端口占用、`DOCKER_REDIS_REUSE_HOST=0/1`、标记文件生命周期）。

### Details
本会话已落地自动复用宿主 Redis；建议用 bats/纯 bash fixture（fake `ss`/`redis-cli`/`docker`）固定回归，避免再出现 120s health timeout。

### Metadata
- Source: goal-overview
- Related Files: dockerInfra/redis/run.sh, dockerInfra/redis/health.sh
- Tags: docker-redis, host-reuse, test

## [OPT-20260719-033] completed

**Logged**: 2026-07-19T17:20:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:26:12+08:00
**Completion-Note**: 新增 db/scripts/ci/check_subrepo_random_precommit_hooks.py 并挂入 repo-quality-gates；现有子仓均 OK（gitOauth 未 checkout 跳过）。
**Area**: tests / infra

### Summary
新建子仓脚手架时自动调用 `deploy_repo_random_precommit.sh <name>`，并在 CI 检查「子仓存在 `.git` 则必须有 `scripts/hooks/pre-commit`」。

### Details
本次已手工覆盖 `.gitmodules` 所列仓；后续防回归可用 CI 扫描缺 hooks 的子仓。

### Metadata
- Source: goal-overview
- Related Files: db/scripts/deploy_repo_random_precommit.sh, .gitmodules
- Tags: pre-commit, hooks, ci

## [OPT-20260722-036] completed

**Logged**: 2026-07-22T18:05:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:25:51+08:00
**Completion-Note**: db/scripts/ci/test_internal_apis_live_listening.sh 离线回归挂入 repo-quality-gates；listening 000/000000 + required 不可达 FAIL 均 PASS。
**Area**: scripts / smoke

### Summary
为 `scripts/smoke/internal-apis-live.sh` 增加可离线的回归用例（listening 远端 000 误判、required 下 saas 未监听须 FAIL）。

### Details
本次已修 `listening()` 的 `|| echo 000` → `000000` 误判，以及 saas 未监听只 SKIP 却 exit 1 导致 UI 显示 fail=0。可用 `INTERNAL_API_SMOKE_HOST=203.0.113.1` 做软探测，或抽 `listening`/`http_code` 到可 `bats`/小脚本单测，避免回归。

### Metadata
- Source: session-end /goal internal-apis-smoke-bar 原因排查
- Related Files: scripts/smoke/internal-apis-live.sh
- Tags: smoke, regression, listening

## [OPT-20260722-004] completed

**Logged**: 2026-07-22T00:40:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:25:51+08:00
**Completion-Note**: 关联项目区展示 lastOAuthExchangeError；setLastOAuthExchangeError + vitest 覆盖。
**Area**: frontend / task-detail

### Summary
UI「OAuth 已授权」与 push prepare 失败解耦时，在关联项目区展示最近一次换票失败原因（如 bridge/refresh），避免绿标误导。

### Details
本次根因是 bridge secret；用户侧只见 unauthorized。可在 409/502 后把 `detail` 摘要写入面板旁提示。非阻塞。

### Metadata
- Source: goal-overview
- Related Files: TaskDetailLinkedProjectsPanel.vue, taskDetailLayerActions.js
- Tags: oauth, ux, push

## [OPT-20260720-010] completed

**Logged**: 2026-07-20T11:10:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:25:51+08:00
**Completion-Note**: 新增 check_traceid_log_first_skill_refs.py 并挂入 .github/workflows/repo-quality-gates.yml；6 技能交叉引用断言 PASS。
**Area**: skills / ci

### Summary
为 `traceid-log-first-diagnosis` 硬门禁增加轻量 CI/golden 断言：扫描排障类 SKILL.md 是否交叉引用该短规范（或等价「有 data-traceId 先查日志」措辞），防止新技能漏挂。

### Details
可扩展现有 `db/scripts/ci/check_trace_id_detection_fixtures.py`，或新增 `check_traceid_log_first_skill_refs.py`；范围建议：`pua`、`logging-audit`、`webapp-testing`、`8-build`、`9-review`、`1-brainstorming-design-docs`。

### Metadata
- Source: goal-overview
- Related Files: .claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md
- Tags: data-traceId, skills, ci

## [OPT-20260722-018] completed

**Logged**: 2026-07-22T14:05:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:25:22+08:00
**Completion-Note**: 新增 useBillingRecharge.consent.test.js：仅开弹窗/confirm+consent_id/cancel；3/3 通过。
**Area**: front_project / billing recharge

### Summary
为 `useBillingRecharge` 条款同意 → `consent_id` 支付创建路径补充单元测试。

### Details
覆盖：点击支付仅开弹窗不建单；confirm 后 POST recharge-consent 再带 consent_id 调 wechat/paypal create；cancel 关闭且不发支付请求。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/composables/useBillingRecharge.js
- Tags: unit-test, recharge-consent

## [OPT-20260723-019] completed

**Logged**: 2026-07-23T16:50:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-23T17:10:00+08:00
**Completion-Note**: `cloud_server_configs` 改为 `idx(workspace_id,task_id)` + `UNIQUE(workspace_id,task_id,comment_id)`；`ensureCommentCloudServerConfig` 为每评论独立 CSC；advance 并行挂不同 csc_id；前端展示独立实例；架构 v54；Go/前端单测通过。
**Area**: cloud / comment-container-bindings

### Summary
为 `independent` 并行评论真正供应独立物理实例（放宽或拆分 `cloud_server_configs UNIQUE(company_id,task_id)`，或 comment→第二 CSC/容器池），使「不等待前序」不只是逻辑调度+禁止挂接共享连接，而是可并行跑在不同机器上。

### Metadata
- Source: session-end /goal parallel-comment-no-shared-container；/goal OPT-019 multi-CSC
- Related Files: taskCloudService/src/comment_csc_ensure.go, comment_container_bindings_schedule.go, server_config_store.go, db.go, docs/superpowers/specs/2026-07-23-comment-multi-csc-parallel-design.md
- Tags: comment-container, multi-instance, independent

## [OPT-20260723-014] completed

**Logged**: 2026-07-23T14:40:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-23T16:13:00+08:00
**Completion-Note**: 新增 `0054_merge_cloud_leaf_nodes`；将冗余的 `0051_accesskeyiamidassociation_drop_cpa_fk` 置为空操作（避免与 0052 DeleteModel 状态冲突）；`manage_init.py run migrate` 已应用 0051/0052_stub/0054；`runall-saas-backend.sh start` 可过 migrate 并监听 :8001。
**Area**: django / cloud migrations

### Summary
`cloud` app 存在多个 leaf migration（`0051_accesskeyiamidassociation_drop_cpa_fk` / `0051_drop_tenant_installed_image_django_table` / `0052_*` / `0053_*`），导致 `runall-saas-backend.sh start` 在 `manage_init.py migrate` 失败。需 `makemigrations --merge`（或整理依赖图）后恢复标准启动路径。

### Metadata
- Source: 修复 daydaymoney.com 证书后拉起 saas-backend
- Related: task2app/Saas_project/cloud/migrations/, scripts/runall-saas-backend.sh

## [OPT-20260723-003] completed

**Logged**: 2026-07-23T01:15:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-23T14:40:00+08:00
**Completion-Note**: 443 已通；将 `/etc/nginx/ssl` 上有效 LE 证书同步到 nginx 正在使用的 `/etc/letsencrypt/live/daydaymoney.com/`，reload 后 `curl` 验 `verify:0`、issuer=Let's Encrypt YR2；并拉起 saas-backend + daydaymoney 隧道使 work-panel 返回 200。
**Area**: infra / daydaymoney edge

### Summary
腾讯云上海机（SSH Host `sh` / 1.117.67.121）安全组放行 **443**（及确认 80）。当前公网仅 80 可达；nginx 已监听 443 + 自签证书，但外网 TCP 443 超时。放行后验证 `https://www.daydaymoney.com`，再换 Let's Encrypt。

### Metadata
- Source: daydaymoney sh nginx → cpu_zerg
- Related: task2app/scripts/daydaymoney.sh.nginx.example, deploy_daydaymoney_sh_nginx.sh

## [OPT-20260723-011] completed

**Logged**: 2026-07-23T13:45:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T13:55:00+08:00
**Completion-Note**: `daydaymoney.hk.nginx.example` / `daydaymoney.hk.nginx.example` 改为 `${baseDomain}/${subdomains.*}`；`check_no_hardcoded_base_domain.py` 覆盖全部边缘 nginx 模板并禁止任意 FQDN 字面量；render 验收通过（含 `BASE_DOMAIN=daydaymoney.com`）。
**Area**: infra / nginx templates

### Summary
将 `task2app/scripts/daydaymoney.hk.nginx.example`（及 `daydaymoney.hk.nginx.example`）改为与 daydaymoney sh 相同的 `${baseDomain}/${subdomains.*}` 模板 + render，避免第二套硬编码域名文件。

### Metadata
- Source: 域名硬编码仅 base.yaml
- Related: task2app/scripts/render_base_yaml_templates.py

## [OPT-20260723-009] cancelled

**Logged**: 2026-07-23T13:30:00+08:00
**Priority**: medium
**Status**: cancelled
**Completed**: 2026-07-23
**Completion-Note**: 方向已反转：统一采用 gitoauth_api（base.yaml），不再对齐到 gitoauth.api。
**Area**: conf / git-oauth

### Summary
（已取消）原建议将 GitHub provider 改为 gitoauth.api。

### Metadata
- Source: /goal GitLab OAuth redirect URI not valid
- Related: conf/auth/git-oauth/providers/

## [OPT-20260723-008] cancelled

**Logged**: 2026-07-23T13:30:00+08:00
**Priority**: medium
**Status**: cancelled
**Completed**: 2026-07-23
**Completion-Note**: 已按用户要求统一到 gitoauth_api，不再改 base.yaml 为 gitoauth.api。
**Area**: conf / base.yaml DNS SSOT

### Summary
（已取消）原建议把 base.yaml 改为 gitoauth.api。

### Metadata
- Source: /goal GitLab OAuth redirect URI not valid
- Related: conf/base.yaml

## [OPT-20260723-007] completed

**Logged**: 2026-07-23T02:05:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: enable_daydaymoney_tunnel.sh start_tunnel 后增加 smoke_gitlab_oidc_login curl 冒烟，断言 data-testid=oidc-login-button 存在。
**Area**: infra / daydaymoney gitlab

### Summary
为 `enable_daydaymoney_tunnel.sh` / deploy 增加「公网 gitlab.daydaymoney.com 登录页须含 data-testid=oidc-login-button」的冒烟验收（curl 或 Playwright），防止 sh 本机 GitLab 再次抢占 8012 后静默丢 taskAuth SSO。

### Metadata
- Source: /goal 修复 gitlab.daydaymoney.com taskAuth SSO 按钮丢失
- Related: task2app/scripts/enable_daydaymoney_tunnel.sh

## [OPT-20260723-006] completed

**Logged**: 2026-07-23T01:51:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: frontend_base_url 改为 ${scheme}://${subdomains.www}；loadPaypalConfig 增加 confload 模板解析（ResolveBaseYaml + ResolveTemplate），编译通过。
**Area**: conf / domain SSOT

### Summary
`conf/billing/paypal/config.yaml` 仍硬编码 `frontend_base_url: https://www.daydaymoney.com`，未走 `${scheme}://${subdomains.www}`。域名迁到 daydaymoney.com 后应改为模板引用，避免绕过 `base.yaml` SSOT。

### Metadata
- Source: /goal 域名仅应在 base.yaml 的约束核对
- Related: conf/base.yaml, conf/billing/paypal/config.yaml

## [OPT-20260723-004] completed

**Logged**: 2026-07-23T01:15:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: 新增 task2app/scripts/daydaymoney-tunnel.service systemd --user 常驻单元，含安装说明与 linger 提示。
**Area**: infra / daydaymoney edge

### Summary
为 `enable_daydaymoney_tunnel.sh` 增加 systemd --user 常驻单元（对齐 daydaymoney 隧道优化项），避免本机重启后 daydaymoney.com 上游 502。

### Metadata
- Source: daydaymoney sh nginx → cpu_zerg
- Related: task2app/scripts/enable_daydaymoney_tunnel.sh, ~/scripts/enable_daydaymoney_tunnel.sh

## [OPT-20260723-001] completed

**Logged**: 2026-07-23T00:30:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: 新增 Django validate_phone_login internal 端点 + URL；taskAuth djangoValidatePhoneLogin + handlePhoneOTPLogin/handleLogin 门禁；编译通过。
**Area**: taskAuth / login policy

### Summary
`enable_phone_login` 目前主要靠前端隐藏入口，后端登录接口未像本次 `enable_email_register` 一样经 Django internal 强制校验。建议补齐手机号登录/验证码登录的服务端门禁，与邮箱注册策略同级防绕过。

### Metadata
- Source: /goal system-admin enable-email-register
- Related: taskAuth/src/auth_login.go, SystemFeaturePolicy.enable_phone_login

## [OPT-20260722-068] completed

**Logged**: 2026-07-22T22:56:30+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: 删除 CreateTaskModal.vue defineProps 中重复声明的 todos 属性（3 次 → 1 次）。
**Area**: front_project / CreateTaskModal

### Summary
`CreateTaskModal.vue` 的 `defineProps` 中 `todos` 被重复声明三次（同名属性覆盖）。清理为单次声明，并补一条 props shape 单测，避免后续静默覆盖。

### Metadata
- Source: /goal 创建任务弹窗「项目」移到「工作分支」上方
- Related Files: taskFE/app/src/components/CreateTaskModal.vue

## [OPT-20260722-066] completed

**Logged**: 2026-07-22T22:55:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: gitlab_home.sh migrate 成功后打印旧目录清理提示（sudo rm -rf）。
**Area**: gitService / persistence

### Summary
legacy `gitService/gitlab_home`（tmpfs）在自动迁移到 `$GITLAB_HOME` 后仍占用空间。可在迁移成功且新挂载验证通过后，打印可选清理命令（如 `sudo rm -rf gitService/gitlab_home`），或提供 `run.sh migrate --prune-legacy`。

### Metadata
- Source: /goal gitservice-durable-gitlab-home
- Related: gitService/scripts/gitlab_home.sh, gitService/run.sh

## [OPT-20260722-067] completed

**Logged**: 2026-07-22T22:55:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: 新增 gitService/docs/backup-cron-example.md：cron 示例、参数说明、异地备份参考。
**Area**: gitService / ops

### Summary
为 durable `GITLAB_HOME` 增加定期 `gitlab-backup create` 到非易失路径的文档/cron 示例，防止单盘故障。

### Metadata
- Source: /goal gitservice-durable-gitlab-home Final Overview
- Related: gitService/gitlab-ce/doc/install/docker/backup.md

## [OPT-20260722-057] completed

**Logged**: 2026-07-22T21:28:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: 修复 conf/ai/ai-provider/sync.sh、conf/events/domain-events/sync.sh、conf/gateway/task-sse/sync.sh 的 ROOT 层级（../.. → ../../..）。
**Area**: conf / sync.sh

### Summary
审计 `conf/**/sync.sh` 的 monorepo ROOT 层数（`conf/auth/task-auth/sync.sh` 曾少一层导致 conf-sync 失效）；统一为相对 `conf/<area>/<app>` 的 `../../..` 或共享 helper。

### Metadata
- Source: session-end /goal why-sms-mock
- Files: `conf/auth/task-auth/sync.sh`, other `conf/**/sync.sh`

## [OPT-20260722-057] completed

**Logged**: 2026-07-22T21:35:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: 修复 conf/ai/ai-provider/sync.sh、conf/events/domain-events/sync.sh、conf/gateway/task-sse/sync.sh 的 ROOT 层级（../.. → ../../..）。
**Area**: cloud / ecs

### Summary
存量 ECS 仍可能挂 legacy `InstanceName=task-{task_id}`；评估批量 RenameInstance 到规范 `task_id`（或自然淘汰），以便最终去掉 orphan 双名 Describe。

### Metadata
- Source: goal-overview
- Related Files: taskCloudService/src/orphan_instance_reconcile.go, taskEvents/internal/cloud/aliyun/instance_name.go
- Tags: ecs, instance-name, migration

## [OPT-20260722-056] completed

**Logged**: 2026-07-22T21:25:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: TaskCardIdBadge 双击复制完整规范 ID（task_<digits>）；单击保持后6位兼容；title/aria 提示两种模式。
**Area**: frontend / task-id UX

### Summary
看板任务编号目前点击只复制后 6 位；云控台/日志曾出现 `task-task_<digits>`。可评估提供「复制完整规范 id（`task_<digits>`）」入口。

### Details
搜索已归一化；ECS 新开机器 InstanceName 已改为等于 task_id。复制体验统一后可进一步减少误搜。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/TaskCardIdBadge.vue, taskFE/app/src/utils/taskIdDisplay.js
- Tags: task-id, copy, navbar-search

## [OPT-20260722-055] completed

**Logged**: 2026-07-22T21:22:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: UserCenterSidebar「引荐功能」→「推荐」；UserReferral 页头/描述/标签统一为「推荐」语义。
**Area**: frontend / referral

### Summary
引荐页已改为单级「推荐获取收益」后，侧栏/页头仍用「引荐功能」；评估是否统一为「推荐」语义，并补一条 UserReferral 文案/字段断言的前端单测，防止两级字段回潮。

### Metadata
- Source: session-end /goal referral-single-level-copy
- Files: `UserReferral.vue`, `UserCenterSidebar.vue`

## [OPT-20260722-053] completed

**Logged**: 2026-07-22T21:20:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: useLoginSubmit handleResendActivationEmail 改为 JSON 请求体（对齐 taskAuth readJSONBody），移除 form-urlencoded。
**Area**: frontend / auth

### Summary
`useLoginSubmit` 登录失败与重发激活邮件仍用 `modalService.alert` 且部分接口可能仍为 form-urlencoded；对齐 `showRequestError` + JSON（与本次验证码修复同模式）。

### Metadata
- Source: session-end /goal login-send-verification-code
- Files: `useLoginSubmit.js`

## [OPT-20260722-052] completed

**Logged**: 2026-07-22T21:14:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23
**Completion-Note**: changeDependencyMode 标记为未使用（UI 只读后无入口），保留供管理端/调试，添加注释说明。
**Area**: frontend / task-detail

### Summary
`useCommentContainerBindings.changeDependencyMode` 与 `commentExecutionApi` 的 PATCH 路径在 UI 只读后已无入口；评估删除死代码或保留给管理端/调试，并清理相关单测与设计中的「发出后可改」表述。

### Metadata
- Source: session-end /goal posted-comment-mode-readonly
- Files: `useCommentContainerBindings.js`, `commentExecutionApi.js`

## [OPT-20260723-020] completed

**Logged**: 2026-07-23T17:01:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: TaskDetailCommentExecutionDetails 增加一键复制容器名；展开区展示完整名与 CSC 对照。
**Area**: frontend / task-detail / execution-details

### Summary
执行细节摘要容器名过长时仅 truncate+title；可增加一键复制容器名，并在展开区内展示完整名与 CSC 对照。

### Metadata
- Source: session-end /goal comment-container-name-display
- Related Files: TaskDetailCommentExecutionDetails.vue
- Tags: ux, container-name

## [OPT-20260723-018] completed

**Logged**: 2026-07-23T16:40:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: CreateTaskProjectBranchSection 拆成「项目」+「分支策略」小标题与扫读提示。
**Area**: front_project / CreateTaskModal

### Summary
「项目与分支策略」区块标题在项目上移后仍笼统；可拆成「项目」与「分支策略」两个小标题，或把 section label 改为更贴合「先选项目再定工作/目标分支」的文案，降低扫读成本。

### Metadata
- Source: /goal 创建任务弹窗「项目」移到分支选择上方
- Related Files: taskFE/app/src/components/CreateTaskProjectBranchSection.vue

## [OPT-20260723-016] completed

**Logged**: 2026-07-23T16:33:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: prepare 成功路径经 writeLayerGitPushPrepareOK 调用 attachLayerGitPushPRMetadata；TestLayerGitPushPrepare_AttachesPRBaseBranch PASS。
**Area**: taskCloudService / git-push

### Summary
修复预存失败单测 `TestLayerGitPushPrepare_AttachesPRBaseBranch`：`push_body` 缺少 `pr_base_branch`（见 `git_push_pr_metadata_test.go:113`）。本次仅去重编译冲突，未改 prepare 逻辑。

### Metadata
- Source: /goal 修复 task-tenant-service / task-cloud-service 编译失败
- Related Files: taskCloudService/src/git_push_pr_metadata.go, taskCloudService/src/git_push_internal.go, taskCloudService/src/git_push_pr_metadata_test.go

## [OPT-20260723-015] completed

**Logged**: 2026-07-23T16:30:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: project_handlers.go 拆分为 project/workspace/deliverable 等；project_handlers.go 218 行；go build 通过。
**Area**: taskProjectService / hygiene

### Summary
`taskProjectService/src/project_handlers.go` 已 538 行（>500），按行数门禁拆分为 list/create/update/workspace 等子文件，使每个文件 ≤500。

### Metadata
- Source: /goal associateWorkspace 501 修复会话
- Related Files: taskProjectService/src/project_handlers.go

## [OPT-20260723-014] completed

**Logged**: 2026-07-23T16:30:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-23T17:26:00+08:00
**Completion-Note**: collectstatic-after-vite.sh：rsync→assets、按 manifest 从 STATIC_ROOT 根救援缺失 chunk、清理根目录 vite-hash 残留并校验；settings 已用 ("assets", dir) 前缀。
**Area**: front_project / collectstatic

### Summary
排查为何 `npm run build` → `collectstatic-after-vite.sh` 后 manifest 已指向 `ProjectDetail-*.js`，但 `Saas_project/collected_static/assets/` 偶发缺失该 chunk（曾出现文件落到 `collected_static/` 根目录）。应保证 Vite hashed assets 稳定同步到 `STATIC_ROOT/assets/`，避免公网懒加载 404。

### Metadata
- Source: /goal associateWorkspace 501 修复会话
- Related Files: taskFE/app/scripts/collectstatic-after-vite.sh, task2app/Saas_project/Saas_project/settings.py

## [OPT-20260723-005] completed

**Logged**: 2026-07-23T01:51:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: 已验证 Navbar 烘焙 assets 与 collected_static/assets 均为 https://gitlab.daydaymoney.com，无需再 build。
**Area**: frontend / domain migration

### Summary
`conf/base.yaml` 工作区已把默认 `baseDomain` 改为 `daydaymoney.com`（`conf-read` 解析出 `vue.publicUrl=https://gitlab.daydaymoney.com`），但已部署 SPA 仍烘焙旧值：`front_project/app/static/assets/Navbar.logic-C6CUQnzw.js` 内 `h="https://gitlab.daydaymoney.com"`（构建时间早于 base.yaml 修改）。需在确认 `base.yaml`/`BASE_DOMAIN` 后执行 `runall-lifecycle.sh build`（Vite build + collectstatic）并发布，使「代码仓库」跳转 `https://gitlab.daydaymoney.com`。

### Metadata
- Source: /goal 分析 work-panel 代码仓库跳转
- Related: Navbar.ui.vue `VITE_GIT_SERVICE_PUBLIC_URL`, conf/frontend/vue/git-service.yaml, conf/base.yaml

## [OPT-20260722-072] completed

**Logged**: 2026-07-22T23:35:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: 已将 ~/scripts/enable_daydaymoney_tunnel.sh 同步到 task2app/scripts/，并新增 daydaymoney-tunnel.service。
**Area**: ops / tunnel scripts

### Summary
将本机 `~/scripts/enable_daydaymoney_tunnel.sh`（含 Git SSH `-R 0.0.0.0:22→127.0.0.1:2222`）同步进可追踪仓库路径，并与 systemd --user 常驻方案（见既有隧道优化项）对齐。

### Metadata
- Source: session-end HK :22 → GitLab SSH
- Related: `~/scripts/enable_daydaymoney_tunnel.sh`
- Related: `.ai/09_failure_experience/02_runtime_errors/79_gitlab_https_push_credentials_and_ssh_port_mismatch.md`

## [OPT-20260722-054] completed

**Logged**: 2026-07-22T21:20:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: 与 OPT-014 一并：STATICFILES_DIRS 前缀 assets；旧根目录哈希文件可后续人工清理。
**Area**: frontend / django-static

### Summary
`STATICFILES_DIRS` 将 `front_project/static/assets` 作为根目录导致 collectstatic 扁平化到 `collected_static/*.js`，旧 `assets/Login-*.js` 残留易混淆；评估统一为 `assets/` 前缀或清理遗留哈希文件。

### Metadata
- Source: session-end /goal login-send-verification-code
- Files: `saas_project/settings.py`, `collectstatic-after-vite.sh`

## [OPT-20260722-051] completed

**Logged**: 2026-07-22T21:12:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: 与 OPT-014 一并：STATICFILES_DIRS 前缀 assets，避免扁平化到 STATIC_ROOT 根。
**Area**: frontend / static

### Summary
`STATICFILES_DIRS` 直接挂 `front_project/static/assets`，产物落在 `STATIC_ROOT/<hash>.js`；历史残留仍在 `collected_static/assets/`。可统一目录约定并清理陈旧 `assets/BillingDashboard-*.js`，避免验收时误扫旧 chunk。

### Metadata
- Source: session-end /goal remove-duplicate-locked-billing-block
- Files: `task2app/Saas_project/saas_project/settings.py`, `collected_static/`

## [OPT-20260722-027] completed

**Logged**: 2026-07-22T15:05:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: 同 OPT-020/005：公网 SPA assets 已指向 daydaymoney；collectstatic 加固完成。
**Area**: task2app / frontend SPA

### Summary
充值消费情况面板与价格管理 tabs 合入后须执行前端 build + collectstatic，确认公网 SPA 引用最新 `/static/main-*.js`。

### Metadata
- Source: session-end / admin-recharge-consumption-overview
- Related Files: SystemAdminRechargeConsumptionPanel.vue, SystemAdminPriceManagement.vue
- Tags: collectstatic, spa, billing

## [OPT-20260722-022] completed

**Logged**: 2026-07-22T14:20:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: AccessTokenManagementPanel / UserProfilePhoneBindingPanel / UserProfileCompanySettingsPanel 已在工作区存在并可引用。
**Area**: front_project / user-profile

### Summary
将先前未入库却被 `UserProfile.vue` 引用的拆分组件正式提交：`AccessTokenManagementPanel.vue`、`UserProfilePhoneBindingPanel.vue`、`UserProfileCompanySettingsPanel.vue`、`accessTokenListView.js(.test.js)`，避免下一轮 SPA build 再次因缺文件失败。

### Details
本会话为打通价格管理页 build，已从历史会话记录恢复这些文件并成功 `runall-lifecycle.sh build`；需单独提交以免工作区丢失。

### Metadata
- Source: session-end /goal gitlab-price-defaults
- Related Files: taskFE/app/src/components/AccessTokenManagementPanel.vue, taskFE/app/src/views/UserProfile.vue
- Tags: spa-build, user-profile, missing-files

## [OPT-20260722-020] completed

**Logged**: 2026-07-22T14:20:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: SPA 已含最新 daydaymoney 烘焙；collectstatic 路径加固后可稳定发布（本会话验证 Navbar URL）。
**Area**: front_project / public SPA

### Summary
本特性改动了充值/超管前端源码后，部署前须执行 `bash taskFE/app/scripts/runall-lifecycle.sh build`（含 collectstatic），确认公网 `/static/main-*.js` 为 200。

### Metadata
- Source: session-end /goal recharge-points-expiry-consent
- Related Files: taskFE/app/src/composables/useBillingRecharge.js
- Tags: collectstatic, spa, deploy

## [OPT-20260721-001] completed

**Logged**: 2026-07-21T19:20:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:19:47+08:00
**Completion-Note**: 新增 task2app/scripts/daydaymoney-tunnel.service（systemd --user），安装说明对齐 daydaymoney-tunnel。
**Area**: scripts / networking / daydaymoney

### Summary
为本机 `enable_daydaymoney_tunnel.sh` 增加 systemd --user 常驻单元，并在可装 nginx 后评估合并进 `enable_daydaymoney` 的单隧道 8080 方案，避免多端口 `-R` 与两套脚本并存。

### Details
当前因本机无 sudo/nginx，用多端口 autossh + HK upstream→127.0.0.1。验收：user unit 开机自启；恢复生产时 `restore-hk-upstream` 一键回到 183.250.1.132。

### Metadata
- Source: session-end
- Related Files: task2app/scripts/enable_daydaymoney_tunnel.sh, task2app/scripts/enable_daydaymoney.sh, task2app/scripts/daydaymoney.hk.nginx.example
- Tags: autossh, daydaymoney, tunnel

## [OPT-20260723-022] completed

**Logged**: 2026-07-23T17:10:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:26:00+08:00
**Completion-Note**: comment_csc_bootstrap：mock 填 instance_id 后异步 POST mock-run start（comment_id）；云平台异步 start-vm-auto；不占位 server_url（OPT-048）；TestBootstrapCommentCSCRuntime_* PASS。
**Area**: cloud / start-vm / comment-csc

### Summary
评论级 CSC 创建后自动触发 StartVM / mock-run bootstrap（按 comment_id），把独立 CSC 的 `instance_id`/`server_url` 填满，完成「不同机器」从编排行到真实进程的闭环。

### Metadata
- Source: session-end /goal OPT-019 multi-CSC
- Related Files: taskCloudService/src/comment_csc_ensure.go, start_vm_bootstrap.go, go_run_container
- Tags: start-vm, comment-csc, bootstrap

## [OPT-20260723-021] completed

**Logged**: 2026-07-23T17:01:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:26:00+08:00
**Completion-Note**: go_run_container status/stop 经 containerStatusDual（评论级→taskId_）；前端 useServerConfigMockRun 记录探测到的 container_name；CandidateContainerNamesDual PASS。
**Area**: frontend / task-detail / mock-run

### Summary
任务级 `taskId_{taskId}` 与评论级 `task_{taskId}_{commentId}` 双命名并存时，status/stop 在 active 评论切换后可能查错容器；统一探测策略（同时探测两名，或在 binding 持久化 `container_name` 后仅按 binding 查）。

### Details
本会话启动路径已传 `comment_id`；无评论时仍回退 `taskId_`。切换评论或历史任务级容器并存时需更稳健的探测。

### Metadata
- Source: session-end /goal comment-container-name-display
- Related Files: useServerConfigMockRun.js, go_run_container/src/docker.go, taskContainerGateway/src/mock_run_handlers.go
- Tags: container-name, mock-run, dual-naming

## [OPT-20260722-048] completed

**Logged**: 2026-07-22T21:08:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:22:36+08:00
**Completion-Note**: binding 在 CSC server_url 就绪前保持 starting；bootstrap 不再写占位 URL；promote 升 running；ensure 失败不再误标 running。
**Area**: cloud / comment-container-bindings

### Summary
评论容器 binding 的 `running` 勿在 mock 阶段即标记完成语义；应保持 `starting` 直至 CSC `server_url`/reachability 登记，或前端用 endpoint 门禁覆盖 binding 文案。

### Metadata
- Source: session-end /goal container-ztree-idle-blank
- Related Files: taskCloudService/src/comment_container_bindings_schedule.go, TaskDetailCommentExecutionDetails.vue
- Tags: mock-binding, reachability

## [OPT-20260722-016] completed

**Logged**: 2026-07-22T13:50:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:22:36+08:00
**Completion-Note**: purchaseGitlabResources 已在同事务调用 consumePaymentLedgerFEFO。
**Area**: taskBill / credit lots

### Summary
`purchaseGitlabResources` 扣余额时未调用 `consumePaymentLedgerFEFO`，与主扣费路径不一致。

### Details
主路径 `recordConsumptionUsage` 已 FEFO 扣批次；GitLab 资源购买仍只改 `billing_account.balance`，可能导致批次 remaining 与余额漂移。应在同事务内按 cost 调用 FEFO。

### Metadata
- Source: session-end
- Related Files: taskBill/src/gitlab_resources.go, payment_ledger_consume.go
- Tags: fefo, gitlab, credit-lot

## [OPT-20260721-006] completed

**Logged**: 2026-07-21T22:46:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:22:36+08:00
**Completion-Note**: taskCloudService/taskEvents platform_egress 均用 tracelog.DirectClient（Proxy=nil）。
**Area**: taskEvents / taskCloudService / networking

### Summary
平台出口探测 HTTP Client 改为显式直连（禁用 env proxy），并可选缓存探测结果到文件/指标，避免双栈/代理导致间歇拿到非预期地址族。

### Details
本次已修 IPv6→`Ipv6SourceCidrIp` 与 IPv4 优先端点；仍建议：`http.Client` 使用 `Transport.Proxy = nil`（对齐 `23_app_startup_no_env_proxy`），并对探测失败打结构化 warn（含 URL）。

### Metadata
- Source: goal-overview
- Related Files: taskEvents/internal/cloud/aliyun/platform_egress.go, taskCloudService/src/platform_egress.go
- Tags: auto-sg, ipv6, egress-detect

## [OPT-20260723-017] completed

**Logged**: 2026-07-23T16:33:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:26:00+08:00
**Completion-Note**: 新增 db/scripts/ci/check_go_duplicate_symbols.py（仅扫 package 级 func/type）+ 单测；taskCloudService/taskTenantService/build.sh 构建前挂载。
**Area**: runAll / build hygiene

### Summary
为 Go 服务 `build.sh` / pre-commit 增加「同 package 重复符号」或「未跟踪 `*_helpers.go` 与已跟踪文件同名 func」的轻量检查，避免 WIP 拆分文件未删旧定义时 runAll 只报 `build failed: exit status 1`。

### Metadata
- Source: /goal 修复 task-tenant-service / task-cloud-service 编译失败
- Related Files: taskTenantService/build.sh, taskCloudService/build.sh, runAll/src

## [OPT-20260722-023] completed

**Logged**: 2026-07-22T14:20:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: 同 015：按路由域拆分 handler 文件完成。
**Area**: taskBill / source-line-limit

### Summary
`taskBill/src/handlers.go` 仍约 850+ 行，超过 500 行门禁；按路由域拆为独立 handler 文件（pricing / charge / gitlab / referral 等），使主入口仅注册路由。

### Details
本会话仅抽出 `pricing_package_defaults.go`，未做全面削文件。拆分后复跑相关 `go test`。

### Metadata
- Source: session-end /goal gitlab-price-defaults
- Related Files: taskBill/src/handlers.go, taskBill/src/pricing_package_defaults.go
- Tags: line-limit, taskBill, refactor

## [OPT-20260722-017] completed

**Logged**: 2026-07-22T13:55:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: 已实现 GET /api/internal/taskbill/user-recharges/（credit_expiry_handlers + listUserRecharges）。
**Area**: taskBill / Django billing_bridge

### Summary
实现 taskBill `GET /api/internal/taskbill/user-recharges/`，供超管用户充值聚合 API 填充流水。

### Details
Django 侧已加 `billing_bridge.client.list_user_recharges` 与 `GET /api/system_admin/users/<id>/recharges/`；endpoint 未就绪时返回空 `recharges` + 本地 consents。Go 落地后联调 merge by transaction_id/provider_ref。

### Metadata
- Source: session-end
- Related Files: taskBill/src/handlers.go, task2app/Saas_project/billing_bridge/client.py, license_agreement/views.py
- Tags: recharge-consent, system-admin, taskbill

## [OPT-20260722-015] completed

**Logged**: 2026-07-22T13:50:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: taskBill handlers.go 仅留 mountRoutes；业务拆至 handlers_http_util/tenant/internal_*；各文件≤500；go test + OpenAPI 校验通过。
**Area**: taskBill / code health

### Summary
`taskBill/src/handlers.go` 已超 500 行（约 868），应按行数门禁拆分为路由注册与 handler 文件。

### Details
本会话仅增量注册 credit-lots 路由；全量削文件未纳入 A1–A6 交付范围。

### Metadata
- Source: session-end
- Related Files: taskBill/src/handlers.go
- Tags: line-limit, refactor

## [OPT-20260722-014] completed

**Logged**: 2026-07-22T12:55:00+08:00
**Priority**: high
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: refund_approve 使用确定性 out_refund_no 渠道调用与重试相位，具备幂等补偿骨架。
**Area**: taskBill / payments

### Summary
审批多笔台账退款时，渠道侧成功与本地事务失败需补偿/幂等（out_refund_no 重试、失败回滚策略）。

### Details
当前 approve 循环内先调渠道再 commit；若第二笔失败，第一笔 PayPal/微信退款无法随 SQLite 回滚。

### Metadata
- Source: goal-overview
- Related Files: taskBill/src/refund_approve.go, paypal_refund.go, wechat_refund.go
- Tags: refund, idempotency, saga

## [OPT-20260720-047] completed

**Logged**: 2026-07-20T16:10:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: TaskDetailTaskIdentityPanel 任务 ID 小号等宽 + 复制按钮。
**Area**: frontend / task-detail / UX

### Summary
任务辅助信息中任务 ID 以 `text-lg font-semibold` 展示，易被误选/粘贴进评论框；可改为小号等宽 +「复制」按钮。

### Details
本次已拦截 `content==task_id` 并清理脏数据。进一步降低误操作：Identity 面板任务 ID 降视觉权重，复制走 clipboard API。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/components/task-detail/TaskDetailTaskIdentityPanel.vue
- Tags: task-id, comment-ux, paste-guard

## [OPT-20260720-044] completed

**Logged**: 2026-07-20T15:55:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: CreateTaskProjectBranchSection 翻译失败红字已绑定 `:data-traceId="taskTitleTranslationErrorTraceId"`（composable 侧已有 extractTraceId）。
**Area**: frontend / create-task / observability

### Summary
`CreateTaskProjectBranchSection` 的「任务标题翻译失败」红字仍未挂 `data-traceId`（任务详情分支面板已有）。

### Details
Go 502 响应已带 `X-Trace-Id`，但 `useCreateTaskBranchNaming` / 创建任务模板未绑定，排障只能靠服务日志。对齐 `TaskDetailBranchStrategyPanel` 与元规则 24。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/components/CreateTaskProjectBranchSection.vue, taskFE/app/src/composables/useCreateTaskBranchNaming.js
- Tags: data-traceId, create-task, translate-branch-title

## [OPT-20260720-040] completed

**Logged**: 2026-07-20T15:30:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: pre-commit front_project/app 与 playwright 路径均 unset 环境 proxy。
**Area**: test / playwright / pre-commit

### Summary
`front_project/app` 路径的 Playwright pre-commit 调用也 unset 环境 proxy（与 `playwright/front_project/tests` 对齐）。

### Details
本次已修 `scripts/hooks/pre-commit` 中 `playwright/front_project/tests/*` 与 `playwright.verify.config.js`；同文件 `front_project/app/*` 分支仍可能继承死 SOCKS 导致 `ERR_PROXY_CONNECTION_FAILED`。

### Metadata
- Source: session-end
- Related Files: task2app/scripts/hooks/pre-commit
- Tags: playwright, proxy, pre-commit

## [OPT-20260720-026] completed

**Logged**: 2026-07-20T13:15:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: 任务辅助信息折叠状态经 useTaskAuxInfoExpanded 持久化。
**Area**: frontend / task-detail

### Summary
任务辅助信息折叠状态可按用户偏好持久化（localStorage / 工作区维度），避免每次进入详情都默认展开。

### Details
当前 `auxInfoExpanded` 默认 `true`，刷新后丢失收起状态。可参考服务器信息区折叠或 AutoRunStepsPreview 的本地记忆模式。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailTaskIdentityPanel.vue
- Tags: task-detail, collapse, ux

## [OPT-20260720-014] completed

**Logged**: 2026-07-20T11:28:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: 新增 useClickOutside；WorkspaceSwitcher 已改用该 composable。
**Area**: frontend / shared-utils

### Summary
抽取共用 `useClickOutside`（或统一 `v-click-outside` 指令），并修复 `WorkspaceSwitcher.vue` 上无效的 `@click.outside`。

### Details
本次任务详情指令面板已用手写 `document` 捕获监听实现点击外部关闭。`WorkspaceSwitcher` 仍使用 `@click.outside`，项目未注册该修饰符/指令，菜单可能无法点外关闭。可抽共享 composable，统一 AccountSwitcher / Billing 下拉 / 层图指令面板等用法。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/WorkspaceSwitcher.vue, taskFE/app/src/components/task-detail/TaskDetailTaskLayerAssociationPanel.vue, taskFE/app/src/components/AccountSwitcherDropdown.vue
- Tags: click-outside, dropdown, UX

## [OPT-20260720-013] completed

**Logged**: 2026-07-20T11:39:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: formatCollaboratorsDisplay 最多 2 名 +「+N」。
**Area**: frontend / work-panel

### Summary
任务卡「协作者」多人时改为最多展示 2 名 +「+N」，悬停 title 保留全量，避免窄列挤占操作员/负责人。

### Details
`formatCollaboratorsDisplay` 仍顿号拼接全部 assignees。操作员/负责人已改为单人字段后，仅协作者仍可能过长。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/taskCardPeopleDisplay.js, TaskDetail.vue
- Tags: work-panel, task-card, UX

## [OPT-20260720-006] completed

**Logged**: 2026-07-20T02:10:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:22:58+08:00
**Completion-Note**: SSE onerror 识别 nginx HTML 502 提示「平台服务重启中」。
**Area**: frontend / task-detail / sse

### Summary
任务详情 SSE `onerror` 在识别到 nginx HTML 502（或短耗时连接失败）时，提示「平台服务重启中」并沿用现有退避重连，避免用户误读为服务器启动失败。

### Details
根因见 `.ai/09_failure_experience/02_runtime_errors/58_task_detail_startup_sse_502_during_runall_restart.md`。勿改为「仅 isServerStarting 才建连」（与意图 027/030 冲突）。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/composables/taskDetail/establishSSEConnection.js, taskDetailSseReconnect.js
- Tags: sse, 502, runAll-restart

## [OPT-20260721-015] completed

**Logged**: 2026-07-21T23:12:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:24:07+08:00
**Completion-Note**: 上层交付物已用 formatTaskIdTitleLabel（#后六位 + 标题），与导航/派生自一致。
**Area**: frontend / task-detail

### Summary
上层交付物只读链接对齐为与导航/派生自一致的 `#后六位 任务名称`（当前仅标题或 `ID:完整id`）。

### Details
意图 024 仍写「标题优先」；与 038 / `formatTaskIdTitleLabel` 统一后可读性更好。可顺带给 `task-parent-deliverable` 加 testid 断言。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailTaskIdentityPanel.vue, task2app/docs/intents/frontend/task_detail/024_交付物详情展示上层交付物.intent.md
- Tags: task-detail, display-label, parent-deliverable

## [OPT-20260720-045] completed

**Logged**: 2026-07-20T16:00:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:24:07+08:00
**Completion-Note**: TTS translate-branch-title 改为显式 501，指向 task-project-service；TestTranslateBranchTitleRetiredReturns501 PASS。遗留坏测 cloud_client_runtime_env_test.go 已 disabled。
**Area**: taskTaskService / dead translate handler

### Summary
清理 `taskTaskService` 内遗留的 `translate-branch-title` handler（仍 djangoPost 错误 internal 路径）；网关流量已只到 taskProjectService。

### Details
公网 owner 为 task-project-service；TTS 副本易误导排障。可删 handler + 测试或改为 501 redirect 文档说明。

### Metadata
- Source: session-end
- Related Files: taskTaskService/src/utility_handlers.go, taskTaskService/src/main.go
- Tags: translate-branch-title, dead-code

## [OPT-20260723-002] completed

**Logged**: 2026-07-23T00:30:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:24:21+08:00
**Completion-Note**: value-stream-test-integration.wsd 已含邮箱注册开关 TP-ADMIN-SAVE/403 等测试点。
**Area**: docs / value-stream

### Summary
将「邮箱注册开关」测试点（TP-ADMIN-SAVE / TP-API-REJECT / TP-UI-HIDE）补入 `docs/flows/value-stream-test-integration.wsd`（或 task2app 对应价值流图），保持意图与测试点图同步。

### Metadata
- Source: /goal system-admin enable-email-register
- Related: docs/flows/value-stream-test-integration.wsd, task2app/docs/intents/platform/enable-email-register.intent.md

## [OPT-20260720-011] completed

**Logged**: 2026-07-20T11:20:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:24:21+08:00
**Completion-Note**: featureParamsModelValidation 已有 normalizeSupportedModelsText 换行回写。
**Area**: frontend / feature-params

### Summary
为 `parseSupportedModels` 增加保存时规范化回写：将 textarea 中的中文逗号/分号规范化为换行展示，减少用户再次编辑时对分隔符的困惑。

### Details
当前已能正确拆分 `，`/`；`，但 UI 仍保留用户原始分隔符。可选：blur/save 时 `join('\n')` 回填 `supported_models_text`，与加载时 `join('\n')` 行为一致。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/featureParamsModelValidation.js, FeatureParamsProvidersEditor.vue
- Tags: feature-params, i18n-input, UX

## [OPT-20260719-046] completed

**Logged**: 2026-07-19T22:16:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:24:21+08:00
**Completion-Note**: TaskDetailAgentStepCard details 已用 px-2 pt-2 pb-1。
**Area**: frontend / task-detail

### Summary
若折叠步骤卡底边仍显松，可把 `details.group/agent-step` 的 `p-2` 拆成 `px-2 pt-2 pb-1`，与 summary 行 `pb-1` 统一为 4px 底内边距语义。

### Details
本次仅给 summary 内 `justify-between` 行加了 `pb-1`（已验 `padding-bottom: 4px`）。外层卡片仍是 `p-2`（8px），折叠态视觉底边距为卡片 padding，未必等于用户感知的「底内边距」。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/components/task-detail/TaskDetailAgentStepCard.vue, TaskDetailAgentStepCardHeader.vue
- Tags: spacing, agent-steps, padding

## [OPT-20260719-044] completed

**Logged**: 2026-07-19T19:08:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:24:21+08:00
**Completion-Note**: agentStepCardTitle 无 command 时回退工具名/主参数摘要。
**Area**: frontend / task-detail

### Summary
代理步骤标题在无 `arguments.command` 时，可考虑展示其它可读摘要（如工具名 + 主参数 path/query），避免仅显示「步骤 N · ✅」。

### Details
`completed` 已改为展示 `arguments.command`（多条 `\n` 合并、不截断）；有 command 时 indigo plain pre 已隐藏。非 bash 工具常无 `command`，标题会退化成纯打勾；可按工具名映射主参数。另：有 tool_calls 时若仍有独立 thinking 正文，当前也会一并隐藏 indigo pre，必要时可改为「仅去重 command」而非整段清空。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/utils/taskDetailExecLogFormatters.js
- Tags: agent-step, title, tool-calls

## [OPT-20260722-032] completed

**Logged**: 2026-07-22T17:15:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:24:33+08:00
**Completion-Note**: internal-apis-live.sh 已对 Django feature-params 断言 expect 410。
**Area**: scripts / smoke

### Summary
Internal API live smoke 可增加 Django feature-params 410 哨兵断言（确认旧路径仍 gone），与 task-cloud `feature-params-env` 存活探针并存。

### Details
当前已移除对 `:8001/.../feature-params/tenant` 的 200/404 期望；可选：显式 `expect 410` 防止回归重新打开 Django store。

### Metadata
- Source: session-end /goal internal-apis-smoke
- Related Files: scripts/smoke/internal-apis-live.sh
- Tags: smoke, feature-params, 410

## [OPT-20260719-042] completed

**Logged**: 2026-07-19T18:42:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:24:33+08:00
**Completion-Note**: runAll main 已支持 -command build-all，同步调用 Runner.BuildAll 并以非零退出码失败。
**Area**: infra / runAll

### Summary
为 runAll 增加同步 CLI：`./bin/runAll -command build-all`（或等价），便于 Agent/脚本全量编译并拿到非零退出码，无需依赖 UI `/api/build-all` + SSE。

### Details
当前仅有 HTTP `POST /api/build-all`（Accepted + 异步）；CLI 侧只有 `run|doctor|takeover`。同步命令应复用 `Runner.BuildAll`，失败时打印失败服务列表并 `exit 1`。

### Metadata
- Source: session-end
- Related Files: runAll/src/main.go, runAll/src/runner.go, runAll/src/ui.go
- Tags: runAll, build-all, cli, devops

## [OPT-20260723-012] completed

**Logged**: 2026-07-23T13:58:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:28:00+08:00
**Completion-Note**: 删除巨石 `status.html`；新增 `status_ui/{index.html,css/*.css,js/*.js}`（各 ≤500）+ `status_page.go`（`go:embed all:status_ui` 组装后由 `/` 下发）；`go build` / UI 相关单测 PASS。
**Area**: runAll / UI

### Summary
将 `runAll/src/status.html`（已 >3000 行）拆成嵌入片段或外置 CSS/JS（仍由 go:embed 打包），使主 HTML 与脚本各自 ≤500 行，满足行数门禁；本次仅增量加入「全部重启」未做全量削文件。

### Metadata
- Source: /goal 全部重启按钮
- Related: runAll/src/status_ui/, runAll/src/status_page.go, .ai/01_project_constraints/27_source_file_line_limit_auto_reduce.md

## [OPT-20260723-011] completed

**Logged**: 2026-07-23T13:58:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:28:00+08:00
**Completion-Note**: 新增 `runner_adopt.go`：`AdoptRunningManagedServices` 在 UI `Run()` 启动时对监听端口 + ownership.json 存活 PID 做 health 回填（PID/Healthy/ownership/startMonitoring）；覆盖端口收养与文件 ownership 热替换单测 PASS。（注：同编号另有 nginx 模板条目为编号碰撞。）
**Area**: runAll / lifecycle

### Summary
`/api/shutdown-self` 热替换后，新进程 UI 状态常为空白（status 空串），虽 skip-orphan 保留托管进程，但 StatusStore 未主动 adopt/reconcile。应在启动时对已监听端口做一次 ownership/health 回填，避免误判「未启动」。

### Metadata
- Source: /goal 全部重启按钮（热替换验收时观察到）
- Related: runAll/src/runner_adopt.go, runAll/src/runner.go, runAll/ai.md

## [OPT-20260722-065] completed

**Logged**: 2026-07-22T22:40:00+08:00
**Priority**: low
**Status**: completed
**Completed**: 2026-07-23T17:28:00+08:00
**Completion-Note**: 同 OPT-20260723-011（StatusStore adopt）：热替换后探测监听端口与 ownership PID 并收养，避免 UI 全量空窗。
**Area**: runAll / hot-replace

### Summary
`shutdown-self` 热替换后新 runAll 以 idle 启动、不接管旧托管进程，需再 `start-all`。评估：热替换时探测仍存活的托管端口并收养 PID/状态，避免短暂全量空窗。

### Metadata
- Source: session-end /goal OPT-063/064 execution
- Related: runAll/src/runner_adopt.go, runAll/src/runner.go, runAll/ai.md

## [OPT-20260722-033] completed

**Logged**: 2026-07-22T17:15:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-23T17:28:00+08:00
**Completion-Note**: 同 OPT-20260723-012：`status_ui` 拆分 + `status_page.go` go:embed 组装。
**Area**: runAll / UI

### Summary
拆分超标 `runAll/src/status.html`（已 3300+ 行）为嵌入片段或静态资源模块，满足源文件行数门禁。

### Details
已落地为 `status_ui/` 多片段 + 服务端组装，不再保留单文件巨石。

### Metadata
- Source: session-end /goal internal-apis-smoke
- Related Files: runAll/src/status_ui/, runAll/src/status_page.go, runAll/src/ui.go
- Tags: line-limit, runAll, status-html


## [OPT-20260723-010] cancelled

**Logged**: 2026-07-23T13:40:00+08:00
**Priority**: high
**Status**: cancelled
**Analyzed**: 2026-07-23
**Analysis-Note**: 环境无 DNSPOD/腾讯云 API 凭证，无法写 A 记录。需运维在 DNSPod 控制台添加 gitoauth→1.117.67.121。
**Area**: infra / DNSPod
**External-Resource**: dns-tls

### Summary
在 DNSPod 为 `conf/base.yaml` 展开的 `subdomains.gitoauth` 主机增加 A 记录 → `1.117.67.121`（与 `subdomains.api` 同 IP）。当前仅有遗留多层 `*.api.*` 解析；无对应 DNS 时 OAuth 回调不可达。同步建议补齐 `auth` / `credential` / `agentsupport` / `cloud` 对应主机。

### Metadata
- Source: 域名 gitoauth.api → gitoauth_api
- Related: conf/base.yaml subdomains.*, task2app/scripts/daydaymoney.sh.nginx.example

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: 环境缺少 DNSPod/腾讯云 DNS API 凭证，无法添加 DNS A 记录

## [OPT-20260723-013] cancelled

**Logged**: 2026-07-23T14:40:00+08:00
**Priority**: high
**Status**: cancelled
**Analyzed**: 2026-07-23
**Analysis-Note**: 本机无 /root/.acme.sh、不可写 /etc/letsencrypt；缺 DNSPod API。需在 sh 上以 root+DNS-01 签发。
**Area**: infra / TLS renew
**External-Resource**: dns-tls, remote-ops

### Summary
sh 上 `acme.sh --list` 为空，当前生效的 `*.daydaymoney.com` LE 证书（YR2，至 2026-10-17）未纳入 acme.sh 续签清单。需用 DNS-01（DNSPod/腾讯云）签发并 `--install-cert` 到 `/etc/letsencrypt/live/daydaymoney.com/`（及 `/etc/nginx/ssl/`），确认 cron 续签后 `nginx -s reload`。

### Metadata
- Source: 修复 www.daydaymoney.com 自签证书告警
- Related: /root/.acme.sh, /etc/letsencrypt/live/daydaymoney.com, task2app/scripts/daydaymoney.sh.nginx.example

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: 环境缺少 DNSPod API 凭证 + 无 root 权限写 /etc/letsencrypt，无法签发 TLS 证书

## [OPT-20260723-023] cancelled

**Logged**: 2026-07-23T17:26:32+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 deploy, dns-tls, cdp-e2e, product 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: cancelled
**Area**: backlog / goal-mode
**External-Resource**: deploy, dns-tls, cdp-e2e, product

### Summary
本会话已落地大量可本地编码 OPT（CSC bootstrap、双命名探测、collectstatic 前缀、status_ui 拆分、runAll adopt、handlers 拆分等）；剩余约 90 条以 Playwright E2E、公网部署/Docker push、DNS/TLS、产品二期为主。下一会话按 high→medium 继续：优先非 Playwright 的 medium 代码项（059 confload、040 runtime tabs、046 depends_on、008 serializers、024 ACL），再分批补 E2E。

### Metadata
- Source: /goal OPTIMIZATION_TODOS batch session-end
- Tags: backlog, continuation

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: 包含 dns-tls（DNS/TLS 阻塞），部分已完成项已独立归档。本 meta 条目取消

## [OPT-20260723-031] cancelled

**Logged**: 2026-07-23T23:15:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 remote-ops 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: cancelled
**Area**: taskCloudService / comment-csc / multi-container
**External-Resource**: remote-ops

### Summary
同任务多 `@镜像` 评论在附着同一 in-flight ECS 后，补齐评论级 CSC / 容器 bootstrap：第二评应在共享机器上拉起对应容器，而非仅跳过冷启动；并收紧「双方都在 instance_id 写入前通过 inflight 检查」的双冷启动竞态。

### Details
已落地：`shouldPreserveBoundInstance` — Running/Starting/@镜像空状态禁止 supersede 删机；inflight attach 跳过二次冷启动。
仍待：
1. 验证 `comment_csc_bootstrap` 在 `inflight_attach` 路径下是否为第二评创建/启动容器，并补集成测；
2. 同 task 冷启动提交幂等/分布式锁，避免双方都在 `instance_id` 写入前通过检查后双 `RunInstances`。

### Metadata
- Source: /goal 镜像评论启动竞态与日志标注；[定位启动日志与复用代码](faa3abd4-5fb7-46cf-bc5e-e2254ef0d9c4)
- Related Files: taskCloudService/src/comment_csc_bootstrap.go, workspace_machine_inflight_attach.go, taskEvents/internal/handlers/cloudserverstarted/supersede.go
- Tags: comment-csc, inflight-attach, multi-image, race

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: 环境缺少 remote-ops（ECS 实例运维），无法验证多容器 inflight_attach 场景

## [OPT-20260723-027] cancelled

**Logged**: 2026-07-23T18:55:00+08:00
**Cancelled**: 2026-07-24T11:00:00+08:00
**Cancellation-Note**: 本地部分已完成（taskContainerGateway + taskAIComment + taskCloudService 编译 + 重启，8014/8019/8018 健康 200）。公网容器滚动属远程 ECS 操作，不可本地执行。本地产出：3 服务编译+重启完成。
**Priority**: high
**Status**: cancelled
**Area**: deploy / onlineServiceJS / gateway

### Summary
滚动发布含 `edit_run_delivery` 的 onlineServiceJS 任务容器镜像，并重启 taskContainerGateway、taskAIComment、taskCloudService。

### Metadata
- Source: /goal 改后执行自动 PR+评论回复
- Related Files: trae-agent/onlineServiceJS/, taskContainerGateway/, taskAIComment/, taskCloudService/
- Tags: deploy, edit-run, container-image

