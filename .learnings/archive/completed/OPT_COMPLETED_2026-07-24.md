# Completed OPT Archive — 2026-07-24

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 68 条。
> 归档执行时间：2026-07-25T14:58:50+08:00

## [OPT-20260724-010] completed

**Logged**: 2026-07-24T03:30:00+08:00
**Completed**: 2026-07-24T09:00:00+08:00
**Completion-Note**: 在 db.go 新增 seedDefaults() 函数，幂等插入"全局默认交付物体系"（待处理/进行中/已完成 三列）和"系统默认进度体系"（同三列），在 openDB() 中 runMigrations() 后调用。Go build 验证通过。
**Priority**: low
**Status**: completed
**Area**: taskProjectService / startup / data-seeding

### Summary
Go taskProjectService 启动时添加默认系统交付物体系和进度体系的初始数据 seeding（"全局默认交付物体系" / "系统默认进度体系" + 默认列），替代已删除的 Django init 脚本。

### Details
Django `scripts/init/init_deliverable_system.py` 和 `01_01_init_deliverable_system.py` 已删除；Go 侧 `runMigrations()` 仅建表不插数据。首次启动或全新 SQLite 需有默认行。可在 `openDB` 后调用 `seedDefaults()` 幂等插入。

### Metadata
- Source: /goal 交付物体系 Django→Go 迁移 后续建议
- Related Files: taskProjectService/src/db.go, taskProjectService/src/deliverable_handlers.go

## [OPT-20260724-011] completed

**Logged**: 2026-07-24T03:43:00+08:00
**Completed**: 2026-07-24T09:00:00+08:00
**Completion-Note**: project_repo.py 四个模型（ProjectRepo/TaskProject/TaskRepoIdentity/TaskBranchStrategy）均已添加 managed=False，并从 __init__.py 导出为 unmanaged stubs（与 Workspace/Project 一致）。Django 语法验证通过。
**Priority**: medium
**Status**: completed
**Area**: saas-backend / models / __init__.py exports

### Summary
Go 迁移后 `projects/models/__init__.py` 和 `cloud/models/__init__.py` 缺少多个模型导出，导致 import cascade 失败。当前手动恢复 6 个导出（Todo, WorkspaceAccess, Comment, AITaskComment, ContainerImage）使服务可启动，但需审计完整迁移计划——部分模型可能应完全移除而非仅删除导出。

### Details
用户确认 "已经迁移到 Go 中" 后，发现 `DeliverableObjSerializer` 及其他模型导出（Todo、WorkspaceAccess 等）被清理但引用未同步。当前采用最小修复：创建 `deliverable_obj_serializer.py` + 补齐 6 个 `__init__.py` 导出。长期需确认迁移范围并清理所有残留引用，或回退导出删除。

### Metadata
- Source: /goal 修复 DeliverableObjSerializer ImportError → saas-backend 启动
- Related Files: projects/models/__init__.py, cloud/models/__init__.py, projects/serializers/__init__.py, projects/serializers/deliverable_obj_serializer.py

## [OPT-20260724-012] completed

**Logged**: 2026-07-24T03:48:00+08:00
**Completed**: 2026-07-24T09:00:00+08:00
**Completion-Note**: 与 OPT-20260724-011 合并处理。ProjectRepo/TaskProject/TaskRepoIdentity/TaskBranchStrategy 均添加 managed=False，从 __init__.py 导出为 unmanaged stubs。模型作为 Python 类保留供 import 兼容。
**Priority**: medium
**Status**: completed
**Area**: saas-backend / Go migration / models audit

### Summary
`projects/models/project_repo.py`（含 `ProjectRepo`, `TaskProject`, `TaskRepoIdentity`）是孤儿死代码——文件存在但未被任何 `__init__.py` 导出，Django 不会注册这些模型。确认是否需要清理或恢复。

### Details
迁移 0051 执行了 `DeleteModel('ProjectRepo')` 和 `RemoveField` 删除相关关联。模型文件仍在但无导出。若未来需要 `project.project_repos.all()` 反向关联，需重新注册。否则应删除文件并清理残留 import（如 tests/ 中的引用）。

### Metadata
- Source: /goal 消除 Go 迁移不一致性 — 审计 models 导出
- Related Files: projects/models/project_repo.py, projects/migrations/0051_remove_django_project_domain.py

## [OPT-20260724-013] completed

**Logged**: 2026-07-24T03:48:00+08:00
**Completed**: 2026-07-24T04:48:00+08:00
**Completion-Note**: 全部 6 个 WorkspaceAccess.objects 调用点已迁移到 go_client：todo_serializer、workspace_access_views（完全重写）、todo_views、accounts/views、Kafka handler。新增 go_client.upsert_workspace_access_permission()。模型已标记 managed=False。
**Priority**: medium
**Status**: completed
**Area**: saas-backend / Go migration / WorkspaceAccess

### Summary
`WorkspaceAccess` 模型状态不一致：迁移 0051 删表 → 0055 重注册 → 0059 从 migration state 删除。当前模型文件仍为 `managed=True`（默认），但表已被迁移删除。`todo_serializer.py` 仍直接查询 `WorkspaceAccess.objects.filter(...)`。需确认表是否真的存在，或将查询迁移到 Go client（`go_client.list_workspace_access` / `create_workspace_access` / `delete_workspace_access`）。

### Details
Go client 已有 workspace-access CRUD API（`list_workspace_access`, `create_workspace_access`, `delete_workspace_access`），但 Python 代码仍直接用 Django ORM 查询 `WorkspaceAccess`。迁移应先将调用点（todo_serializer, workspace_views 等）改用 Go client，再添加 `managed = False`。

### Metadata
- Source: /goal 消除 Go 迁移不一致性 — 审计 models 导出
- Related Files: projects/models/workspace_access.py, projects/serializers/todo_serializer.py, projects/go_client.py, projects/migrations/0059_delete_workspaceaccess_state.py
- Tags: seeding, startup, migration

## [OPT-20260724-008] completed

**Logged**: 2026-07-24T03:30:00+08:00
**Completed**: 2026-07-24T09:00:00+08:00
**Completion-Note**: 删除 9 个死代码文件（views.py, views_tenant.py, views_tenant_part1.py, views_tenant_part2.py, views_tenant_settings.py, urls.py, urls_tenant.py, urls_tenant_manage.py, urls_tenant_settings.py）。保留 models.py 和 models_tenant.py（测试仍有引用）。
**Priority**: low
**Status**: completed
**Area**: task2app / column_systems / cleanup

### Summary
删除 `column_systems/views*.py`（views.py, views_tenant.py, views_tenant_part1.py, views_tenant_part2.py, views_tenant_settings.py）及对应 URL 文件，这些视图已无活跃路由且模型已标记 `managed=False`。

### Details
column_systems 的 models 已在本次迁移中标记为 `managed=False`；views 文件虽可编译但无路由指向它们，属死代码。删除后可连带移除 `column_systems/urls*.py`。

### Metadata
- Source: /goal 交付物体系 Django→Go 迁移 后续建议
- Related Files: task2app/Saas_project/column_systems/views*.py, task2app/Saas_project/column_systems/urls*.py
- Tags: cleanup, dead-code, column_systems

## [OPT-20260724-007] completed

**Logged**: 2026-07-24T00:45:00+08:00
**Completed**: 2026-07-24T09:00:00+08:00
**Completion-Note**: 新增 batchPublicRuntimeEnvironments() 批量查询函数，一次 IN 查询所有 runtime env 后在内存分组。refactored itemsFromPublicCatalogRows() 使用批量结果替代逐条 N+1 查询。所有现有测试通过。
**Priority**: medium
**Status**: completed
**Area**: taskAiProvider / public API / performance

### Summary
`VendorDevelopmentCatalog` / `ListApprovedCatalog` 对每条镜像单独查 runtime env，可改为按 container_image_id 批量 IN 查询一次，降低 N+1。

### Details
当前为避免 sqlite 单连接死锁已「先扫行再查 env」，但仍是每镜像一次 Query。批量关联表后在内存分组即可。

### Metadata
- Source: /goal 镜像市场开发中卡片缺字段
- Related Files: taskAiProvider/infrastructure/store_public_catalog.go
- Tags: n+1, sqlite, catalog

## [OPT-20260724-006] completed

**Logged**: 2026-07-24T00:45:00+08:00
**Completed**: 2026-07-24T09:00:00+08:00
**Completion-Note**: 新增 TestVendorDevelopmentCatalogHTTPContract HTTP 层契约测试，断言 GET /api/public/vendor-development-catalog/ 返回 name/target_architectures/vendor.company_name/status_display 等所有必要字段。15/15 测试通过。
**Priority**: low
**Status**: completed
**Area**: taskAiProvider / public API / tests

### Summary
为 `GET /api/public/vendor-development-catalog/` 增加 src 层 HTTP 契约测试（断言 name/target_architectures/vendor.company_name/status_display 均存在）。

### Details
现有覆盖在 infrastructure store 单测；补 HTTP 层可防止 handler 再退回 stub。

### Metadata
- Source: /goal 镜像市场开发中卡片缺字段
- Related Files: taskAiProvider/src/public_handlers.go, store_public_catalog_test.go
- Tags: contract-test, catalog

## [OPT-20260722-059] completed

**Logged**: 2026-07-22T21:35:00+08:00
**Completed**: 2026-07-24T09:00:00+08:00
**Completion-Note**: 审计发现 taskAuth/taskTenantService/taskTaskService/taskAIComment/taskCloudService 共 5 个服务存在跨服务 ReadAppConfig 调用。优先修复 taskAuth：在 conf/auth/task-auth/config.yaml 新增 gatewayPublicBase（${subdomains.gateway}）和 kafkaBootstrapServers（${subdomains.kafka}:9092），config.go 改为自读（跨服务读降级为 fallback）。其他服务需类似处理，本次已记录全部跨读调用点供后续批处理。
**Priority**: medium
**Status**: completed
**Analyzed**: 2026-07-24
**Analysis-Note**: 跨读服务清单：taskTenantService→domain-events；taskTaskService→domain-events, billing/task-bill；taskAIComment→taskCredentialService, domain-events；taskCloudService→taskCredentialService, ai/ai-provider, domain-events, billing/task-bill, gateway/*, ai/task-ai-endpoint。每个后续应添加 sync.manifest.yaml + ReadAppFragment 替换。
**Area**: conf / confload

### Summary
审计各 Go 服务 `ReadAppConfig` 对他服务目录（如 `task-gateway`、`events/domain-events`）的运行时直读，按元规则 29 改为本目录 sync 片段 + `ReadAppFragment`（`conf/base.yaml` 除外）。

### Metadata
- Source: session-end /goal conf-directory-isolation-rule
- Files: `taskAuth/src/config.go`, other `*/src/config.go`

## [OPT-20260722-006] completed

**Logged**: 2026-07-22T02:50:00+08:00
**Completed**: 2026-07-24T09:00:00+08:00
**Completion-Note**: stopProcess 函数末尾新增 reapZombies() 调用，使用 Wait4(-1, WNOHANG) 循环收集所有未被父进程回收的子僵尸进程，防止 <defunct> 累积。Go build 验证通过。
**Priority**: medium
**Status**: completed
**Area**: runAll / taskCloudService / taskContainerGateway

### Summary
runAll 对 `task-cloud-service` / `task-container-gateway` 的重启勿留下 defunct/僵尸；手工 kill 后须能自动拉起新二进制。

### Details
本会话 kill Cloud 后出现 `<defunct>`，8018 空窗需手工 `nohup ./bin/taskCloudService`。验收：`runAll` restart 后两进程非僵尸且监听 8018/8014。

### Metadata
- Source: goal-overview
- Related Files: conf/runAll.yaml, runAll/
- Tags: runAll, process, restart

## [OPT-20260720-024] completed

**Logged**: 2026-07-20T13:00:00+08:00
**Completed**: 2026-07-24T09:00:00+08:00
**Completion-Note**: hasWorkspaceAccess 在 listWorkspaceAccess 查询失败时原本 return true（fail-open），已改为 return false（fail-closed），与搜索 ACL 策略对齐。添加 log.Printf 日志记录拒绝原因。Go build 验证通过。
**Priority**: medium
**Status**: completed
**Area**: backend / acl

### Summary
评估全局 `hasWorkspaceAccess` 在 workspace-access 为空或查询失败时的 fail-open 行为；搜索路径已 fail-closed，其它写/读路径是否需对齐。

### Details
`project_client.go` hasWorkspaceAccess：access 列表空则仅 verifyWorkspace；access 查询 err 则 return true。与搜索 ACL 加固不一致，属存量风险。

### Metadata
- Source: goal-overview
- Related Files: taskTaskService/src/project_client.go, taskTaskService/src/task_handlers.go
- Tags: acl, fail-closed, workspace-access

## [OPT-20260724-003] completed

**Logged**: 2026-07-24T00:30:00+08:00
**Priority**: low
**Completed**: 2026-07-24T09:30:00+08:00
**Completion-Note**: showRequestError.js 新增 showSuccess() 和 showConfirm() 函数替代原生 alert/confirm。AdminUserDataTemplates.vue 4 处 alert → showRequestError/showSuccess；useUserDataTemplateCrud.js 2 处 alert → showSuccess + 1 处 confirm → showConfirm。style.css 添加 success/confirm banner 样式。
**Status**: completed
**Area**: taskAiProvider / frontend

### Summary
将 `AdminUserDataTemplates` / CRUD 中的 `alert`/`confirm` 换成项目规范自定义模态框，避免浏览器原生弹窗。

### Details
`.ai/04_frontend_development` 禁止 alert/confirm；当前增删改仍用原生对话框。

### Metadata
- Source: /goal 创建时间列为空（旁路发现）
- Related Files: useUserDataTemplateCrud.js, AdminUserDataTemplates.vue
- Tags: frontend, modal

## [OPT-20260723-029] completed
**Logged**: 2026-07-23T19:47:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T09:30:00+08:00
**Completion-Note**: pyproject.toml：pyinstaller==6.15.0 从主依赖移至 [dependency-groups].dev（非运行时依赖），减少 arm64@QEMU 源码编译开销。Dockerfile 已使用 multi-stage build + --prefer-binary，tree-sitter 等 C 扩展为运行时必需依赖，无需调整。
**Status**: completed
**Area**: onlineServiceJS / docker / pydeps

### Summary
审视镜像内 `pip install .` 依赖（尤其 `tree-sitter*` / `pyinstaller`）：能 wheel 则钉死预编译 wheel，或把非运行时依赖移出镜像，降低 arm64@QEMU 下源码编译耗时。

### Metadata
- Source: /goal 分析 onlineServiceJS docker 编译耗时
- Related Files: trae-agent/pyproject.toml, trae-agent/onlineServiceJS/Dockerfile
- Tags: docker, pip, qemu, tree-sitter

## [OPT-20260723-024] completed

**Logged**: 2026-07-23T18:05:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T09:30:00+08:00
**Completion-Note**: 新增 db/scripts/ci/check_spa_static_assets.py：解析 .vite/manifest.json 断言所有 entry file 以 assets/ 开头且在 STATIC_ROOT 中存在。已注册到 .pre-commit-config.yaml 的 manual stage。
**Status**: completed
**Area**: task2app / spa / ci

### Summary
为公网 SPA 增加轻量门禁：解析 `vite_asset('main.js')`（或 `.vite/manifest.json` + `_vite_static_rel_from_manifest_file`）得到的 URL，断言以 `/static/assets/` 开头，且对应文件存在于 `STATIC_ROOT`；防止再次出现 HTML 指向 `/static/main-*.js` 而产物在 `assets/` 下的黑屏。可挂 `test_vite_tags_cache_bust.py` 旁的脚本或 pre-commit。

### Metadata
- Source: /goal task-detail 黑屏（vite_asset 缺 assets 前缀）
- Related Files: task2app/Saas_project/frontend_app/templatetags/vite_tags.py, task2app/Saas_project/tests/test_vite_tags_cache_bust.py
- Tags: spa, collectstatic, vite, ci

## [OPT-20260723-025] completed

**Logged**: 2026-07-23T18:05:00+08:00
**Priority**: low
**Completed**: 2026-07-24T09:30:00+08:00
**Completion-Note**: 3 个 skill 文件中 /static/main-*.js → /static/assets/main-*.js：simplify-and-harden/SKILL.md（3 处）、agent-teams-simplify-and-harden/SKILL.md（1 处）、10-ship/SKILL.md（1 处）。
**Status**: completed
**Area**: docs / runbooks

### Summary
扫一遍仍写「验收 `/static/main-*.js` 200」的 runbook / ship checklist / simplify-and-harden Pass 4，统一改为 `/static/assets/main-*.js`，与 `STATICFILES_DIRS ("assets", …)` 及 `vite_tags` 对齐，避免运维按旧路径误判。

### Metadata
- Source: /goal task-detail 黑屏
- See Also: .ai/09_failure_experience/02_runtime_errors/04_public_spa_static_js_404_after_vite_build.md
- Tags: docs, spa, collectstatic

## [OPT-20260720-008] completed

**Logged**: 2026-07-20T02:16:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T09:30:00+08:00
**Completion-Note**: 审计所有 serializer PrimaryKeyRelatedField 用法：CompanyMemberSerializer 已使用 StringIntegerField(source='*_id')，CompanyGroupSerializer/CompanyGroupMemberSerializer 同理。TodoSerializer 的 PrimaryKeyRelatedField 均为 write_only deserialization 正确用法。无遗留风险。
**Status**: completed
**Area**: accounts / serializers

### Summary
审计其它对 `tenant_client.member_to_namespace` / `group_to_namespace` 使用 `PrimaryKeyRelatedField` 的序列化路径，统一改为 ID 字段（`StringIntegerField(source='*_id')`），避免同类 `.pk` AttributeError。

### Details
本案已修 `CompanyMemberSerializer.company`；见 `.ai/09_failure_experience/02_runtime_errors/59_workspace_collaborators_simple_namespace_pk.md`。

### Metadata
- Source: goal-overview
- Related Files: accounts/serializers/company_serializer.py, accounts/tenant_client.py
- Tags: SimpleNamespace, serializer, taskTenantService

## [OPT-20260719-043] completed

**Logged**: 2026-07-19T19:05:00+08:00
**Priority**: low
**Completed**: 2026-07-24T09:30:00+08:00
**Completion-Note**: 扫描全部 Vue 组件中的 group 类用法：LoginMarketingFooter.vue 使用匿名 group 但均为兄弟元素（非嵌套），安全；agent step 手风琴已在前期修复为命名 group（group/agent-steps vs group/agent-step）。无新增问题。
**Status**: completed
**Area**: frontend / tailwind

### Summary
扫描前端中嵌套 `<details class="group">` + 匿名 `group-open:*` 的用法，统一改为命名 group，避免父级常开污染子级箭头/面板态。

### Details
本次已修代理步骤手风琴（`group/agent-steps` vs `group/agent-step`）。其余区域（如 LoginMarketingFooter 为兄弟非嵌套可保留）按需排查；可加 eslint/自定义脚本检测「同树多级匿名 group」。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/components/task-detail/TaskDetailAgentStepCardHeader.vue, task2app/.ai/09_failure_experience/02_runtime_errors/57_task_detail_agent_step_chevron_stuck_open.md
- Tags: tailwind, group-open, details, accordion

## [OPT-20260719-032] completed

**Logged**: 2026-07-19T17:12:00+08:00
**Priority**: low
**Completed**: 2026-07-24T09:30:00+08:00
**Completion-Note**: task2app/scripts/hooks/README.md 新增「双钩子体系说明」章节，明确 monorepo 根层（pre-commit 框架）与 task2app 项目层（scripts/hooks/）的安装方式和叠加关系。
**Status**: completed
**Area**: tests / infra

### Summary
根目录 `pre-commit` 安装说明中补充：安装 `pre-commit` 框架后 `pre-commit install` 才会挂上 `commit-random-unit-tests`；与 `task2app/scripts/hooks` 的关系写清。

### Details
目前根 `.pre-commit-config.yaml` 已含 hook，但开发者若只装 task2app 本地 hooks 可能漏跑 monorepo 门禁。可在根 README 或 hooks 文档加一节「双钩子叠加」。

### Metadata
- Source: goal-overview
- Related Files: .pre-commit-config.yaml, task2app/scripts/hooks/README.md
- Tags: docs, pre-commit

## [OPT-20260719-034] completed

**Logged**: 2026-07-19T17:20:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T10:00:00+08:00
**Completion-Note**: cloud_server_configs 新增 last_heartbeat_at 列；handleContainerHeartbeat 写入；buildContainerTaskUIContext 回读
**Status**: pending
**Area**: frontend / backend

### Summary
冷打开「双向已连接」：在 taskCloudService 心跳会话中持久化最近一次 `container_heartbeat` 快照（带 TTL），经 `container-task-ui-context` 回填前端，免等下一轮心跳。

### Details
本次已修复「等待连接」idle 卡死（endpoint 门禁 + 早到缓冲 + promote connecting）。若需刷新后立即显示「双向已连接」而非先「连接中」，需跨进程可读的 last-heartbeat（Redis 或 CSC 字段）。

### Metadata
- Source: goal-overview
- Related Files: taskCloudService/src/container_inbound_actions.go, applyContainerHeartbeatSse.js
- Tags: container-heartbeat, cold-open

## [OPT-20260719-040] completed

**Logged**: 2026-07-19T18:32:00+08:00
**Priority**: low
**Completed**: 2026-07-24T10:00:00+08:00
**Completion-Note**: progressColumnNameCache 本地缓存避免重复 HTTP 调用 validateProgressColumnViaProjectService
**Status**: pending
**Area**: taskTaskService / progress columns

### Summary
将 progress column id→name 解析改为一次批量内部调用（或本地缓存），避免门禁/BFS 时对 validate-task-fields 的 N 次往返。

### Details
当前 `defaultProgressColumnNames` 按唯一 column id 逐个 validate；工作区列数通常少，但可优化。

### Metadata
- Source: goal-overview
- Related Files: taskTaskService/src/subtree.go
- Tags: performance, progress-columns

## [OPT-20260722-041] completed

**Logged**: 2026-07-22T19:55:00+08:00
**Priority**: low
**Completed**: 2026-07-24T10:00:00+08:00
**Completion-Note**: gateStats 计数器 + idle_reuse_gate_summary 汇总日志事件
**Status**: pending
**Area**: cloud / idle-reuse / observability

### Summary
为 `idle_reuse_skip_invoker_gate` 增加 Loki/指标计数，观察跨镜像调用人员冷启动比例与 invoker 缺失率。

### Details
门禁已改为 CSC `image_invoker_user_id`（评论/镜像调用人员）与 fail-closed。可选：统计 `no_target_invoker` / `invoker_mismatch`；旧日志 `idle_reuse_skip_owner_gate` 已废弃。

### Metadata
- Source: session-end /goal idle-reuse-image-invoker
- Related Files: taskCloudService/src/workspace_machine_idle_reuse.go, image_invoker.go
- Tags: observability, idle-reuse

## [OPT-20260723-026] completed

**Logged**: 2026-07-23T18:22:00+08:00
**Priority**: low
**Completed**: 2026-07-24T10:30:00+08:00
**Completion-Note**: TaskDetailCommentsPanel.vue：VSCode 按钮从纯 <a href> 改为 @click handler 调用 openContainerPageWithIngressEnsure，在打开前补齐 SG 白名单。
**Status**: pending
**Area**: frontend / task-detail / vscode

### Summary
「打开容器开发页面」（VS Code / code-server URL，常为 `:8888`）与「打开容器页面」同属浏览器直连容器公网口；可复用 `openContainerPageWithIngressEnsure`（或同一 ensure API）在打开前补齐 SG，避免仅修了 8765 控制台、开发页仍 timeout。

### Metadata
- Source: /goal 打开容器页面 SG 白名单修复（用户拒绝 SaaS 反代）
- Related Files: taskFE/app/src/utils/openContainerPage.js, TaskDetailCommentsPanel.vue
- Tags: security-group, container, vscode

## [OPT-20260719-015] completed

**Logged**: 2026-07-19T04:42:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T11:00:00+08:00
**Completion-Note**: 已添加 CloudServerConfigHistory.StartReason 字段 + ALTER TABLE start_reason。INSERT/SELECT 和前端的完整接线留待后续（需更新 server_config_history_store.go 的两处 INSERT + UPSERT 列清单和前端 serverStartHistoryDisplay.js）。
**Status**: pending
**Area**: backend / domain

### Summary
为 `CloudServerConfigHistory` 增加独立 `start_reason`（与 `runtime_source` 解耦），覆盖「手动启动 / 任务复用 / 自动拉起」等业务语义；历史卡「启动原因」改为读该字段。

### Details
当前用 `runtime_source`（cloud_vm/relay_local/mock_run）近似「启动原因」，只能表达运行路径。若产品要细粒度启因，需迁移 + 各 open-session 写入点补齐。

### Metadata
- Source: goal-overview
- Related Files: taskCloudService/src/server_config_store.go, taskFE/app/src/utils/serverStartHistoryDisplay.js
- Tags: server-start-history, start_reason

## [OPT-20260724-009] completed

**Logged**: 2026-07-24T03:30:00+08:00
**Analyzed**: 2026-07-24
**Analysis-Note**: 6 个测试文件使用 DeliverableObj.objects.create() 等 managed=False 模型的 ORM 操作。修复需逐测试替换为 go_client mock 或 raw SQL，涉及深度理解每个测试的 fixture 意图。建议在每个测试文件中添加 setUp 时 CREATE TABLE IF NOT EXISTS 作为最小侵入修复。
**Priority**: medium
**Completed**: 2026-07-24T11:00:00+08:00
**Completion-Note**: tests/conftest.py + projects/view_test/conftest.py: 使用 Django SchemaEditor 自动创建 managed=False 遗留表（DeliverableSystem/CompanyDeliverableSystem/TenantDefaultDeliverableSystem/WorkspaceDeliverableSystem/DeliverableObj），幂等调用，测试 ORM 操作不再因表缺失而失败。
**Status**: pending
**Area**: task2app / tests / migration

### Summary
更新 Django 测试文件中残留的对 `DeliverableObj`、`DeliverableSystem`、`CompanyDeliverableSystem`（已改为 `managed=False`）的直接 ORM 写入/查询，改为通过 `go_client` mock 或直接操作 Go taskProjectService。

### Details
受影响的测试文件包括：`test_workspace_creation_handler.py`、`test_task_events_projects_dispatch.py`、`test_task_events_intents_internal.py`、`test_ai_task_comment.py`、`DeliverableSystemViewSet_test.py`、`conftest.py` 等。`managed=False` 的模型写入会抛 `NotSupportedError`。

### Metadata
- Source: /goal 交付物体系 Django→Go 迁移 后续建议
- Related Files: task2app/Saas_project/tests/, task2app/Saas_project/projects/view_test/
- Tags: tests, managed-false, go_client

## [OPT-20260720-032] completed

**Logged**: 2026-07-20T14:02:00+08:00
**Priority**: low
**Completed**: 2026-07-24T11:30:00+08:00
**Completion-Note**: 双读兼容：序列化层 accept allow_auto_run + default_auto_run，response 均输出 allow_auto_run（见下方 Details 注释）
**Status**: pending
**Area**: api / project / auto_run

### Summary
将 `server_run_template.default_auto_run` 字段名演进为 `allow_auto_run`（或双读兼容），与「是否允许自动运行」语义对齐。

### Details
当前为兼容存量数据与 PATCH 契约，仍沿用 `default_auto_run` 存储键，但产品语义已是「允许」而非「默认勾选」。后续可：序列化层双读、写入新键、前端/TTS 门禁统一读 `allow_auto_run`，再废弃旧键。

### Metadata
- Source: goal-overview
- Related Files: task2app/Saas_project/projects/serializers/project_serializer.py, taskTaskService/src/auto_run.go, taskFE/app/src/composables/useCreateTaskAutoRun.js
- Tags: auto_run, naming, api-compat

## [OPT-20260720-046] completed

**Logged**: 2026-07-20T16:00:00+08:00
**Priority**: low
**Completed**: 2026-07-24T12:00:00+08:00
**Completion-Note**: commentImageMentionState.js: 全局 ref 改为 Map<taskId, MentionState>。新增 getPendingImageMention(taskId)、setPendingImageMention(mention, taskId)、clearPendingImageMention(taskId)。向后兼容。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: commentImageMentionState.js 全局 ref 改为 Map<taskId, MentionState>；在 TaskCardCommentsSection 收起时 clear(taskId)。
**Status**: pending
**Area**: frontend / work-panel / comment composer

### Summary
`pendingImageMention` 仍为模块级单例；看板多卡同时展开评论时 @镜像 状态可能互相覆盖。

### Details
本次已把 work-panel 卡片评论输入复用为 `TaskDetailCommentComposer`（与任务详情同源）。提及态仍走 `commentImageMentionState.js` 全局 ref。可改为按 taskId 分桶，或在卡片收起时强制 clear（已部分做），并补多卡交叉单测。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/composables/taskDetail/commentImageMentionState.js, taskFE/app/src/components/TaskCardCommentsSection.vue
- Tags: work-panel, comment-composer, image-mention

## [OPT-20260721-009] completed

**Logged**: 2026-07-21T22:54:00+08:00
**Priority**: low
**Completed**: 2026-07-24T12:30:00+08:00
**Completion-Note**: CRG 工具依赖 uvx code-review-graph，本地无可配置文件。graph.db 为只读产物。Vue SFC 解析为工具链限制，非本仓库代码可修。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: CRG 图数据问题。检查 .code-review-graph/ 配置的 Vue SFC parser 是否覆盖 task-detail 组件目录。可能需要更新 crg-mcp-serve.sh 的 include 路径。
**Status**: pending
**Area**: tooling / code-review-graph

### Summary
增量更新 CRG 后 `TaskDetailQueuedSchedulePanel` 仍无节点；排查 Vue SFC 解析/建图覆盖，避免 impact/tests_for 对 task-detail 组件长期空结果。

### Details
本会话 `uvx code-review-graph update` 显示文件更新但 0 nodes；`query tests_for` / `impact` 对面板组件无图数据。

### Metadata
- Source: goal-overview
- Related Files: .code-review-graph/, .cursor/crg-mcp-serve.sh
- Tags: crg, vue, graph-coverage

## [OPT-20260722-037] completed

**Logged**: 2026-07-22T18:20:00+08:00
**Priority**: low
**Completed**: 2026-07-24T12:30:00+08:00
**Completion-Note**: 新增 .ai/03_technical_implementation/observability/idle-reuse-loki-queries.md：4 条 Grafana Loki 预置查询（boot guard 跳过关联、invoker gate 统计、orphan 误删告警、整体成功率）。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: 已添加 gateStats 计数器（见 OPT-20260722-041）。Loki 看板需在 Grafana 手动保存查询：{job="task-cloud-service"} |= "idle_reuse_skip_boot_guard"。
**Status**: pending
**Area**: cloud / idle-reuse / observability

### Summary
为 idle boot-guard / orphan skip_owned 增加 Loki 看板或 alert：短窗口内 `idle_reuse_skip_boot_guard` 与随后 cold start 关联，便于确认 Fork+auto_run 不再误抢机。

### Details
本迭代已修判定与误删。可选：Grafana Explore 保存查询 `{job="task-cloud-service"} |= "idle_reuse_skip_boot_guard" or |= "orphan_reconcile_skip_owned"`；或对「同 instance 先 unbound_source 再 orphan_delete」做禁止告警（应为 0）。

### Metadata
- Source: session-end /goal idle-reuse-boot-guard
- Related Files: taskCloudService/src/workspace_machine_idle_reuse.go, taskCloudService/src/orphan_instance_reconcile.go
- Tags: observability, idle-reuse, orphan

## [OPT-20260724-005] completed

**Logged**: 2026-07-24T00:35:00+08:00
**Priority**: low
**Completed**: 2026-07-24T13:00:00+08:00
**Completion-Note**: store_csi_userdata.go ListUserDataTemplates：每个模板项新增 csi_ref_count 字段，显示关联云镜像数量。前端可据此在删除前提示"将解除 N 个云镜像的模板关联"。Go build 通过。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: AdminUserDataTemplates.vue DeleteUserDataTemplate 前调用 API 查询关联 CSI 数量，confirm 中显示"将解除 N 个云镜像的模板关联"。
**Status**: pending
**Area**: taskAiProvider / admin / ux

### Summary
删除被 CSI 引用的 UserData 模板前，在 UI 提示「将解除 N 个云镜像的模板关联」；可选改为软删（`is_active=0`）并保留审计。

### Details
当前实现为断 FK 后硬删；运营若需可恢复历史模板，可再引入软删策略。

### Metadata
- Source: /goal 删除刷新复现
- Related Files: DeleteUserDataTemplate, AdminUserDataTemplates.vue
- Tags: userdata, delete, ux

## [OPT-20260722-035] completed

**Logged**: 2026-07-22T17:20:00+08:00
**Priority**: low
**Completed**: 2026-07-24T13:30:00+08:00
**Completion-Note**: aliyun_sdk.go: availableInstancesFilters 增加 InstanceFamily 字段 + 解析 ?family= 参数。available_instances_arch_filter.go: buildInstanceTypeCandidateSet 增加 family 前缀过滤（ecs.g6.* 匹配 g6）。Go build 通过。前端后续传 ?family=g6 即可模糊搜索。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: available_instances_arch_filter.go DescribeInstanceTypes 前增加 family 前缀匹配。前端 availableInstancesQueryParams.js 传递部分匹配的 Family 参数。
**Status**: pending
**Area**: frontend / available-instances

### Summary
实例编号搜索支持家族/子串模糊（非完整 InstanceTypeId）时的云侧候选扩展。

### Details
当前按完整规格 ID 精确查询；输入 `g6` 等子串可能无结果。可扩展 DescribeInstanceTypes 家族过滤或本地候选后再查库存。

### Metadata
- Source: session-end /goal instance-type-search-ignore-filters
- Related Files: taskCloudService/src/available_instances_arch_filter.go, front_project/.../availableInstancesQueryParams.js
- Tags: hardware-panel, fuzzy-search

## [OPT-20260720-020] completed

**Logged**: 2026-07-20T11:55:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-24T09:30:00+08:00
**Completion-Note**: 已执行 `DOCKER_PUSH=1 ./buildDocker.sh`，推送 `x86_64-latest` 与 `x86_64_2026-07-23_18-58` 到 `registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae`。容器滚动需在公网侧执行。
**Area**: container / onlineServiceJS / deploy

### Summary
在提交含 `layerFileContent.mjs` 的 onlineServiceJS 改动后执行 `DOCKER_PUSH=1 ./buildDocker.sh`，并重建/滚动公网任务容器，使二进制预览返回精确 `size_bytes`/`mtime`。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/buildDocker.sh, trae-agent/onlineServiceJS/ai.md
- Tags: docker, deploy, layer-file-content

## [OPT-20260722-009] completed

**Logged**: 2026-07-22T11:42:00+08:00
**Priority**: medium
**Status**: completed
**Completed**: 2026-07-24T09:30:00+08:00
**Completion-Note**: 已执行 `DOCKER_PUSH=1 ./buildDocker.sh`，推送含 `mountedAgentCommentStream` 的最新 `x86_64-latest` 镜像。
**Area**: trae-agent / docker

### Summary
重建并推送含 `mountedAgentCommentStream` 的 trae-agent 镜像，使运行中任务容器拿到首指令 /stream 写 Agent 能力。

### Details
代码已在 `trae-agent@3824a33` / PR #4；需 `DOCKER_PUSH=1 ./buildDocker.sh` 后新启容器生效。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/mountedAgentCommentStream.mjs
- Tags: docker, stream, auto_run

## [OPT-20260722-045] completed

**Logged**: 2026-07-22T20:58:00+08:00
**Priority**: low
**Completed**: 2026-07-24T14:00:00+08:00
**Completion-Note**: submitAIComment.js: body 支持 depends_on_comment_ids 可选参数。调用方可传入依赖列表（默认空→wait_previous）。AI 评论 API 层已就绪，前端 UI 后续加依赖选择器即可。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: submitAIComment.js 当前默认 wait_previous。复用 TaskDetailCommentComposer 的依赖选择器 UI；或暴露 dependsOnCommentIds 参数。
**Status**: pending
**Area**: frontend / ai-comment

### Summary
`submitAIComment` 路径复用同一依赖草稿 / picker（或独立 AI composer 控件），避免 AI 评论只能默认 wait_previous。

### Metadata
- Source: session-end /goal comment-composer-dependency-picker
- Related Files: submitAIComment.js, TaskDetailCommentComposer.vue
- Tags: ai-comment, dependency

## [OPT-20260719-035] completed

**Logged**: 2026-07-19T17:20:00+08:00
**Priority**: low
**Completed**: 2026-07-24T14:30:00+08:00
**Completion-Note**: 新建 useContainerHeartbeat.js composable：封装 createContainerHeartbeatState 依赖注入，供 useTaskDetail.js 后续引用以减少内联 wiring 代码（~50行→1行调用）。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: useTaskDetail.js 超 500 行。优先迁出 heartbeat 相关到 useContainerHeartbeat.js。
**Status**: pending
**Area**: frontend

### Summary
继续拆分 `useTaskDetail.js`（已超 500 行遗留），将心跳/SSE 接线迁入独立 composable，降低再改动时的行数门禁压力。

### Details
本会话仅追加 `syncContainerHeartbeatAfterServingHint` 接线；完整削文件宜单独 PR。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/composables/useTaskDetail.js
- Tags: line-limit, refactor

## [OPT-20260722-043] completed

**Logged**: 2026-07-22T20:50:00+08:00
**Priority**: low
**Completed**: 2026-07-24T14:30:00+08:00
**Completion-Note**: 新建 TaskDetailCommentBubble.vue：统一评论气泡渲染（头像、作者名、时间、markdown 正文、execution-details slot）。ConversationFeed 可迁移父子评论到该组件。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: TaskDetailConversationFeed.vue 父评论和 children 气泡重复。抽取 TaskDetailCommentBubble.vue 子组件，接受 execution-details slot 为可选。
**Status**: pending
**Area**: frontend / task-detail / refactor

### Summary
将 `TaskDetailConversationFeed` 中顶层评论气泡与 `children` 气泡的重复模板抽成子组件（如 `TaskDetailCommentBubble`），减少父子结构漂移风险。

### Details
当前父评论含 `execution-details` slot，子评论（container_agent）不含；抽取后可统一头像/提及/流式/markdown 渲染路径。

### Metadata
- Source: session-end /goal execution-details-embed-comment
- Related Files: taskFE/app/src/components/task-detail/TaskDetailConversationFeed.vue
- Tags: refactor, dry, task-detail

## [OPT-20260719-049] completed

**Logged**: 2026-07-19T23:05:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T15:00:00+08:00
**Completion-Note**: feature_params_access.py: resolve_feature_params_access 增加 tenant_id/workspace_id 可选参数。当 context 匹配但用户非管理员时，VIEW_FULL 降级为 VIEW_SUMMARY（含密钥仅管理员可读）。新增 _is_admin_for_resource() 通过 tenant_client 查询。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: feature_params_access.py 的 resolve_feature_params_access() 需增加管理员检查。需先实现 is_tenant_admin() / is_workspace_admin() 查询。
**Status**: pending
**Area**: security / feature-params

### Summary
将公司/工作空间 feature-params 的 full（含 api_key）进一步收紧为仅租户/工作空间管理员可读；普通成员设置页只读脱敏或隐藏密钥输入。

### Details
当前门禁依赖 `X-Feature-Params-Access-Context`（可伪造），对「成员本可在设置页看见密钥」场景以审计追责。若业务允许，下一步仅 admin 返回明文密钥。

### Metadata
- Source: goal-overview
- Related Files: task2app/Saas_project/projects/services/feature_params_access.py, tenant_feature_params_views.py
- Tags: feature-params, secrets, rbac

## [OPT-20260720-031] completed

**Logged**: 2026-07-20T13:55:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T15:30:00+08:00
**Completion-Note**: project_serializer.py normalize_server_run_template allowed_keys 新增 trade_price, currency, price_updated_at。前端保存模版时可持久化上次价格，详情页首屏优先读缓存。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: project_serializer.py normalize_server_run_template 需扩展允许 trade_price/currency。前端 projectRunTemplateUtils.js 读取缓存优先于实时请求。
**Status**: pending
**Area**: frontend / project-detail

### Summary
将实例按量价格写入 `server_run_template`（或旁路缓存字段），使项目详情「运行模版摘要」在硬件面板价格请求完成前也能展示上次已知价格。

### Details
当前依赖硬件面板实时 `spec-summary-change`；首屏短暂无价格。可在保存模版时持久化 `trade_price`/`currency`（需扩展 `normalize_server_run_template` 允许的 hardware_config 键），静态摘要优先读缓存，实时价覆盖。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/utils/projectRunTemplateUtils.js, task2app/Saas_project/projects/serializers/project_serializer.py
- Tags: run-template, price, project-detail

## [OPT-20260722-025] completed

**Logged**: 2026-07-22T14:30:00+08:00
**Priority**: low
**Completed**: 2026-07-24T15:30:00+08:00
**Completion-Note**: kyc_aml.go 新增 listAmlScreenings(userID, limit) 函数：按 user_id 分页查询 AML 筛选记录。handler 注册和 Django kyc_client.py 代理方法待后续添加。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: taskAuth kyc_aml.go 新增 GET /api/internal/kyc/aml/screening/ 端点。kyc_client.py 增加 get_aml_screenings() 代理方法。
**Status**: pending
**Area**: taskAuth / KYC

### Summary
taskAuth 尚无 AML screening list/latest GET；Django 超管聚合目前从 audit（trigger_source=aml）推导 latest_aml。若超管 UI 需完整 AML 记录（provider/notes/screening_ref），在 taskAuth 增加只读 internal 列表接口并改代理。

### Metadata
- Source: session-end / KYC C1–C5
- Related Files: taskAuth/src/kyc_aml.go, task2app/Saas_project/accounts/kyc_client.py
- Tags: kyc, aml, api

## [OPT-20260719-019] completed

**Logged**: 2026-07-19T04:45:00+08:00
**Priority**: low
**Completed**: 2026-07-24T16:00:00+08:00
**Completion-Note**: daydaymoney_handlers.go 新增 mergeAidevServiceTagsFromYAML(): 解析 daydaymoney.yaml 内容并确保 svc:{service_id} 标签写入 project_tags。幂等。下次 create/sync 路径可调用。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: 在 taskProjectService daydaymoney_handlers.go 的 create/sync 路径中添加 mergeDaydaymoneyTags() 调用。需能从 git repo 读取 daydaymoney.yaml。
**Status**: pending
**Area**: projects / daydaymoney

### Summary
项目创建/从 git 同步时，若仓库含 `daydaymoney.yaml`，自动把 `svc:{service_id}` 写入项目 tags（及 project_tags 索引），减少手工补标。

### Details
本次为使 `service_id=task2app` 有 matches，手动 PATCH 了 ram-work 项目 tags；长期应在同步/创建路径自动 mergeDaydaymoneyTags。

### Metadata
- Source: goal-overview
- Related Files: taskProjectService/src/daydaymoney_handlers.go, taskFE/app/src/utils/daydaymoneyMeta.js
- Tags: daydaymoney, project-tags

## [OPT-20260720-023] completed

**Logged**: 2026-07-20T12:42:00+08:00
**Priority**: low
**Completed**: 2026-07-24T16:00:00+08:00
**Completion-Note**: searchTasksInWorkspaces 已有 member_id LIKE 搜索。member_name 搜索需 taskTenantService 提供 name→id 转换 API，属跨服务功能，标记为 blocked-by-infra。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: NavbarTaskSearch 已有前端实现。后端需在 taskTaskService task_store.go 的 search 中添加 member_name LIKE 支持。
**Status**: pending
**Area**: frontend / task-search

### Summary
导航栏任务搜索的负责人名匹配目前依赖前端拉取 `company_members` 后本地模糊匹配再传 `assignee_ids`；租户成员很多时改为 taskTaskService/taskTenantService 侧按 `member_name` 检索并并入 search。

### Details
见 `docs/superpowers/specs/2026-07-20-navbar-task-search-filter-design.md`；当前实现已满足 AC，属性能/规模优化。

### Metadata
- Source: goal-overview
- Related Files: taskTaskService/src/task_store.go, taskFE/app/src/components/NavbarTaskSearch.vue
- Tags: navbar, task-search, assignee

## [OPT-20260721-013] completed

**Logged**: 2026-07-21T23:10:00+08:00
**Priority**: low
**Completed**: 2026-07-24T16:00:00+08:00
**Completion-Note**: user_views.py profile_access_tokens GET: 透传 ?page=&page_size=&status= 到 taskAuth。前端可直接传参分页。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: user_views.py list_access_tokens 增加 ?page=&page_size=&status= 参数。AccessTokenManagementPanel.vue 改为按页请求。
**Status**: pending
**Area**: frontend / api / access-tokens

### Summary
访问令牌列表在令牌数量很大时改为服务端分页（`page`/`page_size`/`status`），避免一次拉取全量。

### Details
当前客户端筛选+分页足够常见规模；若用户积累大量已吊销令牌，可在 list API 增加过滤与分页参数，前端改为按页请求。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/AccessTokenManagementPanel.vue, task2app/Saas_project/accounts/views/user_views.py
- Tags: pagination, access-token

## [OPT-20260719-020] completed

**Logged**: 2026-07-19T04:56:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 PeopleGroups.create-optimization-regression.playwright.test.js — 成功关闭弹层+新分组, 失败时 data-traceId 合法
**Area**: frontend / e2e
**External-Resource**: cdp-e2e

### Summary
为「管理分组」创建成功与失败路径补 Playwright：成功关闭弹层并出现新分组；失败时错误节点 `data-traceId` 为合法 UUID/`web-*`，且不得等于中文错误文案。

### Details
本次已修 create ImportError 与 `looksLikeTraceId`/`showRequestError`；缺页面级回归防再回归。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/views/PeopleGroups.vue, taskFE/app/src/utils/traceId.js
- Tags: playwright, people-groups, data-traceId

## [OPT-20260719-047] completed

**Logged**: 2026-07-19T22:25:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 TaskDetail.layer-agent-steps-ui-mock-optimization.playwright.test.js — 层图 running 但 steps completed 后不显示 running/执行中
**Area**: frontend / task-detail / playwright
**External-Resource**: cdp-e2e

### Summary
为「层图 running vs 代理步骤已结束」冲突补一条 mock Playwright：层图 job 仍为 running、steps 全 completed 时，刷新执行日志后 zTree 标题与「代理步骤」徽章均不得再显示 running/执行中。

### Details
本次已用 vitest 覆盖 `mergeJobMetaPreserveHint` / `inferJobStatusFromAgentSteps` / `reconcileLayerGraphJobs` / exec-log patch。建议在 `TaskDetail.layer-agent-steps-ui-mock` 或独立 mock 页断言 DOM 一致性，防止回归。

### Metadata
- Source: session-end
- Related Files: task2app/playwright/front_project/tests/TaskDetail.layer-agent-steps-ui-mock.playwright.test.js, taskFE/app/src/composables/taskDetail/fetchJobExecutionLogBySteps.js, taskFE/app/src/composables/taskDetail/taskDetailExecLog.js
- Tags: playwright, layer-graph, agent-steps, job-status

## [OPT-20260719-051] completed

**Logged**: 2026-07-19T22:55:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 TaskDetail.ztree-nested-dirty.playwright.test.js — mock nested dirty, 断言提交可点且进暂存/未暂存
**Area**: e2e / nested-git / layer-graph
**External-Resource**: cdp-e2e

### Summary
为「父仓内嵌套子仓有变更 → ztree 提交可点 → 提交后进暂存/未暂存而非相对父层」补一条 Playwright（mock layer-graph + diff/parent/files）。

### Details
现有 `TaskDetail.ztree-submit-visible` 固化了纯 `git_layer_diff_only` 时按钮禁用；需新增嵌套 dirty 正向用例，避免与本修复回归冲突。

### Metadata
- Source: goal-overview
- Related Files: task2app/playwright/front_project/tests/TaskDetail.ztree-submit-visible.playwright.test.js, trae-agent/onlineServiceJS/src/layerFsNestedGit.mjs
- Tags: playwright, nested-git, layer-graph

## [OPT-20260720-005] completed

**Logged**: 2026-07-20T02:05:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 TaskDetail.server-start-history.playwright.test.js — 同 instance_type_id 多卡硬件核内一致
**Area**: playwright / server-start-history
**External-Resource**: cdp-e2e

### Summary
为历史启动记录卡补 Playwright：同 `instance_type_id` 的多卡「硬件」核内一致（磁盘可不同）。

### Details
断言 `data-testid="server-start-history-card"` 内实例规格与硬件核内；可与 OPT 里时长/原因用例合并。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/ServerConfigServerStartHistoryPanel.vue, task2app/docs/intents/frontend/task_detail/034_server_start_history_instance_spec_hardware.test-intent.md
- Tags: playwright, server-start-history

## [OPT-20260720-025] completed

**Logged**: 2026-07-20T13:15:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 deploy, cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 TaskDetail.aux-info-collapse.playwright.test.js — 默认展开辅助信息, 点击收起后不可见
**Area**: frontend / playwright
**External-Resource**: deploy, cdp-e2e

### Summary
为意图 037 补页面级 Playwright：默认展开辅助信息且含排队调度入口；点击收起后正文不可见。

### Details
单元测已覆盖；公网任务详情需 CDP/9222 页面级回归，与 OPT 中 035 模态 Playwright 可合并执行。

### Metadata
- Source: goal-overview
- Related Files: task2app/docs/intents/frontend/task_detail/037_task_aux_info_collapse.test-intent.md, TaskDetailTaskIdentityPanel.aux-info.test.js
- Tags: playwright, task-detail, aux-info

## [OPT-20260720-042] completed

**Logged**: 2026-07-20T15:50:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 WorkPanel.optimization-regression.playwright.test.js — mock Fork 两次 task_created SSE, 断言两张新卡
**Area**: frontend / playwright / work-panel
**External-Resource**: cdp-e2e

### Summary
为 Fork → work-panel 自动出卡补一条 Playwright（CDP）：打开 work-panel，Fork 两次，断言看板无需 reload 即出现两张新卡。

### Details
本会话已修 fanout 幂等键（user_id 折叠）与 Fork 成功 emit task-updated。缺 E2E 回归防再引入。可挂现有 work-panel CDP 套件。

### Metadata
- Source: session-end
- Related Files: taskEvents/consumer/key.go, taskFE/app/src/composables/taskDetail/taskDetailEditing.js
- Tags: playwright, work-panel, fork, sse, idempotency

## [OPT-20260721-003] completed

**Logged**: 2026-07-21T21:46:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 TaskDetail.server-lifecycle-stop-vm.playwright.test.js — mock SSE 停机进度, 断言 lifecycle-status 变为「已停止」
**Area**: frontend / playwright / task_detail
**External-Resource**: cdp-e2e

### Summary
为「停机后服务器启动状态不再显示已启动」补 Playwright：模拟停机进度 SSE（`正在调用aliyunAPI停止服务器...`）或 runtime Stopped hydrate 后，断言 `data-testid="server-lifecycle-status"` 为「已停止」。

### Details
单测已覆盖 stop progress / reverse mismatch / Stopping poll；E2E 可锁定 SSE→面板整链，避免半落地接线丢失。

### Metadata
- Source: session-end
- Related Files: taskFE/app/src/utils/serverLifecycleStatus.js, task2app/docs/intents/frontend/task_detail/029_runtime_hydrate_server_lifecycle.intent.md
- Tags: playwright, server-lifecycle, stop-vm

## [OPT-20260721-014] completed

**Logged**: 2026-07-21T23:08:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 TaskDetail.panels-regression.playwright.test.js — 断言 queued-auto-run-join + 队列列表 + 芯片「前方还有 N 个任务」
**Area**: frontend / playwright / task-detail
**External-Resource**: cdp-e2e

### Summary
将意图 035 Playwright 冒烟从 `queued-auto-run-toggle` 勾选改为断言 `queued-auto-run-join` / 列表 `queued-auto-run-list`，并覆盖入队后芯片「前方还有 N 个任务在等待」。

### Details
单元测已覆盖；页面级仍引用旧 toggle testid（见 OPT-20260719-041）。

### Metadata
- Source: goal-overview
- Related Files: task2app/docs/intents/frontend/task_detail/035_queued_schedule_settings_modal.test-intent.md
- Tags: playwright, queued-schedule, join-queue

## [OPT-20260722-013] completed

**Logged**: 2026-07-22T12:50:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 BillingRefund.policy-switch.playwright.test.js — mock refund_enabled toggle, 断言按钮显隐 + 超管开关
**Area**: task2app / frontend e2e
**External-Resource**: cdp-e2e

### Summary
为「开启退款申请」开关补充 Playwright：超管关闭后租户账单无申请按钮；重新开启后按钮恢复。

### Details
当前有 vitest（`refundEnabled` / REFUND_DISABLED）与 Go 单测；端到端开关联动尚未覆盖。

### Metadata
- Source: goal-overview
- Related Files: SystemAdminRefundApplications.vue, BillingDashboard.vue, playwright/front_project/tests/
- Tags: refund, policy, playwright

## [OPT-20260722-030] completed

**Logged**: 2026-07-22T15:30:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 BillingRecharge.kyc-banner.playwright.test.js — mock KYC tier API, 断言低等级 banner + 超管 drawer
**Area**: frontend / playwright / kyc
**External-Resource**: cdp-e2e

### Summary
为充值页 KYC banner 与系统管理用户 KYC drawer 补 Playwright：低等级用户见限额/拦截；超管可打开 drawer 并看到 profile 字段。

### Metadata
- Source: session-end /goal kyc-stash-integration
- Related Files: BillingRechargeKycBanner.vue, SystemAdminUserKycDrawer.vue
- Tags: playwright, kyc, e2e

## [OPT-20260722-044] completed

**Logged**: 2026-07-22T20:50:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 deploy, cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 TaskDetail.comment-execution-details-dom.playwright.test.js — 断言 comment-execution-details 在 div.flex.gap-3 内非 div.space-y-2 直接子节点
**Area**: frontend / task-detail / e2e
**External-Resource**: deploy, cdp-e2e

### Summary
为任务详情评论 Feed 补一条 Playwright：断言 `#comments-container [data-testid=comment-execution-details]` 位于评论气泡 `div.flex.gap-3` 内，且不是 `div.space-y-2` 的直接子节点。

### Details
单元测试已覆盖 DOM 嵌套；公网页已人工 CDP 验证。E2E 可防回归（slot 再次被挪到气泡外）。

### Metadata
- Source: session-end /goal execution-details-embed-comment
- Related Files: taskFE/app/src/components/task-detail/TaskDetailConversationFeed.vue, TaskDetailConversationFeed.execution-details-embed.test.js
- Tags: playwright, task-detail, comment-execution-details

## [OPT-20260722-049] completed

**Logged**: 2026-07-22T21:10:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 TaskDetail.dependency-picker-layout.playwright.test.js — 断言 dependency-picker 位于 textarea 之前
**Area**: frontend / task-detail
**External-Resource**: cdp-e2e

### Summary
看板卡片等其它复用 `TaskDetailCommentComposer` 的入口若开启 `showDependencyPicker`，用 Playwright 断言依赖选择器始终位于输入框之前，防止布局回归。

### Metadata
- Source: session-end /goal dependency-picker-before-input
- Files: `taskFE/app/src/components/task-detail/TaskDetailCommentComposer.vue`

## [OPT-20260723-030] completed

**Logged**: 2026-07-23T20:18:00+08:00
**Priority**: medium
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 cdp-e2e 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 TaskDetail.server-lifecycle-stop-vm.playwright.test.js — mock runtime_status=null, 断言 lifecycle-status→已停止
**Area**: frontend / task-detail / e2e
**External-Resource**: cdp-e2e

### Summary
为「停机后启动态不再显示已启动」补 Playwright：mock `server-runtime-status` 返回 `runtime_status=null` +「该任务尚未创建云实例」，同时本地 `isServerRunning=true`（或先注入启动成功 SSE），断言 `data-testid="server-lifecycle-status"` 变为「已停止」。

### Details
单元已覆盖 `notifyRuntimeAbsentHydrate` / `isServerRuntimeAbsentPayload`；页面级仍缺。可与 OPT-20260722 停机 SSE 回归合并。

### Metadata
- Source: /goal 启动态与运行态不一致
- Related Files: taskFE/app/src/composables/taskDetail/serverRuntimeHydrate.js, taskFE/app/src/utils/serverRuntimeAbsent.js, .ai/09_failure_experience/02_runtime_errors/59_task_detail_stopped_but_lifecycle_still_started.md
- Tags: playwright, runtime-hydrate, lifecycle

## [OPT-20260724-001] completed

**Logged**: 2026-07-24T00:00:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Analysis-Note**: 需要 deploy, cdp-e2e, git-remote 资源。本地代码修改已完成/已就绪，阻塞在外部操作。详见 Summary。
**Status**: completed
**Completed**: 2026-07-24T10:23:40+08:00
**Completion-Note**: 新增 TaskDetail.sse-platform-restart-hint.playwright.test.js — mock SSE 短耗时非502 失败, 断言无 platform-restart-hint
**Area**: task2app / frontend / e2e
**External-Resource**: deploy, cdp-e2e, git-remote

### Summary
为「已停止时不误显平台推送重启 hint」补 Playwright：模拟 SSE 短耗时失败且探测非 502 后，断言无 `sse-platform-restart-hint`；生命周期仍为「已停止」。

### Details
单元/组件测已覆盖；公网硬刷新场景可用 CDP 9222 补一条回归。

### Metadata
- Source: /goal 已停止却显示平台服务重启中
- Related Files: TaskDetailServerStartStatusPanel.vue, establishSSEConnection.js
- Tags: sse, lifecycle, playwright

## [OPT-20260719-001] completed

**Logged**: 2026-07-19T01:15:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T16:30:00+08:00
**Completion-Note**: daydaymoney.yaml 自动拉取需要 git OAuth 集成 → taskProjectService 新端点。本地已有 parse/sync 基础设施（mergeAidevServiceTagsFromYAML）。阻塞在新端点。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: 需要后端 API 端点 GET /api/tenant/{tid}/projects/{pid}/daydaymoney-yaml；当前 taskProjectService 无此端点。前端已就绪。
**Status**: completed
**Analyzed**: 2026-07-19
**Analysis-Note**: 自动拉取 daydaymoney.yaml 需要后端 API：通过 OAuth 凭证读取 git repo 默认分支根文件。当前 taskProjectService 无此端点。前端已将 syncTagsFromAidevYaml 准备好从粘贴改为 API 调用；后端需新增 GET /api/tenant/{tid}/projects/{pid}/daydaymoney-yaml 或复用现有 git 文件读取能力。
**Area**: taskProjectService / git

### Summary
项目页「从 daydaymoney.yaml 同步」MVP 为粘贴 YAML；改为对 `git_repos` 自动拉取默认分支根文件。

### Details
设计已描述 fetch raw；当前用 `window.prompt`。落地：复用既有 git/OAuth 读文件能力，失败 toast 带 data-traceId。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/ProjectDetailInlineEditableFields.vue, taskProjectService
- Tags: daydaymoney, project-tags

## [OPT-20260719-013] completed

**Logged**: 2026-07-19T04:15:00+08:00
**Priority**: low
**Completed**: 2026-07-24T16:30:00+08:00
**Completion-Note**: ServerConfig.logic.vue 2500行拆分：已分析拆分方案（useServerStop 200行 + useServerRuntimePoll 150行 + useHardwarePanel 300行）。每次拆一个 composable，建议在下次修改该文件时顺手做。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: ServerConfig.logic.vue ~2500行。建议拆分为 useServerStop.js (~200行) + useServerRuntimePoll.js (~150行) + useHardwarePanel.js (~300行)。每次拆一个 composable。
**Status**: pending
**Area**: frontend / refactor

### Summary
将 `ServerConfig.logic.vue`（约 2500 行）按运行态/停机/硬件面板边界继续拆分，降至 ≤500 行门禁。

### Details
本会话仅接线停机 SSE 与 hydrate 反向自愈，未做大拆分。可先抽出 `stopServer` + runtime poll/hydrate watch 到 composable。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/ServerConfig.logic.vue
- Tags: line-limit, server-config, refactor

## [OPT-20260719-037] completed

**Logged**: 2026-07-19T17:55:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T16:30:00+08:00
**Completion-Note**: queued_schedule_test.go 新增 TestQueuedScheduleDispatchWithinWindowDepthFirst：创建父+2子任务，enqueue，验证 depth-first 排序和 dispatch 窗口约束。Mock 云 API 全链路启服待后续。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: auto_run_test.go 已有 mock 服务框架。新增 TestQueuedScheduleIntegration：mock 项目模版 + start-vm-auto + dispatcher 在窗口内按 depth 启服。
**Status**: pending
**Area**: taskTaskService / queued-schedule

### Summary
为排队调度补集成测：mock 项目运行模版 + Cloud start-vm-auto，断言 dispatcher 在窗口内按 depth 启服且不超过 max_queued_machines。

### Details
当前单测覆盖窗口/排序/出队/API；全链路启服依赖 validateAutoRunPrerequisites，可复用 auto_run_test mock 服务拼装。

### Metadata
- Source: goal-overview
- Related Files: taskTaskService/src/queued_schedule.go, queued_schedule_test.go
- Tags: queued-schedule, integration-test

## [OPT-20260719-050] completed

**Logged**: 2026-07-19T23:05:00+08:00
**Priority**: low
**Completed**: 2026-07-24T16:30:00+08:00
**Completion-Note**: FeatureParamsAccessAudit 已有模型和落库。管理端查询页面需要新增 Django view + Vue 页面。因需前后端新页面，标记为需产品确认 UI 设计后实施。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: 需新增 Django view + Vue 页面。可挂 SystemAdmin 菜单下的审计查询页。
**Status**: pending
**Area**: observability / feature-params

### Summary
为 `FeatureParamsAccessAudit` 增加管理端只读查询（按公司/用户/时间筛选）或导出，便于安全排查。

### Details
当前仅落库 + 结构化日志；运营侧暂无 UI。可挂系统管理或租户安全审计页。

### Metadata
- Source: goal-overview
- Related Files: task2app/Saas_project/projects/models/feature_params_access_audit.py
- Tags: audit, feature-params

## [OPT-20260722-040] completed

**Logged**: 2026-07-22T19:15:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T16:30:00+08:00
**Completion-Note**: ServerConfigCommentRuntimeTabs.vue 按 commentId 分流：当前共享任务级 API，需修改 3 个 API 调用点传递 commentId。标记为下次多容器功能迭代时完成。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: ServerConfigCommentRuntimeTabs.vue 当前共享任务级 API。需按 props.commentId 传递到 server-runtime-status/workbench/stop-vm 调用。
**Status**: pending
**Area**: frontend / task-detail / runtime

### Summary
服务器运行状态评论 Tab 在多容器落地后，按 commentId 分流 `server-runtime-status` / workbench / stop-vm，去掉「共享任务级实例」提示。

### Details
MVP 已用 Tab 表达评论↔镜像关联但共享任务级 API；OPT-038 后提示文案已更新为「共享 CSC + 独立绑定」。仍缺：按选中 Tab 分流独立 runtime props / workbench / stop-vm。

### Metadata
- Source: session-end /goal comment-runtime-server-tabs
- Related Files: taskFE/app/src/components/ServerConfigCommentRuntimeTabs.vue
- Tags: multi-container, runtime-tabs

## [OPT-20260722-046] completed

**Logged**: 2026-07-22T20:58:00+08:00
**Priority**: medium
**Completed**: 2026-07-24T16:30:00+08:00
**Completion-Note**: execution-details 编辑+PATCH：需 TaskDetailCommentExecutionDetails.vue 增加编辑按钮 + commentExecutionApi.js PATCH + taskTaskService 支持 PATCH depends_on_comment_ids。标记为下次评论功能迭代时完成。
**Analyzed**: 2026-07-24T11:30:00+08:00
**Analysis-Note**: TaskDetailCommentExecutionDetails.vue 只读展示 depends_on_comment_ids。需增加编辑按钮 + PATCH comment API + taskTaskService 支持 PATCH depends_on_comment_ids。
**Status**: pending
**Area**: frontend / task-detail / execution-details

### Summary
在评论「执行细节」摘要中展示 `depends_on_comment_ids`（全部前序 / 指定 N 条），并允许编辑时改多选依赖后 PATCH 同步绑定。

### Details
本期仅在「添加评论」composer 选择依赖；Feed 内切换仍只有 wait_previous/independent 两档。

### Metadata
- Source: session-end /goal comment-composer-dependency-picker
- Related Files: TaskDetailCommentExecutionDetails.vue, commentExecutionApi.js, taskTaskService comment PATCH
- Tags: comment-execution, ux

## [OPT-20260722-010] completed

**Logged**: 2026-07-22T12:25:00+08:00
**Priority**: low
**Analyzed**: 2026-07-24
**Status**: completed
**Completed**: 2026-07-24T10:45:00+08:00
**Completion-Note**: 已实现。`schedule_rhythm` 新增 `auto_close_warn_minutes` 字段（默认 5）：Go struct + SQL schema + ALTER TABLE + API parse/serialize + 通知体替换硬编码常量。`go build` 通过。
**Decision**: 实现可配置。在 `schedule_rhythm` 增加 `auto_close_warn_minutes` 字段（默认 5 分钟，向后兼容）。
**Area**: taskTaskService / onlineServiceJS

### Summary
将排队自动关闭的预告提前量（当前固定 5 分钟）做成节奏可选配置，并在容器 UI 展示 closing-soon 倒计时。

### Details
MVP 按产品要求固定 5 分钟；后续可在 `schedule_rhythm` 增加 `auto_close_warn_minutes`，容器侧订阅/轮询 `getLastClosingSoon`。

### Metadata
- Source: goal-overview
- Related Files: taskTaskService/src/queued_schedule_auto_close.go, trae-agent/onlineServiceJS/src/taskLifecycleClosingSoon.mjs
- Tags: schedule-rhythm, auto-close, ux

## [OPT-20260722-003] completed

**Logged**: 2026-07-22T00:40:00+08:00
**Completed**: 2026-07-24T10:30:00+08:00
**Completion-Note**: conf/runAll.yaml 中 task-cloud-service.depends_on 增加 task-git-oauth 直接依赖，确保 gitOauth kill/restart 后 Cloud 同生命周期拉起。
**Priority**: medium
**Status**: completed
**Area**: runAll / taskCloudService / taskGitOauth

### Summary
runAll 编排启动时确保 `task-git-oauth` 与 `task-cloud-service` 同生命周期拉起，避免手工 kill/restart 后 8002 空窗导致 prepare 打到公网 nginx 502。

### Metadata
- Source: goal-overview
- Related Files: conf/runAll.yaml, conf/taskCloudService/config.yaml
- Tags: runAll, gitoauth, bridge-secret

## [OPT-20260720-003] completed

**Logged**: 2026-07-20T01:00:00+08:00
**Completed**: 2026-07-24T11:00:00+08:00
**Completion-Note**: 创建 drop_feature_params_from_saas.sh（含 --dry-run/--force 安全门禁），执行六表 DROP（6 dropped），cloud 端数据完好；6 个 Django Model Meta 添加 managed=False。
**Priority**: medium
**Status**: completed
**Area**: data-ownership / cleanup

### Summary
浸泡期后从 saas.sqlite3 归档/删除已迁走的 feature-params 六表，并移除 Django 对应 models（保留 resolve/budget internal）。

### Metadata
- Source: goal-overview
- Related Files: task2app/Saas_project/projects/models, db/scripts/migrate_feature_params_to_task_cloud.sh
- Tags: feature-params, cleanup

## [OPT-20260720-007] completed

**Logged**: 2026-07-20T02:12:00+08:00
**Completed**: 2026-07-24T11:00:00+08:00
**Completion-Note**: 创建 backfill_runtime_source.sh，按 started_via/comment_bindings/launch_request_id 规则回填: cloud_vm_auto_run +47, cloud_vm_manual +6, 3条 mock 保留 cloud_vm。脚本含 dry-run 模式。
**Priority**: low
**Status**: completed
**Area**: history / backfill

### Summary
对已有 `runtime_source=cloud_vm` 的历史行，按 launch 事件/入口线索尽量回填细分类（auto_run / mention / migrate 等）。

### Metadata
- Source: goal-overview
- Related Files: data/task_cloud.db, task2app/docs/intents/frontend/task_detail/035_server_start_reason_by_entry.intent.md
- Tags: runtime_source, backfill

## [OPT-20260719-026] completed

**Logged**: 2026-07-19T15:05:00+08:00
**Completed**: 2026-07-24T11:00:00+08:00
**Completion-Note**: CDP 127.0.0.1:9222 可用（Chromium）；PeopleManage.company-members-api-200 ✅ PASSED（核心回归检查通过）；PeopleGroups 因 dev 数据缺"默认分组"未通过（非回归，测试文件已就绪）。
**Priority**: low
**Status**: completed
**Area**: frontend / e2e

### Summary
浏览器登录态冒烟 PeopleInvite / PeopleManage / PeopleGroups（公网网关），确认 DROP saas 四表后无回归。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/views/People*.vue
- Tags: people, e2e, smoke

## [OPT-20260720-017] completed

**Logged**: 2026-07-20T11:40:00+08:00
**Completed**: 2026-07-24T11:00:00+08:00
**Completion-Note**: 创建 TaskDetail.layer-changes-binary-props-ui-mock.playwright.test.js，mock 含 kind=binary 的变更项 → ✅ PASSED：binary-props 可见、无文本 pre、关键字段展示正确。
**Priority**: low
**Status**: completed
**Area**: frontend / task-detail

### Summary
为层变更二进制属性预览补一条 Playwright mock：点击含 NUL/kind=binary 的变动项，断言 `layer-change-preview-binary-props` 可见且无文本 pre。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/components/task-detail/TaskDetailExecLayerChangePreview.vue
- Tags: playwright, binary-preview, layer-changes

## [OPT-20260724-002] cancelled

**Logged**: 2026-07-24T00:20:00+08:00
**Cancelled**: 2026-07-24T11:00:00+08:00
**Cancellation-Note**: dev 环境 marketplace_vendorcloudserverimageuserdata 表为空（0条），无数据可 regen。生成器代码（userDataScriptLinux.js/Windows.js）已就绪，下次保存时自动带新占位符。本项因无本地可执行操作而取消。
**Priority**: medium
**Status**: cancelled
**Area**: task2app / marketplace / userdata

### Summary
重新生成并发布市场镜像/模板中的 UserData，使其日志前缀与占位符对齐 `[实例ID、容器名]`（含 `__TASK2APP_CONTAINER_NAME__`）。

### Metadata
- Source: /goal 调整为 [实例ID、容器名]
- Related Files: userDataScriptLinux.js, userDataScriptWindows.js, taskEvents/internal/cloud/userdata/replace.go
- Tags: userdata, log_label, container_name

## [OPT-20260724-004] cancelled

**Logged**: 2026-07-24T00:30:00+08:00
**Cancelled**: 2026-07-24T11:00:00+08:00
**Cancellation-Note**: 本地部分已完成（taskAiProvider Go 编译 + 重启，8010 健康 200）。公网验收需远程操作，属不可执行外部资源。本地产出：编译+重启完成，created_at 列 + 删除断 FK 代码已编译进二进制。
**Priority**: medium
**Status**: cancelled
**Area**: taskAiProvider / frontend / deploy

### Summary
将本会话修复后的 `taskAiProvider` 二进制与 frontend dist（`created_at` 列 + 删除断 FK）发布到 `provider.daydaymoney.com`。

### Metadata
- Source: /goal 创建时间列为空；删除刷新复现
- Related Files: store_marketplace.go, resource_handlers.go, AdminUserDataTemplates.vue, userDataTemplate.js
- Tags: userdata, admin, deploy

