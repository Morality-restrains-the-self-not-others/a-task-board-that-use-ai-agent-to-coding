# Completed OPT Archive — 2026-07-21

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 10 条。
> 归档执行时间：2026-07-24T10:15:26+08:00

## [OPT-20260719-038] completed

**Logged**: 2026-07-19T17:55:00+08:00
**Priority**: low
**Status**: completed
**Area**: frontend / task-detail

### Summary
顶层任务详情增加队列快照列表（调用 GET .../queued-auto-run/），展示深度/状态/占用。

### Details
后端队列 API 已就绪；前端 MVP 仅有节奏表单与排队开关。

### Metadata
- Source: goal-overview
- Related Files: TaskDetailQueuedSchedulePanel.vue
- Tags: queued-schedule, ui

**Completed**: 2026-07-21T23:08:00+08:00
**Completion-Note**: 模态内落地 `queued-auto-run-list`（GET 快照）；入队改为「加入自动执行队列」按钮；任务 JSON 增加 `queued_ahead_count`，芯片展示前方等待数。

## [OPT-20260720-029] completed

**Logged**: 2026-07-20T13:40:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend / spa / collectstatic

### Summary
修复 `runall-lifecycle.sh build` 在 Vite 产出新 hash 后，Django collectstatic 未把 `main-*.js` / TaskDetail 大包同步进 `STATIC_ROOT` 的问题。

### Details
本会话中 lifecycle 报「78 copied, 194 unmodified」，manifest 已更新但 `collected_static/assets/main-BE7WAgka.js` 缺失，需 `rsync` 才生效。应在 lifecycle 脚本末尾对 `front_project/static/assets/` → `collected_static/assets/` 做显式同步（或修 Django ignore/哈希策略），并加验收断言。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/scripts/runall-lifecycle.sh
- Tags: collectstatic, vite, spa

**Completed**: 2026-07-21T23:08:00+08:00
**Completion-Note**: `runall-lifecycle.sh build` 在 collectstatic 后增加 `rsync -a --delete` 强制同步 Vite assets，并继续显式拷贝 `.vite/manifest.json`。

## [OPT-20260720-022] completed

**Logged**: 2026-07-20T11:56:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend / build

### Summary
排查为何 `runall-lifecycle.sh build` 的 `collectstatic` 未把新建 hash 的 `WorkPanel-*.js` 拷入 `collected_static/assets`（manifest 已更新），必要时在 lifecycle 脚本中增加「按 manifest 校验并强制同步缺失 chunk」。

### Details
本会话修复后 vite 产出 `WorkPanel-E2A5XzqV.js`，manifest 两边一致，但 `collectstatic` 报告大量 unmodified 且 `collected_static/assets` 一度缺少该文件；最终靠 `rsync` 补齐。公网若只依赖 collectstatic 可能仍吃旧包。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/scripts/runall-lifecycle.sh, taskFE/ai.md
- Tags: collectstatic, spa, work-panel

**Completed**: 2026-07-21T23:08:00+08:00
**Completion-Note**: 与 OPT-20260720-029 一并落地：lifecycle 强制 rsync Vite `assets/` → `collected_static/assets/`。

## [OPT-20260721-010] completed

**Logged**: 2026-07-21T22:55:00+08:00
**Priority**: medium
**Status**: completed
**Area**: tooling / mcp / code-review-graph

### Summary
排查为何 Cursor 会话 MCP catalog 未暴露 `code-review-graph`（仅见 cursor-app-control / cursor-ide-browser），尽管 `.cursor/mcp.json` 与 `.mcp.json` 已配置且本机 `uvx code-review-graph` / `graph.db` 可用。

### Details
根因：Agent lease / `~/.cursor/projects/.../mcps/` 从未注册项目 MCP（相对 `command` 不可靠；日志仅有 `DeleteClient config_server_modified`）。已改为 `${workspaceFolder}` + `type:stdio`，加固 `crg-mcp-serve.sh` PATH，runbook 增 §1.1。需用户侧完全重启 Cursor 并在 Tools & MCP 确认已连接后，新会话 `GetMcpTools` 才会出现 CRG。

### Metadata
- Source: goal-overview
- Related Files: .cursor/mcp.json, .mcp.json, .cursor/crg-mcp-serve.sh, docs/runbooks/code-review-graph.md
- Tags: crg, mcp, cursor

**Completed**: 2026-07-21T23:05:00+08:00
**Completion-Note**: 根因定位 + 配置/脚本/runbook 修复；本会话无法代启 Cursor MCP lease，重启后验收。

## [OPT-20260721-007] completed

**Logged**: 2026-07-21T22:00:00+08:00
**Priority**: high
**Status**: completed
**Area**: runAll / docker-kafka / health-check

### Summary
将 `docker-kafka` 的 runAll 健康检查从「仅 Kafka UI :18080」改为 broker 就绪探活（`dockerInfra/kafka/health.sh`：compose kafka running + `kafka-topics`），避免 broker Exited 时 UI 仍 healthy 导致 saas-backend 503 与下游连锁失败。

### Details
原开放清单误标为 OPT-20260721-006（与已归档 avatar 项撞号），本条以 007 归档。验收：`stop kafka-kafka-1` 后 UI 仍 200，health.sh exit 1；runAll 探针 `exec://bash dockerInfra/kafka/health.sh`，监控出现 retrying/restarting（err: broker not running）并经 `run.sh start` 自愈；配置单测 `TestLoadConfig_ProductionDockerKafkaUsesBrokerExecProbe` 通过；恢复后 50/50 healthy。

### Metadata
- Source: /goal OPT-20260721-006
- Related Files: conf/runAll.yaml, dockerInfra/kafka/health.sh, runAll/src/config_test.go, conf/runAll.yaml.ai.md, .ai/09_failure_experience/02_runtime_errors/70_kafka_broker_down_ui_healthy_false_positive.md
- Tags: kafka, health-check, runAll

**Completed**: 2026-07-21T22:10:00+08:00
**Completion-Note**: docker-kafka 改为 exec broker 探活；热替换 runAll 后 live 验收 UI 假阳性已消除；全栈 50 healthy。

## [OPT-20260721-006] completed

**Logged**: 2026-07-21T21:45:00+08:00
**Priority**: medium
**Status**: completed
**Area**: taskTenantService / accounts / avatar

### Summary
公司成员头像（`member_avatar`）迁至 taskTenantService 后补齐读写与列表暴露；与个人头像形成「租户优先 → 个人回退」。

### Details
（原开放清单误编号为 OPT-20260721-002，与已归档 CRG 项撞号，本条以 006 归档。）

### Metadata
- Source: session-end /goal
- Related Files: taskTenantService/src/member_avatar.go, taskTenantService/src/member_handlers.go, taskProjectService/src/workspace_access_enrich.go, task2app/Saas_project/accounts/tenant_client.py, task2app/Saas_project/accounts/views/user_views.py
- Tags: avatar, tenant, member_avatar_url

**Completed**: 2026-07-21T21:55:00+08:00
**Completion-Note**: 租户侧存盘 + 公开 GET `/members/{id}/avatar`；internal upload/delete；company_members/协作列表/batch-resolve/profile 透传；Django `company-avatar` 转发；本地 200 image/png 验收。

## [OPT-20260721-004] completed

**Logged**: 2026-07-21T21:40:00+08:00
**Priority**: medium
**Status**: completed
**Area**: ci / skills / code-review-graph

### Summary
为十步 × CRG 矩阵增加轻量 golden 检查（如 `check_crg_pipeline_refs.py`）：断言 goal-mode、0-auto-flow 与 Steps 2/6/7/8/9/10 的 SKILL.md 均引用 `references/code-review-graph.md`。

### Details
防止后续改技能时丢掉 CRG 指针。验收：脚本进 `repo-quality-gates.yml`；故意删一行引用则 CI 红。

### Metadata
- Source: session-end
- Related Files: .claude/skills/1-brainstorming-design-docs/references/code-review-graph.md, .github/workflows/repo-quality-gates.yml
- Tags: code-review-graph, ci, skills

**Completed**: 2026-07-21T21:37:22+08:00
**Completion-Note**: 新增 db/scripts/ci/check_crg_pipeline_refs.py(+test)，挂入 repo-quality-gates.yml；本地 OK。

## [OPT-20260721-002] completed

**Logged**: 2026-07-21T21:25:00+08:00
**Priority**: low
**Status**: completed
**Area**: ci / code-review-graph

### Summary
评估将 `.github/workflows/code-review-graph.yml` 的 `fail-on-risk` 从 `none` 提升为 `high`（或仅对受保护路径启用），作为可选合并门禁。

### Details
当前按设计保持软依赖、仅 sticky comment。待团队观察误报率与 monorepo 嵌套仓检出覆盖后再决定。验收：文档写清阈值与豁免；CI 在样本 PR 上行为符合预期。

### Metadata
- Source: session-end
- Related Files: .github/workflows/code-review-graph.yml, .claude/skills/1-brainstorming-design-docs/references/code-review-graph.md
- Tags: code-review-graph, ci, risk-gate

**Completed**: 2026-07-21T21:37:22+08:00
**Completion-Note**: 默认 fail-on-risk=none；PR label crg-gate → high。策略写入 docs/runbooks/code-review-graph.md 与 workflow。

## [OPT-20260721-003] completed

**Logged**: 2026-07-21T21:25:00+08:00
**Priority**: low
**Status**: completed
**Area**: tooling / code-review-graph

### Summary
在开发者本机文档或 runbook 中补充 `crg-daemon` 推荐启停命令，并评估是否为大型子仓分别 `register`（避免根仓一次全量 build 过慢）。

### Details
Cursor hook 已 fail-open 做 `update`；daemon 仍为可选。验收：runbook 或 reference 中有可复制命令；至少验证一个子仓增量更新耗时可接受。

### Metadata
- Source: session-end
- Related Files: .claude/skills/1-brainstorming-design-docs/references/code-review-graph.md
- Tags: code-review-graph, daemon, monorepo

**Completed**: 2026-07-21T21:37:22+08:00
**Completion-Note**: runbook 补充 crg-daemon/子仓 register；taskAuth build≈557ms、update≈237ms；taskAuth .gitignore 忽略图目录。


## [OPT-20260721-005] cancelled

**Logged**: 2026-07-21T21:55:00+08:00
**Cancelled**: 2026-07-24T11:00:00+08:00
**Cancellation-Note**: 本地部分已完成（SPA vite build + collectstatic，多仓推送成败明细 UI 已含）。公网容器滚动属远程 ECS 操作，不可执行。本地产出：SPA 构建完成，前端代码已包含格式化多仓推送明细。
**Priority**: high
**Status**: cancelled
**Area**: onlineServiceJS / frontend / deploy

### Summary
发布含多仓推送成败明细的 onlineServiceJS 镜像 + 前端 SPA，使 ztree 推送失败弹层展示「成功/失败」分行明细。

### Metadata
- Source: session-end
- Related Files: layerGitOauthPushDetail.mjs, formatLayerGitPushMultiRepoDetail.js, Modal.ui.vue, 69_multi_repo_push_coverage.md
- Tags: git-push, multi-repo, ux, deploy

