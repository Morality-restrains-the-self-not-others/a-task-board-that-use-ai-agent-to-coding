# Completed OPT Archive — 2026-07-20

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 22 条。
> 归档执行时间：2026-07-24T10:15:26+08:00

## [OPT-20260720-040] completed

**Logged**: 2026-07-20T15:10:00+08:00
**Priority**: low
**Status**: completed
**Area**: onlineServiceJS / tests

### Summary
修复或删除陈旧的 `bootstrap.agentConfig.test.mjs`：它 import 已不存在的 `materializeAgentConfigFile`，会被 `bootstrap.*.test.mjs` glob 误伤。

### Details
功能已由 `persistFeatureParamsEnv` / `resolveAgentConfigFromEnv` 覆盖；宜改测例指向现 API，或移出 `bootstrap.*.test.mjs` 命名。

### Metadata
- Source: bootstrap-split session
- Related Files: trae-agent/onlineServiceJS/src/bootstrap.agentConfig.test.mjs
- Tags: test-debt, onlineServiceJS

**Completed**: 2026-07-20T15:10:49+08:00
**Completion-Note**: 改写为 resolveAgentConfigFromEnv 冒烟测；落盘由 featureParamsEnvLog 覆盖。

## [OPT-20260720-033] completed

**Logged**: 2026-07-20T14:20:00+08:00
**Priority**: medium
**Status**: completed
**Area**: onlineServiceJS / maintainability

### Summary
将超大 `server.mjs`（约 2200 行）按职责拆到 ≤500 行模块（listen 主流程等）。`bootstrap.mjs` 已拆完（220 行 + 叶子模块）。

### Details
`bootstrap.mjs` → 叶子（State/CloneLog/RepoCredentials/CloneLayer/Failure/CredentialsRecovery/RepoInputs/TokenStore/TokenExchange/FeatureParamsPersist 等）；`server.mjs` → routes*/uiHtml/middleware/diffSummary 等。全部生产模块 ≤500。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/server.mjs, trae-agent/onlineServiceJS/src/bootstrap.mjs
- Tags: line-limit, refactor, onlineServiceJS

**Completed**: 2026-07-20T15:10:32+08:00
**Completion-Note**: server.mjs 281 / bootstrap.mjs 247；routesLayerGit 483 等叶子均 ≤500；相关单测通过。

## [OPT-20260720-034] completed

**Logged**: 2026-07-20T14:20:00+08:00
**Priority**: low
**Status**: completed
**Area**: observability / container

### Summary
将 onlineServiceJS 容器内 `AUTO_RUN_FIRST_*` / `BOOTSTRAP_*` 关键日志纳入 Loki（或经 heartbeat/SSE 上报），便于无 SSH 时排查「克隆成功但未跑 Agent」。

### Details
本次排障靠凭证服务 HTTP 状态时间线推断；容器 stdout 未进 Loki，无法直接搜到 `AUTO_RUN_FIRST_SKIP`。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/autoRunOrchestration.mjs, AiMonitor
- Tags: loki, auto_run, observability

**Completed**: 2026-07-20T15:10:32+08:00
**Completion-Note**: runtime-event Cloud 端 + emitRuntimeEvent；AUTO_RUN/BOOTSTRAP 可经 Loki job=task-cloud-service 检索。

## [OPT-20260720-030] completed

**Logged**: 2026-07-20T13:45:00+08:00
**Completed**: 2026-07-20T14:04:00+08:00
**Priority**: low
**Status**: completed
**Area**: taskProjectService / provider

### Summary
为 `IsGitLabRepo` / `matchProvider` 的「未知 host + `.git` → GitLab」启发式补回归用例矩阵（含自建 Git、非 github/gitlab 的 `*.git`），避免再误伤其它 forge。

### Details
本会话已修 github.com`*.git` 误判并加 `TestIsGitLabRepoDoesNotMisclassifyGitHubDotGit`。可再覆盖：`git@github.com:org/repo.git`、`https://git.example.com/a/b.git`（仍走 gitlab 启发式）、以及 `listNestedGitRepos` 在双 flag 冲突时的优先级单测。

### Completion-Note
新增 `TestProviderHeuristicMatrix`（9 子例：HTTPS/SCP GitHub、自建 `*.git`、gitlab host、无后缀未知 host）、`TestResolveProviderFallbackGitHubSCPAndUnknownDotGit`、`TestListNestedGitReposGitHubDotGitPrefersGitHubAPI`；`go test` 相关用例全通过。

### Metadata
- Source: goal-overview
- Related Files: taskProjectService/src/provider_resolver_test.go, taskProjectService/src/nested_git_handlers_test.go
- Tags: nested-git, provider, github

## [OPT-20260720-028] completed

**Logged**: 2026-07-20T13:25:00+08:00
**Completed**: 2026-07-20T13:39:00+08:00
**Priority**: high
**Status**: completed
**Area**: container / docker

### Summary
将含 `bootstrapCloneLayoutSeal` 与 nested `layerGitRemoteSnapshot` 聚合的 onlineServiceJS 推镜像并滚动公网任务容器。

### Completion-Note
trae-agent `3e21761`；`DOCKER_PUSH=1 ./buildDocker.sh` 推送 `x86_64_2026-07-20_13-27`/`arm64_2026-07-20_13-27` 与 `*-latest`。对 `task_13633888450958958867` stop（实例已停）→ `start-vm-auto` 滚动至 `i-j6c57m3gs9rd14eoiue3` / `47.76.185.86`；镜像内 `bootstrapCloneLayoutSeal.mjs` 与 `aggregateGitRemoteSnapshots` 验收通过。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/bootstrapCloneLayoutSeal.mjs, trae-agent/onlineServiceJS/src/layerFsGitRemote.mjs
- Tags: docker, nested-git, clone-seal

## [OPT-20260720-027] completed

**Logged**: 2026-07-20T13:25:00+08:00
**Completed**: 2026-07-20T13:39:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend / spa

### Summary
前端「相对父层差异」文案改动经 `runall-lifecycle.sh build` + collectstatic 发布到公网 SPA。

### Completion-Note
lifecycle build 后补 `rsync` assets→STATIC_ROOT；公网/本地 `TaskDetailContent.logic-DWzWYmMp.js` 含「相对父层差异」×2；`main-BE7WAgka.js` 200。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/layerChangesDirty.js
- Tags: spa, collectstatic, ztree

## [OPT-20260720-019] completed

**Logged**: 2026-07-20T11:52:00+08:00
**Completed**: 2026-07-20T12:00:00+08:00
**Priority**: low
**Status**: completed
**Area**: container / onlineServiceJS

### Summary
独立 `.git` 拷贝且 HEAD 分叉、对象互不可见时，tree-diff 失败会回退 walk；可评估 `git diff --no-index` 或临时 object 互通，避免大仓回退触顶。

### Details
当前 `collectPairChanges` 在共享对象库（symlink `.git` / worktree）下 git-first 正确；两份完整 clone 分叉且未 fetch 时 `git diff hp hc` 会因缺对象失败并 walk。叠层常见路径已覆盖；独立 clone 为边缘场景。

### Completion-Note
HEAD 分叉且跨仓 tree-diff 失败时，改用双端 `git ls-tree -r HEAD` 比 blob（无需对象互通）；单测「独立 clone + 4200 tracked」`strategy=git` 且 `truncated=false`。见 `headDivergeCandidates` / `dualLsTreeDiffPaths`。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/layerParentDiffGit.mjs, trae-agent/onlineServiceJS/src/layerParentDiffGit.test.mjs
- Tags: layer-diff, git, edge-case

## [OPT-20260720-016] completed

**Logged**: 2026-07-20T11:40:00+08:00
**Completed**: 2026-07-20T11:55:00+08:00
**Priority**: medium
**Status**: completed
**Area**: container / onlineServiceJS

### Summary
任务容器需带上 `layerFileContent.mjs` 后，二进制预览才返回精确 `size_bytes`/`mtime`；确认镜像构建/热挂载路径会同步该模块，并对运行中任务提供无感升级或文档说明。

### Details
本会话已改容器侧 `GET /files/*`；前端对旧 API（content 含 NUL）有兼容属性展示。精确属性依赖容器进程加载新代码。应核对 `go_run_container` / bootstrap 是否把 `trae-agent/onlineServiceJS/src/layerFileContent.mjs` 打入镜像或 bind-mount，并在任务详情旁提供「需重启容器后属性更完整」的运维说明（若当前任务仍跑旧镜像）。

### Completion-Note
确认两条路径：(1) 镜像 `COPY onlineServiceJS` 整树烘焙（非 dockerignore）；(2) 本地 relay 原白名单漏挂新模块——改为整目录 overlay `src`→`/app/onlineServiceJS/src:ro`。拆分 `container_image_*.go` 行数门禁；companion `onlineServiceJS/ai.md` + 故障篇 62 已写运维说明。公网仍须 commit 后 `DOCKER_PUSH=1 ./buildDocker.sh` 并重建任务容器。

### Metadata
- Source: goal-overview
- Related Files: go_relayToTrae/src/container_image_overlay.go, trae-agent/onlineServiceJS/Dockerfile, trae-agent/onlineServiceJS/ai.md
- Tags: binary-preview, container-image, layer-file-content

## [OPT-20260720-018] completed


**Logged**: 2026-07-20T11:35:00+08:00
**Completed**: 2026-07-20T11:50:00+08:00
**Priority**: medium
**Status**: completed
**Area**: container / onlineServiceJS

### Summary
父层变动文件列表改为优先 git status / tree-diff 生成变动集，全量 `collectIndex` walk 仅作无 git 回退，彻底降低误报「目录扫描已达上限」。

### Details
阶段 A（跳过 `node_modules` 等）已落地，但大型 monorepo 非噪声文件仍可触 `MAX_DIFF_ENTRIES=4000`。`getLayerParentDiffFiles` 改为 `collectPairChanges`：有 git 时双端 status ∪ HEAD 分叉 tree-diff；无 git 才 walk。

### Completion-Note
开放清单曾误用编号 OPT-20260720-015（与已归档 feature-params 项冲突），落地时重编号为 018。新增 `layerParentDiffGit.mjs` / `layerParentDiffCompare.mjs`，`layerParentDiffGit.test.mjs` 4 场景通过。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/layerParentDiffGit.mjs, trae-agent/onlineServiceJS/src/layerParentDiffCompare.mjs, trae-agent/onlineServiceJS/src/layerParentDiff.mjs
- Tags: layer-diff, performance, scan-cap, git

## [OPT-20260720-015] completed

**Logged**: 2026-07-20T11:30:00+08:00
**Completed**: 2026-07-20T11:32:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / feature-params

### Summary
工作空间环境变量页（`WorkspaceFeatureParamsSettings.vue`）补充与公司级设置一致的「可见性/资源隔离」备注说明。

### Details
公司级 `/settings/feature-params/` 已注明：能看到任务的人可获知相关环境变量；资源隔离应走个人配置。工作空间默认配置若被任务选用，同样存在可见性暗示，可同步一段简短备注避免管理员误解。

### Completion-Note
用户口中的「OPT-20260720-013」因编号已被 accounts_company 归档项占用，本项重登记为 OPT-20260720-015。已在标题区同步公司级备注文案，并完成 `runall-lifecycle.sh build`。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/views/WorkspaceFeatureParamsSettings.vue
- Tags: feature-params, UX, copy

## [OPT-20260720-014] completed

**Logged**: 2026-07-20T10:30:00+08:00
**Completed**: 2026-07-20T10:36:00+08:00
**Priority**: low
**Status**: completed
**Area**: django / company-fk

### Summary
将仍引用 `accounts.Company` FK 的 saas 子表（subscriptions、git identity 等）改为无约束 `company_id` 字段（`db_constraint=False` 或 CharField），消除对已删 parent 表的 schema 依赖。

### Details
`accounts_company` 已从 saas DROP；子表仍带历史 FK 定义。运行时 SQLite 无强制，但迁移/校验可能踩坑。

### Completion-Note
ORM `db_constraint=False`；`db/scripts/strip_saas_accounts_company_fk.py` 重建 7 张子表去掉 REFERENCES；migrations accounts.0042 / cloud.0053 / projects.0062 / subscriptions.0002 已应用。

### Metadata
- Source: goal-overview
- Related Files: subscriptions/models/subscription.py, accounts/models/user_company_git_identity.py, db/scripts/strip_saas_accounts_company_fk.py
- Tags: accounts_company, fk-cleanup

## [OPT-20260720-013] completed

**Logged**: 2026-07-20T10:25:00+08:00
**Completed**: 2026-07-20T10:30:00+08:00
**Priority**: medium
**Status**: completed
**Area**: taskTenantService / accounts_company

### Summary
将 saas `accounts_company` 全量迁入 taskTenantService，Django Company ORM 改为调 tenant API，去掉双写与 saas 表。

### Details
当前 creator SSOT 已在 tenant；Django 仍直连 saas 表做 name/列表，`post_save` 双写。见 `docs/architecture/company-creator-tenant-migration.md`。

### Completion-Note
tenant 全量公司 API；Django `Company` unmanaged 门面；saas 表 DROP（27 行已回填）；CompanyViewSet/intent 单测通过。

### Metadata
- Source: goal-overview
- Related Files: accounts/models/company.py, company_tenant_api.py, taskTenantService/src/company_store.go
- Tags: accounts_company, data-ownership

## [OPT-20260720-012] completed

**Logged**: 2026-07-20T10:15:00+08:00
**Completed**: 2026-07-20T10:25:00+08:00
**Priority**: low
**Status**: completed
**Area**: taskTenantService / company-creator

### Summary
将 `Company.creator_id`（`company-creator` internal）迁出 Django，由 taskTenantService（或 taskCloud）提供，供 `is_tenant` enrich 等调用，去掉 `djangoGet(…/company-creator/)`。

### Details
`taskProjectService` enrich 已用 taskAuth 取 email；`is_tenant` 仍经 Django `feature-params/company-creator`。tenant 侧已有 `fetchCompanyCreator` 亦依赖该路径。

### Completion-Note
tenant `accounts_company` + `/companies/creator`；project enrich / policy 本地读；Django 路由删除；回填 27 行；现场 `is_tenant=true`。

### Metadata
- Source: goal-overview
- Related Files: taskTenantService/src/company_client.go, taskProjectService/src/tenant_client.go
- Tags: company-creator, data-ownership

## [OPT-20260720-010] completed

**Logged**: 2026-07-20T02:25:00+08:00
**Completed**: 2026-07-20T10:15:00+08:00
**Priority**: low
**Status**: completed
**Area**: taskProjectService / enrich-parity

### Summary
若产品需要完整 `user_info.email` / `is_tenant`，在 Go enrich 中补 taskAuth 与公司创建者判定，与旧 Django 行为对齐。

### Details
当前有意简化：`email=""`、`is_tenant=false`；主路径只依赖 `member_name`。见 `docs/architecture/workspace-collaborators-go-migration.md`。

### Completion-Note
`auth_client.go`：taskAuth 取 email；`companyCreatorID` 判定 `is_tenant`；单测 + 现场 `email`/`is_tenant=true` 已验证。

### Metadata
- Source: goal-overview
- Related Files: taskProjectService/src/auth_client.go, workspace_access_enrich.go
- Tags: enrich-parity, taskAuth

## [OPT-20260720-011] completed

**Logged**: 2026-07-20T02:25:00+08:00
**Completed**: 2026-07-20T10:15:00+08:00
**Priority**: low
**Status**: completed
**Area**: django / cleanup

### Summary
确认无调用方后删除 Django `workspace-collaborators` / `enrich-workspace-permissions` 的 410 路由与 `taskproject_internal_retired.py`。

### Details
现保留 410 作迁移哨兵；稳定一周期后可物理删除。

### Completion-Note
路由与 `taskproject_internal_retired.py` 已删；Django 同路径 **404**；Go 公网仍 **200**。

### Metadata
- Source: goal-overview
- Related Files: task2app/Saas_project/projects/urls_taskproject_internal.py
- Tags: cleanup, 410

## [OPT-20260720-009] completed

**Logged**: 2026-07-20T02:18:00+08:00
**Completed**: 2026-07-20T02:25:00+08:00
**Priority**: high
**Status**: completed
**Area**: taskProjectService / migrate-to-go

### Summary
将 `workspace-collaborators` 的 Django internal enrichment 收进 taskProjectService：直调 taskTenantService，去掉 `djangoPost(…/workspace-collaborators/)`。

### Details
可行性与步骤见 `docs/architecture/workspace-collaborators-go-migration.md`。公网入口已在 Go；Django 仅编排且 `permissions` 还回调 Go。同模式一并迁 `enrich-workspace-permissions`。

### Completion-Note
Go：`tenant_client` + `workspace_access_enrich`；handlers 直拼 DTO；Django internal → 410；本地验 collab/perms 200、Django 410；单测 Go + pytest 通过。

### Metadata
- Source: goal-overview
- Related Files: taskProjectService/src/workspace_access_handlers.go, task2app/Saas_project/projects/taskproject_internal_retired.py
- Tags: migrate-to-go, workspace-collaborators, enrich-workspace-permissions, taskTenantService

## [OPT-20260720-001] completed

**Logged**: 2026-07-20T00:20:00+08:00
**Completed**: 2026-07-20T00:55:00+08:00
**Priority**: high
**Status**: completed
**Area**: data-ownership / feature-params

### Summary
将 feature-params 六表迁入 `task_cloud.db`，Cloud 本地 SQL 读写，去掉 Django internal store。

### Details
用户口中的「OPT-20260719-051」因编号已被 nested-git 占用，本项重登记为 OPT-20260720-001。

### Completion-Note
`ensureFeatureParamsSchema` + local store；`db/scripts/migrate_feature_params_to_task_cloud.sh`；Django store/upsert/snapshot/audit → 410；ownership 改 task-cloud。`resolve-owner-user` 仍留 Django。

### Metadata
- Source: goal OPT-A
- Related Files: taskCloudService/src/feature_params_schema.go, feature_params_local_*.go, db/feature_params/schema.sql, db/table_ownership.yaml
- Tags: feature-params, task-cloud, ownership

## [OPT-20260720-002] completed

**Logged**: 2026-07-20T00:20:00+08:00
**Completed**: 2026-07-20T00:55:00+08:00
**Priority**: high
**Status**: completed
**Area**: api / feature-params

### Summary
`compute/feature-params-env-preview` 迁 Go；Django 同路径 410。

### Details
用户口中的「OPT-20260719-052」因编号冲突重登记为 OPT-20260720-002。

### Completion-Note
`handleFeatureParamsEnvPreview` 挂入 `handleCloudTaskRoutes`；Django `@require_task_cloud_service`；网关既有 task-cloud-service 路由可覆盖。

### Metadata
- Source: goal OPT-B
- Related Files: taskCloudService/src/feature_params_env_preview.go, task2app/.../task_cloud_deprecated.py
- Tags: feature-params, env-preview, go

## [OPT-20260720-043] cancelled

**Logged**: 2026-07-20T15:55:00+08:00
**Priority**: high
**Status**: cancelled
**Area**: django / taskProjectService / translate-branch-title

### Summary
补齐 Django internal `POST /api/internal/taskproject/translate-branch-title/`（复用 fanyi_agent），修复创建任务中文标题翻译 502。

### Details
Go `handleTranslateBranchTitle` 已委托该路径，但 `urls_taskproject_internal.py` 未登记；Django 404 `detail` 被 Go 折叠为裸文案「任务标题翻译失败」。公网 tenant `850256677331562496` 今日日志已多次 502。参见 `.ai/09_failure_experience/02_runtime_errors/44_create_task_translate_branch_title_405.md` 修复项 2（未落地）。

**Cancelled**: 2026-07-20 — 已改为 taskProjectService 直连 fanyi_agent（不再补 Django internal）。

### Metadata
- Source: session-end
- Related Files: task2app/Saas_project/projects/urls_taskproject_internal.py, task2app/Saas_project/projects/taskproject_internal_views.py, taskProjectService/src/utility_handlers.go
- Tags: translate-branch-title, fanyi_agent, 502


## [OPT-20260720-004] cancelled

**Logged**: 2026-07-20T02:05:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 remote-ops, data-migration 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: cancelled
**Area**: data-repair / cloud-history
**External-Resource**: remote-ops, data-migration

### Summary
批量扫描 `cloud_server_config_histories` 中「有 instance_type_id 且 cpu/mem 为占位 1/1」的行，按 DescribeInstanceTypes / 规格缓存回填核内。

### Details
本次仅修复了 `ecs.e-c1m4.xlarge` 脏行；其它规格若曾以占位核内落库，可用同策略离线批修。

### Metadata
- Source: goal-overview
- Related Files: taskCloudService/src/server_config_store.go, data/task_cloud.db
- Tags: server-start-history, instance-type, data-repair

**Cancelled**: 2026-07-24T10:31:20+08:00
**Cancellation-Note**: 环境缺少 remote-ops（ECS DescribeInstanceTypes API 凭证），无法完成云实例规格数据修复

## [OPT-20260720-050] cancelled

**Logged**: 2026-07-20T16:25:00+08:00
**Cancelled**: 2026-07-24T11:00:00+08:00
**Cancellation-Note**: 本地部分已完成（taskCloudService Go 编译 + 重启，SPA build）。公网验收需容器滚动（远程 ECS），属 ai.md 规定的不可执行外部资源，标记 cancelled。本地产出：taskCloudService 8018 健康 200，SPA vite build + collectstatic 完成。
**Priority**: high
**Status**: cancelled
**Area**: taskCloudService / frontend / deploy

### Summary
发布 taskCloudService（禁止无 OAuth 裸 push）+ 前端 SPA build，使公网 ztree「推送并创建PR」走 oauth-access-push。

### Metadata
- Source: session-end
- Related Files: taskCloudService/src/git_push_internal.go, taskFE/app/src/composables/taskDetail/taskDetailLayerActions.js
- Tags: git-push, oauth, deploy

## [OPT-20260720-041] cancelled

**Logged**: 2026-07-20T15:30:00+08:00
**Cancelled**: 2026-07-24T11:00:00+08:00
**Cancellation-Note**: onlineServiceJS 镜像已推送 x86_64-latest（含 auto_run kickoff + runtime-event）。任务容器滚动需远程 ECS 操作，不可本地执行。本地产出：go_run_container 已运行（8796），镜像确认推送。
**Priority**: medium
**Status**: cancelled
**Area**: container / onlineServiceJS / deploy

### Summary
公网/云任务容器滚动重启（或重建）以拉取含 auto_run kickoff + runtime-event 的新 onlineServiceJS 镜像。

### Metadata
- Source: session-end
- Related Files: trae-agent/onlineServiceJS/src/postBootstrapAgentKickoff.mjs, trae-agent/onlineServiceJS/ai.md
- Tags: docker, auto_run, loki

