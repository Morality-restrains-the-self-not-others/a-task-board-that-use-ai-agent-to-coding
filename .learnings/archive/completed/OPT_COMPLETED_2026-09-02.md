# Completed OPT Archive — 2026-09-02

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 22 条。
> 归档执行时间：2026-09-03T10:28:46+08:00

## [OPT-20260902-027] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: Playwright Fork `task_882968373028220928`；层图 idle_done、Git OAuth 已绑定、无 BINDING_MISSING chip
- **Created**: 2026-09-02
- **Context**: 层图「push 失败」根因是 comment L2 空导致 `layer-github-oauth-access-tokens` HTTP 409 `BINDING_MISSING`。credential / task / FE 已修。空 password_hash 补齐后 `/api/auth/` + activate-session 可登录。
- **Action**: (1) `PLAYWRIGHT_INTEGRATION=1` 跑 `TaskDetail.fork-auto-run-binding-missing-fix.playwright.test.sh` Fork 源任务；(2) CDP 等到 `comment-layer-ztree-panel`；(3) 断言无 `layer-ztree-push-error-label` BINDING_MISSING。
- **Why**: 单测不能覆盖真实 GitLab ram-work 自动运行拉票。
- **How to apply**: CDP `http://127.0.0.1:9222`；测例 `taskFE/tests/TaskDetail.fork-auto-run-binding-missing-fix.playwright.test.js`。
- **Completion-Note**: Playwright 已 Fork 出 `task_882968373028220928`（源 `task_882943770763489280`，自动运行）。容器初始化约 24min 后面板 `comment-layer-ztree-panel` 可见，文案含 `idle_done` / `Git OAuth · 已绑定` / `容器 运行中`；`layer-ztree-push-error-label` 数量 0；body 无「push 失败」/BINDING_MISSING。点「刷新」后 `container-layer-graph` HTTP 200（clone 层 `mind_state: idle_done`）。测例层图等待已从 10min 提到 25min。

## [OPT-20260901-018] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: taskEvents rename django prefix; tests green
- **Created**: 2026-09-01
- **Context**: 删除 SMTP YAML 中无效的 Django `EMAIL_BACKEND` 时，`taskEvents/notifications/cfg/settings.go` 仍用 `djangoEmailYAML` / `djangoConfigYAML` /「django fragment」注释。Django 已退役，名称会误导下一次改 SMTP 配置的人。
- **Action**: (1) 将 `djangoEmailYAML` / `djangoSMSYAML` / `djangoConfigYAML` 改为 `emailYAML` / `smsYAML` / `fragmentYAML`（或等价）；(2) 更新过时注释；(3) 现有 overlay 测例保持绿。
- **Why**: 结构体名仍暗示 Django 会读这些键，容易把已删的 `backend` 再加回去。
- **How to apply**: `taskEvents/notifications/cfg/settings.go`；验收：`go test ./notifications/cfg` exit 0。

## [OPT-20260901-014] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: dataMigrate bootstrap deep-merge regression test
- **Created**: 2026-09-01
- **Context**: `dataMigrate/taskAuth/bootstrap_admin_email.py` 已读 `conf-local`，但对 `bootstrapAdmin` 做 `dict.update` 浅合并。第 59 条门禁因此放行，nested 键仍可能丢。
- **Action**: (1) 改为 `overlay_conf_file(root / "conf/auth/task-auth/config.yaml")`；(2) 补 nested overlay 测例。
- **Why**: 与 SSOT 深合并不一致时，conf-local 里 nested 管理字段会被空骨架盖掉。
- **How to apply**: `dataMigrate/taskAuth/bootstrap_admin_email.py`；同目录或 `dataMigrate/` 下现有 bootstrap 测例。

## [OPT-20260901-004] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: runAll SSOT write-cutover-env.sh helper; key set unified
- **Created**: 2026-09-01
- **Context**: clone-run 已把 `cutover.env` 做成启动依赖（`run.sh` / `deploy-sync.sh` / `LoadConfig` 直接 source）。写出该文件的脚本仍有两份几乎相同的 `cat > cutover.env <<EOF`：`runAll/scripts/up-from-config-repo.sh` 与 `runAll/scripts/prepare-ram-deploy.sh`。键集合漂移时会出现「入口能 source、文件缺键」。
- **Action**: (1) 把键列表与 heredoc 收成单一 helper（例如 `write-cutover-env.sh DEPLOY_ROOT`）；(2) 两处生成器改调 helper；(3) 用现有 `test_up_from_config_repo.py` 断言键集合，并给 `prepare-ram-deploy.sh` 加同样断言或共用 fixture。
- **Why**: 文件是 SSOT，生成器不能再分叉，否则消费者 source 到不完整 env。
- **How to apply**: `runAll/scripts/up-from-config-repo.sh` 约 275 行、`runAll/scripts/prepare-ram-deploy.sh` 约 68 行；测试 `runAll/scripts/tests/test_up_from_config_repo.py`。

## [OPT-20260901-012] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: taskBill confload.UnmarshalYAMLMerged + gitService deep-merge
- **Created**: 2026-09-01
- **Context**: 全进程 conf-local 审计后，taskBill `overlayWechatLocalConfig` 仍对 struct 二次 Unmarshal（零值可能盖 overlay），gitService `_yaml_merged` 用 `dict.update` 浅合并。当前密钥都在顶层， nested 密钥会丢。
- **Action**: (1) `loadWechatPayConfig` 改为 `confload.UnmarshalYAMLMerged(root, "billing/wechatPay/conf.yaml", …)` 并删自定义 overlay；(2) gitService `_yaml_merged` 改调 `overlay_conf_file`；(3) 各补一条 nested secret 测例。
- **Why**: 与 ADR-0054 SSOT 不一致时，下一份 nested 密钥会再次出现「空骨架赢过 conf-local」。
- **How to apply**: `go test ./src -run Wechat`（taskBill）；`python3 gitService/scripts/test_load_gitservice_config.py`。

## [OPT-20260901-025] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: taskAiProvider WARN/ERROR tracelog trace_id
- **Created**: 2026-09-01
- **Context**: 排障 `skill version catalog unavailable`（traceId `6f9da1e7-…`）时，catalog 失败的 WARN 被 Promtail 包进 `msg` 且无 `trace_id` 字段，Loki `{job=~".+"} | json | trace_id=` 只能命中 `http_request`。本次已把 catalog/skill-md 失败改为 `tracelog.EmitWithTrace`；`src/helpers.go` 的 `logWarn`/`logError` 以及 auth/OIDC 等路径仍是 `log.Printf`。
- **Action**: (1) 将 `logWarn`/`logError` 改为走 slog/tracelog 并写入 `trace_id`；(2) 迁移 `auth_handlers.go` 等现有调用，从 `r.Context()` 取 ID；(3) 测例断言 WARN JSON 含请求 ctx 的 `trace_id`。
- **Why**: 500 业务原因行若不能按同一 trace 检索，下一次同类故障仍要靠全文搜 `event=`。
- **How to apply**: `taskAiProvider/src/helpers.go`、`auth_handlers.go`。验收：`go test ./src`；Loki `{job="ai-provider"} | json | trace_id="<id>"` 能看到 WARN 行而不仅是 `http_request`。

## [OPT-20260901-027] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: fanyi_agent max_tokens capped, unused keys removed
- **Created**: 2026-09-01
- **Context**: conf 里 `max_tokens: 4096`、`max_retries: 10`、`parallel_tool_calls` 对标题翻译无效。Go 已在请求侧封顶 128 且不读 retries。YAML 会误导下次改配置的人。
- **Action**: (1) 将 tracked `max_tokens` 改为 ≤128 并注释「标题翻译封顶」；(2) 删除或实现 `max_retries`（实现则必须远小于 10，且超时路径禁止连乘 20s）；(3) 补 conf 加载测例。
- **Why**: 大 token 预算是本次空 JSON/20s 超时的促成因素之一；未使用的 retries 若被照抄会把弹窗卡住数分钟。
- **How to apply**: `conf/taskProjectService/config.yaml`；`taskProjectService/src/fanyi_agent.go` `fanyiRequestMaxTokens`。

## [OPT-20260901-016] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: taskAuth EMAIL_SENT/kafka publish trace_id JSON
- **Created**: 2026-09-01
- **Context**: 本次用 Loki `{job=~".+"} |= "<traceId>"` 只命中 `task-auth` `http_request` 与 `task-gateway` access log。同秒的 `[taskAuth] kafka published event=EMAIL_SENT` / `EMAIL_SENT: kafka published` 没有 `trace_id` 字段，无法按 trace 串到消费者。
- **Action**: (1) `publishEmailSent` 及邻近 `log.Printf` 改为 slog/tracelog 并写入 `trace_id`/`otel_trace_id`；(2) 测例断言日志 JSON 含请求 ctx 的 trace。
- **Why**: 密码重置是异步投递，HTTP 200 不等于 SMTP 成功；缺 trace 时排障只能靠收件人邮箱全文搜索。
- **How to apply**: `taskAuth/src/` 中 `publishEmailSent` 调用点；`shareLib/tracelog`。验收：`go test` 相关包 + Loki `{job="task-auth"} | json | trace_id="<id>"` 能看到 publish 行。

## [OPT-20260901-024] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: check_conf_sync runs on temp copy; wired into CI + pre-commit
- **Created**: 2026-09-01
- **Context**: conf 非配置收口后 `check_conf_sync.sh` 已迁到 `db/scripts/ci/`，`conf/README.md` 仍写「CI runs」该脚本，但 `.github/workflows/repo-quality-gates.yml` 未调用。脚本先拷贝 `conf/` 再对**当前工作区**跑 `conf-sync-all.sh`，会改 GENERATED 时间戳，不适合直接进 CI。
- **Action**: (1) 改为在临时副本上跑 sync 再与 HEAD/`git ls-files` 对比，工作区保持干净；(2) 补自测：故意改 GENERATED 片段应失败、干净树应成功；(3) 接入 `repo-quality-gates.yml` 与 `.pre-commit-config.yaml`。
- **Why**: 现在「CI 会跑 conf-sync」只是文档承诺；真漂移只能靠人手跑会污染工作区的脚本。
- **How to apply**: `db/scripts/ci/check_conf_sync.sh`；验收：`bash db/scripts/ci/check_conf_sync.sh` 后 `git -C conf status --porcelain` 为空，且 `rg check_conf_sync .github/workflows/repo-quality-gates.yml` 命中。

## [OPT-20260901-005] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: 夜间 OPT 自动执行完成
- **Created**: 2026-09-01
- **Context**: 同机同时存在 `/tmp/ram-work/taskGateway` 与 `~/bin/daydaymoney-deploy/taskGateway` 时，Docker Compose 默认项目名都是 `taskgateway`，`run.sh stop/start` 会拆掉另一棵树的容器。本次 READINESS 根因是 config.yaml 权限，项目名碰撞会放大故障面。
- **Action**: (1) `taskGateway/run.sh` 在 `DEPLOY_MODE=1` 或 `$DEPLOY_ROOT` 时设置 `COMPOSE_PROJECT_NAME`（例如 `taskgateway-deploy`）；(2) 同步 `TASKGATEWAY_APISIX_CONTAINER` 与 `docker exec`/`docker ps` 硬编码名；(3) 测例断言 seed 与源码仓项目名不同；(4) 文档写明同机双根时 18081 仍只能有一方发布。
- **Why**: 不隔离则 source 仓 `compose down` 会误停 clone-run 网关，排障时难以判断是哪份 bind-mount。
- **How to apply**: `taskGateway/run.sh` 的 `docker_compose`/`taskgateway-apisix-1`；`runAll/scripts/truncate-ram-work-logs.sh` 的容器名默认值。

## [OPT-20260901-006] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: 夜间 OPT 自动执行完成
- **Created**: 2026-09-01
- **Context**: clone-run 的 `taskGateway` 在 `envs/current/`，`conf-local` 在部署根。本次已让 `load_gateway_conf` 向上查找 overlay。`run.sh routes_apply` 仍不导出 `DEPLOY_ROOT`，其它只认 `REPO=ROOT.parent` 的脚本还会漏密钥。
- **Action**: (1) `routes_apply` 在调用 `routes-to-apisix.py` 前 source `$DEPLOY_ROOT/cutover.env` 或向上找到含 `conf-local` 的根并 export；(2) 测例：cwd=envs/current/taskGateway 且无 env 时仍能合并部署根密钥（已有 walk 测例可复用）；(3) 清点其它 `ROOT.parent` 生成器。
- **Why**: 只靠 walk 时，若中间目录误放空 `conf-local` 会提前停。显式 `DEPLOY_ROOT` 与 Go `confload` 一致。
- **How to apply**: `taskGateway/run.sh` `routes_apply`；对照 `setup_tls` 里已有的 deploy_root 上溯。

## [OPT-20260901-019] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: 夜间 OPT 自动执行完成
- **Created**: 2026-09-01
- **Context**: `taskEvents/config.FindMonorepoRoot` 曾从 cwd 走到 `envs/current/conf/base.yaml`，忽略已设置的 `CONF_ROOT`/`DEPLOY_ROOT`，SMTP `host_password` 空串导致 QQ 535。email worker 已改为 `confload.FindMonorepoRoot`。`taskGitOauth/infrastructure/config.go`、`taskTaskService/src/db.go` 等仍 cwd walk。
- **Action**: (1) 搜索 `filepath.Join("conf", "base.yaml")` 且不调 `confload.FindConfigRoot` 的 Go 入口；(2) 改为委托 confload；(3) 每个包加 clone-run 测例：cwd=`envs/current/<svc>` 且 `CONF_ROOT` 指向部署根 `conf/` 时 root 为部署根。
- **Why**: clone-run 机密只在部署根 `conf-local/`；走错根会再次出现「HTTP 200、密钥空、厂商 AUTH 失败」。
- **How to apply**: `taskGitOauth/infrastructure/config.go`；`taskTaskService/src/db.go`；对照 `taskEvents/config/conf_yaml.go` 与 `conf_yaml_root_test.go`。验收：`go test` 相关包 exit 0。

## [OPT-20260901-021] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: 夜间 OPT 自动执行完成
- **Created**: 2026-09-01
- **Context**: `handleSSOBridge` 无会话会 302 `/auth/login/`，但 APISIX `taskauth-accounts-sso` 先跑 forward-auth，无 Cookie 时直接 401 JSON「无法解析登录凭据」。无痕 curl / 过期会话点 SSO 看到的是 raw JSON 而不是登录页。
- **Action**: (1) 对该 GET 路由让 forward-auth 失败返回 302 Location=登录页，或把该 URI 改成 cookie 可选并交给 handleSSOBridge；(2) 测例：无凭据 GET `/api/accounts/sso/ai-provider/admin/` 期望 302 而非 401 JSON。
- **Why**: 已登录半开隧道是本次主因；无会话点击仍会「无法打开」一块 JSON，和 302 登录体验不一致。
- **How to apply**: `taskGateway/routes/routes.yaml` `taskauth-accounts-sso`；`taskAuth/src/gateway_forward_auth.go` 或 APISIX forward-auth 插件。验收：无 Cookie curl `-D -` 见 `Location: .../auth/login/`。

## [OPT-20260901-022] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: 夜间 OPT 自动执行完成
- **Created**: 2026-09-01
- **Context**: `/1-brainstorming` 收口 conf 非配置文件时确认：`deploy_repo_random_precommit.sh` 对 `conf|dockerInfra|sdk|taskGateway|gitService|taskChromePlugin` 一律打 `pre-commit.placeholder`，再强制带上 `commit-msg` / `random_test_runner` / `.claude/settings.json`。conf 将按已批设计 SKIP；同类仓仍会被灌进与业务无关的模板文件。
- **Action**: (1) 把 `SKIP_HOOK_REPOS` 扩到真正无测例/非代码仓（至少 `dockerInfra`、`sdk`，gitService/taskGateway/taskChromePlugin 需逐仓确认是否有值得跑的测例）；(2) `check_subrepo_random_precommit_hooks.py` 同步豁免；(3) 已灌入的 `.githooks/lib` 在下次部署时删除而非 KEEP。
- **Why**: 只 SKIP conf 会留下同一分发器对其它子仓的同样污染；下次有人打开 dockerInfra 又会问「为什么配置/SDK 仓有 commit-msg」。
- **How to apply**: `scripts/deploy_repo_random_precommit.sh` 的 `pick_template` / `SKIP_HOOK_REPOS`；`db/scripts/ci/check_subrepo_random_precommit_hooks.py`。验收：对 SKIP 仓跑分发器后 `git -C <repo> ls-files .githooks` 为空或仅手写允许清单。

## [OPT-20260901-026] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: 夜间 OPT 自动执行完成
- **Created**: 2026-09-01
- **Context**: 排障创建任务标题翻译 502 时读到 `conf/taskProjectService/config.yaml` 的 `ai_agent_config.fanyi_agent.api_key` 仍是已跟踪非空密钥，违反第 58/57 条。本次未改该文件以免把密钥再写进 diff。
- **Action**: (1) 把 `api_key` 抽空写入 tracked YAML；(2) 同相对路径写入 gitignored `conf-local/taskProjectService/config.yaml`；(3) 轮换已暴露的 key；(4) 跑 `python3 db/scripts/ci/check_conf_local_secrets.py`。
- **Why**: 已跟踪 conf 中的 LLM key 等于公开凭据；翻译失败排障也会反复打开该文件。
- **How to apply**: `conf/taskProjectService/config.yaml` `ai_agent_config.fanyi_agent`；overlay `conf-local/taskProjectService/config.yaml`。

## [OPT-20260901-028] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: 夜间 OPT 自动执行完成
- **Created**: 2026-09-01
- **Context**: ADR-0055 种下路径用 `grant:comment:{commentID}:{userID}:{gitsite}`。存量 `applyGrantTicketToIdentities` 仍把 Kafka key 设为 ticket id，且 payload 无 `comment_id`（ticket 消费发生在 insert 之前）。
- **Action**: (1) 在已知 comment id 之后再发 ticket 路径事件，或把 key 改成与 seed 同形；(2) payload 补 `comment_id` + `via=grant_ticket`；(3) 补单测断言 key 含 comment id。
- **Why**: 审计与幂等消费按评论粒度；ticket id 键无法与 seed 事件对账。
- **How to apply**: `taskTaskService/src/comment_oauth_grant.go` `applyGrantTicketToIdentities`；调用点 `prepareAutoRunRepoIdentities` / 评论 handler。

## [OPT-20260901-007] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: taskFE 已登记 CRG 并全量重建 graph（2210 files/14943 nodes），showVendorSsoButton 与 ImageMarket.vue 可搜，crg-daemon watch 已启动
- **Created**: 2026-09-01
- **Context**: 诊断镜像市场 SSO 缺失时 `code-review-graph search "showVendorSsoButton"` 返回 0。根图约 18 files / 114 nodes，未覆盖 `taskFE/app/src/views/ImageMarket.vue`，Step 1 无法用图看 CTA 爆炸半径。
- **Action**: (1) 将 `taskFE` 登记进 CRG（`crg-daemon add` 或仓库文档约定的 register）；(2) `code-review-graph update --brief`；(3) 确认 Vue computed / 模板符号可搜。
- **Why**: 厂商入口这类跨仓漏斗只写在 Vue 里时，图搜为空会把诊断逼回 Loki+手搜，同类问题会重复漏检。
- **How to apply**: `code-review-graph search showVendorSsoButton`（或 `ImageMarket.vue`）输出须包含 `taskFE/app/src/views/ImageMarket.vue`；exit 0 且命中数 ≥ 1。

## [OPT-20260901-020] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: AiMonitor 新增 provider-public + provider-local-8010 blackbox 探针与 provider-edge 告警，promtool 校验通过，线上 probe_success=1 双绿，blackbox-exporter 补 host-gateway；人为停隧道触发验证保留人工（破坏性），配置与规则已生效
- **Created**: 2026-09-01
- **Context**: 本会话「镜像市场管理（SSO）」打不开：taskAuth 已 302，SH nginx 的 provider vhost 因 8010 反向隧道半开而 HTTP 挂死。autossh PID 仍在，runAll 健康检查看不到公网子域。
- **Action**: (1) 在 AiMonitor blackbox / Prometheus 增加 `https://provider.daydaymoney.com/` 与 `http://127.0.0.1:8010/` 探针；(2) 连续失败告警指向 `ensure-edge-tunnels.sh` 半开重建；(3) 告警文案写明不要先改 taskFE SSO href。
- **Why**: 下次半开时用户只会看到浏览器转圈，Loki 里 SSO 还是 302，没有现成红灯。
- **How to apply**: `AiMonitor/` blackbox jobs；对照现有 `www.daydaymoney.com` 探针。验收：人为停 8010 隧道后告警触发。

## [OPT-20260901-023] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: 新增 taskFE/tests/helpers/adminLogin.js 的 playwrightAdminLoginViaApi（admin-login + activate-session 两段式），新增 SystemAdmin.admin-login-api.playwright.test.js 断言请求契约与 /system-admin/container-images/ 侧栏；playwright headless 2 测例通过并推送
- **Created**: 2026-09-01
- **Context**: `/goal` 验收 `https://www.daydaymoney.com/system-admin/container-images/` 时，对 `/auth/admin-login/` 填 `#email` 后 click「管理员登录」在 CDP Chrome 上挂死（>100s 无第二帧）。改用 `POST /api/auth/admin-login/` + 页内 `fetch /api/accounts/users/activate-session/`（`Authorization: Token …`）后 2s 内完成登录，SSO 弹窗可达 `provider.daydaymoney.com/admin`。
- **Action**: (1) 在 `taskFE/tests/` 抽出 `playwrightAdminLoginViaApi(page, { email, password })`（禁止走客户 `playwrightLoginWithLegalAccept`）；(2) 现有系统总览 / 超管 E2E 改调该 helper；(3) 测例断言 `admin-login` 200 且 `activate-session` 200，且随后 `page.goto('/system-admin/')` 可见侧栏「容器镜像列表」。
- **Why**: 表单路径依赖 `document.getElementById` 与原生 submit，CDP/headless 下会假死；无 helper 时下次 `/goal` 又会卡在登录而不是页面点击。
- **How to apply**: `taskFE/app/src/composables/useLoginSubmit.js`、`taskFE/tests/` 现有 playwright 登录 helper。验收：`cd taskFE && npx playwright test` 命中该 helper 的测例 exit 0；禁止再对 AdminLogin 页面 `locator('#email').fill`。

## [OPT-20260901-008] completed

- **Status**: completed
- **Completed**: 2026-09-02
- **Summary**: taskFE unit test 从 2 补到 6 测例，与 taskAiProvider directUpload.unit.test.js 同构断言 hasPostFormFields/no-cors POST 路径/同源 PUT Bearer/buildDirectUploadHeaders；两侧 6/6 全绿并推送。跨仓 SSOT 抽取受两仓独立 git 边界限制（无 monorepo workspace），改以契约孪生注释 + 双测例同构断言防漂移
- **Created**: 2026-09-01
- **Context**: 镜像市场恢复申请表单时，把 `taskAiProvider/frontend/src/utils/directUpload.js` 复制为 `taskFE/app/src/utils/vendorDocsDirectUpload.js`。两套 COS POST Object / 同源 PUT 契约会漂移。
- **Action**: (1) 抽共享模块（或 taskFE 只 re-export 一份 SSOT）；(2) 两边 `uploadIssuedFile` 测例共用 fixture；(3) 断言 `hasPostFormFields` + no-cors POST 路径一致。
- **Why**: 桶 CORS / 预签名字段变更时只改一处，避免一边 403 一边还能传。
- **How to apply**: `npx vitest run src/utils/vendorDocsDirectUpload.unit.test.js` 与 `node --test taskAiProvider/frontend/tests/*directUpload*`（或合并后的单测）均 exit 0；两文件不再各写一套 `hasPostFormFields`。

## [OPT-20260902-004] completed

**Logged**: 2026-09-02
**Priority**: low
**Status**: completed
- **Completed**: 2026-09-02
- **Summary**: ProjectEdit 经 useProjectGitRepoRows + gitRepoRowsFormatError 与创建页共用 isValidGitRepoUrl；ssh:// 可保存、ftp:// 返回 INVALID_REPO_URL_MSG。Validate: npx vitest run src/composables/useProjectGitRepoRows.test.js src/utils/gitRepoUrlUtils.test.js
**Area**: frontend

### Summary
编辑项目页 Git 仓库行复用 `isValidGitRepoUrl`，与创建页同样拒绝非法 scheme。

### Details
`ProjectEdit.vue` 目前只改了 placeholder，提交时不做前端格式校验。应与 `useCreateProjectGitRepoRows` 共用校验，避免编辑页能存入创建页会拦下的 URL。验收：`ssh://git@host/path.git` 可保存；`ftp://` 在编辑页有格式提示。

### Metadata
- Source: goal-overview
- Related Files: taskFE/app/src/views/ProjectEdit.vue, taskFE/app/src/utils/gitRepoUrlUtils.js
- Tags: git-url, create-project, ssh

---

## [OPT-20260902-005] completed

**Logged**: 2026-09-02
**Priority**: low
**Status**: completed
- **Completed**: 2026-09-02
- **Summary**: layerGitRouteHelpers.mjs 再导出 repoMatchKey.mjs 的 repoMatchKeyFromUrl。Validate: node --test src/repoMatchKey.test.mjs
**Area**: trae-agent

### Summary
`layerGitRouteHelpers.mjs` 的 `repoMatchKeyFromUrl` 改为从 `repoMatchKey.mjs` 再导出，去掉双份实现。

### Details
本次为 ssh:// 去端口分别改了两处。后续应 `export { repoMatchKeyFromUrl } from './repoMatchKey.mjs'`。验收：helpers 不再内联 URL 解析；`node --test src/repoMatchKey.test.mjs` 仍绿。

### Metadata
- Source: goal-overview
- Related Files: trae-agent/onlineServiceJS/src/layerGitRouteHelpers.mjs, trae-agent/onlineServiceJS/src/repoMatchKey.mjs
- Tags: ssh, repo-match-key, duplication

---

