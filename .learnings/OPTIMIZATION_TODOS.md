# Session Optimization TODOs — 本地可执行

> 仅存放本地可执行、无阻塞的 pending 项。分流规则见 [OPTIMIZATION_TODOS.ai.md](./OPTIMIZATION_TODOS.ai.md)。
> 阻塞：[BLOCK_TODO_BROWSER.md](./BLOCK_TODO_BROWSER.md) · [BLOCK_TODO_OPS.md](./BLOCK_TODO_OPS.md) · [BLOCK_TODO_INFRA.md](./BLOCK_TODO_INFRA.md)；产品：[PRODUCT_DECISIONS.md](./PRODUCT_DECISIONS.md)；完成：[OPTIMIZATION_TODOS_COMPLETED.md](./OPTIMIZATION_TODOS_COMPLETED.md)。

- **Count**: 11


### OPT-20260911-001 — 有 Docker/MySQL 时补跑 bootstrap-admin 随机密码集成测

- **Status**: pending
- **Created**: 2026-09-11
- **Context**: 本会话将 022 密码改为哨兵并由 `bootstrap-admin` 每环境随机生成；`TestBootstrapAdminGeneratesRandomPassword` / `TestMigrateSeedsConfBootstrapAdminEmail` 等依赖 `OpenTestMySQL`，当前 Cloud Agent 环境无 Docker/MySQL 被 Skip。
- **Action**: (1) 在具备 `docker` + MySQL 的环境执行 `cd taskAuth && go test ./src/ -count=1 -run 'TestBootstrapAdmin|TestMigrateSeedsConf'`；(2) 确认哨兵被替换为 bcrypt、二次 bootstrap 哈希不变；(3) 可选：对 fresh DB 跑 `db/task-auth/init.sh` 冒烟。
- **Why**: 无库单测已覆盖旋转判定与随机生成，但端到端写库路径仍需一次实库确认，避免 INSERT IGNORE + UPDATE 交互回归。
- **How to apply**: `taskAuth/src/bootstrap_admin_test.go`；`taskAuth/src/bootstrap_admin_password.go`；`dataMigrate/taskAuth/022_seed_bootstrap_admin.sql`；`db/task-auth/init.sh`。


### OPT-20260909-001 — 专利文书同步到独立 docs.git 远端

- **Status**: pending
- **Created**: 2026-09-09
- **Context**: 本会话在 meta 扁平树新增 `docs/专利/`（受理通知书 PDF + README）。`.gitmodules` 登记了 `task2money/docs.git`，但当前环境无 `docs/.git` 嵌套仓。
- **Action**: (1) 在具备 `docs` 独立 clone 的环境中，将 `docs/专利/` 与 `docs/README.md` 链接变更提交并推送到 `docs.git`；(2) 若后续恢复 gitlink，确认 meta 指针与 docs 远端一致。
- **Why**: 避免独立 docs 远端与 meta 扁平树长期漂移。
- **How to apply**: `.gitmodules` `[submodule "docs"]`；`docs/专利/`；`docs/README.md`。


### OPT-20260908-001 — 确认各子仓远端是否仍含 COMMERCIAL.md 并在子仓独立清理

- **Status**: pending
- **Created**: 2026-09-08
- **Context**: 本会话在 meta 扁平工作区删除了全部 `COMMERCIAL.md` 与商业授权 README 指引；当前环境无 gitlink，文件以普通路径跟踪。若后续恢复独立 submodule 远端（如 `task2money/conf`），远端历史树可能仍含 `COMMERCIAL.md`。
- **Action**: (1) 对 `.gitmodules` 中各 path 执行 `git ls-tree -r HEAD --name-only | rg COMMERCIAL`（或 clone 远端浅检）；(2) 若仍存在，在对应子仓删文件、改 README、推送后再更新 meta 指针；(3) 确认 CI `check_conf_tracked_allowlist` 在真实 conf.git 上仍通过。
- **Why**: 避免 submodule 重新同步时把已移除的商业授权文件带回主工作树。
- **How to apply**: `.gitmodules`；各子仓 `COMMERCIAL.md`/`README.md`；`db/scripts/ci/check_conf_tracked_allowlist.py`。


### OPT-20260905-007 — 评估清除或文档化 `.runall/no_restart_runall` 以免 :9999 空窗无自动恢复

- **Status**: pending
- **Created**: 2026-09-05
- **Context**: 诊断 `task-events-gitlab-manual-node-fulfillment-queued-1-ops-alert` 时发现编排器已于 08:29 SIGTERM 退出，且 `.runall/no_restart_runall`（写于 2026-08-17）阻止 watchdog 自动拉起 :9999；UI 打开日志面板只显示 `Failed to load logs: Failed to fetch`。手动 `./runAll/run.sh` 后恢复。
- **Action**: (1) 确认该 flag 是否仍为运维意图；若否，删除 flag 并验证 `ensure_services_healthy.py` 可在 :9999 down 时重启；(2) 若保留，在 `runAll/ai.md` / 9999 空窗段补充「有 no_restart 时必须手工 `./run.sh`」；(3) 可选：UI 在 status poll 连续失败时横幅提示 orchestrator down + 指向 `last_orchestrator_exit.json`。
- **Why**: flag 长期存在会使每次 SIGTERM 后出现长时间 :9999 空窗，日志/启停 API 全部不可用，表象像「服务挂了」实为编排器未起。
- **How to apply**: `.runall/no_restart_runall`；`runAll/scripts/ensure_services_healthy.py`；`runAll/scripts/watchdog_notify.py`；`runAll/ai.md`。

### OPT-20260905-006 — status UI 日志拉取网络失败时提示编排器可能已退出

- **Status**: pending
- **Created**: 2026-09-05
- **Context**: :9999 不可达时日志面板仅显示 `Failed to load logs: Failed to fetch`（`TypeError` 原文），无法区分服务无日志与编排器已退出；本次排障靠 `last_orchestrator_exit.json` 才定位。
- **Action**: (1) 在 `runAll/src/status_ui/js/09.js` / `13.js` 的 catch 中识别 `Failed to fetch` / `NetworkError`；(2) 改为可读提示（如「无法连接编排器 :9999，请检查 runAll 是否在跑；退出指纹见 `.runall/last_orchestrator_exit.json`」）；(3) 补最小回归（字符串断言或 UI smoke）。
- **Why**: 空窗场景下用户/Agent 会误判为单个 intent 日志坏了，延误恢复。
- **How to apply**: `runAll/src/status_ui/js/09.js` `fetchLogsOnce`；`13.js` `fetchDevLogsOnce`；可选共享 `formatLogsFetchError(err)`。

### OPT-20260905-004 — 首页页脚版权行品牌与「云端Coding 平台」对齐

- **Status**: pending
- **Created**: 2026-09-05
- **Context**: 首页页脚品牌区已从「SaaS Platform」改为「云端Coding 平台」；同页版权行仍为 `© 2023 SaaS Platform. All rights reserved.`，与品牌文案不一致。
- **Action**: (1) 将 `Home.vue` 中 `data-testid="home-footer-copyright"` 文案改为含「云端Coding 平台」（或统一品牌常量）；(2) 在 `Home.footer-links.test.js` 断言版权行不再含 `SaaS Platform`；(3) 搜索其它 PageFooter / LoginMarketingFooter 是否仍硬编码 SaaS Platform 一并处理。
- **Why**: 用户可见品牌不一致会削弱品牌识别；版权行仍暴露旧名。
- **How to apply**: `taskFE/app/src/views/Home.vue`；`taskFE/app/src/views/Home.footer-links.test.js`；`rg -n 'SaaS Platform' taskFE`。

### OPT-20260905-003 — 将源仓 conf icpBeian 持久同步进 daydaymoney-deploy conf

- **Status**: pending
- **Created**: 2026-09-05
- **Context**: 本次为上线页脚备案号，在 `$DEPLOY_ROOT/conf/frontend/vue/config.yaml` 手工补了 `icpBeian`；源仓 `task2money/conf` 已含该键并已 push，但 deploy 树 conf 属于 `daydaymoney-deploy` 工作区，未走常规 sync。
- **Action**: (1) 确认 deploy conf 与源仓 `conf/frontend/vue/config.yaml` 的 `icpBeian` 一致；(2) 按 ADR-0052 双写/导出流程把该键提交或纳入下次 conf 同步，避免手工漂移。
- **Why**: 下次从 deploy 侧读 conf 构建时若缺键，页脚会隐藏备案号。
- **How to apply**: `$DEPLOY_ROOT/conf/frontend/vue/config.yaml`；对照 `/tmp/ram-work/conf/frontend/vue/config.yaml`。

### OPT-20260905-002 — ResetPassword 页脚补齐配置驱动的 ICP 备案号

- **Status**: pending
- **Created**: 2026-09-05
- **Context**: Home / PageFooter / LoginMarketingFooter 已改为读 `VITE_ICP_BEIAN`；`ResetPassword.vue` 自建页脚版权区仍无备案号链接。
- **Action**: (1) 在 `ResetPassword.vue` 版权区按 PageFooter 模式加 `v-if="icpBeian"` + MIIT 链接；(2) 补 unit test 断言文案来自 env、源码无硬编码。
- **Why**: 同站多页脚展示不一致，合规页脚应统一。
- **How to apply**: `taskFE/app/src/views/ResetPassword.vue`；复用 `import.meta.env.VITE_ICP_BEIAN`。

### OPT-20260904-009 — 精准重启后验证 git_pr 回复经 SSE 无需刷新出现在 Feed

- **Status**: pending
- **Created**: 2026-09-04
- **Context**: 任务详情打开时，层图/auto-run 异步创建的 git_pr 子评论原先需整页刷新才可见；已改为创建时发 SSE_MESSAGE(task_git_pr_reply_created) + FE 重拉 Feed。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」task-task-service + taskFE（及确认 task-events-sse-message-1-send-sse-message / task-sse 健康）；(2) 打开任务详情保持不刷新；(3) 触发带 PR 的推送/auto-run 或层图回填，确认父评论下出现「一键合并」气泡且无整页刷新。
- **Why**: 需运行时确认 Kafka→SSE→FE 全链路与线上 bundle 已加载。
- **How to apply**: 登记已写入 `.runall/precise_restart_services.txt`；对照设计 `docs/superpowers/specs/2026-09-04-git-pr-reply-sse-fanout-design.md`。
- **Deploy (2026-09-05 夜间)**: 服务与 bundle 已就绪——taskTaskService bin 19:07 构建含 c2e86ee；线上 FE 已提升至 release 20260904190805（含 FE 741f302，TaskDetailContent chunk 含 task_git_pr_reply_created 标记）。仅剩运行时验收：需对真实项目触发带 PR 的推送/auto-run 生成 git_pr 子评论，副作用大，留待交互/授权会话。

### OPT-20260904-007 — 精准编译重启后验证项目详情换镜像不再误报完整模版

- **Status**: pending
- **Created**: 2026-09-04
- **Context**: Loki trace `78764b42-4701-459a-bc58-010603200f0b`：PATCH 换镜像时 FE 附带硬件面板「有地域无实例」半成品，后端 400「更换镜像须同时提交完整运行模版」。已修 FE/BE 忽略半成品并回退库内完整模版。
- **Action**: (1) http://10.2.150.68:9999/ 「精准编译重启」task-project-service + taskFE；(2) 打开 `https://www.daydaymoney.com/tenant/882864422450655232/projects/proj_883640741912408064/`；(3) 在已有完整运行模版下更换同架构镜像，应 200 且无 `project-inline-edit-error`。
- **Why**: 需运行时确认线上 bundle/服务已加载修复。
- **How to apply**: 精准重启登记已写入 `.runall/precise_restart_services.txt`。
- **Deploy (2026-09-05 夜间)**: 服务与 bundle 已就绪——taskProjectService bin 19:07 构建含 7cc4fd7；线上 FE 已提升至 release 20260904190805（含 FE fd55001）。仅剩运行时验收：E2E 账号访问目标项目 proj_883640741912408064 被重定向到租户项目列表（仅 ram-work--gitlab-sh-1 / somanyad-github 可见），且换镜像会改动真实运行项目，需有该 project 权限的账号/交互会话执行。

