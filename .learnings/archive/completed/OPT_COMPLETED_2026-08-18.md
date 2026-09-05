# Completed OPT Archive — 2026-08-18

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 12 条。
> 归档执行时间：2026-08-19T16:47:13+08:00

## [OPT-20260817-031] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: taskBill/src/wechat_pay.go wechatPrepay live 失败分支打结构化 JSON 日志（appid/mchid/out_trade_no/err_code），新增 wechatPrepayCall 注入接缝与 wechat_pay_fail_log_test.go，taskBill 953406b 已推送
- **Created**: 2026-08-17
- **Context**: TraceId `3a96a817-7fab-41b9-98eb-ddcf0ec6bb99` 微信支付 Native 预下单返回 `APPID_MCHID_NOT_MATCH`。Loki 仅有 `task-bill` POST `/pay/` 500 与 `task-auth` KYC 门禁，错误路径未记录实际 `appid`/`mchid`，只能靠 conf 文件 mtime 反推。
- **Action**: (1) 在 `wechatPrepay` live 失败分支打结构化日志，字段含 `appid`、`mchid`、`out_trade_no`、微信错误码；禁止写 `api_v3_key` / 私钥 (2) 补 `wechat_pay` 单测：注入失败 Prepay，断言日志/记录器含 appid 与 mchid (3) `go test` 覆盖该测例
- **Why**: 同类绑定错误再次出现时，不必靠配置时间线猜测请求参数。
- **How to apply**: `taskBill/src/wechat_pay.go` 的 `wechatPrepay`；测例放同包 `wechat_pay_*_test.go`。验收：`cd taskBill && go test ./src -count=1 -run 'WechatPrepay.*Appid|WechatPrepayFailLogsAppid'` 退出码 0；Loki `{job="task-bill"} | json | msg=~"wechat.*prepay"` 失败行含 `appid` 与 `mchid` 字段。

## [OPT-20260817-022] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: taskEvents/commandhttp/client.go Dispatch 出站改 tracelog.ApplyOutboundHeaders，ctx 缺 span 时补 SpanID/TraceID；client_test.go 补 TestDispatchAppliesOutboundHeaders 断言 X-Trace-Id/X-Parent-Span-Id/traceparent；taskEvents a70c63d 已推送
- **Created**: 2026-08-17
- **Context**: 举一反三搜索发现 `taskEvents/commandhttp/client.go` 仅 `Set(tracelog.Header, tid)`，未带 `X-Parent-Span-Id`。当前下游多是 Django 未启用 `RejectTraceIdOnlyHTTP`，但若下游改为 shareLib Middleware 会复现评论 bootstrap 同类 400。
- **Action**: (1) 将 `Dispatch` 中单设 Header 改为 `tracelog.ApplyOutboundHeaders(req, ctx)`，并保证 ctx 带 SpanID (2) 补单测断言 Parent-Span / traceparent 存在 (3) 全量 `go test ./commandhttp/`
- **Why**: 与本次 Starting 根因同模式；提前堵住跨服务严格 Middleware 回归。
- **How to apply**: `taskEvents/commandhttp/client.go`；对照 `shareLib/tracelog.ApplyOutboundHeaders` / `RejectTraceIdOnlyHTTP`

## [OPT-20260817-032] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: conf/billing/wechatPay/conf.yaml.ai.md 修正 Native appid 绑定说明，明确登录 type=qr 网站应用不可作支付 appid（APPID_MCHID_NOT_MATCH 根因）；conf b7bd5e3 已推送
- **Created**: 2026-08-17
- **Context**: `conf/billing/wechatPay/conf.yaml.ai.md` 写「appid 与微信登录开放平台网站应用一致」。报错当时支付 appid 正是登录用网站应用 `wx625802b55b33608f`。官方 Native 可绑定类型为已认证服务号/公众号、小程序、企业微信、移动应用，不含网站应用。
- **Action**: (1) 改 companion 为：支付 `appid` 必须是商户号已绑定的服务号/公众号/小程序/移动应用 (2) 明确登录 `type=qr` 网站应用不可直接作 Native `appid` (3) 不改未提交的 `conf.yaml` 工作区 WIP
- **Why**: 按现 companion 配网站应用 appid 会稳定触发 `APPID_MCHID_NOT_MATCH`。
- **How to apply**: `conf/billing/wechatPay/conf.yaml.ai.md`。验收：`rg -n '网站应用一致' conf/billing/wechatPay/conf.yaml.ai.md` 无命中；`rg -n '服务号|公众号|小程序|移动应用' conf/billing/wechatPay/conf.yaml.ai.md` 有命中。

## [OPT-20260817-017] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: docs/superpowers/specs 2026-08-06/08-10 两旧 spec 加 superseded 指向 2026-08-17-vendor-apply-on-provider-portal-design，入口/按钮文案改「申请认证」；store_auth.go 错误文案已由他会话 WIP 同步；docs 90580ec 已推送
- **Created**: 2026-08-17
- **Context**: 入口已迁到厂商门户，但 `docs/superpowers/specs/2026-08-06-vendor-application-entry-design.md` 与 `2026-08-10-vendor-application-kyc-docs-design.md` 仍写「镜像市场申请成为厂商门户」及旧错误文案。
- **Action**: (1) 在上述 specs 文首加 superseded 指向 `2026-08-17-vendor-apply-on-provider-portal-design.md` (2) 把入口/按钮文案改成厂商门户「申请认证」 (3) 同步 `store_auth.go` 现网错误「请先在厂商门户申请认证」
- **Why**: 后续会话若按旧 spec 实现会把申请入口加回镜像市场。
- **How to apply**: `docs/superpowers/specs/2026-08-06-vendor-application-entry-design.md`；`docs/superpowers/specs/2026-08-10-vendor-application-kyc-docs-design.md`；意图 F-089

## [OPT-20260817-024] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: taskFE/app/src/components/FeatureParamsProvidersEditor.vue 选 deepseek 预填 https://api.deepseek.com/v1，base_url 含 /anthropic 且非 anthropic/claude 时行内告警；新增 deepSeekBaseUrl 单测 4 例 + 原 subTokenUi 测试全绿；taskFE 42f4034 已推送
- **Created**: 2026-08-17
- **Context**: 误配来自人工填写 Anthropic 兼容地址给 deepseek provider；写路径已强制改写，但 UI 仍易再次填错。
- **Action**: 在 FeatureParamsProvidersEditor / 推荐供应商导入处：选 deepseek 时默认 `https://api.deepseek.com/v1`；若检测到 `/anthropic` 给出行内警告。
- **Why**: 减少配置侧再引入同类 404。
- **How to apply**: `taskFE/app/src/components/FeatureParamsProvidersEditor.vue`；对照 claude-agent 的 anthropic base_url 仅用于 provider=anthropic

## [OPT-20260817-036] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: config.go 解析 gitoauth_timeout_seconds 启动接线 gitHTTPClient.Timeout（下限 10s），单测全绿，提交 1504342 已推送
- **Created**: 2026-08-17
- **Context**: 修复 GitHub OAuth Refresh 超时导致 auto_run 跳过时，发现 `conf/taskProjectService/config.yaml` 的 `gitoauth_timeout_seconds` 仅为文档性配置，实际 `gitHTTPClient.Timeout` 硬编码；本次已把硬编码与 conf 值都改为 25s，但仍未从 YAML 加载。
- **Action**: (1) 在 `taskProjectService/src/config.go` 解析 `gitoauth_timeout_seconds` (2) 启动时用该值设置 `gitHTTPClient.Timeout`（下限保护，如 ≥10s）(3) 补单测断言 conf 覆盖生效
- **Why**: 运维调超时需改 conf 而非重编译；死配置易与真实行为漂移。
- **How to apply**: `conf/taskProjectService/config.yaml`、`taskProjectService/src/config.go`、`gitoauth_client.go`

## [OPT-20260816-038] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: Loki 按 trace_id 检索已验证：promtail-local 采集 logs/*.log，JSON 含 trace_id（与响应头 X-Trace-Id 首段一致）+ otel_trace_id；{job="task-container-gateway"} | json | trace_id="7815a013033a3d278efdc087" 实时命中（404 时间线 1 行）；task-cloud-service 同验；Grafana datasource http://loki:3100 同源。验收命令已补入 .ai/01_project_constraints/24_frontend_error_data_trace_id.md。
- **Created**: 2026-08-16
- **Context**: 任务详情文件树 404 带 `data-traceId=87d18b17-707a-4ab7-aa36-6d796f0f0a26`。Loki `{job=~".+"} | json | trace_id="…"` 1h/24h 为空，采集当时几乎只有 `collection_status`；同 ID 只能在 `logs/task-container-gateway.log` 里 grep 到。排障无法走「有 traceId 先查 Grafana」的强制路径。
- **Action**: (1) 核对 Promtail/Alloy 是否采集 `logs/*.log` 且 JSON 解析 `trace_id`/`traceId` (2) 网关与 cloud 出站/入站日志统一写入该字段 (3) 用该样例 ID 在 Grafana 复验能拉出 gateway 404 时间线 (4) 补一条采集探针或文档验收命令
- **Why**: 页面已强制带 `data-traceId`，日志管道接不上就等于钥匙打不开锁，后续同类故障只能靠磁盘 grep。
- **How to apply**: Loki `http://127.0.0.1:3100`；采集配置在 `AiMonitor/` / promtail；样例日志 `logs/task-container-gateway.log` `GET .../layers/.../children` 404
- **2026-08-17 凌晨复验**: gateway / cloud JSON 日志均含 `trace_id` 字段，promtail 已按 JSON+结构化元数据提取；`{job="task-container-gateway"} | trace_id="..."` 曾命中。但 promtail 配置 05:46/06:06 两次被他会话修改、容器在重启，查询结果不稳定（同 trace 曾 2 条后转 0），原样例 UUID 已轮转无法复验。暂不关闭，待他会话完成 promtail 配置后再验收。

## [OPT-20260817-018] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: taskCloudService 4d7d3d9 已推送；comment_csc_bootstrap 新契约（loadStartEventPayload）依赖先有 start 事件，binding 测试 setup 同步补种 task-level start event（82fdbaa），src 整包测试全绿
- **Created**: 2026-08-17
- **2026-08-18 夜已实现但提交被 WIP 测试破坏阻断**: `serverSchedulingLogMessage` 已加 `isGitCloneRepoURL` / `insertRepoURLAfterFailedRepo`（`失败 <name>:` 冒号前插 URL）+ `TestServerSchedulingLogMessageInsertsRepoURLAfterFailedRepo` 6 子例全绿；但 taskCloudService 全包 `go test ./src` 因他会话 WIP（idle-reuse 拆除 / comment_csc_bootstrap 改造）致 `TestCommentContainerBindingLogsTimeline` / `TestCommentContainerBindingsIndependentParallel` / `TestCommentContainerBindingsWaitPreviousBlocksUntilComplete` 失败（stash 验证无本改动也失败），pre-commit 阻断，未违规 --no-verify。改动已还原，待 WIP 批提交后按本条目 Action 重新实现。
- **2026-08-18 凌晨复验（重新实现 + 确认硬阻断）**: 目标文件 `comment_container_binding_server_log.go`/`_test.go` 确认 git-clean 且不在他会话 WIP 批内；重新实现同一方案（`insertRepoURLAfterFailedRepo` 冒号前插 URL，与 `formatBootstrapCloneRepoFailureMessage` 的 `<name> <url>:` 对齐），新增 6 子例表测 + 既有 `TestPublishTaskSSE*` 全绿；全包失败集恰为上述 3 个 WIP 测例（变更前后一致，无回归）。确认子仓 pre-commit 硬阻断（`core.hooksPath=.githooks` → random_test_runner 对暂存包 100% 跑 `go test -count=1 ./src/...`，3 WIP 测例致 exit 1，无配额豁免），故未提交。实现已存补丁 **`.runall/opt-20260817-018-repo-url.patch`**（124 行，含代码+测例），工作树已还原避免卷入 WIP 批。WIP 批提交后可直接 `git apply .runall/opt-20260817-018-repo-url.patch` 恢复并提交。
- **Context**: 评论执行细节冷打开只看到 `失败 ram-work: git exit 128`，binding 日志 512 字截断会吃掉文案末尾。已在前端用关联仓库目录回填 URL，并在 onlineServiceJS 新失败文案内嵌 URL；旧容器镜像仍发无名 URL 的 SSE。
- **Action**: (1) 在 `taskCloudService/src/comment_container_binding_server_log.go` 的 `serverSchedulingLogMessage`（或 git-clone-progress 专用路径）若 `statusData.repo_url` 为 git URL 且 message 尚未包含它，则插到 `失败 <name>` 之后 (2) 保证插入发生在 512 截断之前 (3) 补 binding log 单测：冷打开解析能抽出 URL
- **Why**: 未重建容器时新失败事件仍会写入无名 URL 的 binding 日志，仅靠前端 catalog 在关联项目未加载或 basename 冲突时会漏按钮。
- **How to apply**: `handleGitCloneProgress` → `publishTaskSSE` → `logServerSchedulingToBindingBestEffort`；对照 `96_comment_clone_fail_no_manual_retry.md`

## [OPT-20260818-014] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: taskCloudService comment_start_vm_inflight.go: per-comment mutex + bootstrap skip + duplicate_skipped idempotent 200; tests in comment_start_vm_inflight_test.go
- **Created**: 2026-08-18
- **Context**: 分析任务 `task_877443925097345024` 时发现评论 `cmt_877444137035526144`（`@trae-agent`、`execution_mode=independent`）在约 1 秒内写入两条 `cloud_server_events(start)`：一条无 `csc_id`（`task-events` @镜像消费者直调 `start-vm`），一条带 `csc_id`（`ccbStartBinding` → `bootstrapCommentCSCRuntime` 自调用 `start-vm`）。两次 `RunInstances` 使用相同 `InstanceName=task_<task>_<comment>`，但 `ClientToken` 因每次新发的 `userdata_access_token`/`trace_id` 而不同，导致阿里云同名双实例；CSC 最终只保留后者（`i-m5eaujdxa3ksi2dfeu2m`），前者（用户在控制台看到的 `i-m5eiydfa7p5sybgb45x7`）成为孤儿。
- **Action**: (1) 在 `bootstrapCommentCSCRuntime` / `postCommentCSCStartVM` 与 `TASK_COMMENT_IMAGE_MENTIONED` 冷启动之间加互斥：若 comment 已有 pending/starting 的 start 事件或 binding 已由 mention 路径发起 start-vm，则 bootstrap 跳过 (2) 或让 mention 路径只负责发事件、统一由 binding bootstrap 建机（单一入口）(3) 补回归测：independent @镜像只产生一次 `RunInstances`/`start` 事件 (4) 对现存同名孤儿走 `reconcileOrphanInstancesByName` 验收
- **Why**: 双建机会产生未绑定 ECS 持续计费，且控制台 InstanceName 与库内 `instance_id` 不一致，排障易误判。
- **How to apply**: `taskEvents/internal/handlers/taskcommentimagementioned/handler.go`、`taskCloudService/src/comment_csc_bootstrap.go`、`comment_container_bindings_schedule.go`、`orphan_instance_reconcile.go`；对照本任务 CSC `csc_-2298388069504333747` 与 binding logs 760–767

## [OPT-20260818-017] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: 本机 task_bill 已 apply_datamigrate 043/044（2 已应用）；其他环境仍须 9999
- **Created**: 2026-08-18
- **Context**: `043_gitlab_region_tencent_sh_1.sql` 与 `044_order_item_gitlab_region.sql` 已入库目录，尚未经 `InitAllDatabases` 落到 `task_bill`。业务进程禁止自迁移。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点「初始化全部数据库」或对 task_bill 跑 `apply_datamigrate.sh` (2) 确认 `billing_gitlab_region` 有 `tencent-sh-1` 且 `gitlab_api_base` 为公网 HTTPS (3) 确认 `billing_resource_order_item.region` 列存在
- **Why**: 未迁移则购买/下单会因缺列或缺区域行失败。
- **How to apply**: `dataMigrate/taskBill/043_*.sql`、`044_*.sql`；入口 `db/registry.yaml` → `apply_datamigrate.sh`

## [OPT-20260818-019] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: 已拆除 /tmp/ram-work-runall-clean（仅未跟踪 conf 符号链接，无独有提交）；cleanup_stale_worktrees.py --scan 现为 0
- **Created**: 2026-08-18
- **Context**: 清理 `*-wt` 后扫描仍见 `/tmp/ram-work-runall-clean`（detached、未跟踪 `conf/`、mtime 2026-08-10）。不属于本次 `feat/terminal-release-comment-csc`。
- **Action**: (1) 核对是否仍有未拣选改动 (2) 无价值则 `git -C runAll worktree remove /tmp/ram-work-runall-clean` (3) 有价值则拣选到 main 再拆除
- **Why**: 脏 detached worktree 会挡住后续 `worktree` 操作，也会让 `--scan` 一直非空。
- **How to apply**: `git -C /tmp/ram-work/runAll worktree list`；`runAll/scripts/cleanup_stale_worktrees.py --scan`

## [OPT-20260818-033] completed

- **Status**: completed
- **Completed**: 2026-08-18
- **Summary**: 公网 chunk https://www.daydaymoney.com/static/assets/WorkspaceSettingsGitlabConnection-DTEoUoG-.js 与 dist 字节一致；含「磁盘单价」「流量单价」，不含「锁价」/(锁价)/（锁价）。vitest WorkspaceSettingsGitlabConnection.test.js 8/8 通过。
- **Created**: 2026-08-18
- **Context**: 租户 GitLab 连接设置页资源配额区磁盘/流量单价标签已从「磁盘单价（锁价）」「流量单价（锁价）」改为「磁盘单价」「流量单价」。单测已绿、`taskFE/app` 已 `npm run build`，需精准编译重启后公网才能看到。
- **Action**: (1) :9999「精准编译重启」`taskFE` (2) 硬刷新 `https://www.daydaymoney.com/tenant/877397588196749312/settings/gitlab-connection/` (3) 断言资源配额区 `dt` 为「磁盘单价」「流量单价」，全文不再出现「（锁价）」或「(锁价)」
- **Why**: 公网 SPA 读 dist；未重启/未硬刷新会继续展示旧标签。
- **How to apply**: `taskFE/app/src/views/WorkspaceSettingsGitlabConnection.vue`；`data-testid=gitlab-builtin-resources`

