# Learnings

## [LRN-20260903-001] pitfall

**Logged**: 2026-09-03T09:45:00+08:00
**Priority**: high
**Status**: pending
**Area**: backend / cloud / sse

### Summary
`publishTaskSSE` 对 `status=error` 会走 `persistCommentCSCStartFailure`，把 CSC `last_runtime_status` 改成 Failed。VM 已 Running、只是容器未 `register-reachability` 时不得走这条路径。

### Details
trace `dbd651948b1849c6d17177d5`：Aliyun 实例已 Running + SSH 通，但 `server_url` 空。若用 `publishTaskSSE` 发可达超时，测试里 `last_runtime_status` 被写成 Failed。应收口 binding + `error_reason`，SSE 用 `publishSSEMessage`。

### Suggested Action
容器未登记类失败：`appendCommentContainerBindingLogMessage` + `error_reason` UPDATE + `markCommentBindingFailedAfterStartError` + `publishSSEMessage`。禁止 `persistCommentCSCStartFailure`。回归：`TestReconcileStaleStartingReachabilityFailsOldRunningWithoutServerURL`。

### Metadata
- Source: conversation
- Related Files: taskCloudService/src/stale_starting_reachability.go, taskCloudService/src/events.go, taskCloudService/src/comment_csc_start_failure_persist.go
- Tags: reachability, sse, last_runtime_status
- See Also: OPT-20260903-006

---

## [LRN-20260902-002] pitfall

**Logged**: 2026-09-02T16:05:00+08:00
**Priority**: high
**Status**: pending
**Area**: frontend / ops / spa

### Summary
clone-run 下 `taskfe-nginx` 挂的是 `$DEPLOY_ROOT/envs/current/taskFE/app/public`，源码仓 `taskFE/app/public/html` 构建不会自动出现在 https://www.daydaymoney.com。且 `collect-deploy-binaries.sh` 曾只认 `public/index.html`，atomic-vite 的 `public/html/index.html` 会被跳过，精准编译仍打成部署树旧包。

### Details
OAuth 探测超时徽标已在源码 `d7c3396` + 源码 release `20260902142508`，但公网仍加载 `index-DBfw65qI.js`（build-time 2026-09-01），`useCommentGitOauthAccessTokenProbe` catch 仍 `showRequestError`。trace `7fa891f7-4d25-4e87-989c-47794f21b5a8` 弹窗来自该旧包。切 deploy `html` symlink 到新 release 后公网变为 `index-BjaN9aln.js`。

### Suggested Action
改 taskFE 后除 source `npm run build` 外，须确认 `curl -s https://www.daydaymoney.com/ | rg 'index-.*\\.js'` 的 hash 与 `taskFE/app/public/html/index.html` 一致；collect 须把 `html/index.html` 视为 live。

### Metadata
- Source: conversation
- Related Files: runAll/scripts/collect-deploy-binaries.sh, taskFE/app/scripts/atomic-vite-build.sh
- Tags: clone-run, taskFE, spa, oauth-badge
- See Also: OPT-20260902-010, ADR-0056

---

## [LRN-20260902-001] pitfall

**Logged**: 2026-09-02T09:05:00+08:00
**Priority**: high
**Status**: pending
**Area**: backend / ops / sse

### Summary
clone-run 下 Node `findRepoRoot` 若只 walk 到第一个 `conf/base.yaml`，会停在 `envs/current/`，读不到部署根 `conf-local`，taskSSE `gatewayInternalSecret` 空串 fail-closed，已登录 SSE 返回 403 `forbidden: gateway internal secret required`。

### Details
trace `cb9b0ffb7d6bdc9ce9fd2a424a8fff95`：APISIX `task-sse` 已 forward-auth + proxy-rewrite，taskSSE 仍 403。进程有 `CONF_ROOT`/`DEPLOY_ROOT` 但旧 `config.mjs` 不读它们。本地 `:8798` 无头 403、带 48 字密钥 200。修复：`findRepoRoot` 对齐 `confload.FindConfigRoot`。同类 Go 坑见 LRN-20260901-001 / OPT-20260901-019。

### Suggested Action
新增 Node 配置加载必须先读 CONF_ROOT/DEPLOY_ROOT；SSE 403 先看启动日志 `gatewaySecret=MISSING(fail-closed)` 与 `expected_secret_empty_fail_closed`。

### Metadata
- Source: conversation
- Related Files: taskSSE/src/config.mjs, taskSSE/src/server.mjs, shareLib/confload/root.go
- Tags: clone-run, conf-local, task-sse, 403, CONF_ROOT
- See Also: LRN-20260901-001, OPT-20260901-019, OPT-20260902-001

---

## [LRN-20260901-001] pitfall

**Logged**: 2026-09-01T17:50:00+08:00
**Priority**: high
**Status**: pending
**Area**: backend / ops / email

### Summary
密码重置 API HTTP 200 只表示 Kafka `EMAIL_SENT` 入队成功，不表示 SMTP 已投递。本次 QQ `535 Login fail` + 11 次重试进 DLT，用户收不到信。

### Details
根因不是授权码失效，而是 clone-run worker 的 `config.FindMonorepoRoot` 从 cwd 命中 `envs/current/conf/base.yaml`，忽略 `CONF_ROOT`，读不到部署根 `conf-local/events/domain-events/email.yaml`，SMTP 密码空串 → QQ 535。trace `85ddb59f-661a-4af4-b8f1-d4030cf9687d`：`task-auth` 200 → Kafka 入队 → 消费者 535 重试进 DLT。Kafka publish 日志无 `trace_id`。18:48 改为 `confload.FindMonorepoRoot` 后 `password_configured=true`；Playwright 公网重置（trace `3a80284e-6016-415a-a499-3426a7c203aa`）Loki 出现 `EMAIL_SENT delivered to [author@example.com]`。

### Suggested Action
其它服务同类 cwd walk 见 OPT-20260901-019；publish 日志带 trace（OPT-20260901-016）；仅当 overlay 后仍 535 才轮换 QQ 授权码（OPT-20260901-011）。

### Metadata
- Source: conversation
- Related Files: taskAuth/src/auth_password_reset.go, taskEvents/notifications/delivery.go
- Tags: smtp, password-reset, dlt, traceid
- See Also: OPT-20260901-011, OPT-20260901-015, OPT-20260901-016, OPT-20260901-017

---

## [LRN-20260831-002] best_practice

**Logged**: 2026-08-31T06:52:00+08:00
**Priority**: high
**Status**: pending
**Area**: frontend / backend / nfr

### Summary
页面数据 GET 不得内嵌 GitLab 探活；即使缩短超时或 lite 内部接口，仍会挡住首屏。探活必须是独立请求。

### Details
任务详情 GET 在 GitLab 不可达时固定约 15s 才 200（trace `544a7a87-842d-42b9-aebd-c87b62e281c6`）。根因是 `loadProjects` → GET project → `enrichProjectGitReposStatus`。首版设计把 lite repo-urls + 300ms 超时放在任务 GET 内，被否决：数据必须立即返回，badge 走已有 `POST validate-git-repos`。

### Suggested Action
同类「展示数据 + 外部探活」一律两阶段：DB GET 首屏，探活独立且失败 fail-open。见 B-086。

### Metadata
- Source: user_feedback
- Related Files: taskTaskService/src/task_store.go, taskProjectService/src/project_handlers.go, taskFE/app/src/composables/taskDetail/useTaskLinkedRepoGitProbe.js
- Tags: gitlab-probe, first-paint, timeout
- See Also: B-086, OPT-20260831-004, OPT-20260831-005

---

## [LRN-20260831-001] pitfall

**Logged**: 2026-08-31T05:40:00+08:00
**Priority**: high
**Status**: pending
**Area**: ops / project catalog / task_projects

### Summary
任务详情上的项目胶囊若显示裸 `proj_*` ID，说明 `project_entries` 已无该行，而 `task_projects` 仍在。昨夜 MySQL dump 恢复不会带回 8 月 29 日之前已从目录删除的项目。应用 8 月 28 日 dump 的 `INSERT IGNORE` 可找回。

### Details
- `proj_880498883115905024`（somanyad--self-gitlab）在 mysql-dump-20260827/28 有完整 `project_entries`+`project_repos`+`project_workspaces`，从 20260829 dump 起消失。
- 同批消失：`proj_878549993831559168`、`proj_880070089976606720`。
- `handleDeleteProject` 不清理跨服务 `task_projects`。前端曾把缺失目录的 ID 当成 `project_name` 做成可点链接。

### Suggested Action
目录行从更早 dump `INSERT IGNORE` 回填；任务详情对 `project_missing` 显示「项目已删除」且不链到 404。删除项目补领域事件拆关联（OPT-20260831-001）。

### Metadata
- Source: conversation
- Related Files: taskFE/app/src/utils/taskProjectsWithDetails.js, taskProjectService/src/project_handlers.go
- Tags: dangling-fk, project-delete, dump-restore
- See Also: OPT-20260831-001, OPT-20260830-025, LRN-20260830-001

---

## [LRN-20260830-001] pitfall

**Logged**: 2026-08-30T22:35:00+08:00
**Priority**: high
**Status**: pending
**Area**: ops / mysql / ramsync / P4 deploy

### Summary
tmpfs 重启后，`ramsync-daemon.sh` `ensure_data_dir` 会把磁盘上「有目录名、无 ibdata1」的 MySQL 空壳拷回 RAM 并记 restored。P4 再 symlink 该目录；`detect_corrupt_data.sh` 命中后 `mv` 符号链接并冷启动空库，9999 init 只留下 bootstrap。登录成功 ≠ 数据还在。

### Details
- 2026-08-30 约 21:33 本机重启，uptime 与 `/tmp/ramsync.log` 对齐。
- `dockerInfra/mysql/data/` 被 ramsync EXCLUDE；磁盘 `ram-mount` 侧长期是 Aug 10 空壳。真正落盘是 `/home/ljy/ramwork-recovery/mysql-dumps/`（latest 21:27）以及 root 拷的 `/home/ljy/gitClone/ram-work_0830_2115/`（21:16，含 ibdata1）。
- 21:42 从 `/tmp/ram-deploy` 起 MySQL：`data.corrupt.20260830_214230` → ram-work 空壳；新 `data` 为种子库。`auth_user` 从 dump 中 12 行变成 live 2 行（bootstrap-admin + 新会话用户）。

### Suggested Action
损坏检测拒绝静默 rebuild；`ensure_data_dir` 校验 InnoDB 文件；恢复走 dump 或 21:16 物理副本（需停库、sudo 拷贝）。不要把 ram-work 空壳绑回 3306。

### Metadata
- Source: conversation
- Related Files: dockerInfra/mysql/run.sh, dockerInfra/mysql/detect_corrupt_data.sh, /home/ljy/bin/ramsync-daemon.sh, runAll/scripts/materialize-p4-root.sh
- Tags: mysql, tmpfs, ramsync, p4-deploy, data-loss
- See Also: OPT-20260830-025, OPT-20260812-014

---

## [LRN-20260829-002] pitfall

**Logged**: 2026-08-29T21:45:00+08:00
**Priority**: high
**Status**: pending
**Area**: frontend / CSS grid / vendor portal

### Summary
固定列数的 CSS Grid 若子元素多于轨道，多出来的节点会掉进隐式第二行、占用第一列，表现为「操作按钮消失」。长字段（镜像 URL）必须 `grid-column: 1 / -1` 单独占下一行，不能和 `auto` 操作列抢同一条轨道。

### Details
- 厂商门户 `.version-row` 曾是 5 列，但行内有 id/version/skill/status/url/actions 六个子节点。URL 吃掉 `auto` 轨，按钮折到 ID 列。
- 同时 `withdraw` 只调用 `WithdrawPending()`，已上架行即使按钮可见也会 400。

### Suggested Action
版本行：操作列放 URL 之前；URL `grid-column: 1 / -1`。领域：`VendorWithdraw` 对 `approved` 走 `Unpublish` 并清 `is_active`；激活版本禁止删除。

### Metadata
- Source: conversation
- Related Files: taskAiProvider/frontend/src/views/VendorPortal.css, taskAiProvider/frontend/src/lib/vendorVersionRowActions.js, taskAiProvider/domain/entities.go
- Tags: css-grid, version-row, vendor-withdraw
- See Also: OPT-20260829-021, F-112, B-083

---

## [LRN-20260829-001] best_practice

**Logged**: 2026-08-29T11:25:00+08:00
**Priority**: high
**Status**: pending
**Area**: backend / cloud start-vm

### Summary
`start-vm` / `start-vm-auto` 在 HTTP ACK 时必须已经插入 `cloud_comment_container_bindings`，并把 comment CSC `csc_id` 写入 pending `cloud_server_events`。不能等任务详情页 `ensureCommentContainerBinding`。

### Details
- 案例：评论 `cmt_881353149581914112` 09:42 已 ACK 第一条 start event（无 `csc_id`、无 binding）；75 分钟后打开任务详情才 insert binding 并二次 `RunInstances`，阿里云 CreationTime 变成 10:57。
- 异步 `RunInstances` 失败只能 drain 到 binding；没有行时 UI 看起来像「评论很早、机器很晚」。
- 前端 `syncBindingsForComments` 是补偿不是主路径。

### Suggested Action
走共享 `finalizeStartVmInGo`：`ensureCommentBindingForStartVM` → bootstrap tokens → `fillStartVmEventCommentCSCID` → insert event → `markCommentBindingStartingAfterStartAck`。

### Metadata
- Source: conversation
- Related Files: taskCloudService/src/compute_start_vm_comment_scope.go, taskCloudService/src/compute_start_vm_finalize.go, taskFE/app/src/composables/taskDetail/useCommentContainerBindings.js
- Tags: start-vm-auto, comment-container-binding, creation-time-lag
- See Also: OPT-20260829-004, OPT-20260829-005

---

## [LRN-20260827-003] best_practice

**Logged**: 2026-08-27T01:55:00+08:00
**Priority**: high
**Status**: pending
**Area**: observability / gateway catalog

### Summary
Grafana `:3000` 适合叠加 **runtime**（Tempo service map、HTTP RED），**不能**当全站网址/接口 SSOT。既定拓扑必须从 `routes.yaml` + `api_route_ownership.yaml` + taskFE router 生成；「核心路径」要拆成 declared（价值流）与 empirical（QPS），禁止用流量高低代替业务主路径。

### Details
- 现成面板 Lightweight APM / HTTP API Latency / Distributed Trace View 已能看热路径与调用图。
- 零流量网关路由、SPA 页面、deny/null-upstream 孤儿路径在 Tempo 里不存在。
- 落地：`docs/architecture/service-url-api-catalog.md` + Grafana uid `service-url-api-catalog`。

### Suggested Action
改路由后跑 `python3 db/scripts/ci/build_service_url_api_catalog.py`；浏览用 Grafana catalog dashboard，不要手改生成物。

### Metadata
- Source: goal
- Related Files: db/scripts/ci/build_service_url_api_catalog.py, docs/superpowers/specs/2026-08-27-service-url-api-catalog-design.md
- Tags: grafana, catalog, gateway, core-path
- See Also: OPT-20260827-008, OPT-20260827-009

## [LRN-20260827-002] best_practice

**Logged**: 2026-08-27T01:40:00+08:00
**Priority**: high
**Status**: pending
**Area**: Git OAuth / Path A / provider resolve

### Summary
未命中 YAML 的自托管 Git 仓（尤其是 **裸 IP Path A GitLab**、URL 以 `.git` 结尾）**禁止**回退 `gitlab:default`。默认 CE 的 token 属于另一套实例，会表现为「未检测到可用授权」或 401。必须用租户 `gitlab:tenant-{company_id}`（catalog + host 对齐），前端 catalog 请求必须带 `company_id`。

### Details
- 项目页 `http://115.29.110.74/example-user/somanyad.git`：heuristic 不含 `gitlab` 子串 → 无 OAuth 按钮。
- 后端曾把未知 `*.git` 映射到 `gitlab:default`。
- 区域 GitLab 已有同类禁令（failure 108）；Path A IP 被漏掉。

### Suggested Action
改 provider 解析与 catalog 合并；克隆凭证 YAML miss 须走 `gitsite:{host}`（failure 120），禁止把已绑定 Path A 标成 missing。回归：`TestMatchProviderUnknownDotGitDoesNotBorrowDefault`、`TestGitOauthProvidersCatalogIncludesTenantPathA`、`TestBuildCredentialsPathAIPGitLabYAMLMissStillFetches`。

### Metadata
- Source: goal
- Related Files: taskProjectService/src/provider_resolver.go, taskGitOauth/src/providers_catalog.go, taskFE/app/src/utils/repoOAuthAuthorizeUtils.js, taskCredentialService/application/clone_provider.go
- Tags: gitlab, oauth, path-a, provider-key
- See Also: OPT-20260827-002, .ai/09_failure_experience/02_runtime_errors/118_path_a_ip_git_borrows_gitlab_default.md, 108_ztree_push_gitlab_borrow_other_ce_token.md, 120_fork_bound_clone_path_a_yaml_miss.md

## [LRN-20260827-001] best_practice

**Logged**: 2026-08-27T01:30:00+08:00
**Priority**: medium
**Status**: pending
**Area**: /5-nfr skill / Redis cache

### Summary
`/5-nfr` 把 Redis 用在幂等键持久化备选和分片 key 前缀上，没有把 Redis 当读路径缓存来设计。硬门禁不强制缓存是对的；缺角色拆分（缓存 vs pub/sub vs 锁 vs 总线）会让 Agent 把 Redis 写成 MQ，读缓存则默认 locmem。仓内可抄模式是 PDP：进程缓存 + `membership_rev` 失效 + Redis 宕机降级 TTL。

### Details
- 技能 713 行、3 处 Redis/缓存命中；性能类别无 TTL/命中率/stampede。
- 约 279 份 NFR：7% 提 Redis、1.4% 同时写缓存；Redis 条目以 docker-redis / Streams 为主。
- 审计全文：`docs/superpowers/specs/2026-08-27-5-nfr-redis-cache-audit.md`。

### Suggested Action
补可选附录，不要升 Hard Gate。见 OPT-20260827-004 / 005。

### Metadata
- Source: goal
- Related Files: .claude/skills/5-nfr/SKILL.md, docs/superpowers/specs/2026-08-27-5-nfr-redis-cache-audit.md
- Tags: nfr, redis, cache, skill-design
- See Also: OPT-20260827-004, OPT-20260827-005

## [LRN-20260823-001] ops_forensics

**Logged**: 2026-08-23T06:55:00+08:00
**Priority**: high
**Status**: completed
**Area**: docker-mysql / nightly cron / Cursor agents

### Summary
mysqld CPU 爆表时「多个测试进程并行建库重放 DDL」不是线上流量，也不是第二份 sweep 漏锁。外部触发是 **ljy crontab 夜间窗口**叠 **多路 Cursor `/goal` 提交门禁** 和 **OPT 无头执行器**。sweep 自身 flock 有效（06:20 触发 1 秒退出）；并行来自不同父进程同时 `go test ./src`。

### Details
- crontab：`*/20 0-7 * * * nightly-test-sweep.sh`、`*/30 0-7 * * * nightly-opt-runner.sh`。journal：`2026-08-23T06:00:01` 两任务同时拉起。
- sweep 06:00:01–06:39:08 三轮，`taskBill`/`taskAuth`/`taskAIComment` 等 `picked=1` 即整包 `go test -count=1 ./src/...`（含 TestMarkOrderPaid）。R3 `FIXED=1` → `88eeb3c` 06:35:54，修复 agent 再跑一遍全包。
- 同时段交互会话提交（均走 pre-commit 全包）：taskAuth `104832e` 06:19:44、taskCloud `18e20db` 06:26:49（会话记录整包 394s）、taskBill `bd4748a` 06:30:06。
- OPT runner 06:00–06:12 与 06:30–06:42；meta auto-commit 触发 `run_commit_random_unit_tests.py`。
- 进程树现证：`cron → sh -c nightly-test-sweep.sh → python3 nightly_test_sweep.py → random_test_runner.sh run-go`。

### Suggested Action
再遇本机 mysqld CPU 飙高：先 `pgrep -af 'nightly_test_sweep|nightly-opt-runner|go test'` 和 `crontab -l`，不要先猜业务慢查询。互斥只在同类 cron 之间，交互会话与 sweep 无共享 MySQL 测试锁。

### Metadata
- Source: goal
- Related Files: scripts/nightly-test-sweep.sh, scripts/nightly_test_sweep.py, scripts/nightly-opt-runner.sh, logs/nightly-test-sweep-report-20260823.md
- Tags: mysql, cpu, nightly-test-sweep, cron, pre-commit, go-test
- See Also: OPT-20260823-019, OPT-20260823-020

## [LRN-20260822-003] bug_fix

**Logged**: 2026-08-22T21:20:00+08:00
**Priority**: high
**Status**: completed
**Area**: taskGitOauth / GitLab OAuth refresh

### Summary
GitLab `/oauth/token` 与 GitHub 共用 `ResponseHeaderTimeout=4s`。上海 2GiB GitLab 常 >4s 才回响应头，但 Doorkeeper 已旋转 refresh。客户端超时后新 refresh 无法落库，DB 仍持作废 token，后续全部 `invalid_grant`。nested-git 的 sanitize 还把含 `/oauth/token` 的超时文案误判成授权失效。

### Details
GitLab 独立 `gitlabHTTPClient`（25s header timeout）；`sanitizeGitOauthResolveError` 先匹配 timeout。运维可在自有 GitLab 用 `OauthAccessToken.create!` 补发 refresh 写入 `git_oauth_appusercredential`，无需用户重新点 OAuth。

### Suggested Action
再遇 `invalid_grant`：先查 `oauth_access_tokens` 是否刚有新行而 MySQL `updated_at` 未变；再查 git-oauth 是否 `timeout awaiting response headers`。

### Metadata
- Source: goal
- Related Files: taskGitOauth/infrastructure/outbound_http.go, taskProjectService/src/gitoauth_client.go
- Tags: gitlab, oauth, refresh_token, ResponseHeaderTimeout, invalid_grant
- See Also: LRN-20260822-002, OPT-20260822-050, taskGitOauth@ae86e75

## [LRN-20260822-002] bug_fix

**Logged**: 2026-08-22T20:50:00+08:00
**Priority**: high
**Status**: completed
**Area**: taskGitOauth / GitLab OAuth refresh

### Summary
任务自动运行评论泄漏 `gitlab refresh http 400` 不是「缺 client_secret」单一原因。当时运行中的 taskGitOauth 二进制早于补齐 `redirect_uri` 的提交；GitLab Doorkeeper 对缺 `redirect_uri` 的 refresh 直接 400。并行 probe / access-for-user 的 `GetCredentialByIDForUpdate` 原先没有 `FOR UPDATE`，refresh 旋转会把仍有效的 refresh 打成 `invalid_grant`。旧进程错误换票窗口过后，存量 refresh 无法用代码救活，必须用户重新绑定。

### Details
修复：`RefreshGitLabToken` 空 `redirect_uri` 禁止发 HTTP；`GetCredentialByIDForUpdate` 真正行锁；probe/access-for-user/merge 收口 `issueAccessTokenFromCredential`。人性化文案：「Git OAuth 授权已失效，请重新绑定」。部署后若日志已是 `invalid_grant`，不要再猜缺字段。

### Suggested Action
再遇 GitLab refresh 400：先看 Loki 是否带 `invalid_grant` 后缀、进程启动时间是否晚于含 `redirect_uri` 的二进制。`invalid_grant` → 重新绑定 + `force_auto_run`（OPT-20260822-050）。

### Metadata
- Source: goal
- Related Files: taskGitOauth/infrastructure/oauth_gitlab.go, taskGitOauth/infrastructure/db.go, taskGitOauth/src/issue_access_token.go
- Tags: gitlab, oauth, refresh_token, invalid_grant, FOR UPDATE
- See Also: OPT-20260822-030, OPT-20260822-050, taskGitOauth@5ffd6a5

## [LRN-20260822-001] correction

**Logged**: 2026-08-22T17:47:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend / session-forensics

### Summary
用户纠正：系统管理顶栏「今天才看不见」时，只查当天下午会话，不要把 8 月 9 日引入 `v-if="!navbarCollapsed"` 的提交当成「今天下午引进的会话」。代码引进日 ≠ 用户感知日。

### Details
今天下午证据：17:00 Profile 会话选择器含可见 `nav.bg-white/90`；17:06 进入带「控制台导航」的租户任务详情；17:33 报 `/system-admin/` 顶栏丢失。收起态写在全局 localStorage，跨页带到没有恢复入口的系统管理页。

### Suggested Action
追「哪次会话引进」时先用用户给出的时间窗过滤 transcript；下午才出现的症状优先对齐当天下午页面跳转与持久化状态，而不是 git blame 最早提交。

### Metadata
- Source: conversation
- Related Files: taskFE/app/src/utils/navbarCollapsedVisibility.js, taskFE/app/src/components/Navbar.ui.vue
- Tags: navbar, localStorage, session-forensics
- See Also: 6c9d7249-0907-4b43-9e9a-11d2c7328ece, 984ea2f6-ad72-45fb-8d71-82289f0cdfb7

## [LRN-20260821-001] bug_fix

**Logged**: 2026-08-21T18:50:00+08:00
**Priority**: high
**Status**: completed
**Area**: taskCloudService / taskTaskService / taskEvents

### Summary
任务改「已取消」后看板机器绿环仍亮：`TASK_STATUS_CHANGED` 的 release-servers 在 cloud `:8018` connection refused 时重试耗尽进 DLT，SSE 扇出仍成功所以 UI 进度已变。孤儿对账只查任务是否存在，取消任务仍存在故不释放。heartbeat 恢复后继续把 CSC 标 Running。补偿：terminal-kinds API + 入站 410 同步释放 + reconcile 同 tick 清终态 CSC。另：`markTerminalReleased` 曾只 upsert 空 comment 模板行，评论级 `instance_id` 仍在，`machineRuntimeCountsAsStarted` 继续为真。

### Metadata
- Source: goal
- Related Files: taskCloudService/src/inbound_terminal_guard.go, taskCloudService/src/terminal_release_migrate.go, taskTaskService/src/internal_tasks_terminal.go
- Tags: terminal-release, dlt, heartbeat, work-panel
- See Also: docs/intents/backend/cloud/terminal_release_comment_csc.intent.md

## [LRN-20260722-001] bug_fix

**Logged**: 2026-07-22T00:40:00+08:00
**Priority**: high
**Status**: completed
**Area**: taskCloudService / taskGitOauth / git-push

### Summary
ztree 推送 `unauthorized` 而 UI 显示「OAuth 已授权」：Cloud prepare→access-for-user 缺 `X-GitOauth-Bridge-Secret`（bridgeSecret 空且无回退），gitOauth 401；Django summary 有 secret 故绿标。修复：对齐 bridgeSecret + 与 taskProjectService 相同回退链；runAll 注入 `GITOAUTH_BASE_URL` loopback。

### Metadata
- Source: goal
- Related Files: conf/taskCloudService/config.yaml, taskCloudService/src/config.go, taskCloudService/src/git_push_internal.go
- Tags: git-push, oauth, bridge-secret
- See Also: .ai/09_failure_experience/02_runtime_errors/74_ztree_push_unauthorized_gitoauth_bridge_secret.md

## [LRN-20260720-004] bug_fix

**Logged**: 2026-07-20T16:25:00+08:00
**Priority**: high
**Status**: completed
**Area**: taskCloudService / frontend / git-push

### Summary
ztree 推送报 `Username for https://github.com: terminal prompts disabled`：多仓 prefer_container_remote 无 OAuth 时回退裸 git/push。修复：默认 409；仅 allow_bare_git_push；多仓传 repo_url hint。

### Metadata
- Source: goal
- Related Files: taskCloudService/src/git_push_internal.go, taskDetailLayerActions.js
- Tags: git-push, oauth, ztree
- See Also: .ai/09_failure_experience/02_runtime_errors/67_ztree_push_terminal_prompts_disabled.md

## [LRN-20260720-003] bug_fix

**Logged**: 2026-07-20T16:25:00+08:00
**Priority**: high
**Status**: completed
**Area**: onlineServiceJS / auto_run / delivery

### Summary
auto_run 首指令 completed 后 ztree 仍「可推送」：交付失败也写 `auto_run_delivery.done` 锁死重试；`runtime-event` 未转发 Cloud → Django 404。修复：仅成功/干净跳过写 done、嵌套 commit、启动补跑、转发 runtime-event、OAuth 返回 pr_base。

### Metadata
- Source: goal
- Related Files: trae-agent/onlineServiceJS/src/autoRunOrchestration.mjs, autoRunDeliveryHooks.mjs, taskAgentSupport/src/handlers.go
- Tags: auto_run, delivery, push, runtime-event
- See Also: .ai/09_failure_experience/02_runtime_errors/66_auto_run_delivery_done_locks_unpushed.md

## [LRN-20260720-002] refactor

**Logged**: 2026-07-20T15:20:00+08:00
**Priority**: medium
**Status**: completed
**Area**: onlineServiceJS / observability

### Summary
OPT-033/034：bootstrap/server 拆至 ≤500；容器 BOOTSTRAP_/AUTO_RUN_ 经 runtime-event → Cloud LogForwardStage 进 Loki。

### Metadata
- Source: goal
- Related Files: trae-agent/onlineServiceJS/src/runtimeEventLog.mjs, taskCloudService/src/container_runtime_event.go
- Tags: line-limit, loki, auto_run

## [LRN-20260720-001] bug_fix

**Logged**: 2026-07-20T14:20:00+08:00
**Priority**: high
**Status**: completed
**Area**: onlineServiceJS / auto_run

### Summary
auto_run 在 `repo-clone-credentials` 先 409、后经恢复克隆成功时，不会启动首条 Agent：kickoff 只挂在 listen 后主路径，恢复成功分支漏调用。修复：共用 `postBootstrapAgentKickoff` + `kickoffAfterCredentialsRecovery`；残缺 at_mention 回退 auto_run。

### Metadata
- Source: goal
- Related Files: trae-agent/onlineServiceJS/src/postBootstrapAgentKickoff.mjs, bootstrap.mjs, server.mjs
- Tags: auto_run, credentials-recovery, bootstrap
- See Also: .ai/09_failure_experience/02_runtime_errors/58_auto_run_skipped_after_credentials_recovery.md

## [LRN-20260719-006] bug_fix

**Logged**: 2026-07-19T15:55:00+08:00
**Priority**: high
**Status**: completed
**Area**: onlineServiceJS / frontend

### Summary
文件变动「目录扫描已达上限」：`collectIndex` 未跳过 node_modules 占满 4000；截断琥珀提示补 `data-traceId`（`fetch_trace_id`）。

### Metadata
- Source: goal
- Related Files: trae-agent/onlineServiceJS/src/layerParentDiffCollect.mjs, TaskDetailExecLayerChangesHints.vue
- Tags: layer-diff, data-traceId, scan-cap
- See Also: .ai/09_failure_experience/02_runtime_errors/55_layer_changes_scan_cap_node_modules_and_trace_id.md

## [LRN-20260719-005] bug_fix

**Logged**: 2026-07-19T15:50:00+08:00
**Priority**: medium
**Status**: completed
**Area**: frontend / task-detail

### Summary
任务详情「文件变动」分栏上下堆叠：根因是 `flex-col` + `md:flex-row`（视口 &lt;768 或桌面期望不符）。改为固定 `flex-row`，左栏内联 width 兜底。

### Metadata
- Source: goal
- Related Files: taskFE/app/src/components/ResizableSplitPane.vue
- Tags: split-pane, layout
- See Also: .ai/09_failure_experience/02_runtime_errors/54_task_detail_split_pane_stacked_not_row.md

## [LRN-20260719-004] bug_fix

**Logged**: 2026-07-19T15:35:00+08:00
**Priority**: high
**Status**: completed
**Area**: onlineServiceJS / taskContainerGateway

### Summary
任务详情文件树选文件预览恒 `not found`：children 相对 primary 无仓库前缀，而 files/* 要求带前缀；TCGW 整段 PathEscape 把 `/` 编成 `%2F`。

### Details
修复：`listLayerChildren` 对齐多仓前缀；`resolveAbsolutePathForLayerListedFile` 解码 `%2F` 并兼容无前缀；TCGW `pathEscapeRelPosix` 按段编码。

### Metadata
- Source: goal
- Related Files: trae-agent/onlineServiceJS/src/layerChildren.mjs, trae-agent/onlineServiceJS/src/layerFs.mjs, taskContainerGateway/src/l0_registry.go
- Tags: file-tree, preview, path-prefix
- See Also: .ai/09_failure_experience/02_runtime_errors/53_task_detail_file_preview_not_found_path_prefix.md

## [LRN-20260719-003] best_practice

**Logged**: 2026-07-19T05:30:00+08:00
**Priority**: high
**Status**: completed
**Area**: backend / ownership-migration

### Summary
人员邀请/成员/分组从 Django 迁 Go 时，勿把 `accounts_company` 一并搬走；四表归 `taskTenantService`，公司 creator 经 Django internal HTTP，成员写入经 `tenant_client`，公网 ViewSet 卸除后 gateway 白名单必须同步。

### Details
CompanyMember 在 saas 深度耦合 → Go SSOT + ORM 过渡回退，DROP 旧表需运维窗口。workspace-access 创建要传 `company_member_id`。事件首版可 log stub，但意图文档须声明映射。

### Metadata
- Source: goal
- Related Files: taskTenantService/, task2app/Saas_project/accounts/tenant_client.py, taskGateway/routes/routes.yaml
- Tags: task-tenant, people, go-migration, single-service-ownership

## [LRN-20260719-002] bug_fix

**Logged**: 2026-07-19T04:45:00+08:00
**Priority**: high
**Status**: completed
**Area**: gateway / taskProjectService

### Summary
`GET /api/tenant/*/daydaymoney/resolve` 经公网 404：Go 已实现，但 APISIX `task-project-service` 白名单漏登 `daydaymoney`，请求落入 Django `ApiJson404Middleware`。

### Details
对照：直连 `:8016` → 200；经 `:18081` / example.com → Django `detail`+`path` 404。修复：`routes.yaml` 增加 `/api/tenant/*/daydaymoney*` 后 `routes-apply`（须 `TASK_GATEWAY_APISIX_IN_DOCKER=1`）。另：项目 tags 缺 `svc:task2app` 时 matches 为空，已为 ram-work 项目补标。

### Metadata
- Source: goal
- Related Files: taskGateway/routes/routes.yaml, taskProjectService/src/daydaymoney_handlers.go
- Tags: gateway, daydaymoney, routes, 404
- See Also: .ai/09_failure_experience/02_runtime_errors/51_gateway_daydaymoney_resolve_404.md, LRN related to 21_gateway_access_tokens_post_405

## [LRN-20260719-001] bug_fix

**Logged**: 2026-07-19T03:55:00+08:00
**Priority**: high
**Status**: completed
**Area**: frontend / nested-repos

### Summary
子仓 staging→移入父仓后，「子仓库克隆状态」全显「未开始」：根因是全局完成后清空 SSE 进度 Map，且 `bootstrapCloneDone` 从未接线。

### Details
状态不查磁盘；文件树有子仓目录只说明 clone/relocate 成功。修复：接线引导日志完成态；无进度时解析日志段「已移入」/失败；容器就绪拉取 `container-bootstrap-clone-log`；空 SSE text 不覆盖已有日志。

### Metadata
- Source: goal
- Related Files: taskFE/app/src/utils/nestedRepoCloneStatusUtils.js, taskFE/app/src/composables/taskDetail/taskDetailCloneProgress.js, taskFE/app/src/components/task-detail/TaskDetailNestedReposCloneStatus.vue
- Tags: nested-repos, clone-status, bootstrap-clone-log
- See Also: .ai/09_failure_experience/02_runtime_errors/50_nested_repo_clone_status_idle_after_relocate.md

## [LRN-20260718-003] bug_fix

**Logged**: 2026-07-18T16:26:00+08:00
**Priority**: high
**Status**: completed
**Area**: container-token

### Summary
容器重启后预埋 ACCESS_TOKEN 已作废时，exchange-refresh 401 未回退 refresh-access，导致任务详情全线「Invalid or missing access token」。

### Details
16:03:26 仍 upstream 200，16:03:27 exchange-refresh 401 后心跳中断、全线 401。修复：credential 在 scope 已有 refresh 时对废弃 access 返回 403 AlreadyDone；onlineServiceJS 对 403/401 均可凭落盘 refresh 回退 refresh-access；并修正 403 判定不依赖文案中的 `refresh-access` 子串。

### Metadata
- Source: goal
- Related Files: trae-agent/onlineServiceJS/src/bootstrap.mjs, taskCredentialService/application/services.go
- Tags: access-token, exchange-refresh, clone-log
- See Also: .ai/09_failure_experience/02_runtime_errors/44_task_detail_invalid_access_token_after_container_restart.md

## [LRN-20260718-002] bug_fix

**Logged**: 2026-07-18T16:20:00+08:00
**Priority**: high
**Status**: completed
**Area**: frontend

### Summary
exec-log 轮询不得无条件 bump 项目文件树；提交日志预览失败须挂 `data-traceId`。

### Details
`ingestLayerChangesFromExecutionPayload` 被 `activeJobExecLogPoller`（2s）反复调用并 bump nonce，导致文件树持续 `fetchFiles` 闪烁。修复为 fingerprint 变化才 bump。`container-layer-git-log` 401 的「Invalid or missing access token」此前只抛 Error(detail)，预览错误节点无 `data-traceId`。

### Metadata
- Source: goal
- Related Files: taskFE/app/src/composables/taskDetail/taskDetailZTreeExecLogState.js, taskFE/app/src/components/task-detail/TaskDetailProjectFileTree.vue, taskFE/app/src/components/task-detail/TaskDetailExecLayerChangePreview.vue
- Tags: file-tree, polling, data-traceId, performance
- See Also: .ai/09_failure_experience/02_runtime_errors/47_task_detail_file_tree_poll_refresh_and_git_log_trace_id.md

## [LRN-20260718-001] best_practice

**Logged**: 2026-07-18T11:12:00+08:00
**Priority**: high
**Status**: pending
**Area**: backend

### Summary
`cloud/compute/*` 经 APISIX 打到 taskCloudService 后，OpenAPI/Django 已有的写接口若未在 `handleCloudTaskRoutes` 登记，会统一 501「not yet ported」；前端无 `detail` 时只显示兜底文案（如「保存 GitHub 账号失败」），易误判为业务失败。

### Details
任务 `task_13419614134018489866` 保存 GitHub 账号时，`github-credential-status` 已代理，但 `github-credential-approve` 未注册 → 501。修复：对称登记 `handleGithubCredentialApprove` 并 `proxyDjangoRequest`；前端同时读 `detail`/`message` 并挂 `data-traceId`。

### Suggested Action
迁服时对 status/list 与 approve/create/delete 成对登记；curl/网关日志确认非 501。

### Metadata
- Source: goal
- Related Files: taskCloudService/src/compute_handlers.go, taskFE/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue
- Tags: 501, github-credential-approve, route-gap, taskCloudService
- Pattern-Key: harden.compute_route_pair_registration
- See Also: .ai/09_failure_experience/02_runtime_errors/45_github_credential_approve_501_not_ported.md

## [LRN-20260717-009] best_practice

**Logged**: 2026-07-17T18:15:00+08:00
**Priority**: high
**Status**: pending
**Area**: infra

### Summary
runAll 批量 `build failed: exit status 1` 时先查 `/tmp/ram-work` tmpfs 是否已满，勿直接归因代码编译错误。

### Details
十余个 `task-events-*` 与 `task-task-service` / `task-cloud-service` / `task-ai-comment` 同时失败；`go build` 底层为 `no space left on device`。根分区仍有空间，但工作区 tmpfs（约 10G）被日志与二进制占满。截断 `logs/` 与 `taskGateway/logs/` 后重建全部成功。

### Suggested Action
runAll build 前增加磁盘余量预检；维护脚本定期截断大 access log。

### Metadata
- Source: conversation
- Related Files: taskEvents/run.sh, runAll/src/runner.go, conf/runAll.yaml
- Tags: tmpfs, disk-full, build-failed, runAll, taskEvents
- Pattern-Key: harden.build_disk_preflight
- See Also: .ai/09_failure_experience/01_compilation_errors/02_ram_work_tmpfs_no_space_build_failed.md

## [LRN-20260717-008] best_practice

**Logged**: 2026-07-17T16:20:00+08:00
**Priority**: high
**Status**: pending
**Area**: frontend

### Summary
inline 请求错误除文案外须同步挂载 `data-traceId`；`apiFetch` 已有 `response.traceId` 时禁止只写 `detail`。

### Details
任务详情拉取 `container-clone-log` 401（`Invalid or missing access token`）时，`TaskDetailExecCloneLogSection` 的 `p.text-red-600` 无 `data-traceId`。根因是 `refreshZTreeExecutionLog` 未读取 `r.traceId`、组件链未下传。修复：traceId ref + 模板绑定。

### Suggested Action
扫任务详情其余 `text-red-600` 自绘错误路径；新增失败 UI 对照元规则 24 验收。

### Metadata
- Source: conversation
- Related Files: taskFE/app/src/composables/taskDetail/taskDetailExecLog.js, taskFE/app/src/components/task-detail/TaskDetailExecCloneLogSection.vue
- Tags: data-traceId, task-detail, clone-log, observability
- Pattern-Key: harden.request_error_dom_data_trace_id
- See Also: .ai/09_failure_experience/02_runtime_errors/38_task_detail_clone_log_error_missing_data_trace_id.md

## [LRN-20260717-007] best_practice

**Logged**: 2026-07-17T15:30:00+08:00
**Priority**: high
**Status**: pending
**Area**: backend

### Summary
展示列依赖的嵌套对象须含 UI 字段；CREATE/PATCH 禁止 stub 200；同模态提交的关联字段须同路径落库。

### Details
厂商门户保存区域运行环境后 UserData 列仍「—」：ListAssociations 只回 `{id}`；CSI create 把 userdata_template_id 写死 NULL、PATCH 空转；关联 POST 忽略 userdata_template_id。修复为 JOIN 回 name/version、真正持久化、关联保存时 SetCloudServerImageUserdataTemplate。

### Suggested Action
列表契约测试覆盖展示字段；表单可选字段与 SQL 写入同批测例。

### Metadata
- Source: conversation
- Related Files: taskAiProvider/infrastructure/store_marketplace.go, taskAiProvider/src/vendor_container_actions.go, taskAiProvider/frontend/src/utils/userdataTemplateDisplay.js
- Tags: userdata, association, vendor-portal, display-contract
- Pattern-Key: harden.list_enrich_and_persist_form_fields
- See Also: .ai/09_failure_experience/02_runtime_errors/36_vendor_userdata_cell_dash_after_save.md

## [LRN-20260716-001] best_practice

**Logged**: 2026-07-16T04:29:35.412608+00:00
**Priority**: high
**Status**: pending
**Area**: backend

### Summary
CLOUD_SERVER_STOPPED 幂等键不能用 company_id，否则同租户后续停止被静默跳过。

### Details
IdempotencyKeyFromEnvelope 字段序曾把 company_id 放在 task_id 之前；停止事件载荷始终含 company_id，导致 MemoryStore.Seen 在同租户首次成功停止后吞掉所有后续停止。日志表现为 received→dispatch_ok 且无 stop_vm_begin。

### Suggested Action
域事件幂等键与业务重复边界同粒度；为每次用户触发的停机写入 stop_request_id。

### Metadata
- Source: conversation
- Related Files: taskEvents/consumer/key.go, taskEvents/internal/handlers/cloudserverstopped/handler.go
- Tags: kafka, idempotency, stop-vm, mock
- Pattern-Key: harden.idempotency_key_grain

## [LRN-20260716-002] best_practice

**Logged**: 2026-07-16T04:37:39.052596+00:00
**Priority**: medium
**Status**: pending
**Area**: backend

### Summary
启动类事件须专用幂等键（event_id），且 START_AUTO 链式 STARTED 要回填 event_id。

### Details
审计确认正常 start finalize 已带 event_id；但缺省时旧逻辑仍可能落到 task_id/company_id。已对 STARTED/START_AUTO 做专用键，并在 chain 前从 pending event 回填 event_id。

### Suggested Action
新增云域事件时先定义幂等粒度字段，禁止默认扫 company_id。

### Metadata
- Source: conversation
- Related Files: taskEvents/consumer/key.go, taskEvents/internal/handlers/cloudserverstartauto/handler.go
- Tags: kafka, idempotency, start-vm
- Pattern-Key: harden.idempotency_key_grain
- See Also: LRN-20260716-001

## [LRN-20260716-003] best_practice

**Logged**: 2026-07-16T05:12:00+00:00
**Priority**: medium
**Status**: pending
**Area**: backend

### Summary
列表 API 若前端读 `billing_unit.name`，后端必须返回嵌套对象，不能只回 FK 字符串。

### Details
taskBill transactions/usages 曾把 `billing_unit` 序列化为 ID 字符串；Vue 用 optional chaining 取 `.name` 得到 undefined，整列显示「-」。充值无 unit 为预期；消费有 unit_id 却仍显示「-」才是契约破坏。

### Suggested Action
对外展示字段做「前端表达式可解析」的契约测试；FK 展示名在 owner 服务内 join/展开。

### Metadata
- Source: conversation
- Related Files: taskBill/src/handlers.go, taskFE/app/src/views/BillingTransactions.vue
- Tags: billing, api-contract, serialization
- Pattern-Key: harden.api_nested_display_object

## [LRN-20260716-004] best_practice

**Logged**: 2026-07-16T05:15:00+00:00
**Priority**: medium
**Status**: pending
**Area**: backend

### Summary
用量列表须与交易列表共用 enrich，否则用户列只显示成员 ID。

### Details
BillingUsage.vue 已写 `user_name || user_id`；taskBill `handleUsagesList` 未调用 `djangoEnrichTransactions`，而 `list_filtered` 交易列表已调用。`user_id` 常为 company_member_id，缺 `user_name` 时整列落成裸 ID。

### Suggested Action
新增/对照 billing 展示列表时默认走同一 enrich；用「返回含 user_name」的契约测试防回归。

### Metadata
- Source: conversation
- Related Files: taskBill/src/handlers.go, taskFE/app/src/views/BillingUsage.vue
- Tags: billing, usage, display-name, enrich
- Pattern-Key: harden.billing_list_enrich_parity

## [LRN-20260716-004] best_practice

**Logged**: 2026-07-16T05:25:00+00:00
**Priority**: medium
**Status**: pending
**Area**: backend

### Summary
充值流水也需要 billing_unit：种子展示单元 + 写入路径设 FK + 历史回填，不能只修列表序列化。

### Details
嵌套展开后消费正常，但 recharge INSERT 未写 billing_unit_id，且无 recharge 单元行。004 迁移种子「积分充值」并 UPDATE 历史；creditRecharge 在事务前 ensureBillingUnit。

### Suggested Action
新增交易类型时同步定义展示用 billing_unit 与写入契约；历史 NULL FK 用迁移回填。

### Metadata
- Source: conversation
- Related Files: taskBill/src/credit.go, taskBill/migrations/004_seed_recharge_billing_unit.sql
- Tags: billing, recharge, api-contract
- Pattern-Key: harden.api_nested_display_object
- See Also: LRN-20260716-003

## [LRN-20260716-005] best_practice

**Logged**: 2026-07-16T05:35:00+00:00
**Priority**: high
**Status**: pending
**Area**: frontend

### Summary
Chrome 扩展 Popup 主路径（登录）不得无超时 await chrome.storage；登录 API 不得附带内存脏 Authorization。

### Details
Popup 启动态已对 storage 加超时，但 handleTokenLogin 仍先 await saveBaseUrl，storage 挂起时点击登录无任何反馈。同时 API.init(undefined token) 不清空内存 token，login 走通用 request 会带旧 Authorization。

### Suggested Action
用户触发的关键操作：先更新 UI，storage 带 withTimeout，失败继续；公开登录用 requestUnauthenticated。

### Metadata
- Source: conversation
- Related Files: taskChromePlugin/popup/popup.js, taskChromePlugin/lib/api.js, taskChromePlugin/background/service-worker.js
- Tags: chrome-extension, popup, login, storage
- Pattern-Key: harden.popup_storage_timeout
- See Also: .ai/09_failure_experience/02_runtime_errors/20_chrome_plugin_popup_login_storage_hang.md

## [LRN-20260716-006] best_practice

**Logged**: 2026-07-16T05:40:00+00:00
**Priority**: medium
**Status**: pending
**Area**: infra

### Summary
taskAuth 新增公网方法时须在网关 routes.yaml 显式登记；仅 GET 的 /users/* 通配不会覆盖 POST。

### Details
POST /api/accounts/users/access-tokens/ 落入 django-default 返回 405，直连 :8003 正常。前端走 /profile/access-tokens/ Django 代理故未被发现。

### Suggested Action
Go 服务新 handler + 网关路由 + api_route_ownership 三件套同 PR；用 18081 做方法级冒烟。

### Metadata
- Source: conversation
- Related Files: taskGateway/routes/routes.yaml, taskAuth/src/handlers.go
- Tags: apisix, gateway, access-tokens, routing
- Pattern-Key: harden.gateway_route_method_coverage
- See Also: .ai/09_failure_experience/02_runtime_errors/21_gateway_access_tokens_post_405.md

## [LRN-20260716-007] best_practice

**Logged**: 2026-07-16T06:45:00+00:00
**Priority**: medium
**Status**: pending
**Area**: backend

### Summary
列表嵌套展示对象只放 UI 字段；enrich 只传 ID，名称回填到原行，避免嵌套体与余额字段双向放大。

### Details
usages/transactions 每行嵌套完整 billing_unit（含 price_points/is_active）再整页 POST 给 enrich-transactions，请求体随页膨胀。UI 仅需 name/unit；Django 仅需 user/project/workspace/task id。

### Suggested Action
嵌套 FK 展示用 display-shaped 子集；enrich 用 slimRows + mergeDisplayNames；非 billing 的 projects batch-get 保持 lite=False 默认拿完整 detail。

### Metadata
- Source: conversation
- Related Files: taskBill/src/handlers.go, taskBill/src/django_client.go, task2app/Saas_project/projects/go_client.py
- Tags: billing, payload-size, enrich, batch-get
- Pattern-Key: optimize.list_nested_display_slim_enrich

## [LRN-20260716-008] best_practice

**Logged**: 2026-07-16T06:50:00+00:00
**Priority**: low
**Status**: pending
**Area**: frontend

### Summary
大屏「最近交易」须带 page/page_size，勿依赖无分页大 LIMIT 再 slice。

### Details
Dashboard 曾拉 `/billing/transactions/` 全量后 `slice(0,5)`；无分页上限下调后仍应显式 `page=1&page_size=5`，并兼容 `{results}` 分页包络。

### Suggested Action
列表「预览 N 条」调用方默认走服务端分页；前端大页数用省略号分页组件，禁止 `v-for="page in totalPages"`。

### Metadata
- Source: conversation
- Related Files: useBillingDashboard.js, BillingPaginationNav.vue, taskBill/src/handlers.go
- Tags: billing, pagination, dashboard
- Pattern-Key: optimize.list_preview_use_server_page

## [LRN-20260716-009] best_practice

**Logged**: 2026-07-16T09:42:00+00:00
**Priority**: medium
**Status**: pending
**Area**: frontend

### Summary
账单套餐卡勿硬编码计费项子集；须与 tenant_pricing_view 字段对齐（含 GitLab 磁盘/流量），并确认运行中的 taskBill 二进制非旁路 worktree。

### Details
路由页 BillingDashboard 曾内联只渲染 4 类价目，API 已返回 gitlab_disk_points_per_gb_per_month 却不展示；拆分组件 BillingDashboardPricing 已含该项但未接入。

### Suggested Action
新增计费项时同步：后端 JSON → pricingPackageDisplay helper → 当前套餐/可切换卡/切换弹窗；优先复用已拆分组件而非再写一份硬编码 dl。

### Metadata
- Source: conversation
- Related Files: BillingDashboard.vue, BillingDashboardPricing.vue, pricingPackageDisplay.js, SwitchPricingPackageModal.vue
- Tags: billing, pricing, gitlab-disk, frontend
- Pattern-Key: harden.pricing_ui_field_parity

## [LRN-20260716-010] best_practice

**Logged**: 2026-07-16T12:10:00+00:00
**Priority**: medium
**Status**: pending
**Area**: infra

### Summary
改 Django 视图后须 `runall-saas-backend.sh restart`，勿只 `kill -HUP` gunicorn。

### Details
HUP 后 worker 可能仍返回旧 JSON（缺新字段）。完整 stop（按 :8001 清理）再 start 才能加载新模块。

### Suggested Action
视图/路由变更验收前执行 bash task2app/scripts/runall-saas-backend.sh restart。

### Metadata
- Source: conversation
- Related Files: task2app/scripts/runall-saas-backend.sh, run.sh.ai.md
- Tags: gunicorn, django, reload
- Pattern-Key: harden.django_full_restart_not_hup

## [LRN-20260716-023] bug_fix

**Logged**: 2026-07-16T13:00:00.000000+00:00
**Priority**: high
**Status**: pending
**Area**: backend

### Summary
厂商门户 SSO「bridge 缺少 sub」：主站签发 string sub，Go claimInt64 未解析 string。

### Details
sso_bridge_token.py 按 ID 字符串规范写 `"sub": str(user_id)`；ExchangeBridge 用 claimInt64(sub)，原先只认 float64/int/json.Number，导致误报缺少 sub。

### Suggested Action
跨服务 JWT/claim 的业务 ID 解析必须接受 string；SSO 回归覆盖 string sub。

### Metadata
- Source: conversation
- Related Files: taskAiProvider/infrastructure/config_jwt.go, task2app/Saas_project/accounts/sso_bridge_token.py
- Tags: sso, jwt, snowflake, ai-provider
- Pattern-Key: harden.jwt_string_id_claims


## [LRN-20260716-024] best_practice

**Logged**: 2026-07-16T14:15:00.000000+00:00
**Priority**: high
**Status**: pending
**Area**: backend

### Summary
Go 默认 http.Client 继承失效本地 SOCKS（127.0.0.1:1234）会导致 registry 拉取 proxyconnect refused。

### Details
resolve-target-architectures 用默认 Client 拉 Aliyun OCI manifest；开发机/runAll 父进程常带 https_proxy=socks5h://127.0.0.1:1234，代理未起时 detail 返回 proxyconnect connection refused。runAll use_proxy=false 会 strip 子进程代理，但 Client 侧仍应显式 Proxy=nil 作纵深防御。

### Suggested Action
访问公有云/国内 registry 的 http.Client Clone DefaultTransport 后设 Proxy=nil；启动脚本 unset 代理；回归测死代理环境。

### Metadata
- Source: conversation
- Related Files: taskAiProvider/infrastructure/registry_manifest.go, taskAiProvider/run.sh
- Tags: proxy, registry, ai-provider, http-client
- Pattern-Key: harden.http_client_ignore_dead_proxy

## [LRN-20260716-010] best_practice

**Logged**: 2026-07-16T14:34:00+08:00
**Priority**: high
**Status**: pending
**Area**: frontend

### Summary
厂商门户版本列表误用 `image_group_id`，而 API 返回 `image_group`，导致添加镜像后展开组始终「暂无版本」。

### Details
`getGroupVersions` 用 `im.image_group_id` 过滤；`scanContainerImageRows` 序列化为 `image_group` 字符串 ID。另组列表缺 `versions_count`，删除守卫失效。已用纯函数工具 + 契约单测锁住真实响应字段名。

### Suggested Action
列表过滤字段以线上 JSON 样例写单测；禁止按 ORM 列名臆测前端属性。

### Metadata
- Source: conversation
- Related Files: taskAiProvider/frontend/src/utils/containerImageGroup.js, taskAiProvider/frontend/src/views/VendorPortal.vue
- Tags: api-contract, vue, marketplace
- Pattern-Key: harden.api_field_name_contract
- See Also: LRN-20260716-003


## [LRN-20260716-025] best_practice

**Logged**: 2026-07-16T15:10:00.000000+00:00
**Priority**: medium
**Status**: pending
**Area**: backend

### Summary
元规则「应用启动禁用环境 Proxy」落地：run.sh unset + tracelog disableDefaultEnvProxy + Python trust_env=False。

### Details
存量服务统一启动脚本 unset；Go 在 tracelog.Init/InitConsumer 替换 DefaultTransport.Proxy=nil，覆盖绝大多数业务进程；无 tracelog 的 go_run_container 在 init 自处理；Python 热点 Session 显式 trust_env=False。

### Suggested Action
新增服务必须：run.sh unset、调用 tracelog.Init（或等价禁用）、出站 Client 不信任环境代理。

### Metadata
- Source: conversation
- Related Files: shareLib/tracelog/direct_http.go, .ai/01_project_constraints/23_app_startup_no_env_proxy.md
- Tags: proxy, runsh, tracelog, http-client
- Pattern-Key: harden.app_startup_no_env_proxy_rollout

## [LRN-20260716-026] best_practice

**Logged**: 2026-07-16T15:30:00.000000+00:00
**Priority**: medium
**Status**: pending
**Area**: backend

### Summary
Django urllib 用 ProxyHandler({}) 的 urlopen_direct；Aliyun Tea 用 Config/RuntimeOptions 清空代理覆盖 env。

### Details
同机 Go 转发客户端改为 urlopen_direct；阿里云 Python SDK 按官方优先级 RuntimeOptions>Config>env，在 direct_network 助手中置空 http_proxy/https_proxy/socks_5proxy 并设 no_proxy。Go 侧 taskCloudService/taskEvents 已有 stripOutboundProxyEnv + Config 空代理。

### Suggested Action
新增 urllib 出站用 urlopen_direct；新增 Tea Config/Runtime 经 direct_network 助手创建。

### Metadata
- Source: conversation
- Related Files: task2app/Saas_project/core/utils/urlopen_direct.py, task2app/Saas_project/cloud/providers/aliyun/direct_network.py
- Tags: proxy, urllib, aliyun, tea-sdk
- Pattern-Key: harden.urlopen_and_aliyun_direct

## [LRN-20260717-001] best_practice

**Logged**: 2026-07-17T00:40:00+00:00
**Priority**: high
**Status**: pending
**Area**: frontend

### Summary
Chrome 扩展登录成功后禁止 await 全标签页 auth 广播；API 200 仍卡在登录页通常是 sendResponse 被广播拖死。

### Details
login-with-access-token 已返回 token，但 SW 在 return success 前 await broadcastAuthStateChanged→tabs.sendMessage；部分 discarded/frozen 标签页永不 resolve，Popup 收不到 success。修复：scheduleAuthBroadcast 异步、单标签 withTimeout、Popup 收到 success 立即 showLoggedInUI。

### Suggested Action
登录/登出响应路径只做持久化（带超时）+ fire-and-forget 广播；用 Playwright Popup E2E 锁回归。

### Metadata
- Source: conversation
- Related Files: taskChromePlugin/lib/login-finalize.js, taskChromePlugin/background/service-worker.js, taskChromePlugin/popup/popup.js, taskChromePlugin/e2e/popup-token-login.playwright.test.js
- Tags: chrome-extension, popup, login, broadcast
- Pattern-Key: harden.popup_login_no_await_broadcast
- See Also: LRN-20260716-005, .ai/09_failure_experience/02_runtime_errors/20_chrome_plugin_popup_login_storage_hang.md


## [LRN-20260717-002] best_practice

**Logged**: 2026-07-17T12:05:00+08:00
**Priority**: high
**Status**: pending
**Area**: frontend

### Summary
DevTools Panel 依赖父页 postMessage 的数据，监听/缓冲必须在首个 await 之前就绪；空列表兜底也必须刷新 UI。

### Details
TaskPlugin 请求列表卡在「正在加载请求列表...」：onShown 的 initRequests 在 await refreshAuthState 期间丢失；SW getRecentRequests 空数组时不调用 applyRequestFilters；setRequestLoading(false) 为空操作。

### Suggested Action
head 同步缓冲 + init 首行接通 consumer；凡结束 loading 的路径覆盖 length===0；用 panel-request-bootstrap 单测锁回归。

### Metadata
- Source: conversation
- Related Files: taskChromePlugin/panel/panel.js, taskChromePlugin/panel/panel.html, taskChromePlugin/lib/panel-request-bootstrap.js, taskChromePlugin/devtools/devtools.js
- Tags: chrome-extension, devtools, postMessage, race
- Pattern-Key: harden.panel_postmessage_before_await
- See Also: LRN-20260717-001, .ai/09_failure_experience/02_runtime_errors/30_chrome_plugin_panel_request_list_loading_stuck.md

## [LRN-20260717-003] best_practice

**Logged**: 2026-07-17T12:16:47+08:00
**Priority**: high
**Status**: pending
**Area**: frontend

### Summary
onlineServiceJS 控制台凡发送 X-Trace-Id 必须同时带 X-Parent-Span-Id（或 traceparent），否则 traceMiddleware 一律 400。

### Details
`GET /api/repos/bootstrap-clone-log` 在容器 UI 轮询时 400，响应 detail 为 trace propagation incomplete。根因不是业务参数，而是 `static/index.html` 的 `api()` 只写了 X-Trace-Id。补发 16-hex `X-Parent-Span-Id` 后同接口 200。远程复现：补 Parent-Span 即可立刻 200，说明服务端策略已生效、缺的是客户端传播头。

### Suggested Action
浏览器/前端写 X-Trace-Id 时同步写 Parent-Span；出站复用 traceHeadersForOutbound。调试 API 400 先读 detail 是否为 trace propagation。

### Metadata
- Source: conversation
- Related Files: trae-agent/onlineServiceJS/static/index.html, trae-agent/onlineServiceJS/src/traceId.mjs, trae-agent/onlineServiceJS/src/server.mjs
- Tags: tracing, onlineServiceJS, bootstrap-clone-log, 400
- Pattern-Key: harden.browser_trace_parent_span
- See Also: .ai/09_failure_experience/02_runtime_errors/31_online_bootstrap_clone_log_trace_id_only_400.md

## [LRN-20260717-004] correction

**Logged**: 2026-07-17T12:44:13+08:00
**Priority**: high
**Status**: pending
**Area**: backend

### Summary
厂商门户保存「区域运行环境」用 `cloud_server_image`，后端只认 `cloud_server_image_id`，且 SetAssociations 先全删再静默跳过，导致运行环境被清空；任务 auto_run 表现为「自动运行为是但服务器未启动」。

### Details
日志 `镜像市场返回空的运行环境列表`。同镜像 11:40 曾成功 start-vm-auto，11:41 起多次 POST association 后关联为空。修复：兼容字段、校验后删除、单区域 Upsert、列表返回嵌套 CSI；前端改发 cloud_server_image_id。

### Suggested Action
关联类 API 做写入/读出字段契约测；禁止 DELETE-ALL + 解析失败仍 200。

### Metadata
- Source: conversation
- Related Files: taskAiProvider/infrastructure/store_marketplace.go, taskAiProvider/src/vendor_container_actions.go, taskAiProvider/frontend/src/views/VendorPortal.vue
- Tags: marketplace, association, auto_run, start-vm-auto
- Pattern-Key: harden.association_field_contract_no_silent_wipe
- See Also: .ai/09_failure_experience/02_runtime_errors/32_vendor_runtime_env_association_field_wipes.md

## [LRN-20260717-005] best_practice

**Logged**: 2026-07-17T12:58:51+08:00
**Priority**: high
**Status**: pending
**Area**: frontend

### Summary
taskAiProvider SPA 凡发送 X-Trace-Id 必须同时带 X-Parent-Span-Id；请求错误的 p.msg 必须挂 data-traceId。

### Details
provider.daydaymoney.com 上可见 `trace propagation incomplete`。生产 curl：仅 X-Trace-Id → 400；补 16-hex X-Parent-Span-Id → 200。根因是 `api.js` 只写 Trace-Id，被 `shareLib/tracelog.RejectTraceIdOnlyHTTP` 拒绝；同时 VendorPortal 等 `p.msg` 未绑定 data-traceId，违反元规则 24。

### Suggested Action
前端写 Trace-Id 时同步写 Parent-Span（复用 buildOutboundTraceHeaders）；所有请求错误 DOM 挂 data-traceId。调试 400 先读 detail 是否为 trace propagation。

### Metadata
- Source: conversation
- Related Files: taskAiProvider/frontend/src/api.js, taskAiProvider/frontend/src/utils/traceId.js, taskAiProvider/frontend/src/views/VendorPortal.vue, shareLib/tracelog/strict.go
- Tags: tracing, taskAiProvider, data-traceId, 400
- Pattern-Key: harden.browser_trace_parent_span
- See Also: LRN-20260717-003, .ai/09_failure_experience/02_runtime_errors/33_provider_frontend_trace_id_only_400.md

## [LRN-20260717-007] best_practice

**Logged**: 2026-07-17T13:50:00+08:00
**Priority**: high
**Status**: pending
**Area**: backend

### Summary
onlineServiceJS 配置读取：本地 `service_config.yaml` miss 时应回源 SaaS `feature-params-env`，失败文案引导 traceId 而非「请先上传」。

### Details
`GET /api/config` 仅读本地文件时，引导未完成或文件被删的任务会 404，控制台误导用户手动上传。正确路径与 bootstrap 一致：`feature-params-env` → `persistFeatureParamsEnv`。UI `#cfgErr` 须挂 `data-traceId`。

### Suggested Action
配置类 GET 遵循 local → SaaS 回源；错误 UI 用 formatConfigErrMsg + data-traceId。

### Metadata
- Source: conversation
- Related Files: trae-agent/onlineServiceJS/src/ensureServiceConfig.mjs, trae-agent/onlineServiceJS/src/server.mjs, trae-agent/onlineServiceJS/src/formatConfigErrMsg.mjs, trae-agent/onlineServiceJS/static/index.html
- Tags: onlineServiceJS, config, saas-fallback, data-traceId
- Pattern-Key: harden.config_local_then_saas
- See Also: .ai/09_failure_experience/02_runtime_errors/29_online_cfgerr_raw_json_not_found.md

## [LRN-20260717-009] best_practice

**Logged**: 2026-07-17T16:05:00+08:00
**Priority**: high
**Status**: pending
**Area**: frontend

### Summary
子仓库克隆失败行必须同时展示具体错误与「重新克隆」；子仓不在 `project.git_repos` 时，`repo-reclone` allowlist 须经 nested 发现校验父仓归属。

### Details
任务详情 `task-nested-repos-clone-row` 仅有「克隆失败」徽章无 err、无重试。修复：`formatNestedRepoCloneErrorDetail` + reclone 按钮；`onRepoReclone` 支持 `{repoUrl,parentRepoUrl,cloneAlias}`；Django `_allow_nested_repo_reclone`。

### Suggested Action
失败态 UI = 原因 + 动作；凡按 URL 鉴权的接口对 nested 子仓显式校验。

### Metadata
- Source: conversation
- Related Files: taskFE/app/src/components/task-detail/TaskDetailNestedReposCloneStatus.vue, task2app/Saas_project/cloud/services/forward_container_repo_reclone.py
- Tags: nested-repos, reclone, task-detail, ux
- Pattern-Key: harden.failure_row_reason_and_retry
- See Also: .ai/09_failure_experience/02_runtime_errors/37_nested_repo_clone_fail_no_reclone.md

## [LRN-20260717-008] best_practice

**Logged**: 2026-07-17T14:20:00+08:00
**Priority**: high
**Status**: pending
**Area**: frontend

### Summary
多仓并行引导克隆失败时，日志摘要与任务详情仓库行必须点名失败仓，且「重新克隆」不得仅依赖 container endpoint 已注册。

### Details
用户日志仅见「已结束（存在失败）」，实际失败为 `relayToTrae.git` / `scripts.git`（GitLab not found）。修复：bootstrap 页脚列出 `name — url（err）`；前端解析失败 URL 驱动 error 徽章；`shouldShowRepoRecloneButton` 在 relay 模式或已判定 error 时也展示。

### Suggested Action
并行批处理失败汇总始终枚举实体；失败重试控件与成功路径前置条件解耦。

### Metadata
- Source: conversation
- Related Files: trae-agent/onlineServiceJS/src/bootstrap.mjs, taskFE/app/src/utils/taskDetailContainerCloneProgress.js, taskFE/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue
- Tags: bootstrap-clone, task-detail, reclone, ux
- Pattern-Key: harden.parallel_batch_name_failures
- See Also: .ai/09_failure_experience/02_runtime_errors/35_bootstrap_clone_fail_no_repo_name.md

## [LRN-20260717-006] best_practice

**Logged**: 2026-07-17T13:40:00+08:00
**Priority**: high
**Status**: pending
**Area**: frontend

### Summary
模态内 API 失败不得只写被遮罩挡住的页面级 p.msg；「不选」清空须与后端 UpsertAssociation(csiID=0) 语义对齐。

### Details
provider 区域运行环境保存发 cloud_server_image_id:null → 400；saveRegionEnv 用 setMsgError 写页面 p.msg，全屏 modal-bg 挡住故视口无提示。修复：模态内 regionEnvError+data-traceId；UpsertAssociation 在 csiID=0 时仅清除该 platform+region。

### Suggested Action
模态请求失败挂模态内错误节点；UI 清空选项与 API clear 语义契约测试。

### Metadata
- Source: conversation
- Related Files: taskAiProvider/frontend/src/views/VendorPortal.vue, taskAiProvider/infrastructure/store_marketplace.go
- Tags: modal, data-traceId, association, vendor-portal
- Pattern-Key: harden.modal_inline_request_error
- See Also: .ai/09_failure_experience/02_runtime_errors/34_vendor_region_env_modal_error_hidden_null_csi.md
