# Completed OPT Archive — 2026-08-17

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 35 条。
> 归档执行时间：2026-08-19T16:47:13+08:00

## [OPT-20260816-058] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 环境变量日志脱敏默认开启，仅显式 0/false/off 关闭；trae-agent 00aa672 已推送
- **Created**: 2026-08-16
- **Context**: 审计 `trae-agent` 密钥时确认版本库无真实密钥，但 `onlineServiceJS` 的 `INIT_LOG_REDACT` / `FEATURE_PARAMS_ENV_LOG_REDACT` / `ENV_LOG_REDACT` 默认关闭，`init.log` 与 `feature-params-env.log` 可按原值落盘 `ACCESS_TOKEN`、api_key。
- **Action**: (1) 将 `isInitLogRedactEnabled` / `isFeatureParamsEnvLogRedactEnabled` 改为默认开启，仅显式 `0/false/off` 关闭 (2) 更新对应单测断言默认 redact (3) 文档注明生产必须脱敏
- **Why**: 日志目录虽 gitignore，容器或共享磁盘上的明文 token 仍可被运维拷贝或误上传。
- **How to apply**: `trae-agent/onlineServiceJS/src/envLogRedact.mjs`；`initLog.mjs`；`featureParamsEnvLog.mjs` 及其 `*.test.mjs`

## [OPT-20260816-049] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: OpenAI 非官方 base_url 回退 chat.completions；trae-agent 2a4d951 已推送
- **Created**: 2026-08-16
- **Context**: `OpenAIClient` 固定走 Responses API（`/v1/responses`）。SaaS LLM 代理与多数兼容网关只实现 `/v1/chat/completions`。若 workspace 把 provider 配成 openai 且 base_url 指向代理，会得到永久 404，重试也无法恢复。
- **Action**: (1) 当 `base_url` 不是 `api.openai.com` 时改用 `OpenAICompatibleClient` 的 chat.completions (2) 补测：自定义 base_url 不调用 `responses.create`
- **Why**: 与网关瞬时 404 不同，错误 API 路径重试只会浪费时间后仍失败。
- **How to apply**: `trae-agent/trae_agent/utils/llm_clients/openai_client.py`、`llm_client.py`

## [OPT-20260816-050] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: postJson 401 走 refresh-access 换 token 重试，无 refresh 不盲重试；trae-agent b1a6802
- **Created**: 2026-08-16
- **Context**: 同一时刻除 LLM 404 外，另一 trace `dfc792b3b5a337f47f08adf0` 对 heartbeat/layer-graph-push 返回 401（1–2ms 本地拒）。SaaS 重启后旧 access 可能失效，当前 `postJson` 对 401 不重试。
- **Action**: (1) 识别 HTTP 401 (2) 走已有 `proactiveAccessRefresh` / refresh-access (3) 换新 token 再 POST 一次 (4) 401 单测保持「无 refresh 则不盲重试」
- **Why**: 只修 LLM 404 不够覆盖重启后 token 失效；心跳 401 会让页面误报容器掉线。
- **How to apply**: `trae-agent/onlineServiceJS/src/saasPostJson.mjs`、`proactiveAccessRefresh.mjs`、`saasTaskCloud.heartbeat.test.mjs`

## [OPT-20260816-051] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 瞬时重试窗口拉长至 15 次/40s，reachability 退避覆盖 30s；trae-agent b1a6802
- **Created**: 2026-08-16
- **Context**: LLM 侧 `max_retries` 默认 10 + 指数退避可达数分钟，足够覆盖重启。但 `postJson` 默认 5 次、backoff 400ms（约 4s），reachability 仅 `[0,200,500,1000]ms`。完整 runAll/SaaS 重启常 30–90s，心跳/注册仍会在窗口内耗尽。
- **Action**: (1) 把 `TASK_API_POST_JSON_TRANSIENT_RETRIES` 默认和 reachability `retryDelaysMs` 拉到至少覆盖 ~30s (2) 补测：连续 404 超过旧窗口、在新窗口内成功
- **Why**: 只修 LLM 404 后，容器仍可能在 SaaS 未起来时注册失败或心跳掉线。
- **How to apply**: `trae-agent/onlineServiceJS/src/saasPostJson.mjs` `postJsonTransientRetryConfigFromEnv`；`reachability.mjs` `registerReachabilityAfterBootstrap`

## [OPT-20260816-041] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: job-stream chunk 按 job_id 短窗口合并，step/终态立即 flush；trae-agent a519e5d
- **Created**: 2026-08-16
- **Context**: `recordJobEvent(..., 'chunk')` 每个 stdout 片段都异步 POST `job-stream-push`，再打 Kafka。步骤/生命周期必须实时；chunk 只用于 SSE 直播，高频会打满 inbound 与 broker。
- **Action**: (1) 在 `saasJobStreamPush.mjs` 对 phase=chunk 按 job_id 做短窗口合并（例如 100–250ms 或按字符上限）(2) step/start/终态立即 flush 并绕过 debounce (3) 补测：连续 chunk 只产生少量 POST，step 不被吞
- **Why**: 长命令刷屏会放大 Kafka/SSE 流量，拖垮 Cloud inbound。
- **How to apply**: `trae-agent/onlineServiceJS/src/saasJobStreamPush.mjs`；参考 `mountedAgentCommentStream.mjs` 的 chunk buffer

## [OPT-20260816-045] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: start-all/start-group/单服务入口磁盘重载配置，失败保留内存；runAll cf076a0
- **Created**: 2026-08-16
- **Context**: 精准编译重启已在解析登记名前 `LoadConfig` 磁盘 YAML，新增 `task-events-*` 可被编译重启。但「全部启动 / 启动本组 / 单行 ↻」仍只用进程启动时的内存配置，UI 网格在热替换前看不到新服务，也无法只靠分组启动拉起它。
- **Action**: (1) 把 `reloadConfigFromDisk` 抽到 start-all / start-group / 单服务 start 入口（失败保留内存配置，与 PreciseRestart 相同）(2) 补测：内存缺名、磁盘 YAML 有名时 StartGroup 能启动该服务 (3) 确认 StatusStore 仍用 EnsureNames 而非 Init
- **Why**: 只修精准编译重启的话，运维在 UI 点「启动本组」仍会漏掉进程启动之后才写入 YAML 的消费者。
- **How to apply**: `runAll/src/precise_restart_reload.go`；调用点在 `runner.go` 的 StartAll / StartGroup / StartService；测例可仿 `precise_restart_reload_test.go`

## [OPT-20260816-046] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: GET 精准重启登记对 unresolved 名做一次磁盘重载；runAll cf076a0
- **Created**: 2026-08-16
- **Context**: 页头按钮在 `resolvable===false` 时标题会标「未知服务」，但点击仍会走 PreciseRestart 热加载并成功。登记列表 API 只查内存配置，容易让人以为点了也没用。
- **Action**: (1) `handlePreciseRestartRegistrations` 若有条目 `resolveRegisteredServices` 为空，则调用一次 `reloadConfigFromDisk` 再解析 (2) 禁止每次 2s 轮询都 LoadConfig，只在存在 unresolved 时触发 (3) 补 API 测例：内存缺名、磁盘有名时 entries[].resolvable 为 true
- **Why**: 避免 UI 把可热加载的新服务显示成未知，减少误判为配置写错。
- **How to apply**: `runAll/src/ui_precise_restart.go`；`precise_restart_api_test.go`

## [OPT-20260816-032] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: relayToTrae 状态推送改事件驱动，删除 1.5s ticker，allowlist 移除；go_relayToTrae c0cf8a7 + db dba463c
- **Created**: 2026-08-16
- **Context**: `statusPushLoop` 每 1.5s 扫本进程 `registeredTasks` 推后端。状态只活在 sidecar 内存，搬到 taskEvents 会变成定时打 sidecar，更差。
- **Action**: (1) 在 `/v1/register` `/v1/start` `/v1/stop` 以及改变 status 的子进程事件上同步调用 `pushStatusToBackend` (2) 删除 `statusPushLoop` / `ensurePushThread` 的 ticker (3) 从 allowlist 去掉 `go_relayToTrae/src/push.go` (4) 单测：无 `NewTicker`；register 至少一次 push；stop 后不再周期请求 (5) 若产品仍要心跳保活，改为后端按最后一次 push 超时判定离线，不在 sidecar 盲推
- **Why**: 1.5s 全量推是 CPU/网关噪音，且把「循环模式」留在业务 sidecar 上。
- **How to apply**: 计划 §032；`go_relayToTrae/src/push.go`；`go_relayToTrae/src/handlers.go`；现有 `push_test.go`

## [OPT-20260816-054] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 厂商门户新增站内 Markdown 渲染页渲染容器→SaaS SSOT 表格；taskAiProvider 2dd1a3a
- **Created**: 2026-08-16
- **Context**: 已在顶栏用真实 `<a href="/saas-machine-container.md">` 打开 SSOT。当前以 `text/plain`/`text/markdown` 展示，表格在浏览器里可读性差。
- **Action**: (1) 增加站内路由页拉取同一 SSOT 并渲染 Markdown 表格 (2) 顶栏链接仍保持真实 href，可指向该页或保留 `.md` 原文 (3) 补前端单测断言表格标题可见
- **Why**: 厂商对照 inbound action 时更不容易漏字段。
- **How to apply**: `taskAiProvider/frontend/src/App.vue` 顶栏；`docs/skills/saas-container/saas-machine-container.md`；`GET /saas-machine-container.md`

## [OPT-20260816-052] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: domain-events 9 个未编入 intent 补进 runAll 编排 + prometheus file_sd + 迁出 unmanaged allowlist；conf 2448557 + db 9081c96 + AiMonitor 75b5fe4
- **Created**: 2026-08-16
- **Context**: 排查 fanout 健康检查端口拷错时对照 `conf/events/domain-events/*/config.yaml` 与 `conf/runAll.yaml`，发现 9 个已登记 intent 没有 runAll 条目（`user_created/2_sync_user_profile`、4 个 `relay_lifecycle`、`task_created`×2、`task_deleted`、`billing_order_comment_created`）。精准编译重启不会拉起它们。
- **Action**: (1) 为每个 groupId 按现有 task-events 模板补 `build/start/stop` 与 `${INFRA_HOST}:<SSOT port>` 健康检查 (2) 同步 `AiMonitor/prometheus/file_sd/runall-health-targets.json` (3) 跑 `python3 db/scripts/ci/check_task_events_runall_health_ports.py` 确认无端口漂移 (4) 在 9999 精准编译重启这 9 个服务
- **Why**: 看板 SSE fanout（TASK_CREATED/DELETED）、用户资料同步、relay 生命周期审计等消费者在编排外运行，重启/部署后会静默缺席。
- **How to apply**: SSOT `conf/events/domain-events/<event>/config.yaml` 的 `intents.*.port`/`groupId`；编排 `conf/runAll.yaml` `domain-events-intents` 组；门禁 `db/scripts/ci/check_task_events_runall_health_ports.py`；每补进一个 groupId 须从 `LEGACY_UNMANAGED_GROUP_IDS` **删除**该项（只减不增）

## [OPT-20260816-026] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: taskEvents 944729e + conf 5cfe612 + taskBill 6f63654 + db 5e9ec5a 已推送；billing_referral_settle_scan/1_settle_due timer（18067）POST taskBill settle-due，startReferralSettleLoop 移除，allowlist 清一行，CI 双门禁绿
- **Created**: 2026-08-16
- **Context**: `taskBill/src/main.go` `startReferralSettleLoop` 每小时调 `settleDueReferralCommissions`。一次性入口已有：`POST /api/internal/taskbill/referral/settle-due/`（`handleInternalReferralSettleDue`）。
- **Action**: (1) 按 `taskpostexpiryscan` 新增 timer `billing_referral_settle_scan/1_settle_due`（**勿用 18062**：已占用 `sse_message/2_persist_job_execution_event`，另选空闲端口）POST 该路径 (2) AllIntents + run.sh + `conf/events/domain-events/` + runAll（depends_on task-bill，tick=1h）(3) 删除 `startReferralSettleLoop` 及 main 里 `go` 调用 (4) 从 allowlist 去掉 `taskBill/src/main.go`（若 029 仍引用 wechat 文件则只删 main 这一行）(5) 补 timer 单测与门禁
- **Why**: 结算时钟与 HTTP 计费进程耦合，taskBill 挂了结算也停、且没有独立 health。
- **How to apply**: 计划 §026；handler `taskBill/src/referral_commission_handlers.go`；模板 `taskEvents/cmd/task_post_expiry_scan/1_expire_posts/main.go`

## [OPT-20260816-031] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: taskEvents 54ed8ce + conf df1a46e + taskReferral 3e46436 + db e8c44fe 已推送；referral_code_expiry_scan/1_expire timer（18066）POST taskReferral expire-codes，startReferralExpiryLoop 移除，allowlist 清一行，CI 双门禁绿
- **Created**: 2026-08-16
- **Context**: `startReferralExpiryLoop` 每小时 `expireReferralCodes`。读路径已按 `expires_at` 过滤，循环只为把 `status` 写成 expired 给管理列表。
- **Action**: (1) `POST /api/internal/referral/expire-codes/` 调 `expireReferralCodes` (2) timer `referral_code_expiry_scan/1_expire`（建议 18066，1h）(3) 删除 loop 与 main 调用 (4) 从 allowlist 去掉 `taskReferral/src/referral_code.go` (5) 已有 `referral_code_test.go` 过期用例须仍绿
- **Why**: 管理列表状态与 HTTP 进程耦合；迁出后过期回写可独立失败而不拖垮推荐码 API。
- **How to apply**: 计划 §031；`taskReferral/src/referral_code.go`；`taskReferral/src/main.go`

## [OPT-20260816-029] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: taskEvents f27b7a8 + conf bb059fc + taskBill 7afc4a1 + db c68a501 已推送；billing_profit_sharing_scan/1_process_pending timer（18064）POST taskBill process-pending，runProfitSharingDaemon 移除，allowlist 清一行，OpenAPI 内部路由补齐，CI 双门禁绿
- **Created**: 2026-08-16
- **Context**: `runProfitSharingDaemon` 在 taskBill `main` 里每小时 `processPendingProfitSharings`；微信未 live 时只打日志 skip。
- **Action**: (1) `POST /api/internal/taskbill/profit-sharing/process-pending/` 调该函数；未 live 返回 200+skipped (2) timer `billing_profit_sharing_scan/1_process_pending`（建议 18064，1h）(3) 删除 `go runProfitSharingDaemon` 与 daemon 循环 (4) 从 allowlist 去掉 `taskBill/src/wechat_profit_sharing.go` (5) 单测 + 门禁
- **Why**: 分账是资金路径，时钟必须能单独重启、单独看 health，不能绑在计费 HTTP 进程上。
- **How to apply**: 计划 §029；`taskBill/src/wechat_profit_sharing.go`；内部鉴权复用 `requireInternalSecret`

## [OPT-20260816-044] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: taskCloudService a87d9ad 已推送；sibling start 回退日志改 logInfo + tracelog.NewTraceID()，新增 TestLoadStartEventPayload_SiblingFallbackEmitsTraceID 断言 trace_id≠task_id
- **Created**: 2026-08-16
- **Context**: `loadStartEventPayload` 复用同任务其它评论的 start 事件时用 `log.Printf`，没有 `trace_id` 字段。第二条 @镜像 bootstrap 排障仍难和页面启动 TraceId 对齐。
- **Action**: (1) 将 `event=start_event_payload_sibling_fallback` 改为 `logInfo(..., independentTrace)` (2) 补测：捕获日志含 `trace_id=` 且 ≠ task_id
- **Why**: 现在只能靠 msg 文本搜 comment_id，和「有 TraceId 先查 Loki」门禁不一致。
- **How to apply**: `taskCloudService/src/comment_csc_bootstrap.go` 的 sibling fallback 分支；`comment_csc_bootstrap_test.go`

## [OPT-20260816-020] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: taskFE 1c1fd9f 已推送；删除失效 ServerConfig.logic.featureParamsSourcesAvailable.test.js，来源可用性已由 useServerConfigFeatureParams.sourcesAvailable.test.js（company true/false）+ taskDetailFeatureParamsBridge.test.js 覆盖，vitest 8 passed
- **Created**: 2026-08-16
- **Context**: `ServerConfig.logic.featureParamsSourcesAvailable.test.js` 仍 stub `ServerConfigFeatureParamsBlock` 并断言 `fp-block-stub`。智能体资源配置已迁到评论 composer，`ServerConfig.logic.vue` 不再挂该 block，测例 2 条必然找不到 stub。
- **Action**: (1) 删除或改写该文件，改为对 `TaskDetailCommentComposer` / `taskDetailFeatureParamsBridge` 断言 `:sources-available` (2) 用 vitest 确认新测例覆盖 company true/false
- **Why**: 失效测例会在抽测门禁里制造噪声，也掩盖 composer 接线回归。
- **How to apply**: `taskFE/app/src/components/ServerConfig.logic.featureParamsSourcesAvailable.test.js`；接线在 `TaskDetailCommentComposer.vue` 与 `taskDetailFeatureParamsBridge.js`

## [OPT-20260816-053] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: runAll task-events intent_path SSOT 端口解析落地：runAll LoadConfig 从 domain-events YAML 读 port 生成 health/liveness URL（groupId/name 不匹配加载即报错）；conf/runAll.yaml 46 条 task-events 迁移 intent_path+health_path（去手写端口）；db CI 检查改为 intent_path 解析 + 禁手写端口；AiMonitor prometheus 生成器兼容 intent_path。runAll 0d3f9dc / conf 77a9770 / db d072a86 / AiMonitor 9db20e2 已推送。bin/runAll 已重建（skip-orphan marker 通过），运行中实例待下次重启生效。
- **Created**: 2026-08-16
- **Context**: ADR-0012 先用 CI 堵住探活端口与 listen SSOT 漂移，runAll.yaml 仍手写 `${INFRA_HOST}:<port>`。Agent 复制相邻条目时仍可能写出错误端口（门禁会拦提交，但本地未跑 hook 时仍会把错误配置写进工作区）。
- **Action**: (1) 给 `task-events-*` 增加 `intent_path`（或等价字段）指向 `conf/events/domain-events/<event>` + intent key (2) runAll `LoadConfig` 用该 path 读 YAML `port` 生成 health/liveness URL，禁止再手写端口 (3) 迁移现有 34 条条目并补表征测试，确认探活语义不变
- **Why**: 双份端口是本次事故的结构根因；CI 是防护网，从编排 schema 消掉第二份端口才能让拷贝不再有「改 name 漏改 port」的坑。
- **How to apply**: `runAll` 配置加载（`runAll/src`）；`conf/runAll.yaml` `domain-events-intents`；SSOT `conf/events/domain-events/`；门禁 `db/scripts/ci/check_task_events_runall_health_ports.py` 可改为断言「无手写端口」或保留对照

## [OPT-20260816-060] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 2026-08-17 经 9999 全部重新编译 64/64、初始化全部数据库 status=ok、全部重启 72/72 且 /api/status healthy=72。覆盖本条目所列服务的编译重启与 dataMigrate。
- **Created**: 2026-08-16
- **Context**: 本轮已推送任务帖定价/续费单价、job execution event 表、小米 MiMo 推荐模型替换等 SQL，以及 taskCloudService / taskEvents / taskFE / runAll 等源码。业务进程不跑迁移；现网库与进程仍是推送前状态。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「初始化全部数据库」使 `dataMigrate/taskBill/040_*`、`041_*`、`dataMigrate/taskCloudService/021_*`、`023_*` 生效；(2) 对 task-cloud-service、task-events、task-bill、taskFE、runAll、task-container-gateway、task-agent-support、task-credential-service、go-relay-to-trae、task-ai-provider 执行「精准编译重启」；(3) 确认各服务 health 与 job-stream 落库路径可用。
- **Why**: 只推源码不跑 9999 迁移则 job execution 表缺失、定价仍是旧值；不重启则容器/前端仍跑旧二进制。
- **How to apply**: 迁移入口 `http://10.2.150.68:9999/`；服务名单见 `conf/runAll.yaml`；SQL 在 `dataMigrate/taskBill/` 与 `dataMigrate/taskCloudService/`。

## [OPT-20260817-001] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: runAll UI 监听丢失立即退出（uiListenerLost 置 skip + cancel）+ 启动残留 runAll 扫描（SIGKILL opt-in RUNALL_CLEANUP_RESIDUAL=1，daemon 排除）；回归 6 例全绿，runAll 5701cd7 已推送。
- **Created**: 2026-08-17
- **Context**: 经 9999 做全部编译/初始化/重启前，主机上除当前 UI 进程外还有两个 `bin/runAll`（PID 2122292、3909745，后者已跑 2 天）不监听 :9999。它们仍跑健康检查循环；对其中任一发 SIGTERM 会按「关闭自身」路径去停托管服务。本次用 SIGKILL 清掉后 72/72 healthy。
- **Action**: (1) 在 `runAll/src/main.go` 对「UI ListenAndServe 返回 / shutdown-self 完成 / 未能占用 ui-port」三条路径断言进程退出；(2) 启动时若发现同二进制、非本 PID、且未监听 ui-port 的残留进程，记录并可选清理（不要 SIGTERM 以免误停托管服务）；(3) 补回归：模拟 bind 失败与 shutdown-self 后不得残留进程。
- **Why**: 残留监督进程会与新实例抢健康检查、在 SIGTERM 时误杀正在跑的服务，导致 9999 空白或 orphan 端口清理把整组服务杀掉。
- **How to apply**: `runAll/src/main.go`（killPreviousRunAllProcess / ListenAndServe）；`runAll/src/ui.go` shutdown-self；用 `lsof -iTCP:9999 -sTCP:LISTEN` 与 `pgrep -af bin/runAll` 对照验收。

## [OPT-20260816-034] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 删除未挂载的 TaskDetailNestedReposCloneStatus（components/ 与 task-detail/ 两份 + 三份测试副本，均已无生产 import）；克隆进度与开关由评论执行细节/composer 承担。taskFE dbb7ea9 已推送，clone-progress/composer 单测 52 例全绿。
- **Created**: 2026-08-16
- **Context**: 任务详情已不再 import `TaskDetailNestedReposCloneStatus`（`components/` 与 `task-detail/` 各有一份），开关 UI 现复制在 `CommentComposerRepoIdentity.vue`。克隆进度仍应由评论执行细节承担。
- **Action**: (1) 确认无生产 import 后删除未挂载组件与重复测试副本 (2) 若执行细节仍需子仓状态列表，只保留列表、不要第二套开关 (3) 可选：抽出 `AutoCloneNestedReposToggle` 供 composer / 项目详情复用
- **Why**: 死组件会让后人继续往错误挂载点加功能；两套开关 DOM 也容易 testid 冲突。
- **How to apply**: `taskFE/app/src/components/TaskDetailNestedReposCloneStatus.vue`；`taskFE/app/src/components/task-detail/TaskDetailNestedReposCloneStatus.vue`；`CommentComposerRepoIdentity.vue`

## [OPT-20260816-047] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: WorkPanel initData 抽出 resolveWorkPanelInitialWorkspace：URL workspace_id 优先，/me/ 仅在 URL 未指定时回退；删除覆盖后删 query 的 replaceState。taskFE 4d2b46a 已推送，5 例单测全绿。
- **Created**: 2026-08-16
- **Context**: 修导航栏「打开任务」时发现 `WorkPanel.vue` `initData` 在 `userData.current_workspace` 与 URL `workspace_id` 不一致时，会改用 /me/ 的当前工作空间，并用 `history.replaceState` 删掉 URL 上的 `workspace_id`。整页进入带 `workspace_id`/`task_id` 的工作面板深链时，可能加载错误看板且深链打不开目标任务。
- **Action**: (1) URL 带 `workspace_id` 时以 URL 为准设置 `currentWorkspace` (2) 删除「不一致就从 URL 抹掉 workspace_id」的 replaceState (3) 补测：/me/ 返回另一 workspace 时仍加载 URL 指定 workspace 的 todos，并保留 query
- **Why**: 导航栏「工作面板」链接仍走 work-panel 深链；不修则跨工作空间跳转会静默落到用户上次工作空间。
- **How to apply**: `taskFE/app/src/views/WorkPanel.vue` 的 `initData`；`useWorkPanelTaskIdDeepLink.js` 可一并断言 query 未被清掉

## [OPT-20260816-036] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: comment-execution-panel-ztree 拆 v-if(能力开关)+v-show(Tab)，切走保留 LayerGraphZtree 展开态；单测改 style.display 断言（jsdom isVisible 恒 false）。taskFE 6e3ea1c 已推送，34 例全绿。
- **Created**: 2026-08-16
- **Context**: 035 把 ztree 做成独立 Tab 后，面板用 `v-if` 与「执行细节 / 服务器运行状态」一致。切走再切回会卸载 `LayerGraphZtree`，树展开/滚动丢失（选中节点仍在父级 state）。
- **Action**: (1) 将 `comment-execution-panel-ztree` 改为 `v-show`（或三面板统一 v-show 并改既有 `exists()===false` 断言为可见性）(2) 补测：切到执行细节再回来，slot-ztree 仍是同一挂载实例或展开态保留
- **Why**: 可写层树节点多，反复展开成本高；用户会在任务关联与执行细节之间来回看连接状态。
- **How to apply**: `TaskDetailCommentExecutionDetails.vue` 的 `comment-execution-panel-ztree`；测例 `TaskDetailCommentExecutionDetails.ztree-tab.test.js`

## [OPT-20260816-042] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: SSE phase=step 即时按 step_number 合并 step_number/delivery_summary/state 进 layerJobExecutionPayload.steps；GET 作 hydrate/对账，迟到缺页经 mergeAgentStepsByNumber 不丢步。taskFE 1e7aadd(接线)+69673b4(修复) 已推送，taskDetail 65 文件 433 例全绿。
- **Created**: 2026-08-16
- **Context**: 前端收到 `phase=step` 会立刻 `refreshZTreeExecutionLog` 打 GET。persist 消费者是异步的，可能读到缺最新一步的 DB。
- **Action**: (1) 在 `updateServerStatus.js` 的 `container_job_stream`/`step` 分支把 `step_number`/`delivery_summary` 合并进 `layerJobExecutionPayload.steps` (2) GET 仍作 hydrate/对账，不要用空页覆盖已有步骤 (3) 补测：SSE step 先到、GET 后到仍保留该步
- **Why**: 用户会看到「SSE 有输出、步骤列表晚一拍或闪一下」。
- **How to apply**: `taskFE/app/src/composables/taskDetail/updateServerStatus.js`；`mergeAgentStepsByNumber`；`taskDetailExecLog.js`

## [OPT-20260816-021] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 删除死代码 hasAnyFeatureParamsEnvSource（可用性已由后端 env_var_sources_available 驱动）及其单测；空状态文案统一为「暂无可用智能体资源配置」并同步 Playwright 断言。taskFE 731dba0 已推送，16 例单测全绿。
- **Created**: 2026-08-16
- **Context**: 可用性已改由后端 `env_var_sources_available`（含 LLM 配置）驱动，前端 `hasAnyFeatureParamsEnvSource` 仍只数 extra_env_vars key，且生产路径不再调用。空状态仍写「暂无可用环境变量」，与「智能体资源配置」称谓不一致。
- **Action**: (1) 删除 `hasAnyFeatureParamsEnvSource` 及其单测，或改为转调后端标志语义 (2) 将 hint 改为「暂无可用智能体资源配置」并更新 Playwright 断言
- **Why**: 死代码会让后人再次按 extra_env_vars 口径改可用性；文案不一致则用户仍会以为只缺自定义变量。
- **How to apply**: `taskFE/app/src/utils/envParamsSourceSelection.js`；`ServerConfigFeatureParamsBlock.vue`；`taskFE/tests/WorkPanel.create-task-feature-params-unavailable.playwright.test.js`

## [OPT-20260816-057] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: missing 前序标「未加载」（不在当前页），空态/缺失行提示可加载更早评论；taskFE 231eec9 已推送
- **Created**: 2026-08-16
- **Context**: `listCommentPredecessors` 只在当前 `displayComments` 里解析前序。Feed 分页时，更早的前序可能尚未加载；显式 `depends_on` 会变成 `missing`+「未知」。
- **Action**: (1) 评估「加载更多」后刷新列表，或对 missing id 用已有评论摘要缓存 (2) 空态改为「前序不在当前页，可加载更早评论」并补测
- **Why**: 长线程串行等待时，用户会误以为前序丢失而不是未加载。
- **How to apply**: `useCommentExecutionContext.js` 的 `listCommentPredecessors`；`TaskDetailCommentPredecessorList.vue` 空态/missing 行。

## [OPT-20260816-035] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: pending 启机 TraceId 暂存落 MySQL 旁路表，跨重启/多副本可回填；taskCloudService a5074b7 + dataMigrate 907f1d9 已推送
- **Created**: 2026-08-16
- **Context**: start-vm persist 早于 binding INSERT 时现已用进程内 `sync.Map` 暂存，ensure INSERT 后 drain。taskCloudService 重启或多副本时暂存丢失，已启动中的评论冷打开仍可能缺 TraceId。
- **Action**: (1) 用 binding 旁路表或 `INSERT ... ON DUPLICATE KEY` 把 pending trace 落到 MySQL (2) ensure/list 时 drain 落库值 (3) 补测：persist → 模拟进程重启（清空 map）→ create binding 仍能回填
- **Why**: 内存暂存修的是同进程竞态；跨重启/多副本仍会丢第二条评论的启动 TraceId。
- **How to apply**: `taskCloudService/src/comment_container_binding_start_trace.go` 的 `pendingBindingStartTrace`；DDL 走 `dataMigrate/taskCloudService/`

## [OPT-20260816-027] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 孤儿 CSC 对账改走 taskTaskService /api/internal/tasks/exists/ 批量 HTTP，去掉 task-task 库直连；taskTaskService cfdda99 + taskCloudService cbb8680 已推送
- **Created**: 2026-08-16
- **Context**: `startOrphanTaskCSCReconcileTicker` 默认每 10 分钟扫 CSC；`server_orphan_reconcile.go` 用 `ResolveMySQLDSN("task-task")` 跨服务查任务是否存在。
- **Action**: (1) 在 taskTaskService 提供批量「task_id 是否存在」internal API，或复用/补强 `TASK_DELETED` 后只做补偿 (2) `POST /api/internal/cloud/compute/reconcile-orphan-csc/` 只走 HTTP，删除对 task-task DSN 的直连 (3) 单测：他库 DSN 不再出现在该文件 (4) 时钟不要单独拉起，与 028 共用 `cloud_csc_reconcile/1_sweep`（本条先把 Once API 做绿）(5) 从 allowlist 去掉本文件（若 ticker 仍在 main，则等 028 同一 PR 再删调用）
- **Why**: 跨库扫任务违反单库单表所有权；进程内 ticker 无法独立探活。
- **How to apply**: 计划 §027；`taskCloudService/src/server_orphan_reconcile.go`；`resolveAndCacheTaskDBPath` 必须消失

## [OPT-20260816-028] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 孤儿/泄漏 CSC 对账迁 taskEvents cloud_csc_reconcile/1_sweep timer（先 POST orphan 再 POST leaked，300s）。taskCloudService 去 2 进程内 ticker、新增 reconcile-leaked-servers 端点并修复 clear 带 instanceID 精确命中泄漏行；openapi-internal 补 2 端点文档；runAll/conf 登记 18063 服务；db allowlist 移除 2 文件。taskCloudService 6f3dbb6+35a81fb、taskEvents 9e16146、conf 685ff2b、db 7817908 已推送。待 runAll 重载配置后启新 timer。
- **Created**: 2026-08-16
- **Context**: `startLeakedServerReconcileTicker` 默认每 5 分钟清 `terminal_released=1` 但仍有 `server_url` 的 CSC。orphan（027）先打标，本条再清资源，应是同一条流水线。
- **Action**: (1) `POST /api/internal/cloud/compute/reconcile-leaked-servers/` 调 `reconcileLeakedServerURLs` (2) 新建 timer `cloud_csc_reconcile/1_sweep`（建议 18063）：一次 tick 先 POST orphan 再 POST leaked (3) 从 `main.go` 删除 `startLeakedServerReconcileTicker` 与 `startOrphanTaskCSCReconcileTicker` (4) 两文件都移出 `LEGACY_INTERNAL_TICKERS` (5) runAll depends_on task-cloud-service；间隔建议 300s（可覆盖原 5min/10min）
- **Why**: 两个业务 ticker 会漏掉「先标后清」顺序；独立 timer 才能在 cloud 重启时仍扫。
- **How to apply**: 计划 §028；依赖 027 的 orphan API；`taskCloudService/src/server_release_reconcile.go`

## [OPT-20260816-030] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 排队自动开跑迁 taskEvents queued_auto_run_scan/1_dispatch timer（30s，POST dispatch-once）。taskTaskService 去 30s 进程内 ticker、新增 /api/internal/tasks/queued-schedule/dispatch-once/ 端点；taskEvents 登记 18065；conf/runAll 登记；db allowlist 移除。taskTaskService 07e4e3a、taskEvents 0a639e0、conf 2416827、db 7124ccc 已推送。待 runAll 重载配置后启新 timer。
- **Created**: 2026-08-16
- **Context**: `startQueuedScheduleTicker` 每 30s 跑 `runQueuedScheduleDispatchOnce` + `runQueuedScheduleAutoCloseOnce`，嵌在 taskTaskService HTTP 进程内。
- **Action**: (1) `POST /api/internal/tasks/queued-schedule/dispatch-once/` 顺序调用这两个 Once (2) timer `queued_auto_run_scan/1_dispatch`（建议 18065，30s）(3) 删除 `startQueuedScheduleTicker` 及 main 调用 (4) 从 allowlist 去掉该文件 (5) 单测 dispatch-once 幂等 + 门禁
- **Why**: 节奏窗口调度失败时现在只会跟着任务服务一起沉默；独立 worker 才能告警。
- **How to apply**: 计划 §030；`taskTaskService/src/queued_schedule_dispatch.go`；`taskTaskService/src/main.go`

## [OPT-20260817-002] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: live Describe 缓存带宽到 CSC（026 迁移），auth_missing 回退读缓存注入 instance_attribute.body；taskCloudService 7e98877 + dataMigrate 854baf2 已推送，026 已应用生产 task_cloud；双测全绿
- **Created**: 2026-08-17
- **Context**: 本次已从 DescribeInstances 映射并在评论运行态实例详情展示带宽计费模式与公网出/入带宽。`auth_missing` 回退只拼 Status/公网 IP，CSC 也不存带宽，授权失效时网格仍缺流量带宽。
- **Action**: (1) 启动会话或 CSC 持久化 `internet_charge_type` / `internet_max_bandwidth_out` (2) `buildAuthMissingRuntimeFallback` 写入 `instance_attribute.body` (3) 补测授权缺失时仍能展示带宽行
- **Why**: 授权过期后用户仍能看到实例在跑，但带宽信息再次消失，和本次缺口同类。
- **How to apply**: `server_runtime_auth_fallback.go`；`CloudServerConfig` / history；`buildServerRuntimeStatusDetails` 已能消费这些字段，无需再改展示层

## [OPT-20260817-007] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: exec_queue.json 落盘；UI 启动 LoadExecQueueFromDisk；热替换回读 pending 单测通过且不占用 bulk 锁
- **Created**: 2026-08-17
- **Context**: 本次把执行队伍进度改由 runAll `/api/status` 的 `execution_queue` 下发，刷新后可恢复。队列状态仍只在进程内存；`shutdown-self` 热替换后 pending/current 会清空。
- **Action**: (1) 将 `execQueueState` 与 current run_id 写入 `.runall/exec_queue.json` (2) UI 模式启动时回读并接到 ProgressBroadcaster.Latest (3) 补热替换前后刷新仍能看到排队项的单测
- **Why**: 热替换是 runAll 日常发布路径，只靠内存则刷新修复在替换窗口会再次「看不见执行队伍」。
- **How to apply**: `runAll/src/exec_queue.go`；路径约定对齐 `bulk_op.lock` / `.runall/`

## [OPT-20260817-008] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: formatProgressDetail 按 op 区分等待文案；组装页不再含「编译/健康检查中…」
- **Created**: 2026-08-17
- **Context**: `bulkQueue.formatProgressDetail` 在 `current` 为空且未 done 时固定文案「编译/健康检查中…」。全部启动的健康等待阶段也会显示这句，刷新恢复后尤其容易误解成还在编译。
- **Action**: (1) 按 `progress.op` / `event.operation` 区分 start/stop/build/precise-restart 的等待文案 (2) 补 queue-status-bar 片段断言
- **Why**: 刷新后执行队伍是用户判断「现在在干什么」的唯一条，文案错会让人以为卡住或在编不该编的服务。
- **How to apply**: `runAll/src/status_ui/js/07.js` `formatProgressDetail`；`queue_status_bar_ui_test.go`

## [OPT-20260817-011] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 公网 https://provider.daydaymoney.com/ 已硬刷新验收：顶栏「镜像Demo」href=https://github.com/task2money/trae-agent，target=_blank，rel=noopener noreferrer。SPA 由磁盘 dist 即时生效，无需等全量精准重启。
- **Created**: 2026-08-17
- **Context**: 厂商门户顶栏已加「镜像Demo」外链到 `https://github.com/task2money/trae-agent`，并已登记 `ai-provider`。现网 `https://provider.daydaymoney.com/` 仍是旧 SPA；登记文件里还有大量他会话 WIP 服务，不宜点「精准编译重启」一次全编。
- **Action**: (1) 待 WIP 批提交后，在 http://10.2.150.68:9999/ 对 `ai-provider` 执行编译重启（或单独重启该服务）(2) 硬刷新 `https://provider.daydaymoney.com/` (3) 确认顶栏「镜像Demo」`href` 为 GitHub trae-agent 且新标签打开
- **Why**: 未重建前端 dist / 未重启进程则公网页看不到新链接。
- **How to apply**: 服务名 `ai-provider`；源码 `taskAiProvider/frontend/src/App.vue`；testid `nav-image-demo`

## [OPT-20260817-012] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: INDEX 2.10 已补 F-088 saas_machine_container_skill_link；厂商门户申请认证顺延为 F-089
- **Created**: 2026-08-17
- **Context**: `docs/intents/frontend/provider/saas_machine_container_skill_link.intent.md` 已存在且有测试意图，但 `docs/intents/INDEX.md` 未收录。本次只登记了同目录新意图 F-087。
- **Action**: 在 INDEX「2.10 厂商门户 / 镜像市场」增加 F-088 行，指向 `frontend/provider/saas_machine_container_skill_link` 与 `saasMachineContainerSkill.unit.test.js`
- **Why**: 目录漏登会导致 `/intent-test` 与人工对照时以为该入口没有意图文档。
- **How to apply**: `docs/intents/INDEX.md` 节 2.10；意图文件 `docs/intents/frontend/provider/saas_machine_container_skill_link*.md`

### OPT-20260817-030 — 清理 CommentsSection 仅供已删 fallback 的任务级 props

- **Status**: completed
- **Created**: 2026-08-17
- **Completed**: 2026-08-17
- **Completion-Note**: 已从 Section defineProps 与 commentsSectionProps 移除 serverStatus/statusLogs/containerHeartbeat*/runtimeStatus/statusTraceId 等；单测 noEmptyCommentExecutionFallback 守住。
- **Context**: 已移除零评论 `execution-details-fallback`，但 `TaskDetailCommentsSection` 仍声明并接收 `serverStatus`/`statusLogs`/`containerHeartbeat*`/`runtimeStatus` 等任务级 props，仅曾给 fallback 用；父级 bindings 仍透传。
- **Action**: (1) 从 Section `defineProps` 删除未再读取的任务级运行态 props (2) 同步 `taskDetailSectionBindings.js` / `useTaskDetailCommentsSectionBindings` 停止透传 (3) 源码扫描单测守住无回退
- **Why**: 减少死 props 与误读「任务级状态仍注入评论区」的维护成本。
- **How to apply**: `TaskDetailCommentsSection.vue`；`taskDetailSectionBindings.js`；对照本次 `noEmptyCommentExecutionFallback` 测例

## [OPT-20260817-033] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: 热替换孤儿 current 已改为 abandon→pending；cancel 可清幽灵；单服务 8min 超时；runAll 已重建并自动续跑，已越过原卡点 fanout-work-panel-sse（22+/53）
- **Created**: 2026-08-17
- **Context**: 修复零评论空 comment-id「执行细节」fallback 后，9999 精准编译重启卡在 53 服务中的第 21 个 `task-events-task-status-changed-2-fanout-work-panel-sse`；浏览器未登录任务详情 HTTP 403，无法在真实页面 DOM 验收。
- **Action**: (1) 清理/取消卡住的 precise-restart 队列，必要时只重启 taskFE (2) 用已登录 Chrome CDP/Playwright 打开该任务页，断言无 `comment-execution-details-fallback-wrap` 且无 `data-comment-id=""` 的 details
- **Why**: 公网已出新 hash JS，但缺少登录态 E2E 闭环；卡住的 bulk restart 会影响后续部署。
- **How to apply**: `http://10.2.150.68:9999/`；`taskFE/tests` 或人工硬刷新；对照 `comment_execution_details.test.intent.md` T4

## [OPT-20260817-041] completed

- **Status**: completed
- **Completed**: 2026-08-17
- **Summary**: formatGitLabAPIError 接入 list-branches/resolve-commit/probe-repo/read-file；单测覆盖 401/403/429/限流文案；不再把通用 403 误标为 rate limit
- **Created**: 2026-08-17
- **Context**: 修复 GitHub 分支列表 403 只显示硬编码文案时，发现 `fetchGitLabBranches` / `probeGitLabRepo` / `fetchGitLabCommit` 在 401/403/default 同样丢弃响应体（403 甚至误标为 rate limit）。
- **Action**: (1) 抽取 `formatGitLabAPIError` 解析 JSON `message` (2) 替换 GitLab 各调用点硬编码 (3) 补单测覆盖 401/403/限流
- **Why**: 与 GitHub 同类问题：用户看不到 Git 网站真实拒绝原因，排障只能猜。
- **How to apply**: `taskProjectService/src/git_branches.go` `fetchGitLabBranches`；`git_repo_validate.go` `probeGitLabRepo`；`git_resolve_ref.go`；可复用 `github_api_error.go` 模式新建 `gitlab_api_error.go`

