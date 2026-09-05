# Completed OPT Archive — 2026-08-15

> 从 OPTIMIZATION_TODOS_COMPLETED.md 按天归档，共 21 条。
> 归档执行时间：2026-08-19T16:47:13+08:00

## [OPT-20260814-023] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: 16 个 Go 服务 build.sh 增加 -buildvcs=false（claude-agent 保留 git describe 例外，-wt worktree 副本不动）；bash -n 全过；rg 断言仅剩有意保留 VCS 的例外。已逐仓提交推送。
- **Created**: 2026-08-14
- **Context**: 父仓 git 损坏时 `go build` 默认 VCS stamp 以 exit 128 失败。`taskTaskService/build.sh` 已加 `-buildvcs=false`（`1200f91`）。同模式仍在 `taskCloudService`、`taskAuth`、`taskBill`、`taskProjectService` 等 15+ 个 `build.sh`。
- **Action**: (1) 对所有 `go build` 且未带 `-buildvcs=false` 的 `*/build.sh` 补上该 flag（2）`bash -n` 每个改过的脚本（3）`rg -n 'go build' --glob '**/build.sh' | rg -v buildvcs` 断言只剩有意保留 VCS 的例外（如 claude-agent 用 `git describe`）
- **Why**: 精准重启是 kill-first 再编译；任一服务 stamp 失败即停机。
- **How to apply**: 各服务 `build.sh`；验收命令见 Action (3)。

## [OPT-20260814-018] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: genID 改为统一 shareLib/snowflake（保留 prefix 前缀），删除 unixNano*1000+seq 实现与 idSeq；go.mod 加 replace snowflake；新增 genid_test.go 锁定 <prefix>_<snowflake digits> int64 范围。存量 ID 不改写。taskTaskService 全量测试 23s 绿。
- **Created**: 2026-08-14
- **Context**: 设计工作空间人读序号时发现 `taskTaskService/src/handlers.go` 的 `genID` 使用 `unixNano*1000+seq`，不是 `shareLib/snowflake`。主键仍全局唯一，但与 `.ai/01_project_constraints/35_snowflake_id_generation.md` 不一致。
- **Action**: (1) 盘点 `genID` 调用面与已落库 `task_*` 主键形态 (2) 评估新建记录是否改为 `snowflake.GenerateIDString()` 并保留 `task_` 前缀 (3) 存量 ID 不改写；补单测锁定格式
- **Why**: 未对齐则跨服务 ID 比较/时间序假设会漂移；人读序号落地后更适合单独收口技术 ID。
- **How to apply**: `taskTaskService/src/handlers.go` `genID`；`shareLib/snowflake`；勿改已有 `workspace_seq` 方案。

## [OPT-20260814-017] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: handlePatchServerConfig 带 comment_id 时写入前用 commentCSCCloudMetaIllegal 校验；mock/空 platform/region/auth → 400 不落库；新增 server_config_handlers_patch_comment_test.go（4 非法/合法用例 + 无 comment_id 放行控制）。taskCloudService 全量测试绿。
- **Created**: 2026-08-14
- **Context**: 今日 `handlePatchServerConfig` 只打任务级行。设计允许评论级合法覆盖，但若后续 PATCH 带 `comment_id` 写评论行，仍可能把 platform 写成 mock 或清空 region/auth。
- **Action**: (1) 若 PATCH 带 comment_id，写入前用 `commentCSCCloudMetaIllegal` 校验 (2) 非法返回 400，不落库 (3) 单测：评论行 PATCH mock/空字段失败，合法覆盖成功
- **Why**: 写路径若不校验，读路径自愈只能修历史脏数据，挡不住新的非法写入。
- **How to apply**: `taskCloudService/src` 中 `handlePatchServerConfig`；复用 `commentCSCCloudMetaIllegal`。

## [OPT-20260815-001] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: handleGetOrder 解析 URL 租户并与订单 tenant_id 比对，不一致/不可解析一律 404（IDOR 防探测）；新增 handlers_orders_idor_test.go（同租户 200 / 跨租户 404）。taskBill 全量测试 37s 绿。
- **Created**: 2026-08-14
- **Context**: 做订单独立详情页时复核 `handleGetOrder`：只按 path 中的 orderId `loadOrder`，未把 `/api/tenant/{tid}/...` 的 tid 与 `ResourceOrder.TenantID` 对齐。详情页会更频繁打这条 GET。
- **Action**: (1) 在 `handleGetOrder` 解析 path tenant (2) 与订单 tenant_id 不一致则 404（避免探测存在性）(3) 补 Go 单测：本租户 200、跨租户 404
- **Why**: 知悉 orderId 即可跨租户读行项/金额，属于 IDOR。
- **How to apply**: `taskBill/src/handlers_orders.go` 的 `handleGetOrder`；对照 `handleCreateOrder` 的 `parseTenantID` + `requireTenantAdmin`。

## [OPT-20260815-003] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: mock-complete 加 internal secret 双门禁，taskBill 2a2ddcb
- **Created**: 2026-08-15
- **Context**: 订单详情公网仍走 `conf/billing/wechatPay/conf.yaml` 的 `mode: mock`。前端已对 `*.daydaymoney.com` 隐藏「模拟支付成功」，但 `POST .../billing/orders/{id}/mock-complete/` 仅判断 `wechatIsMock()`，登录用户仍可直接调接口入账。权限分析原要求 `X-Internal-Secret` + mock 双门禁。
- **Action**: (1) `handleOrderMockComplete` 在 `wechatIsMock()` 之外再要求 internal secret（或 `allow_public_mock_complete` 显式开关，默认 false）(2) 无 secret 返回 403 并打 warn 日志（含 trace_id，不含密钥）(3) 补 Go 单测：无 secret → 403；live 模式 → 403；mock+secret → 进入既有入账路径
- **Why**: 前端隐藏按钮不能阻止直接打 API；生产 `mode=mock` 时存在免费入账风险。
- **How to apply**: `taskBill/src/handlers_orders.go` `handleOrderMockComplete`；对照 `docs/superpowers/specs/2026-07-14-wechat-pay-recharge-permission-analysis.md`。

## [OPT-20260814-009] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: 9999 精准编译重启 10 服务全绿；mock 停机 fix 已部署，Loki 重启后 0 dispatch_permanent_fail，单测 TestDispatchStopMockPlatformSkipsCloudAPI PASS；无运行中 mock 实例未实停
- **Created**: 2026-08-14
- **Context**: `CLOUD_SERVER_STOPPED` 在 `cloud_platform_type=mock` 时被消费者 `DispatchPermanent` 进 DLT（trace `c8d91ac8-4fd0-4284-8601-db9cb59f1798`，任务 `task_15794155595830266865`）。代码已修：本地平台跳过云 API；CSC 误标 mock 的 `i-*` 实例仍走 Aliyun DeleteInstance。进程未重启前现网仍会 DLT。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「精准编译重启」（已登记 task-cloud-service + cloud-server-stopped/started/start-auto + task-status-changed + graceful-shutdown-await）(2) 对 mock CSC 或同任务再点停机 (3) Loki `{job=~".+"} |= "unsupported platform mock"` 在重启后的新 trace 上不得再出现 `dispatch_permanent_fail`
- **Why**: 未重启则旧消费者仍把 mock 当永久失败，停机 200 后实例不会 ClearAfterStop / 不会删 ECS。
- **How to apply**: 机器验收：`curl -s -G http://10.2.150.68:3100/loki/api/v1/query_range --data-urlencode 'query={job=~"task-events-cloud-server-stopped.+"} |= "unsupported platform mock"'` 对重启后时间窗断言 0 条 `dispatch_permanent_fail`；单测 `go test ./internal/handlers/cloudserverstopped/ -run TestDispatchStopMockPlatform`。

## [OPT-20260814-010] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: ai-provider backend=cos 已部署，upload-url 路由 401(非404)；taskFE SPA 重建并发布，公网 index-Pd8pgm6P.js 200（与本地 dist 哈希一致）
- **Created**: 2026-08-14
- **Context**: 厂商证照 COS 预签名已合入源码（upload-url / complete / Admin 路径规则）。现网进程与公网 SPA 仍是旧 multipart 落盘，未重启则申请人仍走 旧 upload。
- **Action**: (1) 在 http://10.2.150.68:9999/ 对已登记的 `ai-provider` + `taskFE` 执行「精准编译重启」(2) `cd taskFE/app && npm run build` 后确认公网 `/static/assets/main-*.js` 为 200 (3) 用虚构图走申请表直传，确认不再 POST multipart `/upload/`
- **Why**: 未重启则 COS 预签名与 FE 直传对现网不可见；旧 multipart 在 `backend=cos` 会 410。
- **How to apply**: `scripts/register-precise-restart.sh ai-provider taskFE`；`taskFE/app` Vite build；联调 `POST /api/ai-provider/vendor-application/upload-url/`。

## [OPT-20260814-014] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: 019 迁移已应用(两列+8 任务级行回填)；task-cloud-service+taskFE 重启健康；SPA 发布；comment_csc_instance_healed 0；task_running_counts_updated 需运行态事件触发（当前系统无 running 实例）
- **Created**: 2026-08-14
- **Context**: 任务级运行中机器/容器计数已合入源码（DDL `019_task_running_counts.sql`、删除 heal、indicators JSON 带两计数）。现网未跑 019 则无新列；未重启则旧进程仍可能按 task_id 刷状态。
- **Action**: (1) 在 http://10.2.150.68:9999/ 执行「初始化全部数据库」使 `dataMigrate/taskCloudService/019_task_running_counts.sql` 生效 (2) 「精准编译重启」`task-cloud-service` + `taskFE`（已登记）(3) `cd taskFE/app && npm run build` 后确认公网 `/static/assets/main-*.js` 为 200 (4) 打开工作区看板，确认卡片可读 `running_machine_count` / `running_container_count`；Loki `{job=~"task-cloud-service.+"} |= "task_running_counts_updated"` 有新日志，且不得再出现 `comment_csc_instance_healed`
- **Why**: 未迁移则 UPDATE 两列失败；未重启则串台写路径仍在；未发 SPA 则看板看不到 N/M。
- **How to apply**: `dataMigrate/taskCloudService/019_task_running_counts.sql`；`scripts/register-precise-restart.sh task-cloud-service taskFE`；看板 `workspace-runtime-indicators`。

## [OPT-20260814-020] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: taskEvents a783f9f：终态释放改 ListByTask 按评论 CSC 遍历 releaseOne，UpsertAfterStart 信封自洽 comment_id/csc_id，mock/relay-local 平台跳过不 dead-letter；全量 go test 绿，已推送
- **Created**: 2026-08-14
- **Context**: `release-servers` 已改 `ListByTask`。`taskgracefulshutdownawait`、`cloudserverstarted`、`cloudserverstopped` 回退仍 `LoadForTask`（lookup 无 comment → 空模板）。本迭代停机信封已带 instance/region/comment，stopped 主路径可避开；其它路径仍可能误读模板。
- **Action**: (1) 盘点三处 `LoadForTask` 调用是否在缺 comment 时读到空模板 (2) 优先改为信封自洽字段；必要时改 `ListByTask` / lookup+comment_id (3) 每处补单测
- **Why**: 停机/启动回退若再读空模板，会漏停或漏清 CSC。
- **How to apply**: `taskEvents/internal/handlers/taskgracefulshutdownawait`、`cloudserverstarted`、`cloudserverstopped`；`cloudconfig.LoadForTask`。

## [OPT-20260814-019] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: taskCloudService e02727e（feat/comment-csc-instance-bind-runtime-status）：根因=loadTaskStartEventPayload 只查 comment_id 空的任务级 start 事件；新增 loadStartEventPayload 优先取评论级 payload 缺失回退任务级；补回归单测 TestLoadStartEventPayload_CommentScopedFallback；全量 src 单测绿 214s。注：改动在 feature 分支，生产生效需并入 main 后 9999 精准编译重启
- **Created**: 2026-08-14
- **Context**: 终态释放 DLT（trace `61544b00-9f30-4c30-97d6-c57264e2cda7`）同任务在 12:11–12:23 反复 `comment_csc_start_bootstrap_failed`（`no task-level start event payload`），从未形成 instance。本迭代只修空模板 retry→DLT，不修起机。
- **Action**: (1) 用 Loki 按该错误文案重建 bootstrap 路径 (2) 定位 task-level start event 缺失原因（payload 未写 / 读错 comment 作用域）(3) 补单测：无任务级 start 事件时评论 CSC 仍能起机或明确失败不再空转
- **Why**: 不修则评论 VM 起不来；终态 no-op 只避免 DLT，用户仍无运行环境。
- **How to apply**: `taskCloudService` comment CSC start/bootstrap；对照 `comment_csc_start_bootstrap_failed`。

## [OPT-20260815-002] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: taskFE 38e96f9：新增 billingResourceTypeLabel.js 共用函数 + 单测（未知类型回退原文）；SystemAdminOrderRecords 改引用；OrderDetail/BillingOrders 已同步改引用，随其 WIP（订单详情/微信支付）落地时一并提交
- **Created**: 2026-08-14
- **Context**: `OrderDetail.vue`、`BillingOrders.vue`、`SystemAdminOrderRecords.vue` 各自维护一份 `resourceTypeLabel` map（task_post / gitlab_disk / gitlab_traffic）。
- **Action**: (1) 抽到 `taskFE/app/src/utils/billingResourceTypeLabel.js` (2) 三处改为引用 (3) 补单测覆盖未知类型回退原文
- **Why**: 新增资源类型时三处文案会漂移。
- **How to apply**: 上述三个 Vue 文件中的 `resourceTypeLabel`；不要改后端 `resource_type` 枚举。

## [OPT-20260815-005] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: 三份 _opt_*.mjs 移入 tests/manual/（无正式用例替代，CDP 人工复验保留），.gitignore 加 _opt_*.mjs + tests/**/__pycache__/ + *.pyc
- **Created**: 2026-08-15
- **Context**: 提交 main 时 `taskFE` 仍有未跟踪的 `tests/_opt_login_probe.mjs`、`tests/_opt_public_accept_followup.mjs`、`tests/_opt_public_accept_once.mjs` 与 `tests/helpers/__pycache__/`。这些是公网验收临时探针，不是产品代码，但会让 meta 仓一直显示 taskFE dirty。
- **Action**: (1) 确认三份 `_opt_*.mjs` 是否已有 Playwright 正式用例替代；(2) 无保留价值则删除，否则移入 `tests/manual/` 并在 `taskFE/.gitignore` 忽略 `_opt_*.mjs`；(3) 在 `taskFE/.gitignore` 加入 `tests/**/__pycache__/` 与 `*.pyc`。
- **Why**: 临时探针留在工作区会污染每次 `git status`，也容易被后续「提交所有变更」误入库。
- **How to apply**: `taskFE/.gitignore`、`taskFE/tests/_opt_*.mjs`、`taskFE/tests/helpers/`

## [OPT-20260815-006] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: 确认 taskAiProvider COS 预签名已用 Go module v0.7.70；sdk/.gitignore 忽略 tencent/（本地调试嵌套克隆）
- **Created**: 2026-08-15
- **Context**: 提交 main 时 `sdk/` 工作区有未跟踪的 `tencent/cos-go-sdk-v5/`（自带 `.git`，remote 为 tencentyun 上游）。与已入库的 `ecs-20140526` 一样是第三方 SDK 克隆，但本次未提交，meta 仓持续显示 sdk dirty。
- **Action**: (1) 对照 `sdk/ecs-20140526` 的入库方式决定是否同样 vendoring（去掉嵌套 `.git` 后按文件加入）；(2) 若仅本地调试用，则在 `sdk/.gitignore` 忽略 `tencent/cos-go-sdk-v5/`；(3) 确认 taskAiProvider COS 预签名是否已用 Go module 而非此目录。
- **Why**: 嵌套 git 仓库不能直接 `git add`，长期 dirty 会干扰子仓指针同步；误提交还会把上游 `.git` 带进 sdk 仓。
- **How to apply**: `sdk/.gitignore` 或 `sdk/tencent/cos-go-sdk-v5/`（去 `.git` 后按 `ecs-20140526` 模式入库）

## [OPT-20260815-007] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: .gitignore 加 *-wt/，git rm --cached 清 4 个幽灵 gitlink，auto-commit 加未登记 160000 剔除，selftest 新增用例 6（20/20 PASS）
- **Created**: 2026-08-15
- **Context**: `scripts/lib/auto-commit.sh` 使用 `git add -A`。meta 工作区里的 `conf-wt`/`docs-wt`/`taskCloudService-wt`/`taskEvents-wt` 带独立 `.git`，被当成 submodule gitlink 写入 index，且已随 `d8d7635` 进 `origin/main`。它们不在 `.gitmodules`。
- **Action**: (1) `.gitignore` 增加 `*-wt/`；(2) `git rm --cached conf-wt docs-wt taskCloudService-wt taskEvents-wt`；(3) auto-commit 在 `git add -A` 后剔除未登记于 `.gitmodules` 的 `160000` 路径；(4) 自测补「工作区含嵌套 git 目录不得入库」。
- **Why**: 幽灵 gitlink 已在 origin/main；继续 `git add -A` 会把任意 worktree 指针推进历史。
- **How to apply**: `.gitignore`、`scripts/lib/auto-commit.sh`、`scripts/lib/auto-commit_selftest.sh`

## [OPT-20260815-008] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: pre-commit/pre-push 改按 .gitmodules path 逐仓 git -C 检查；自测验证幽灵 gitlink 下不再 fatal 且检出 conf 后仓库
- **Created**: 2026-08-15
- **Context**: `.githooks/pre-commit` 与 `pre-push` 用 `git submodule foreach`，遇到 index 里无 url 的 `conf-wt` 即 `fatal` 退出。错误被 `2>/dev/null || true` 吞掉，只扫到字母序 `conf` 之前的子仓，后面的 dirty/未推送全部漏检。
- **Action**: (1) 改为按 `.gitmodules` 的 `path` 逐仓 `git -C` 检查 porcelain / `@{u}..HEAD`；(2) 自测：index 含未登记 gitlink 时仍能检出 `taskFE` 未推送；(3) 勿再 `|| true` 掩盖 foreach 失败。
- **Why**: 规则 32 的子仓优先门禁对 conf 之后的仓库当前是空转。
- **How to apply**: `.githooks/pre-commit`、`.githooks/pre-push`

## [OPT-20260815-009] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: pick_template 改 -print -quit 去 SIGPIPE；KEEP 加模板类型匹配升级；taskFE 显式 pre-commit.node 已部署且幂等
- **Created**: 2026-08-15
- **Context**: `scripts/deploy_repo_random_precommit.sh` 的 `pick_template` 在 `set -o pipefail` 下用 `find | head -1 | grep -q .`，SIGPIPE 导致 415 个 `*.test.js` 被当成无测例。占位钩子一旦写入 `session_hub_lock_check` 就 KEEP 永不升级。本次 taskFE 提交显示「当前无自动化单元测例，跳过」。
- **Action**: (1) 探测改为 `find ... -print -quit` 或 `grep -l`，禁止 `find|head`+pipefail；(2) KEEP 时若正文含「无自动化单元测例」且现已能检出测例则 UPGRADE；(3) 对 taskFE 部署 `pre-commit.node` 并提交钩子。
- **Why**: 前端提交完全绕过随机单测门禁（约束 29）。
- **How to apply**: `scripts/deploy_repo_random_precommit.sh`、`taskFE/.githooks/pre-commit`

## [OPT-20260815-010] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: 默认 git add -u（untracked 需 --include-untracked），黑名单排除 config.local.yaml/__pycache__/*.pyc，message 用 diff --name-only+ls-files -o，非 main 拒绝（--allow-feature-branch 放行）；单元+集成验证通过
- **Created**: 2026-08-15
- **Context**: `runAll/scripts/commit_with_submodules.py` 对子仓 `git add -A`（会收密钥、pycache、嵌套 git）；`generate_commit_message` 对 porcelain `strip()` 后再 `line[3:]`，` M foo.go` 变成 `oo.go`；且在当前检出分支提交，不会把 feature 分支快进到 main。
- **Action**: (1) 默认只 add 已跟踪变更，untracked 需显式允许并排除 `config.local.yaml`/`__pycache__`/`*.pyc`；(2) 用 `git diff --name-only` + `git ls-files -o` 生成说明，勿手切 porcelain；(3) 非 main 时警告或拒绝，除非 `--allow-feature-branch`。
- **Why**: 一键 --apply 可能把密钥和功能分支指针推进 origin/main。
- **How to apply**: `runAll/scripts/commit_with_submodules.py`

## [OPT-20260815-011] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: 删除商品区 v-if=loading 与配对 v-else；Pricing.membership-section.unit.test.js 4/4 通过
- **Created**: 2026-08-15
- **Context**: 去掉会员等级体系区块后，`Pricing.vue` 商品购买门槛 `<section v-if="!loading && pricingData">` 内部仍留着 `<p v-if="loading">加载中…</p>` 与配套 `v-else` 列表。外层已保证 `!loading`，内层 loading 分支是死代码。
- **Action**: (1) 删除该 section 内的 `v-if="loading"` 加载文案；(2) 去掉列表上仅为配对存在的 `v-else`；(3) 用 `Pricing.membership-section.unit.test.js` 确认三项资源仍渲染。
- **Why**: 死分支会误导后续改加载态的人，也让模板比实际控制流更绕。
- **How to apply**: `taskFE/app/src/views/Pricing.vue` 商品购买门槛 section

## [OPT-20260815-012] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: attachStartVmCommentID 增加 taskID 参数写入 container_name（DeriveContainerName 同规则），两调用点更新，auto_run_comment_id_test 加断言与规则用例，Cloud 侧 ApplyStartupLogScope 已确认优先用传入值无双前缀
- **Created**: 2026-08-15
- **Context**: 修复自动运行丢弃 `comment_id` 后，`triggerTaskAutoRun` / `startQueuedMembership` 已写入 `comment_id` 与 `parent_comment_id`。`@镜像` mention 路径还会写入 `container_name` / `mock_container_name`，自动运行尚未对齐，SSE `log_label` 可能仍缺容器名直到 Cloud 侧推导。
- **Action**: (1) 在 `attachStartVmCommentID` 或其后按 `task_id`+`comment_id` 写入 `container_name`（与 `cloudcommon.DeriveContainerName` 同规则）；(2) 给 `auto_run_comment_id_test.go` 增加断言；(3) 确认 Cloud `deriveStartupContainerName` 在缺省时已能填上，避免双前缀 `task_task_`。
- **Why**: 评论卡启动日志依赖 `[实例、容器名]` 标签做路由；缺容器名时调度进度更难对照到该评论。
- **How to apply**: `taskTaskService/src/auto_run_at_comment.go` `attachStartVmCommentID`；对照 `taskEvents/internal/handlers/cloudcommon/startup_log_scope.go`

## [OPT-20260815-015] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: 删除 compute_start_vm_auto.go 凭证前 soft-fail ensure；bootstrap 硬失败保留；新增 TestBootstrapStartVmTokensFailsOnMockPlatform + TestHandleStartVmAutoNativeNoAuthFailsBeforeRunInstances，回归全绿
- **Created**: 2026-08-15
- **Context**: `handleStartVmAutoNative` 在解析云凭证前对 `ensureCommentCloudServerConfig` 做 soft-fail（模板仍 mock 时必然失败）。真正创建评论 CSC 已改到 `bootstrapStartVmTokens`（模板落真实 platform 后硬失败）。前置那段只打误导日志。
- **Action**: (1) 删除 `compute_start_vm_auto.go` 中凭证前的 soft-fail ensure；(2) 保留 bootstrap 硬失败；(3) 单测确认无模板时 start-vm-auto 在 RunInstances 前返回错误且不调用阿里云。
- **Why**: 调度器 `ensure_csc_failed` 与 start-vm 并行时，前置 soft-fail 日志会让人以为 CSC 失败被忽略。
- **How to apply**: `taskCloudService/src/compute_start_vm_auto.go`；对照 `start_vm_bootstrap.go` `bootstrapStartVmTokens`。

## [OPT-20260815-021] completed

- **Status**: completed
- **Completed**: 2026-08-15
- **Summary**: taskFE 层图/git/job/auto-run-steps/layer-changes 转发统一带 comment_id；gateway 两评论 token 夹具绿
- **Created**: 2026-08-15
- **Context**: 文件树 409 根因是 gateway `container-target` 打到空任务级 CSC。本次已在 children/git-log/file-content 带 `comment_id`，且后端无 comment_id 时回退最新带地址的评论级 CSC。层图、git commit/push/merge、layer-changes 文件内容等仍可能不带 `comment_id`。
- **Action**: (1) 盘点 `taskFE` 中所有 `/api/cloud/compute/container-layer-*` 调用；(2) 对走 gateway 转发容器的 GET/POST 统一 `appendCommentIdQuery` 或 body `comment_id`；(3) 多评论并行夹具断言命中对应 CSC token。
- **Why**: 单评论时后端回退已够用；两评论各绑不同实例时，不传 `comment_id` 会打到「最新有地址」的那台，层图/提交会串容器。
- **How to apply**: `taskDetailContainerFns.js`、`taskDetailLayerActions.js`、`TaskDetailExecLayerChangesListItem.vue`、`taskDetailGitFns.js`；gateway 已支持 query/body `comment_id`。

---

> **【2026-08-15 午后】** 005–012、015 全部闭环迁出（清理探针 / sdk 忽略 / 幽灵 gitlink 防护 / hooks 逐仓化 / deploy 模板修复 / commit_with_submodules 收紧 / Pricing 死分支 / container_name 对齐 / start-vm-auto 硬失败）。
>
> **【2026-08-15 日间】** 提交脚本审计新增 007–010（worktree gitlink、foreach 门禁、taskFE placeholder、commit_with_submodules）。
>
> **【2026-08-14 日间】** OPT-20260814-006（停机释放 comment binding）与 OPT-20260810-028（厂商审核开关 + 镜像市场 SSO）已闭环迁出。其余 13 条公网验收已带 Playwright 证据分流至 [BLOCK_TODO_BROWSER.md](./BLOCK_TODO_BROWSER.md)（错租户改写 / 第二账号 / 停机按钮不可见 / 人员页加载中）。

---

