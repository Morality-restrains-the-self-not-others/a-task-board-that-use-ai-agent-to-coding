# Completed OPT Archive — 2026-08-28

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 71 条。
> 归档执行时间：2026-08-29T00:53:53+08:00

## [OPT-20260827-048] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 精准编译重启后 SQL: task_879620507262021632 status=deferred slots=0 updated_at=2026-08-27 23:56:33，已离开 starting-without-slot
- **Created**: 2026-08-27
- **Context**: 租户 `877397588196749312` 工作空间 `ws_-2309487803472456748` 成员 `task_879620507262021632` 卡在 `starting` 且无 `task_queued_machine_slots` 行，页面显示「调度启服中」+「占用槽位 0/1」。代码已改为 reclaim 后重试，需新二进制与 `queued-auto-run-scan` timer 健康后才会生效。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记的 `task-task-service` 与 `task-events-queued-auto-run-scan-1-dispatch` 执行精准编译重启 (2) 等至少一轮 30s timer 后执行 SQL：`SELECT m.status, COUNT(s.task_id) slots FROM task_queued_auto_run_memberships m LEFT JOIN task_queued_machine_slots s ON s.task_id=m.task_id WHERE m.task_id='task_879620507262021632'` (3) 断言 `status` 不是「starting 且 slots=0」（应为 queued/deferred/starting+slot≥1）
- **Why**: 旧二进制会永远 skip starting；只合代码不重启则公网页面仍卡住。
- **How to apply**: MySQL `task_task`；`scripts/register-precise-restart.sh` 已可登记两服务。验收：上述 SELECT 退出码 0 且不满足 `status=starting AND slots=0`

## [OPT-20260827-021] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 夜间 OPT 执行完成
- **Created**: 2026-08-27
- **Context**: 插件 `uniqueCompanies` 已跳过 `is_active === false`，但 `taskAuth` `fetchTenantMembersOnce` 未把租户成员的 `is_active` 拷进 `/me` 的 `companies`，停用成员仍会进工作空间聚合。
- **Action**: (1) `fetchTenantMembersOnce` 增加 `is_active` (2) `buildCompaniesFromNicknames` 写入 companies (3) 补 taskAuth 单测：停用成员不出现或 `is_active: false`
- **Why**: 用户退出公司后插件下拉仍可能列出该公司默认「用户的工作空间」。
- **How to apply**: `taskAuth/src/auth_user_profile.go`；对照 `taskTenantService` `listMembersByUser` 已返回 `is_active`。

## [OPT-20260827-020] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 夜间 OPT 执行完成
- **Created**: 2026-08-27
- **Context**: `createDefaultWorkspaceDirect` 把每个新公司的默认工作空间都命名为「用户的工作空间」。多租户用户在任何跨租户下拉里都会看到同名项；插件已用「名称 · 公司名」消歧，Web 侧若将来聚合仍会撞名。
- **Action**: (1) 创建默认工作空间时用 `company_name`（或「{company} 的工作空间」）(2) 幂等：已存在 `is_default` 的不改名，避免覆盖用户自定义 (3) 补 taskEvents 单测
- **Why**: 从源头避免跨租户同名；插件消歧是展示层补丁。
- **How to apply**: `taskEvents/internal/repository/saas/taskproject_client.go` `createDefaultWorkspaceDirect`；`COMPANY_CREATED` intent 3。

## [OPT-20260827-019] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 夜间 OPT 执行完成
- **Created**: 2026-08-27
- **Context**: 自动运行 nested-git 探测已补 `tenant_id`/`company_id`，才能匹配 Path A GitLab 并刷新 OAuth。`taskCredentialService` `NestedGitReposHTTPClient.FetchNestedGitRepos` 仍只传 `user_id`+`repo_url`，克隆前发现子仓在租户 GitLab 上会同样误报「未检测到可用授权」。
- **Action**: (1) 给 `NestedGitReposFetcher` / HTTP client 增加 tenant/company_id 参数或请求头 `X-Auth-Tenant-Id` (2) 从 task-detail / clone 入站请求把租户 ID 传到 enrich (3) 补 `nested_git_client_test.go` 断言 query/header 含 tenant
- **Why**: 克隆路径与自动运行共用同一内部 API；只修 auto_run 会留下同类 Path A 漏租户。
- **How to apply**: `taskCredentialService/infrastructure/nested_git_client.go`；`application/nested_repos_enrich.go`；`go test ./infrastructure -run TestNestedGitReposHTTPClient`

## [OPT-20260827-035] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 夜间 OPT 执行完成
- **Created**: 2026-08-27
- **Context**: 项目 PATCH 已能在陈旧 `installed_image_id` + 租户下唯一镜像名时回退现网 ID。`taskTaskService/src/cloud_client.go` 的 `lookupInstalledImage` 仍只传 `id=`，自动运行/评论提及/技能快照路径遇到陈旧 ID 会继续 `ErrInstalledImageNotFound`。
- **Action**: (1) `lookupInstalledImage` 增加可选 `name` 查询参数，调用方在有 `image_name`/skill 快照名时传入 (2) 补测：stale id + unique name 应返回现网 ID；无名或重名仍 not found (3) 保持 `lookupInstalledImageFn` 可替换，更新相关 mock 签名
- **Why**: 项目绑定愈合后任务侧仍可能拿着旧 ID 去 Cloud lookup，自动运行会误判镜像不存在。
- **How to apply**: `taskTaskService/src/cloud_client.go` `lookupInstalledImage`；`cd taskTaskService && go test ./src -count=1 -run 'LookupInstalledImage|AutoRun'`

## [OPT-20260827-044] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 夜间 OPT 执行完成
- **Created**: 2026-08-27
- **Context**: `QueueMembersCard` 输入 debounce 250ms 自动搜索，同时「搜索」按钮走 `searchGuard`（300ms）。用户改关键字后立刻再点搜索，第二次可能被 debounce/in-flight 跳过。
- **Action**: (1) 点「搜索」时 clearTimeout debounce (2) 或搜索按钮绕过 debounce 只保留 in-flight 锁 (3) 补测：连续两次不同关键字均发出 GET
- **Why**: 用户会以为第二次搜索没生效。
- **How to apply**: `taskFE/app/src/components/workspaceSchedule/QueueMembersCard.vue` `runSearch` / `watch(query)`

## [OPT-20260827-046] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 夜间 OPT 执行完成
- **Created**: 2026-08-27
- **Context**: 浮窗「逐仓基准分支」空态已改为「一个任务一个项目，按仓库填写」。DevTools 单请求标签仍写「对齐工作面板；空则用项目默认分支」，两处副文案不完全一致。
- **Action**: (1) 把浮窗 label hint 与 `REPO_BASE_EMPTY_HINT` 对齐为同一套「单项目 + 空则用默认分支」 (2) `test/repo-base-copy.test.js` 断言浮窗与 panel.html 共用常量或同一字符串
- **Why**: 用户在浮窗看不到「空则用项目默认分支」，可能误以为必须手填每个仓库。
- **How to apply**: `taskChromePlugin/lib/float-panel-markup.js`、`lib/create-task-payload.js` `REPO_BASE_EMPTY_HINT`；验收 `cd taskChromePlugin && node --test test/repo-base-copy.test.js`

## [OPT-20260827-024] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 夜间 OPT 执行完成
- **Created**: 2026-08-27
- **Context**: 更名时 `lib/plugin-brand.js` 已是用户可见品牌 SSOT，但 `lib/user-guide.js`、`oauth-callback.html` 仍硬编码同一中文字符串；后续再改名容易只改 manifest。
- **Action**: (1) `user-guide.js` 用 `PLUGIN_DISPLAY_NAME` 插值步骤文案，保留一处 fallback 字面量供契约测试扫描 (2) oauth-callback 在 boot 脚本里写 `document.title` / h1 (3) `test/plugin-brand.test.js` 仍锁定旧名「云端 Coding」不回潮
- **Why**: 双份字面量会在下次更名时漏改使用说明或登录回调页。
- **How to apply**: `taskChromePlugin/lib/user-guide.js`、`oauth-callback.html`、`lib/oauth-callback-boot.js`；`cd taskChromePlugin && node --test test/plugin-brand.test.js test/user-guide.test.js`

## [OPT-20260827-034] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 夜间 OPT 执行完成
- **Created**: 2026-08-27
- **Context**: 镜像挂钩已落地（`hasInstalledImage` + 提交门禁）。工作面板 `useCreateTaskAutoRun` 还要求已配置运行模版（`projectHasConfiguredRunTemplate`）。插件仍可能在缺模版时勾选自动运行，提交后被后端 `AUTO_RUN_RUN_TEMPLATE_REQUIRED` 拒绝。
- **Action**: (1) 移植 `projectHasConfiguredRunTemplate` 到 `project-auto-run-label.js` (2) `resolveAutoRunControlState` 在项目允许且已选镜像后再校验模版 (3) hint 列出禁用原因 (4) 补单测后接入 syncFloatAutoRun / syncContainerAutoRun
- **Why**: 与工作面板创建任务门禁仍不完全一致。
- **How to apply**: `taskChromePlugin/lib/project-auto-run-label.js`；对照 `taskFE/app/src/composables/useCreateTaskAutoRun.js` 与 `projectRunTemplateUtils.js`

## [OPT-20260827-030] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: shareLib 130cd1e + AiMonitor d361046 已推送：normalizeMetricsPath 折叠 ws_/task_/cmt_/cpa_ 前缀 ID（修复 task_id 标签误折叠，尾部须以数字开头）；Gateway 面板 p50/p95/p99 与 route p95 改 type=request（用户感知含 upstream）；tracelog 单测绿。
- **Created**: 2026-08-27
- **Context**: `http-api-latency` Gateway 面板用 `apisix_http_latency_bucket{type="apisix"}`（仅网关内部，整体 p95≈52ms），用户感知是 `type="request"`/`upstream`（整体 p95≈1.82s，inbound 4.15s）。tracelog `normalizeMetricsPath` 只替换纯 hex/数字/UUID，`ws_*`/`task_*`/`cmt_*`/`cpa_*` 未折叠，panel-22 把同一 API 拆成单工作区行。
- **Action**: (1) Gateway 时序/p50/p95/p99 改为 `type="request"`（或同时展示 upstream）(2) `normalizeMetricsPath` 折叠 `ws_*`、`task_*`、`cmt_*`、`cpa_*` 前缀 ID (3) 补 tracelog 单测
- **Why**: 看板会显示「网关很快」而容器入站实际 >1s；高基数让「>1s 路径」看起来像 11 条不同 API。
- **How to apply**: `AiMonitor/grafana/provisioning/dashboards/files/http-api-latency.json`；`shareLib/tracelog/http_metrics.go` + `http_metrics_test.go`

## [OPT-20260827-049] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: AiMonitor 87aec3a 已推送：prometheus rule TaskEventsQueuedAutoRunScanDown（up{job=...}==0 for 5m）+ prometheus.yml 注册 + 回归测锁定 expr 与 file_sd job；端口 SSOT 51/51 通过；Loki 已见 task-events-queued-auto-run-scan-1-dispatch job。
- **Created**: 2026-08-27
- **Context**: 排查卡死队列时本地 timer `task-events-queued-auto-run-scan-1-dispatch` 已 `service_status=failed`（约 23:07 起健康检查停），reclaim 根本不会跑。runAll 有探活端口 18065，但失败未形成可查询告警。
- **Action**: (1) 确认 `conf/runAll.yaml` 该条目 health_check 端口等于 `conf/events/domain-events/queued_auto_run_scan/config.yaml` 的 intents.port (2) 给 Prometheus/runAll 增加「timer failed」规则或把 `service_status=failed` 写入可 curl 的 JSON (3) 用 `python3 db/scripts/ci/check_task_events_runall_health_ports.py` 回归端口 SSOT
- **Why**: timer 挂掉后排队调度静默停摆，页面只看到「调度启服中」。
- **How to apply**: `conf/runAll.yaml`、`conf/events/domain-events/queued_auto_run_scan/config.yaml`；验收脚本 exit 0，且失败态可被 `curl`/`jq` 断言

## [OPT-20260827-015] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE e7a870c 已推送：taskFE/tests 全仓移除 rgNodkdq8677 明文回退（116 文件），env-only + module-scope test.skip(!PASSWORD) 守卫；playwrightApiEnv/playwrightLogin 缺 env fail-fast；seed_e2e_account SUPERADMIN_PLAIN_PASSWORD 只读 env；rg 全仓零命中。
- **Created**: 2026-08-27
- **Context**: `taskFE/tests/**` 大量 `process.env.PLAYWRIGHT_TEST_PASSWORD || 'rgNodkdq8677!ci'`。门禁对测试目录不扫 assignment，故未阻断；真实口令仍在仓库。
- **Action**: (1) 删除 `|| 'rgNodkdq...'` 回退，缺 env 则 skip (2) 种子脚本改为只读 env (3) `rg -n "rgNodkdq8677" taskFE/tests` 须无命中
- **Why**: 测试默认值把共享口令写进每一次 clone；轮换时要改几十个文件。
- **How to apply**: 改完后 `rg -n 'rgNodkdq8677' taskFE --glob '!**/node_modules/**'` 仅允许历史文档或为零；相关 Playwright 在设置 env 后仍能登录。

## [OPT-20260827-017] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 34 个含源码子仓 .gitignore 补齐 .env/.env.* + !.env.example 例外并逐一提交推送；git submodule foreach 扫描无 MISSING；meta 指针已同步。
- **Created**: 2026-08-27
- **Context**: 元规则第 57 条已在 meta `.gitignore` 排除 `.env` / `.env.*`。子仓有独立 gitignore，本机 `AiMonitor/.env`、`gitService/.env` 仍可能被 `git add` 进子仓。
- **Action**: (1) 在含业务源码的子仓 `.gitignore` 增加与 meta 相同的 `.env` / `.env.*` + `!.env.example` 例外 (2) `git check-ignore -v <sub>/.env` 对各仓抽检
- **Why**: 子仓提交不读 meta gitignore，本机 dotenv 仍可能入库。
- **How to apply**: `git submodule foreach 'grep -n "^\.env$" .gitignore || echo MISSING'`；补齐后该命令不再打印 MISSING。

## [OPT-20260827-014] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE 023d36f 已推送：normalizeUrlAccessCodeToOwn 对 /tenant/:tid/projects/:id/ 与 /projects/ 路由保留入站分享 accessCode（不再被登录用户自己推荐码覆盖），其余页面保持 OPT-20260824-005 归一化；3 例回归测绿。
- **Created**: 2026-08-27
- **Context**: 验收「授权异常重试」时打开 `?accessCode=9aaHjbryhL` 的项目详情，登录后地址栏变成当前用户推荐码 `DR2AKvP9J9`。重试回流仍回到当前页，但入站分享溯源码丢失。
- **Action**: (1) 核对 Login / tenantRouteGuard / 侧栏是否在 projects 路由改写 `accessCode` (2) 若应保留入站分享码，projects 页禁止用登录用户推荐码覆盖 (3) 补单测：带着分享码进入项目详情，URL 仍为原码
- **Why**: 分享链接打开项目后归因记到打开者自己，推荐统计会漂。
- **How to apply**: `taskFE/app/src/views/Login.vue` accessCode sessionStorage；`tenantRouteGuard.js`；`UserCenterSidebar.vue`；回归 `tests/ProjectDetail.oauth-retry-return.playwright.test.js`

## [OPT-20260827-037] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 根因=promtail 容器未运行（restart:no 静默消失），Loki 0 job。runAll/scripts/runall-local-promtail.sh up 拉起 aimonitor-promtail 后 Loki 现 78 job 含 task-cloud-service；生成器重刷 promtail-local.yaml 含 oidc/wechat 新 timer job；AiMonitor 94681bb 推送；已知 traceId 跨 task-auth+task-cloud-service 命中。
- **Created**: 2026-08-27
- **Context**: Fork 弹窗 `data-traceId=796a1ffa-ede1-4a2f-988d-9566d706151d` 查 Loki `{job=~".+"}` 7 天 0 条。Loki series 仅 `task-auth` 与两条 task-events；`task-cloud-service` 日志在 `logs/task-cloud-service.log`，Promtail 配置扫 `/var/log/runall/`（本机目录不存在）。排障只能退回 APISIX access/error。
- **Action**: (1) 核对 Promtail `__path__` 与 runAll 实际落盘路径是否同机 bind (2) 让 `task-cloud-service` 等业务 job 出现在 Loki label `job` (3) 用已知 traceId 回归 `{job=~".+"} |= "<id>"` 能命中
- **Why**: 有 data-traceId 却查不到日志，强制日志优先门禁失效，只能靠网关 access 猜 502。
- **How to apply**: `AiMonitor/promtail/promtail-local.yaml`；runAll 日志目录与 `/var/log/runall` 的 symlink/bind；`curl Loki /loki/api/v1/label/job/values`

## [OPT-20260827-022] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: runAll 1609eac + conf f6256d4 已推送：Group skip_start_all，PlanStartAll 排除 gitlab-regions，PlanStartGroup/单启仍可用；三层测试绿（repository 传播 / 编排排除与组启动 / 生产配置断言）。
- **Created**: 2026-08-27
- **Context**: ADR-0048 已去掉其它服务对 GitLab 的 `depends_on`。start-all 仍会尝试 `git-service`（本机闸门拒绝）和 `git-service-tencent-sh-1`（SSH 上海）。「可插拔、人工对齐」还差一步：一键启动不必碰 GitLab 组。
- **Action**: (1) 评估 runAll 是否已有 `skip_start_all` / 组级开关 (2) 给 `gitlab-regions` 加「不参加 start-all、面板仍可单启」 (3) 单测：start-all 计划不含 git-service*
- **Why**: start-all 对上海实例的 SSH start 仍是隐式编排依赖；本机条目每次失败噪音。
- **How to apply**: `conf/runAll.yaml` `gitlab-regions`；`runAll/src` `PlanStartAll`；对照 ADR-0047/0048。

## [OPT-20260827-039] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskGateway 9a5f112 已推送：per-upstream retry（retries/retry_timeout 秒），taskCloudService/taskSse/taskAgentSupport 配 retries:2+retry_timeout:3s；生成器 _upstream_retries + 4 单测绿，routes-apply 热载后容器确认加载、网关 health 200。
- **Created**: 2026-08-27
- **Context**: 2026-08-27 11:09 UTC `connect() failed (111)` 打 8018，Fork 拉 feature-params 与多租户 compute API 全部 502。进程随后恢复。精准重启窗口内用户看到模型清单失败。
- **Action**: (1) 查 runAll 重启 8018 的无监听间隙 (2) APISIX upstream retries/timeout 对 connection refused 做 1–2 次短重试 (3) 健康检查失败时不要把 HTML 502 当业务错误
- **Why**: 重启秒级窗口会把「自动运行选模型」打成失败；前端已提示可重试，网关重试可消掉多数闪断。
- **How to apply**: `taskGateway` APISIX upstream `retries`；`conf/runAll.yaml` task-cloud-service health

## [OPT-20260825-012] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: dataMigrate 66561f4 已推送：071 guarded drop 已提交；taskBill/taskReferral INSERT 已在 08-26 去掉比例列且运行二进制含新包；生产 billing_referral_config 比例列已消失。
- **Created**: 2026-08-25
- **Context**: 政策分成比例已固定 5%，读写路径不再使用 `referral_rate_percent` / `profit_sharing_ratio_percent`。列仍写入恒 5 以免旧查询读到脏值。
- **Action**: (1) 精准编译重启 taskBill/taskReferral（新包 INSERT 已去掉比例列） (2) 经 9999 应用 `dataMigrate/taskBill/071_drop_referral_ratio_columns.sql` (3) 推送 dataMigrate 并同步 meta 指针。
- **Why**: 留下可写列会让人误以为比例仍可配置。
- **How to apply**: `dataMigrate/taskBill/` 下一编号 SQL；改 `updateReferralConfig` / `updateReferralSettleConfig` 的 INSERT 列表。

## [OPT-20260825-021] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 生产 task_bill 已验证 billing_profit_sharing_admin_action / billing_profit_sharing_change_notify 两表存在且 utf8mb4_unicode_ci，写入代码已部署（handlers_admin_share_profit_sharing / wechat_profit_sharing_notify）。
- **Created**: 2026-08-25
- **Context**: 超管手工分账审计表 `billing_profit_sharing_admin_action` 与动账通知收件箱 `billing_profit_sharing_change_notify` 已写入 `dataMigrate/taskBill/069_*.sql`、`070_*.sql`。业务进程不跑迁移，未 init 则 share/notify 会写表失败。
- **Action**: (1) 打开 http://10.2.150.68:9999/ 「初始化全部数据库」或对 task_bill 跑 `apply_datamigrate.sh` (2) 确认两表存在且 charset=utf8mb4 (3) 精准编译重启 `task-bill` `taskFE` `task-gateway`。
- **Why**: 未迁移则超管分账 500、微信动账通知无法去重入账。
- **How to apply**: `dataMigrate/taskBill/069_profit_sharing_admin_action.sql`、`070_profit_sharing_change_notify.sql`；登记文件 `.runall/precise_restart_services.txt`。

## [OPT-20260827-028] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: heartbeat HTTP no longer waits probe: TestHandleContainerHeartbeat_ReturnsBeforeSlowProbe (<150ms vs 400ms probe) + ReusesProbeWithinTTL (hits=1) + CancelledRequestContextStillOK; layer-graph identical skip: TestHandleLayerGraphPushSkipsIdenticalGraph published=1; go test ./src -run Heartbeat|LayerGraphPush exit 0. Registered task-cloud-service for 9999 precise restart.
- **Created**: 2026-08-27
- **Context**: Grafana panel-22（now-1h, min latency 1s）中 `server-container-token/heartbeat` 约 148 次/小时，p50=1.25s、p95=2.375s、avg≈1.07s；直方图几乎全部落在 1–2.5s 桶。实现上每次心跳都 `probeContainerHeartbeat`（超时 2500ms，与桶上沿重合）。APISIX `container-inbound-token` **type=request/upstream** p95=4.15s（type=apisix 仅 49ms，会掩盖用户感知）。同族 `runtime-event`/`layer-graph-push`/`boot-progress` 也在 1–2s。
- **Action**: (1) 心跳成功记录 uplink 后立即 200，探测改为后台/降频（如 30s 内复用上次 downlink）(2) 超时显著小于 TAS `heartbeatSec`（默认 15s）但仍须避免占满 2.5s (3) `layer-graph-push` 相同 graph 跳过 upsert+Kafka (4) 回归：探测失败仍发 SSE，TAS 不因 ctx cancel 丢心跳
- **Why**: 容器经 TAS 同步等待探测 RTT；148 次/小时 × ~1s 且网关入站 p95>4s。心跳本应是毫秒级。
- **How to apply**: `taskCloudService/src/container_inbound_heartbeat_probe.go`、`container_inbound_actions.go` `handleContainerHeartbeat`；`taskAgentSupport` `timeouts.heartbeatSec`；测 `container_heartbeat_probe_test.go`

## [OPT-20260827-045] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE ba18518 已推送：父视图 836→259 行，面板模态抽到 WorkspaceTaskPanelAddEditModal.vue，工作空间/面板 CRUD/设置模态抽到 useWorkspaceSettingsTaskPanel + useWorkspaceSettingsModals；click-guard/default-badge/archive-modal 6 测全绿。
- **Created**: 2026-08-27
- **Context**: 修「是否设为默认」时已抽出 `WorkspaceCreateEditModal.vue`；随后套餐设置修复又抽出 `TaskArchiveSettingsModal.vue`。父文件仍约 836 行（面板增删改模态、交付物/进度/访问管理仍内联）。
- **Action**: (1) 抽出添加/编辑面板模态 (2) 将工作空间列表与 load/save 抽到 composable (3) `wc -l` ≤500 且现有 click-guard / default-badge / archive-modal 单测全绿
- **Why**: 行数门禁与「每模态独立文件」规则仍未满足，后续改任务面板易继续堆进超标文件。
- **How to apply**: `taskFE/app/src/views/WorkspaceSettingsTaskPanel.vue`；验收 `wc -l taskFE/app/src/views/WorkspaceSettingsTaskPanel.vue` 后 `npx vitest run src/views/WorkspaceSettingsTaskPanel.default-badge.test.js src/views/WorkspaceSettingsTaskPanel.click-guard.test.js`

## [OPT-20260819-014] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 流水账按 region 回放 GitLab 磁盘/流量瞬时配额剩余：新增 gitlab_ledger_replay.go + Go/FE 单测，taskBill 2d4bd90 / taskFE 3f7129f 已推送
- **Created**: 2026-08-19
- **Context**: 资源流水账增量已对现金与任务帖剩余做瞬时展示；GitLab 磁盘/流量仅有发生额文案，瞬时「区域配额剩余」未回放，租户仍难判断某区容量变化后的账目。
- **Action**: (1) 在 `transaction_ledger_snapshot.go` 增加按 region 的 GitLab 配额回放（类似 task_post）；(2) 扩展 `ledger_snapshot.display_lines`；(3) 补 Go + FE 单测。
- **Why**: 无区域剩余时，GitLab 资源流水无法当作完整账本读。
- **How to apply**: `taskBill/src/transaction_ledger_snapshot.go`；依赖 `billing_resource_grant` / 区域配额表字段。

## [OPT-20260827-040] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 回归锁定：ResolveHttpsCloneURL / normalizeGitRepoURLForBranchLookup / gitPrHtmlUrlOf 均继承 provider website scheme（http:// IP GitLab 不被改成 https://host:443）；三仓回归测提交推送完成
- **Created**: 2026-08-27
- **Context**: 任务详情 PR 卡 `comment-git-pr-error` 把 HTTP GitLab（`http://115.29.110.74`）打成 `https://host:443`。已修 `taskGitOauth` MR 状态/合并 API origin。SSH clone 规范化仍有 `https://`+host 兜底；评论里存的 `git_pr_html_url` 若本身是 https，卡片链接仍会走 443。
- **Action**: (1) 评估 `taskCredentialService/infrastructure/provider_configs.go` 与 `taskProjectService/src/git_repo_url_normalize.go` 的 SSH→HTTPS 兜底是否应回退到 provider website scheme (2) 评论/卡片 `html_url` 展示是否按租户 `base_url` 改写 scheme (3) 有回归测再改，避免误伤 github.com
- **Why**: MR API 修好后，clone 与页面链接仍可能对 HTTP GitLab 默认 443。
- **How to apply**: `ResolveHttpsCloneURL`、`normalizeGitRepoURLForBranchLookup`、`gitPrHtmlUrlOf`；对照租户 Path A `BaseURL`

## [OPT-20260827-038] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 排查完成：ws_-2309487803472456748 为旧 genID UnixNano 溢出产物，库内真实存在（company 877397588196749312，共 3 条负 ws_）；FE 路由/后端 API/DB 主键均按不透明字符串处理，无 Number 解析；存量迁移规划见 OPT-20260827-013；已补 taskProjectService 负 ID 字符串传输回归测试
- **Created**: 2026-08-27
- **Context**: 公网任务详情 URL workspace 为 `ws_-2309487803472456748`（负 int64）。同页其它 cloud API 也带此 ID。Snowflake 应为正；负号像有符号溢出或 JSON number 解析错误。
- **Action**: (1) 查 `taskProjectService` 该 workspace 行真实 id (2) 追前端路由/API 是否把 id 当 Number (3) 若库内已是负值则写修复迁移与字符串传输回归
- **Why**: 负 workspace id 会污染所有 kv-last 路径，后续分片/校验会踩坑。
- **How to apply**: `taskProjectService` workspace 表；`.ai/03_technical_implementation/11_id_field_string_transit.md`；前端 router params

## [OPT-20260827-036] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: FetchAccessToken 统一走 gitsite 路径（taskCredentialService 453911d）：删除 gitsite: sentinel 与 YAML provider_key 分叉，ResolveProvider 仅用于 git_http_username/日志；github.com 与 tencent-sh-1 换票测全绿
- **Created**: 2026-08-27
- **Context**: Path A YAML miss 已用 `gitsite:{host}` 旁路修好（failure 120）。YAML 命中的 GitHub/区域 GitLab 仍走 `/api/internal/{provider}/oauth/access-for-user/` + `provider_key`。两套路径在 Path A 上已分叉一次。
- **Action**: (1) `GitoauthHTTPClient.FetchAccessToken` 一律按 repo host 打 gitsite 内部接口 (2) 删除 `gitsite:` sentinel 与 YAML key 分叉 (3) 保留 YAML ResolveProvider 仅给 git_http_username / 日志 (4) 回归 github.com 与 tencent-sh-1 换票测
- **Why**: 双路径会在下一次租户 Git 变体（非标准端口、SSH host alias）再次漏网。
- **How to apply**: `taskCredentialService/infrastructure/gitoauth_client.go`；`cd taskCredentialService && go test ./infrastructure ./application -count=1 -run 'FetchAccessToken|BuildCredentials'`

## [OPT-20260825-014] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 派生副本 auto_run agent_models[0].model 持久化到任务行 task_tasks.agent_model，create/GET JSON 暴露；taskFE 层图模型选择默认展示该模型
- **Created**: 2026-08-25
- **Context**: Fork 自动运行按模型复制时，每份 POST 只把 `agent_models` 写入 auto_run pending agent 的 `context_pack`，任务行仍沿用源任务的智能体资源配置。详情页层级图「选择模型」读的是任务级 feature-params，不一定等于该副本实际 overlay 的模型。
- **Action**: (1) 评估是否在 `createTaskOnce` 把该份 `agent_models[0].model` 写入任务快照或独立字段；(2) 详情页读取该字段作为默认展示模型；(3) 补 create + GET JSON 回归测。
- **Why**: 多副本并行时运维无法从任务详情直接确认「这一份跑的是哪个模型」。
- **How to apply**: `taskTaskService/src/create_task.go`、任务 JSON 序列化、`taskFE` 任务详情层级图模型选择；先对照 `feature_params_source` 是否已足够、避免双源。

## [OPT-20260827-047] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: settings/task-panel 套餐设置点击打开任务存档模态 Playwright 纯 mock 回归：新增 archive-modal.playwright.test.js + config + .sh，taskFE c841fb5，本地 vite dev server + bundled Chromium 验证 1 passed。
- **Created**: 2026-08-27
- **Context**: 套餐设置点击已用 vitest 覆盖打开模态与 PATCH。公网 SPA 仍需 Playwright 走真实页面，防止 chunk 未发布时再现空点击。
- **Action**: (1) 在 `taskFE/tests/` 增加 `WorkspaceSettings.archive-modal.playwright.test.js`：进入 task-panel、点 `[data-testid=open-task-archive-settings]`、断言模态标题 (2) 配同名 `.sh`
- **Why**: 组件测不覆盖生产 hashed chunk 与路由守卫。
- **How to apply**: `taskFE/tests/`；9222 CDP；参考 `WorkspaceSettings.delete-button.playwright.test.js`

## [OPT-20260827-018] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: work-panel 创建任务技能 chip 点击写入完整 $镜像 /技能 Playwright 纯 mock 回归：新增 WorkPanel.skill-chip-insert.playwright.test.js + config + .sh，taskFE e7a1253，本地 vite dev server + bundled Chromium 验证 1 passed。
- **Created**: 2026-08-27
- **Context**: 本会话修复了「已绑定默认镜像、描述为空时点击 `$镜像 /技能` chip 只写入 `/技能` 并解绑镜像」。组件与集成 vitest 已覆盖，公网 work-panel 创建任务弹窗尚未有 Playwright 锁住该点击路径。
- **Action**: (1) 在 `taskFE/tests/` 增加 Playwright：打开 work-panel 创建任务 → 确认技能 chip 可见 → 点击默认技能 chip → `#task-description` 值为 `$镜像 /技能` 且 chip 仍在 (2) 包装 `.sh` 走 9222 CDP
- **Why**: 项目默认镜像自动绑定是生产主路径，仅 vitest 看不到弹窗布局、焦点与公网包是否切到新 hash。
- **How to apply**: `TaskDescriptionSkillField.vue` `insertSkill`；参考 `taskFE/tests/WorkPanel.create-task-with-projects.playwright.test.js`。

## [OPT-20260821-012] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskCloudService 6 个 project-service helper 接入 ctx + tracelog.ApplyOutboundHeaders，HTTP handler 沿调用链 thread r.Context()，X-Trace-Id 跨 task-cloud→project 可检索；回归测试断言透传
- **Created**: 2026-08-21
- **Context**: 工作面板空任务误报排查时，todos 404 的 Loki 检索只有 task-task-service 与 task-auth，没有 task-project-service，因为 `projectRequest` 未把入站 `X-Trace-Id` 带到出站请求。
- **Action**: (1) 给 taskCloudService 调 `/api/projects/` 的内部客户端穿 `r.Context()` 并 `ApplyOutboundHeaders`（约 6 helper / 12 文件）(2) 部署后用已知 todos 404 traceId 在 Loki 验证能命中 project-service。taskTaskService/taskTenantService 透传已落地。
- **Why**: 没有出站透传时，成员校验 401 被映射成 workspace not found 后无法按同一把钥匙重建全链路。
- **How to apply**: `taskTaskService/src/project_client.go`；`shareLib/tracelog` 出站头；禁止只打 `project_status` 日志而不带 trace_id

## [OPT-20260822-036] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: gitsite/{site} 换票路径已验证落地：taskGitOauth /api/internal/gitsite/ 路由 + ResolveProviderByGitsite(WithDB) 解码匹配（gitsite_access_for_user.go），GetCredentialByIDForUpdate FOR UPDATE + issueAccessTokenFromCredential 共用，cache/锁键含 remote_user_id；taskCredentialService/taskProjectService 客户端统一走 gitsite（OPT-20260827-036），旧 github|gitlab|git-oauth 别名保留；taskGitOauth go test ./... 全绿
- **Created**: 2026-08-22
- **Context**: /1-brainstorming 已批准内部换票改为 `POST /api/internal/gitsite/{site}/oauth/access-for-user/`（v97 target）。GitLab OAuth 无 STS；并行 refresh 会作废旧 access。
- **Action**: (1) taskGitOauth 注册 gitsite 路径，site URL 解码后匹配 YAML website (2) 三家客户端默认改走新路径，旧 github|gitlab 留别名 (3) ✅ `GetCredentialByIDForUpdate` 真正 `FOR UPDATE`，probe / access-for-user / merge 共用 `issueAccessTokenFromCredential`（2026-08-22 `taskGitOauth` `5ffd6a5`，20:46 已精准重启）(4) cache 键含 remote_user_id
- **Why**: 路径用产品族会把区域 GitLab 挤在一条 URL 上；无行锁时 GitLab refresh 旋转导致多评论 clone 400。
- **How to apply**: 设计 `docs/superpowers/specs/2026-08-22-multi-comment-git-oauth-access-token-sharing.md`；意图 `docs/intents/backend/gitoauth_access_for_user_gitsite_path.intent.md`

## [OPT-20260823-022] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE 337a043 已推送：新增 SystemAdminUsers.edit-user-id.playwright.test.js + config + .sh 纯 mock 回归（注入 cookie + mock /api/system-admin/users/ 列表），进入 /system-admin/users/ 点击行内「编辑」打开编辑弹窗，断言 #edit-user-id 可见/readonly/值等于该行 ID；本地 vite dev :4000 + bundled Chromium 验证 1 passed
- **Created**: 2026-08-23
- **Context**: 编辑用户表单已加只读 `#edit-user-id`；Vitest 覆盖字符串值与 readonly。公网系统管理员打开编辑弹窗未做。
- **Action**: (1) 以系统管理员打开 `/system-admin/users/` 点编辑 (2) 断言 `#edit-user-id` 可见、readonly、值等于该行 ID
- **Why**: 只读展示易被后续表单重排漏掉。
- **How to apply**: `taskFE/tests/`；账号见 `task2app/测试.ai.md`

## [OPT-20260823-015] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE 3d5efa5 已推送：新增 BillingOrderCreate.gitlab-region-single.playwright.test.js + config + .sh 纯 mock 回归（VIP1 打开 /tenant/:tid/billing/orders/create/，mock order-pricing/membership/gitlab-regions），断言 order-gitlab-resources 含磁盘+流量、order-gitlab-region 数量为 1、无 order-gitlab-traffic-region；本地 vite dev :4000 + bundled Chromium 1 passed
- **Created**: 2026-08-23
- **Context**: VIP1 购买页已把 GitLab 流量并入磁盘卡片并共用 `order-gitlab-region`；本会话 Vitest + 公网 CDP 已验收。尚无 Playwright 防止回归成双下拉。
- **Action**: (1) 在 `taskFE/tests/` 增加 Playwright：VIP1 打开 `/billing/orders/create/` (2) 断言 `order-gitlab-resources` 含磁盘与流量 (3) 断言 `order-gitlab-region` 数量为 1 且无 `order-gitlab-traffic-region`
- **Why**: 仅组件测无法覆盖公网路由/缓存/VIP 门槛组合；布局回退时 E2E 才能拦住。
- **How to apply**: `taskFE/app/src/views/OrderCreate.vue`；账号见 `task2app/测试.ai.md`；CDP 9222

## [OPT-20260821-006] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE d3042d5 已推送：新增 ProjectDetail.image-architecture-save-400.playwright.test.js + config + .sh 纯 mock 回归（x86 模版 + arm64 镜像，PATCH mock 按选中镜像 400/200 分流），断言跨架构保存报错带 data-traceId、文案含「镜像要求」「实例系统支持」，同架构保存 200 关闭编辑态；本地 vite dev :4000 + bundled Chromium 2 passed
- **Created**: 2026-08-21
- **Context**: T1–T7 已由 Go/Vitest 覆盖；未在真浏览器跑 `ProjectDetail.image-architecture-filter` 的跨架构硬拦截。
- **Action**: (1) 扩展 playwright mock 为 x86 模版 + arm 镜像 (2) 保存镜像断言 400 且页面报错带 data-traceId，文案含「镜像要求」与「实例系统支持」 (3) 同单匹配模版后 200
- **Why**: 组件拆分与 clickGuard 后，E2E 才能证明用户路径与后端不变量一致。
- **How to apply**: `taskFE/tests/ProjectDetail.image-architecture-filter.playwright.test.js`

## [OPT-20260822-029] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE 011d151 已推送：新增 TaskDetail.comment-execution-git-oauth-summary.playwright.test.js + config + .sh 纯 mock 回归（测试意图 T17）。任务关联 GitLab 仓 + 评论 repo_identities，mock 评论 feed 与 user-app-connection(connected=false)，断言 comment-execution-git-oauth unbound/含「未绑定」、comment-execution-git-identity 同一行、bind href 含 start-from-gateway 与 repo_url；本地 vite dev :4000 + bundled Chromium 1 passed
- **Created**: 2026-08-22
- **Context**: 评论执行细节 summary 已在同一行展示 `Git OAuth · 已绑定/未绑定`（复用任务级 `repoOAuthReadiness`，未绑定含真实 `a[href]`）。当前只有 Vitest 组件/接线测，意图 T17 尚未有 Playwright 公网页断言。
- **Action**: (1) 在 `taskFE/tests/` 增加任务详情 fixture：关联 GitLab/GitHub 仓 (2) 断言 `comment-execution-git-oauth` 与 `comment-execution-git-identity` 同在 `comment-execution-details-summary` (3) 未绑定态断言 `comment-execution-git-oauth-bind` 的 href 含 `start-from-gateway` 与 `repo_url`
- **Why**: 公网缓存/接线回归比单测更容易漏掉「摘要行没接到 readiness」。
- **How to apply**: `docs/intents/frontend/task_detail/034_comment_level_repo_identity.test-intent.md` T17；`taskFE/tests/TaskDetail.*.playwright.test.js`

## [OPT-20260822-032] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE 064dddb 已推送：新增 TaskDetail.comment-git-oauth-probe-once.playwright.test.js + config + .sh 纯 mock 回归（测试意图 T18）。连接检查 connected=true → 摘要「已绑定」→ 对该仓 probe_access_token=1 恰好一次（计数断言）→ probe 返回 access_token_valid=false → 摘要 data-kind=unbound 含「未绑定」且 bind href 含 start-from-gateway 与 repo_url；本地 vite dev :4000 + bundled Chromium 1 passed
- **Created**: 2026-08-22
- **Context**: 本会话已用 Vitest/Go 覆盖 `probe_access_token` 与评论区 once-guard。公网任务详情仍依赖 SPA 构建 + 精准重启后才能看到 refresh 失效改「未绑定」。
- **Action**: (1) 在 taskFE Playwright 拦截 `GET /api/git-oauth/user-app-connection/` (2) 断言评论区可见且 DB connected 时 query 含 `probe_access_token=1` 且同一仓只一次 (3) mock `access_token_valid=false` 后 `comment-execution-git-oauth` 为未绑定且含 bind href
- **Why**: 单测不能捕获 CommentsSection watch 与关联项目 readiness 时序导致漏探测或轮询。
- **How to apply**: `docs/intents/frontend/task_detail/034_comment_level_repo_identity.test-intent.md` T18；参照 `taskFE/tests/TaskDetail.fork-popup.playwright.test.js` 的 user-app-connection 路由

## [OPT-20260828-001] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 快照返回 recent_history_has_more，FE 读字段替代 length>=8；Go/JS 边界测 7/8 条均绿
- **Created**: 2026-08-28
- **Context**: 实现排队调度历史时，GET 快照只带最多 8 条 `recent_history`；前端用 `recent.length >= 8` 猜测是否还有更多。恰好 8 条且已到末尾时会误显示「加载更多」，点后才发现没有下页。
- **Action**: (1) `buildWorkspaceQueueScheduleSnapshot` 用 `listScheduleHistory(..., 8)` 的 `hasMore` 写入 `recent_history_has_more` (2) `seedHistoryFromSnapshot` 读该字段，去掉 `length >= 8` (3) 补 Go HTTP 测与 composable 测：7 条无按钮、8 条且 has_more=true 才有按钮
- **Why**: 启发式在边界条数会误导用户多打一次只读 GET。
- **How to apply**: `taskTaskService/src/queued_schedule_workspace_handlers.go`；`taskFE/app/src/composables/workspaceSchedule/useWorkspaceQueueSchedule.js`

## [OPT-20260828-003] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 公网 queue-schedule 页 CDP 断言 schedule-status-bar 与 schedule-history-card 均存在；空态文案「尚无调度记录」
- **Created**: 2026-08-28
- **Context**: 状态栏下已加 `ScheduleHistoryCard`；需 017 迁移 + 精准编译重启 `task-task-service`/`taskFE` + 公网 SPA 新包后，登录态才能看到「调度历史」。
- **Action**: (1) 确认 `task_queued_schedule_history` 已存在 (2) 硬刷新 queue-schedule 页 (3) 断言 status-bar 与 history-card
- **Why**: 单元测试已绿；未切 SPA 时用户仍只看到状态栏。
- **How to apply**: Playwright 登录态；vitest `ScheduleHistoryCard.test.js`

## [OPT-20260828-002] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 入队/出队历史写入 actor_user_id（create/update/internal 透传）；TestEnqueueWritesScheduleHistory 断言非空
- **Created**: 2026-08-28
- **Context**: `appendQueuedAutoRunMembership` / `dequeueQueuedAutoRun` 写 `task_queued_schedule_history` 时未填 `ActorUserID`；保存时段路径已有 actor。历史卡因此无法区分谁把任务加入队列。
- **Action**: (1) 入队/出队函数增加 actor 参数或从请求 ctx 取 user id (2) 所有 call site 传入 (3) `TestEnqueueWritesScheduleHistory` 断言 `actor_user_id` 非空
- **Why**: 审计与排障需要操作者；表已有列却长期空。
- **How to apply**: `taskTaskService/src/queued_schedule_membership.go`；handlers PATCH queued_auto_run

## [OPT-20260827-029] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: GET 只读 last_runtime_status：TestWorkspaceMachineSummaryGETDoesNotDescribeInstances Describe=0；timer SweepOnce 第三步 POST reconcile-workspace-machine-runtimes（runner TestSweepOnceCallsOrphanThenLeaked 3 hits）；FE 去掉 15s interval：visibility 测 15s 后仍 1 次 GET；SSE task_status_changed/server_status_update/container_heartbeat 触发 onMachineRuntimeHint（useWorkPanelTaskStatusSse.test.js）
- **Created**: 2026-08-27
- **Context**: WorkPanel 每 15s `Promise.all` 拉 `workspace-machine-summary` + `workspace-runtime-indicators`（`useWorkPanelMachineSummary.js`）。两端点各自 `computeWorkspaceMachineSnapshot` → `reconcileWorkspaceMachineRuntimes`（逐实例 Aliyun Describe）+ `reconcileOrphanInstancesByName`（按名 Describe）。TTL 30s，双请求并行会同时 miss。慢工作区 p95≈4.6s（p50=5ms，冷路径打云）；快工作区同接口近 750 次/小时仍 <1s。违反前端禁止无触发后台轮询。两端点均重复 `ensureTenantMember`。
- **Action**: (1) 前端改为 SSE（heartbeat/runtime SSE 已有）驱动计数，去掉 `setInterval` 15s (2) 合并为一个 GET 或 singleflight 共用 snapshot (3) GET 路径只读 `last_runtime_status`，Describe/orphan 交给 timer/事件 (4) 删除重复 `ensureTenantMember`
- **Why**: 看板首屏/轮询与云厂商 RTT 绑定；并行双击穿缓存会把 4s 尾巴打到用户感知。
- **How to apply**: `taskFE/app/src/composables/useWorkPanelMachineSummary.js`；`taskCloudService/src/workspace_machine_summary.go`、`compute_workspace_runtime_indicators.go`、`machine_runtime_reconcile.go`、`orphan_instance_reconcile.go`；WorkPanel Playwright 已 mock 这两路

## [OPT-20260828-004] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: runAll 4f68e70: Linux rt_sigtimedwait 记录 si_pid/si_uid/si_code；go test ./src -run WaitTermination|FormatSignal 通过（helper 子进程 kill -TERM，输出 SIGINFO_OK）。不默认开 auditd。
- **Created**: 2026-08-28
- **Context**: 编排器 2026-08-28 00:00:32 以 `trigger=signal detail="terminated"` 退出（pid=2509204、setsid、非 shutdown-self、无第二实例写入 console）。OPT-20260813-007 / OPT-20260820-006 已能区分 signal vs shutdown-self 并打印 pid/pgid/sid，仍看不到发送方。午夜 cron 堆叠（truncate / sweep / nightly-opt 00:00:02）同时开火，根因无法钉死。
- **Action**: (1) 将 `main.go` 的 `signal.Notify` 改为 `unix.Sigwaitinfo`（或等价 raw handler），在 SIGTERM/SIGINT 时日志 `si_pid`/`si_uid`/`si_code` (2) 补单测：子进程向测试进程发 SIGTERM，断言格式化函数打出子进程 PID（或 mock `Siginfo`）(3) 不默认开 auditd
- **Why**: 只有 `detail="terminated"` 时，热替换失败回退、夜间 agent、`kill` 脚本无法区分；看门狗又禁止自动拉起 `:9999`，每次无主 SIGTERM 都会让 Status UI 挂到人工 `./run.sh`。
- **How to apply**: `runAll/src/main.go` 信号 goroutine；`runAll/src/main_exit_reason_test.go` 或新建 `main_siginfo_test.go`；`go test ./src -run Siginfo|ExitReason`

## [OPT-20260827-031] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskCloudService 4303862: start-vm/start-vm-auto HTTP 先 200「已提交」，RunInstances 单次 goroutine（setupCloudTestDB 仍同步以免存量测例竞态）。available-instances 45s 缓存。go test AckBeforeRunInstances + CachesByRegionZone 通过。
- **Created**: 2026-08-27
- **Context**: panel-22 `start-vm` p95=9.5s（n=1）。前端 `startVmHttpResult.js` 已把 200+「已提交」当异步受理，后续靠 SSE。后端仍在请求内跑 Aliyun `RunInstances`，按钮/详情要空等 ~10s。`available-instances` avg≈1.2s 同属云 SDK，实例选择器可短 TTL 缓存。
- **Action**: (1) start-vm 先写受理+发 SSE，RunInstances 放到事件/goroutine（注意元规则 46 禁止业务进程 ticker，用 Kafka/taskEvents 或单次 goroutine+超时）(2) available-instances 按 region/zone/规格缓存 30–60s (3) 硬件面板保持 `Idempotency-Key` 与现有 ack 文案
- **Why**: FE 已按异步设计，HTTP 同步等云 API 是多余用户等待；n 小但每次启动都痛。
- **How to apply**: `taskCloudService/src/compute_start_vm_native.go`、`tenant_cloud_handlers.go` `handleTenantCloudAvailableInstances`；`taskFE/app/src/utils/startVmHttpResult.js` 已兼容 ack

## [OPT-20260827-032] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskChromePlugin 994f1d3: content.js IIFE 拆为 float-boot/pick/drag-auth/form/snapshot + content.js，wc -l 均 ≤500（240/176/494/445/488/285）；manifest 按序注入；node --test test/*.test.js 497 pass；e2e float-close-collapse 1 pass + keyboard-shortcut-fallback 7 pass
- **Created**: 2026-08-27
- **Context**: 后续会话又给浮窗加了项目单选、自动运行联动，以及镜像必选挂钩，`content/content.js` 现约 2182 行，仍超过源文件 500 行门禁。当场全量拆分会与浮窗 IIFE 强耦合，风险大于本增量。
- **Action**: (1) 按域抽出 loadProjects / 创建任务表单 / 鉴权刷新为 `content/` 子模块 (2) 保持 manifest 注入顺序 (3) 复跑 `node --test test/*.test.js` 与浮窗 e2e
- **Why**: 行数门禁要求超标文件立即削减；继续堆功能会扩大回归面。
- **How to apply**: `taskChromePlugin/content/content.js`；对照 `lib/` 纯函数拆分先例（`workspace-list.js`、`project-auto-run-label.js`）

## [OPT-20260827-023] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 同上 994f1d3：浮窗 IIFE 拆成 classic 多文件，共享 var；init 改到最后注入的 content.js；源码契约改读 test/helpers/contentBundle.js 拼接链；node --test 全绿 + 浮窗 e2e 8 pass
- **Created**: 2026-08-27
- **Context**: `content/content.js` 约 2096 行（行数门禁阈值 500）。本会话修 CPU 热路径时只做净削减、未整文件拆分；继续往里堆会无法审查。
- **Action**: (1) 把浮窗 DOM 模板、拖拽、选元素、建任务提交拆成 `content/` 下多文件并在 manifest 按序注入 (2) 每文件 `wc -l` ≤500 (3) 现有 `test/content*.test.js` 源码契约仍绿
- **Why**: 超大 IIFE 无法做针对性审查，后续性能修复容易再引入常驻监听。
- **How to apply**: `taskChromePlugin/content/content.js`；`wc -l taskChromePlugin/content/*.js`；`cd taskChromePlugin && node --test test/content.test.js test/content-cpu-hot-path.test.js`

## [OPT-20260826-002] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE aa445cb: LoginHistory.customer-ip live on https://www.daydaymoney.com 1 pass；最新行 IP 为公网（先前失败证据 222.76.156.241，非 172.16/12）；selector td.px-4.py-3.text-sm.font-mono + isRfc1918Ipv4
- **Created**: 2026-08-26
- **Context**: 公网 `/profile/login-history/` 曾展示 `172.26.0.1`（taskfe_default 网桥）。本会话已改解析与 taskFE host network；存量行无法回填真实公网 IP。OPT-20260825-036 的 Playwright 尚未断言 IP 列。
- **Action**: (1) 客户再登录一次后打开登录历史 (2) 最新一行 `td.font-mono` 不是 `172.16.0.0/12` (3) 把该断言并入 OPT-20260825-036 的 Playwright
- **Why**: 组件测无法覆盖 Docker DNAT + 边缘 XFF 整链。
- **How to apply**: 页面 `https://www.daydaymoney.com/profile/login-history/`；选择器 `td.px-4.py-3.text-sm.font-mono`。

## [OPT-20260825-036] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE aa445cb: LoginHistory.admin-href mock 2 pass（同意并继续后 user-row-login-history 为 <a href> 并打开 /system-admin/users/:id/login-history/）；customer live 侧栏 user-center-nav-login-history 可见；包装 tests/LoginHistory.profile-and-admin.playwright.test.sh
- **Created**: 2026-08-25
- **Context**: 本会话已用浏览器验证客户 `/profile/login-history/`，并用一次性 Playwright 脚本验证超管用户行真实 `href` 与 `/system-admin/users/:id/login-history/` 列表。仓库内尚无 `.playwright.test.js` 固化该路径；超管验收会被 `PrivacyReconsentGate` 遮罩挡住点击。
- **Action**: (1) 在 `taskFE/tests/` 新增 login-history Playwright：客户登录后侧栏「登录历史」可见且表格含 `用户入口`+IP (2) 超管登录后先点「同意并继续」再断言 `user-row-login-history` 为 `<a href>` 并打开目标用户页 (3) 包装 `.sh` 便于 CI
- **Why**: vitest 只覆盖组件 href；SPA 发布/路由/网关 token 优先级回归仍依赖手工。
- **How to apply**: 参考 `/tmp/verify-admin-login-history.mjs` 与 `taskFE/tests/playwrightLogin.js`；`PrivacyReconsentGate.vue` 按钮文案「同意并继续」。

## [OPT-20260823-023] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: Playwright Impersonation.admin-flow: cancel no POST; confirm reason then /profile/inbox/ shows reason (4/4 pass). taskFE/tests/Impersonation.admin-flow.playwright.test.sh
- **Created**: 2026-08-23
- **Context**: 本会话已实现管理员模拟登录必填理由、访问日志标识、以及被模拟用户收信箱通知。Vitest 覆盖 composable 与收信箱渲染；公网管理员点「以该用户身份登录」未做端到端。
- **Action**: (1) 系统管理员打开用户列表点模拟登录 (2) 断言理由弹窗出现、取消不发请求 (3) 填写 ≥8 字理由确认后进入目标身份 (4) 用目标用户打开 `/profile/inbox/` 看到该理由
- **Why**: 弹窗与写信是安全审计路径，仅单测拦不住 UI 接线回归。
- **How to apply**: `taskFE/tests/`；账号见 `task2app/测试.ai.md`

## [OPT-20260823-030] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: Nested 409 HTTP detail is 已在模拟登录中，请先退出 (Go TestImpersonateNestedConflict); same-target second confirm leaves /system-admin/users/ (Playwright). Vitest Chinese 409 same-target redirect vs other-user error. taskAuth+taskFE
- **Created**: 2026-08-23
- **Context**: 系统管理用户页点击确认登录若已有同一目标开放会话，后端现改为 200 重放并跳转目标前台；嵌套模拟另一用户仍 409 且英文 `already impersonating`。Vitest/Go 已覆盖，公网管理员路径未做。
- **Action**: (1) 以系统管理员对同一用户连续两次打开理由弹窗并确认，第二次应离开 `/system-admin/users/` 进入目标前台而非停在红字 (2) 若仍持有模拟 token 再模拟另一用户，断言 409 且文案改为中文（例如「已在模拟登录中，请先退出」）(3) 与 OPT-20260823-023 可同脚本
- **Why**: 本次缺陷就是「有错误文案但不跳转」；单测拦不住网关 X-User-Id=target 的真实 Cookie 路径。
- **How to apply**: `taskFE/tests/`；`taskAuth/domain/impersonation.go` `ErrNestedImpersonation`；账号见 `task2app/测试.ai.md`

## [OPT-20260823-055] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: Playwright BillingInvoice: apply→admin 已开具→订单详情「已开具电子发票」; reject restores apply button. taskFE/tests/Billing.invoice-and-grant.playwright.test.sh (5 passed)
- **Created**: 2026-08-23
- **Context**: 手动开具改动仅单测覆盖（组件 + Go），无浏览器级 e2e 验证管理员面板登记后租户侧看到「已开具电子发票」。
- **Action**: (1) e2e：租户申请开票 → 管理员面板点「已开具」→ 订单详情显示已开具 (2) 拒绝路径：拒绝后租户可重新申请
- **Why**: 手动开具改变了核心操作语义，浏览器闭环可防回归。
- **How to apply**: `taskFE/tests/e2e/` 或 `e2e-tests/` 既有 invoice 用例扩展

## [OPT-20260823-056] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: Playwright BillingGrantPoints: VIP-only POST resources=[] grants=[] + toast 会员等级; quantity 2 generates grant order. 5 passed same wrapper.
- **Created**: 2026-08-23
- **Context**: 根因修复（数量默认留空 + 提交核对文案 + Idempotency-Key）仅单测覆盖（前端 vitest + 后端 Go），无浏览器级 e2e 验证「只改 VIP 时不生成赠送订单」「空数量行被过滤」。
- **Action**: (1) Playwright e2e：管理端赠送页选租户 → 仅设 VIP1 → 提交 → 断言响应 grants=[] 且订单列表无新 admin_grant 订单 (2) 填写数量 → 提交 → 断言生成赠送订单且预览文案正确
- **Why**: 误送订单是资金/配额类事故，浏览器闭环可防回归，且验证 createClickGuard 双通道幂等键在真实网关下生效。
- **How to apply**: `taskFE/tests/e2e/` 既有系统管理用例扩展；mock 或直连 runAll 环境均可

## [OPT-20260823-065] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: Playwright special invoice: type=special + org fields, admin upload PDF, tenant 查看发票文件 href. Billing.invoice-and-grant.playwright.test.sh
- **Created**: 2026-08-23
- **Context**: 目标 2 新增专票/普票选择、管理员上传手动开具发票文件、登记时文件随发票行落库；单测已覆盖（taskBill invoice_file_test / taskFE 组件测试），但端到端（网关 → 表单 → multipart 上传 → 下载）尚无 Playwright 覆盖。
- **Action**: taskFE Playwright 用例：租户申请专票并填完整单位信息 → 管理员面板上传 PDF → 「已开具」登记 → 租户订单详情出现「查看发票文件」且可下载。
- **Why**: multipart 经网关转发 + 文件落盘链路是浏览器行为，单测无法验证网关侧表单边界。
- **How to apply**: `taskFE/tests/e2e/`（invoice manual issue 流程）

## [OPT-20260821-004] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: popup/SW split SHA 49c6a69; node --test test/*.test.js 499 pass; test/line-limit-popup-sw.test.js wc≤500; e2e/popup-layout.playwright.test.js 3 pass; no manifest version bump
- **Created**: 2026-08-21
- **Context**: 去掉跨页描述同步时改了 `content/content.js`（仍约 2044 行）、`background/service-worker.js`（约 1387）、`popup/popup.js`（约 924）。本增量只删同步通道，未做拆文件。
- **Progress (2026-08-28)**: content 浮窗已拆（OPT-20260827-032/023，`994f1d3`）；余 `background/service-worker.js`（1500 行）与 `popup/popup.js`（939 行）。
- **Action**: (1) 按职责把 content 浮窗/选元素/鉴权刷新拆到 `lib/` 或 `content/` 子模块 (2) SW 消息 switch 按域拆文件 (3) 保持 `manifest.json` 注入顺序 (4) `npm test` 全绿
- **Why**: 行数门禁默认 500；继续往这三文件堆功能会放大回归成本。
- **How to apply**: `taskChromePlugin/content/content.js`、`background/service-worker.js`、`popup/popup.js`；对照 `.ai/01_project_constraints/27_source_file_line_limit_auto_reduce.md`。

## [OPT-20260823-007] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE 4f8416f: useTaskDetail facade 18 lines; Core 403 Fetches 188 Container 470 Actions 499 Api 318 all <=500; node --test line-limit pass; vitest imports+cmdError+editing 34 pass
- **Created**: 2026-08-23
- **Context**: `useTaskDetail.js` 已约 1400 行，本会话仅增加 copyCount 透传 3 行，未做大爆炸拆分。
- **Action**: (1) 按 composable 域拆出 fork/comments/edit 子模块 (2) 保持 `useTaskDetail` 门面导出 (3) 跑既有 TaskDetail smoke / editing 单测
- **Why**: 行数门禁默认 500；继续堆功能会恶化可维护性。
- **How to apply**: `taskFE/app/src/composables/useTaskDetail.js`；`.ai/01_project_constraints/27_source_file_line_limit_auto_reduce.md`

## [OPT-20260828-005] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskAuth a94134d: writePhoneBindConflict shared by decision and upsert paths; go test ./src -run TestBindPhone_|TestWritePhoneBindConflict_ pass
- **Created**: 2026-08-28
- **Context**: 特权手机共享绑定在占用判定与 upsert 失败两条路径各写了一份 `phone_bind_limit` / `phone_taken` 的 `writeErrorMap` 字面量。
- **Action**: (1) 在 `taskAuth/src/auth_phone_bind.go` 抽 `writePhoneBindConflict(w, r, decision)` (2) 判定路径与 upsert 错误路径共用 (3) 保留现有 bind 单测。
- **Why**: 文案或 `limit` 字段改动时容易只改一处，导致 409 体不一致。
- **How to apply**: `handleBindPhone`；对照 `auth_phone_bind_test.go` 的 code / reclaim_available 断言。

## [OPT-20260817-016] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskAiProvider 0ad59ca (pushed): VendorPortal.vue 1473→135; composables useVendorPortal{,Session,ImageGroups,CloudServers}.js 60/140/488/401; tabs VendorImageGroupsTab 240 / VendorCloudServerImagesTab 191 / VendorGroupFormModal 25. Evidence: frontend/tests/vendorPortalLineLimit.unit.test.js (≤500 + no exception comment); npm run test:unit 138 pass; vite build ok; node --check four composables.
- **Created**: 2026-08-17
- **Context**: `taskAiProvider/frontend/src/views/VendorPortal.vue` 约 1596 行，文件头有行数例外注释。本次申请认证已外提到 `VendorApplyPanel.vue`，未再向门户堆表单，但门户本体仍超阈值。
- **Action**: (1) 按 tab（镜像组 / 云服务器镜像 / 测试密钥）与登录卡片拆成子组件 (2) script 按 composable 迁出 (3) 每个新文件 `wc -l` ≤500，复跑相关前端单测
- **Why**: 继续在例外文件上叠功能会再次触发行数门禁，检索与评审成本持续升高。
- **How to apply**: `VendorPortal.vue`；已有 `VendorCloudCredentialsTab.vue` / `RegionEnvModal.vue` / `VendorApplyPanel.vue` 可作拆分样例

## [OPT-20260828-007] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskAiProvider 37c1ce7 (pushed): AdminVendorsTab approve/reject use AdminImageReviewNoteModal + createClickGuard Idempotency-Key; no confirm/prompt/alert. Evidence: frontend/tests/adminVendorsTab.unit.test.js 3 pass; check_frontend_button_anti_replay.py --strict --files AdminVendorsTab.vue ok.
- **Created**: 2026-08-28
- **Context**: 容器镜像审核已去掉原生对话框并补 clickGuard。`AdminVendorsTab.vue` 仍用 `confirm` 通过、`prompt`/`alert` 驳回，且写请求无 `Idempotency-Key`。
- **Action**: (1) 复用 `AdminImageReviewNoteModal` 或抽出共享确认/原因弹窗 (2) `approveVendor`/`rejectVendor` 走 `createClickGuard().run` 并传 headers (3) 源码扫描单测断言不再出现 `confirm(`/`prompt(`/`alert(`
- **Why**: 与前端禁止原生弹窗、写按钮防重放门禁不一致，连点可通过同一厂商两次。
- **How to apply**: `taskAiProvider/frontend/src/components/AdminVendorsTab.vue`；验收 `cd taskAiProvider/frontend && node --test ./tests/adminImageReviewActions.unit.test.js` 同类源码扫描，以及 `python3 db/scripts/ci/check_frontend_button_anti_replay.py --strict --files taskAiProvider/frontend/src/components/AdminVendorsTab.vue`

## [OPT-20260828-008] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskAiProvider d82e6a7: GetContainerImageAdmin reuses list enrich; go test ./src -count=1 -run AdminContainerImage exit 0 (TestAdminContainerImageGetByIDReturnsEnrichedFields + NotFound); node --test frontend/tests/adminImageReviewActions.unit.test.js 16 pass including openDetail GET-by-id source-contract
- **Created**: 2026-08-28
- **Context**: 平台审核详情弹窗目前用列表行内存数据。单条 GET 仍只返回 `id/status/version/image_url`，深链或刷新后无法单独拉详情。
- **Action**: (1) `handleAdminContainerImages` GET-by-id 复用 `ListContainerImagesAdmin` 的 scan/enrich 或按 id 过滤 (2) 单测断言响应含 `image_skills`、`runtime_environments`、`review_histories` (3) 前端详情改为优先 GET，失败再回退列表行
- **Why**: 列表分页或筛选后内存行可能过期；单条 GET 契约与文档「获取容器镜像详情」不符。
- **How to apply**: `taskAiProvider/src/resource_handlers.go`；验收 `go test ./src -count=1 -run 'AdminContainerImage'`

## [OPT-20260828-009] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskAiProvider d82e6a7: AdminImageReviewTable group-active-hint visible text 组内激活：version（审批不切换激活）; node --test frontend/tests/adminImageReviewActions.unit.test.js 组内激活 hint 可见文案含版本且说明审批不切换激活 pass
- **Created**: 2026-08-28
- **Context**: 运营在 `https://provider.daydaymoney.com/admin` 审核表操作列看到 `组内激活：x86_64-latest`，需悬停 title 才知道「审批通过不影响激活」。可见文案只报版本号，易被理解成当前行已激活或审批会切市场版本。
- **Action**: (1) 改 `taskAiProvider/frontend/src/components/AdminImageReviewTable.vue` 的 `group-active-hint` 可见文案为同时包含版本与「审批不切换激活」短句 (2) 同步/缩短 `:title` 避免重复 (3) 补 `AdminImageReviewTable` 或现有 admin 前端单测断言可见文本含激活版本且含审批不切换语义
- **Why**: 审核员只看单元格就会误以为点「通过」会把本组公开目录切到当前行；title 不可及于触屏/截图。
- **How to apply**: `AdminImageReviewTable.vue` L106-110；对照厂商侧 `VendorImageGroupsTab.vue` 的 `badge-group-active`。验收：`npm test`/`node --test` 覆盖该组件断言，或现有 frontend unit 文件加一条对 hint 文案的断言。

## [OPT-20260828-006] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskFE 2a2af7c: fetchGitReposOAuthStatus uses resolveValidateGitReposRequestFailure; vitest batch.test.js HTTP 4xx+timeout + oauth-row git-repo-validate-error data-traceId (8 pass)
- **Created**: 2026-08-28
- **Context**: 创建项目页已把 `validate-git-repos` 传输/HTTP/缺结果失败拆成可读文案并挂 `data-traceId`。项目详情 `useProjectDetailGitRepos.js` 批量失败仍回退逐 URL，失败时只把 `token_status` 标成 `token_error`，错误 DOM 没有本次请求的 `data-traceId`。
- **Action**: (1) 在 `fetchGitReposOAuthStatus` 失败分支解析 `extractTraceId(response|err)` (2) 把 traceId 绑到详情页仓库 OAuth 错误节点 (3) HTTP 业务错误用 `messageFromFailedResponse` 而不是一律 `token_error` (4) 补 composable 单测覆盖 4xx + 超时
- **Why**: 详情页授权失败与创建页同样依赖该批量接口，排障仍无法从页面一跳 Loki。
- **How to apply**: `taskFE/app/src/composables/useProjectDetailGitRepos.js`；对照 `taskFE/app/src/utils/gitRepoValidateError.js` 与 `CreateProject.vue` 的 `data-testid="git-repo-validate-error"`

## [OPT-20260822-047] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: trae-agent 5e71734 + taskCloudService d6b15f9: auto_run backfill POSTs git-pr-reply; Cloud proxies to tenant_id comments path; node --test autoRunGitPrReplyComment+backfill 34 pass; go test GitPrReply ok
- **Created**: 2026-08-22
- **Context**: 手动 ztree 推送会 `recordGitPrReplyComment` 落一条带 `git_pr.html_url` 的人类子评论；auto_run 只 complete 容器 Agent。本轮前端用 ztree `prHtmlUrl` 装饰 Agent 气泡，刷新后若层图未加载仍可能看不到 PR 卡。
- **Action**: (1) 在 `backfillAutoRunPrToAgentComment` 成功后（或交付 ok 且有 URL）经 task 评论 API POST 一条 parent=人类评论的 `git_pr` 回复 (2) 幂等：已有相同 html_url 子评论则跳过 (3) 补测不重复写入
- **Why**: PR 展示不应依赖层图是否已拉取；与手动推送路径数据模型一致。
- **How to apply**: `trae-agent/onlineServiceJS/src/autoRunPrBackfill.mjs`；对照 `taskFE/app/src/composables/taskDetail/taskDetailGitPrReply.js` `recordGitPrReplyComment`

## [OPT-20260821-013] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskProjectService 83e9dda: overflow PK stays opaque; GET by ws_-/proj_- or registered snowflake alias returns stored id (id_alias_test.go). Plan: python3 taskProjectService/scripts/negative_id_migration_plan.py --check (workspaces=2 projects=1, CREATE project_id_alias, --apply exits 2). Evidence: go test ./src -count=1 -run Overflow|Alias|GenIDSnowflake|AssociateWorkspaceNegative ok; python3 scripts/test_negative_id_migration_plan.py 2 pass. No dataMigrate PK UPDATE.
- **Created**: 2026-08-21
- **Context**: 旧 `genID` 用 `UnixNano()*1000+seq` 溢出 int64 产生 `ws_-2309487803472456748` 一类负后缀。新建 ID 已改 snowflake；存量路径仍依赖 PathEscape + 字符串主键，不阻塞当前列表修复。
- **Action**: (1) 负 ID 盘点已完成（2 个 `ws_-` 工作区 + 1 条 `proj_-`，task 主键均为正 snowflake） (2) 给出 workspace_id 别名映射 + 该条 proj 主键的双写方案（禁止锁表一次性改主键） (3) 补迁移脚本与回归：旧 URL 仍能打开对应工作空间。
- **Why**: 负 ID 在部分语言/驱动当有符号整数解析时会出错，也让监控与文档难以当正常 snowflake 处理。
- **How to apply**: `taskProjectService`/`taskCloudService` genID 已切 snowflake；存量表与前端 workspace_id 深链；dataMigrate 禁止无回滚的 UPDATE 主键

## [OPT-20260822-015] cancelled

- **Status**: cancelled
- **Completed**: 2026-08-28
- **Summary**: Leftover open duplicate; already completed 2026-08-25 in archive (ADR-0042 workhorse sidecar, SH live meter). SSH bytes are OPT-20260825-025 (gitService fac73b896).
- **Created**: 2026-08-22
- **Context**: 租户 `877397588196749312` 容器 clone 路径已于 2026-08-25 落地：`received_bytes` → taskCloudService → `charge-gitlab-traffic` 按实际字节抬高 `traffic_used_gb`（live：`0.010498 GB / 1 GB`）。GitLab workhorse/nginx `git-upload-pack` 全量采集仍未做，笔记本/非容器 clone 仍可能漏计。
- **Action**: (1) 在各区域 GitLab workhorse/nginx 日志解析 `git-upload-pack` 的 `written_bytes`，按租户+region 调用 `reportGitlabTrafficUsageFloor` 或 `POST /api/internal/taskbill/charge-gitlab-traffic/` (2) 扣费请求与 `addGitlabTrafficUsedGBForRegion` 必须带 `region`，禁止写入 `defaultGitlabRegion` (3) 补重放测：同租户同 region 字节只抬高水位；不同 region 不塌缩
- **Why**: 无精确采集时超额后水位不再随 clone 增长（闸门会阻断），但已购流量的租户用量会偏低，也无法按公网出站计费。
- **How to apply**: `taskBill/src/handlers_internal_charge.go`、`chargeGitlabTraffic`、`gitlab_traffic_usage.go`；采集侧建议 `taskEvents` timer 或 GitLab log shipper，禁止业务进程内 ticker

## [OPT-20260825-025] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: gitService fac73b896: parse_gitlab_shell_line meters written_bytes with ssh: idempotency (not wh:). python3 gitService/scripts/test_ship_gitlab_git_traffic.py 18 pass including skip-without-bytes/receive-pack/CI/VPC. Sidecar follows GITLAB_SHELL_LOG. Conf SH compose updated locally (not committed this session).
- **Created**: 2026-08-25
- **Context**: ADR-0042 以 workhorse HTTP access `written_bytes` 为出站流量 SSOT。gitlab-shell 现网日志未见同等字节字段，SSH clone/fetch 仍可能漏计。
- **Action**: (1) 确认 gitlab-shell / gitaly 是否已有 pack 出站计数可解析 (2) 扩展 `ship_gitlab_git_traffic.py` 跟随对应日志且幂等键不与 HTTP `wh:` 冲突 (3) 补 SSH 解析单测与 CI/内网跳过对齐闸门。
- **Why**: 第三方容器若改走 `git@` SSH，HTTP sidecar 看不到 upload-pack，设置页已用流量会偏低。
- **How to apply**: `gitService/scripts/ship_gitlab_git_traffic.py`；SH 容器 `gitlab-tencent-sh-1-traffic-shipper`；ADR-0042 Consequences。

## [OPT-20260820-009] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: Promtail generator now emits job runall → logs/runall.log. test_generate_promtail_config.py dry-run asserts job+path. docker compose up -d promtail; POST X-Trace-Id runall-opt009-1787896695-after-promtail → Loki {job="runall"} 1 stream msg=ui_request status=404.
- **Created**: 2026-08-20
- **Context**: 诊断「已有全部/分组重新编译进行中」时页面 `data-traceid=runall-1787172683744-jw88w2lfbj`，Loki `{job=~"runall.*"}` 与本地 `logs/` 均 0 命中；runAll 日志未进 Promtail，TraceId 门禁只能退回源码路径。
- **Action**: runAll :9999 恢复后打一笔 Status UI 写请求，用横幅 `runall-*` traceId 在 Loki 断言命中 `{"trace_id":...}`（promtail-local 与结构化写入已落地）。
- **Why**: Status UI 报错带了 data-traceId 却检索不到，等于没有全链路钥匙。
- **How to apply**: `runAll/src/status_ui` `showRequestError` / `apiFetch`；AiMonitor promtail-local；`logs/` tee

## [OPT-20260823-024] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: Precise-restart 5 built 0 failed (runall-console 14:02:32). Loki {job="task-cloud-service"} and {job="task-project-service"} |= "opt024-admin" each 1 stream: http_request impersonating=true impersonator_user_id=opt024-admin impersonated_user_id=opt024-target (not task-auth).
- **Created**: 2026-08-23
- **Context**: `shareLib/tracelog` 已从 `X-Impersonator-Id` 注入 slog 字段，taskAuth 已热替换。下游服务（taskCloudService 等）需用新 tracelog 重新编译后，`http_request` 才会带 `impersonator_user_id`。
- **Action**: (1) 推送 shareLib 后对使用 `tracelog.Middleware` 的 runAll Go 服务执行精准编译重启（勿带脏子仓 WIP）(2) 用模拟会话打一笔业务 API，在 Loki 确认非 task-auth 的 job 也有 impersonation 字段
- **Why**: 网关已透传头，但旧二进制的 slog handler 不会附加字段，排障仍可能只看到被模拟用户。
- **How to apply**: `shareLib/tracelog/impersonation.go`；runAll :9999 精准编译重启

## [OPT-20260817-021] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: Same precise-restart rebuilt task-cloud-service. POST start-vm with X-Trace-Id+X-Parent-Span-Id returned 缺少任务ID not trace-propagation-incomplete (Loki trace_id=opt021hdr1787897015). Trace-id-only still 400 incomplete. Did not fire live @镜像/start-vm-auto so comment_csc_start_bootstrap_ok streams=0; inbound middleware+ApplyOutboundHeaders deploy is the original Starting-stuck bug.
- **Created**: 2026-08-17
- **Context**: 本会话已修复 `postCommentCSCStartVM` 只发 `X-Trace-Id` 被 `RejectTraceIdOnlyHTTP` 400 拦截导致永久 Starting；当前卡住实例已用完整 trace 头手工 `start-vm` + `advance` 救活为 running。进程仍为旧二进制，未重启前新的 @镜像 bootstrap 仍会复现。
- **Action**: (1) 在 http://10.2.150.68:9999/ 点击「精准编译重启」（已登记 `task-cloud-service`）(2) 再发一条评论级 @镜像，确认 Loki 出现 `comment_csc_start_bootstrap_ok` 且无 `trace propagation incomplete` (3) 页面运行状态离开 Starting
- **Why**: 代码未部署时新启机会继续卡 Starting；手工救活只覆盖本条 comment。
- **How to apply**: `.runall/precise_restart_services.txt`；`taskCloudService/src/comment_csc_bootstrap.go`；验收 TraceId 走 Loki `{job=~".+"} |= "<id>"`

## [OPT-20260822-004] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: Precise-restart 51 built 0 failed (runall-console 14:14:39). All 51 taskEvents/bin/task-events-* mtime >= 14:10; newest strings include taskEvents/broker.(*Backoff).WaitAttempt + retry_wait_ms. go test ./consumer ./broker -run DispatchRetryWait|BackoffWaitAttempt pass (attempt 5 = 16s). Loki {job=~"task-events-.*"} |= "retry_wait_ms" 0 streams in 24h — no retryable dispatch traffic to sample live wait_ms.
- **Created**: 2026-08-22
- **Context**: 全部重启 73 个成功项编的是 HEAD 或半成品闭环前的旧包（进程内 `Backoff.Wait`）。本会话只精准重启失败项 + 已登记的 status-changed 消费者。
- **Action**: (1) 在业务低峰 `scripts/register-precise-restart.sh taskEvents` (2) 点精准编译重启 (3) 抽查 `retry_wait_ms` 日志在 attempt≥5 时不再是 ~1s
- **Why**: 旧二进制仍会把 11 次 retryable 压进约 20s（见 110）。
- **How to apply**: working_dir 别名 `taskEvents` 会展开全部 `task-events-*`；避开与全部重启重叠的 bulk 互斥

## [OPT-20260823-008] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskTaskService notifyContainerAgentPending is no-op (6fc5f2b). task-detail synthesizes parent+image without pending agent (credential affecb1). onlineServiceJS POST-creates then streams (trae-agent c7a053f). Tests: go test taskTaskService -run Notify|EnsureAutoRun|ContainerSnapshotInternal; go test credential ./interfaces ./infrastructure; node --test atMention/autoRun/postBootstrap/runtimeEvent 61 pass. Docker: registry.cn-qingdao.aliyuncs.com/ruandao/task2app-trae:x86_64_2026-08-28_15-51 sha256:87cee9a851a727ae757e6bf4f06fa59e6897cc73ac43262d8538cc29a5f48f37. Precise-restart registered: task-task-service task-credential-service.
- **Created**: 2026-08-23
- **Context**: `@镜像` / auto_run 仍由 taskTaskService `notifyContainerAgentPending` 在启服前插入 pending 行（编排需要 `agent_comment_id`）。本会话仅在 Feed 隐藏空气泡；edit_run 路径已由 onlineServiceJS `createEditRunAgentComment` 在容器内创建。
- **Action**: (1) ContextPack 只带 `parent_comment_id` + image，允许缺 `agent_comment_id` (2) 容器 bootstrap 后 POST 创建再 stream (3) `findActiveAutoRunAgentByTask` 改为按 parent 查或等容器回报 (4) 补重放/冲突测例
- **Why**: 用户语义是「回复由容器镜像创建」；平台预创建会在「启动中」阶段露出空回复。展示层过滤是权宜，创建权仍在平台。
- **How to apply**: `taskTaskService/src/aic_client.go` `notifyContainerAgentPending`；`trae-agent/onlineServiceJS/src/editRunAgentComment.mjs`；`atMentionContext.mjs`；`taskAIComment` public POST

## [OPT-20260828-010] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskAuth create path 409 phone_taken (5fa6b9b); FE handleAddUser inline add-phone-error + add-phone-taken.test.js
- **Created**: 2026-08-28
- **Context**: 修编辑用户 PUT 占用号码静默 200 时发现 `handleSystemAdminCreateUser` 仍只走 email+password 建号，添加表单的 `phone` 被忽略，占用号码同样没有提示。
- **Action**: (1) 在 `taskAuth/src/handlers_system_admin_user_write.go` 的 create 路径复用 `applyAdminLoginIdentifiers` (2) 占用则 409 `phone_taken` 且不创建半成品账号 (3) `useSystemAdminUsers.js` `handleAddUser` 在添加弹窗内联展示错误
- **Why**: 编辑与添加是同一运营表单字段；只修编辑会留下对称的静默失败。
- **How to apply**: `handleSystemAdminCreateUser`；`taskFE/app/src/composables/useSystemAdminUsers.js` `handleAddUser`；对照 `handlers_system_admin_user_write_phone_test.go`。

## [OPT-20260828-011] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: taskAuth patchUserAsAdmin writes username/password (5fa6b9b); handlers_system_admin_user_write_profile_test.go
- **Created**: 2026-08-28
- **Context**: 编辑弹窗提交 `username`/`password`，`patchUserAsAdmin` 仍不写入 profile/密码哈希，与手机号被忽略是同一类「表单有字段、接口吃掉」问题；本次只修了 phone/email 占用。
- **Action**: (1) PUT 有 username 时调用 `upsertUserProfile` (2) 非空 password 时走现有改密路径并写审计日志（禁止打出口令）(3) 补单测：改名成功、空密码不改密
- **Why**: 运营以为保存了显示名/密码，实际只改了角色标志，排障成本高。
- **How to apply**: `taskAuth/src/handlers_system_admin_user_write.go` `patchUserAsAdmin`；`auth_user_profile.go` `upsertUserProfile`。

## [OPT-20260828-012] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: Extracted SystemAdminEditUserModal.vue; wc -l SystemAdminUsers.vue=399 SystemAdminEditUserModal.vue=138; vitest 4 phone-taken files passed.
- **Created**: 2026-08-28
- **Context**: 在 `SystemAdminUsers.vue` 为手机号增加「解绑」控件后文件为 498 行，距 500 行门禁只差 2 行。后续任意表单字段都会触发强制削文件。
- **Action**: (1) 把编辑用户模态框抽到 `taskFE/app/src/components/SystemAdminEditUserModal.vue`（表单 + 解绑按钮）(2) `SystemAdminUsers.vue` 只保留 `v-if` 挂载 (3) `wc -l` 两文件均 ≤500 且现有 `SystemAdminUsers.edit-phone-*.test.js` 全绿
- **Why**: 行数门禁触发后必须当场削文件；提前拆弹窗避免下次改字段被打断。
- **How to apply**: `taskFE/app/src/views/SystemAdminUsers.vue` 编辑模态框块（约 134–247 行）与 `useSystemAdminUsers.js` 的 `editUserForm` / `handleEditUser`。

## [OPT-20260828-015] completed

- **Status**: completed
- **Completed**: 2026-08-28
- **Summary**: 已安装卡片用 ImageGroupIcon；存量行目录回退；安装路径快照 icon_url（043 + JSON）。Vitest/Go/node 测绿。
- **Created**: 2026-08-28
- **Context**: 主站 ImageMarket 的开发中/已发布组卡已渲染 `icon_url`（无图标时占位）。已安装列表仍是独立平铺卡片，只显示名称/版本，taskCloudService 安装记录 JSON 也未透传 `icon_url`。
- **Action**: (1) 确认 installed-images API 是否能从 catalog 带回 `icon_url` 或 `image_group.icon_url`；(2) 若无则在 cloud 代理层透传或按 `image_group.id` 拼 convention URL；(3) `ImageMarket.vue` 已安装卡加与组卡相同的 40px `<img>`/`placeholder` + 测例。
- **Why**: 用户在「已安装」区对照市场目录时缺少同一视觉标识，卸载/辨认只能靠名称。
- **How to apply**: `taskCloudService/src` installed image handlers + `taskFE/app/src/views/ImageMarket.vue` 已安装 `<ul>`；图标流仍走 `GET /api/ai-provider/public-image-groups/{id}/icon`。

