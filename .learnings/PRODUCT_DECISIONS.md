# Product Decisions — 需产品确认

> 需人工拍板的待办。**本文件只保留 `pending`（尚未确认）**。
> 确认后：补 `- **Decision**:`、Status 改 `decided`，**立即分流**——未落地 → [OPTIMIZATION_TODOS.md](./OPTIMIZATION_TODOS.md)；已落地 → [OPTIMIZATION_TODOS_COMPLETED.md](./OPTIMIZATION_TODOS_COMPLETED.md)。
> 分流规则见 [OPTIMIZATION_TODOS.ai.md](./OPTIMIZATION_TODOS.ai.md)「产品决策分流规则」。**禁止**在此堆积 `decided` / `accepted`。

本地可执行 pending 见 [OPTIMIZATION_TODOS.md](./OPTIMIZATION_TODOS.md)。

## 编号与状态

- **编号**：`OPT-YYYYMMDD-NNN`（`python3 .learnings/_move_opt.py --next-id`，与开放清单共享编号空间）
- **Status**：仅 `pending`
- **字段**（与 OPT 模板对齐）：`Status` / `Created` / `Context` / `Action` / `Why` / `How to apply`；确认后追加 `Decision` 再分流

- **Count**: 8

### OPT-20260823-033 — 微信关联账号查单是否放开 nickname 模糊 LIKE（前缀/全文索引）

- **Status**: pending
- **Created**: 2026-08-25
- **Context**: 超管「微信关联账号」查单当前按等值匹配（nickname/openid/unionid + 已绑定登录标识），并有 nickname 等值索引。运营若需「粘贴一段昵称也能命中」，需放开 LIKE（前缀或全文索引），有写放大与误命中风险。
- **Action**: (1) 产品确认是否允许模糊昵称查询 (2) 若允许：评估 nickname 前缀/全文索引与写放大，明确索引方案（非临时改 SQL）(3) 为 wechat_linked_account_lookup / admin_order_wechat_account_query 保留 latency 直方图观测（taskAuth/taskBill 已挂载 MetricsMiddleware + Grafana 看板，见 OPT-20260823-033 completed 摘要）
- **Why**: 严格等值避免误伤；模糊放开是搜索行为变更，需产品拍板且要有可观测性与明确索引方案，不能临时改 SQL。
- **How to apply**: `dataMigrate/taskAuth/038_wechat_identity_nickname_index.sql`；`taskAuth/src/auth_wechat_linked_account.go`；AiMonitor Grafana `admin-wechat-account-latency` 看板

### OPT-20260817-026 — cancelled 前序对下游串行的产品策略（跳过 vs 永久阻塞）

- **Status**: pending
- **Created**: 2026-08-17
- **Context**: 当前 `ccbIsBlockedByDependencies` 仅 `completed` 放行；用户终止中间评论后，依赖它的后续串行会永久 waiting（与 failed 一致）。可能需「跳过已终止前序」或「级联提示终止下游」。
- **Action**: (1) 与产品确认：cancelled/failed 前序应跳过还是阻塞 (2) 若跳过：改 `ccbIsBlockedByDependencies` 并补单测 (3) 若阻塞：在前序列表对 cancelled 行加「已终止，后续需手动终止」提示
- **Why**: 长串行链上只终止一条会留下「卡死」等待态，易被误认为 bug。
- **How to apply**: `taskCloudService/src/comment_container_bindings_schedule.go`；FE `TaskDetailCommentPredecessorList.vue`

### OPT-20260817-038 — 软跳过启服时 Agent pending 是否创建可配置

- **Status**: pending
- **Created**: 2026-08-17
- **Context**: 软跳过（Git/子 Git 探测失败 → `StartSkipReason` 非空）仍创建【自动运行】评论（正文含「未启动服务器：{reason}」）并调用 `notifyContainerAgentPendingFn` 通知容器 agent「有新 pending 评论待处理」；服务器未启动即通知 agent，agent 可能在容器未就绪时立即尝试运行。注：原条目全文在迁移时丢失，本条目按标题与 `auto_run_at_comment.go` 现状重建，供产品参考。
- **Action**: (1) 与产品确认：软跳过启服时是否仍通知容器 agent 创建 pending（当前代码已保证 pending 通知失败不阻断 start-vm）(2) 若需可配置：增加配置项（如 feature-params 开关）控制软跳过时是否跳过 `notifyContainerAgentPendingFn` (3) 补单测
- **Why**: 服务器未启动就通知 agent 处理 pending，agent 可能空转/误报；需产品决定预期语义。
- **How to apply**: `taskTaskService/src/auto_run_at_comment.go` 的 `notifyContainerAgentPendingFn` 调用点；对照 `composeAutoRunAtCommentContent` 的 `StartSkipReason` 分支

### OPT-20260822-051 — 已消耗配额是否按剩余比例退款

- **Status**: pending
- **Created**: 2026-08-22
- **Context**: 订单详情支付成功横幅已展示退款入口，并注明「已消耗的资源无法退回」。当前获批仍按订单**全额**原路退，只收回剩余未使用配额、不删除已创建任务帖。
- **Action**: (1) 产品确认：已消耗部分是否扣减可退金额 (2) 若扣减：按 remaining/granted 折算 frozen_points 并走渠道部分退款 (3) 同步超管审批弹层金额展示
- **Why**: 全额退款 + 保留已消耗任务对平台偏慷慨；改金额属资金策略，不能在未确认时落地。
- **How to apply**: `taskBill/src/refund_apply.go` frozen_points；`refund_order.go` revoke；OpenAPI RefundApplication

### OPT-20260819-013 — 存量三节订单号在租户分片后的定位表

- **Status**: pending
- **Created**: 2026-08-19
- **Context**: `ORD-YYYYMMDD-NNN` 与 ADR-0017 三节号解析不到 tenant。当前 `loadOrderByID` 可在单库按主键找行；按 `tenant_id` 分片后只拿订单 Snowflake 无法选片。
- **Action**: (1) 分片上线前为存量三节号建 `order_id → tenant_id` locator（或全量回填展示号，产品需拍板）(2) `loadOrderByID` / 支付回调改为先查 locator 再带 `tenant_id` 入片 (3) 新四段号可不写 locator
- **Why**: 回调、退款审批、管理端按 id 查单在分片后会扫错片或扫全片。
- **How to apply**: ADR-0018 Consequences；`taskBill/src/orders_load.go` `loadOrderByID`；禁止把订单 `id` 哈希当分片键

### OPT-20260823-054 — 退款冲红重开链路确认是否同步改手动

- **Status**: pending
- **Created**: 2026-08-23
- **Context**: 主开票链路已改手动（issue_mode=manual），但 `reissueBlueInvoice`（退款后按剩余金额重开）仍自动调用 `issueWechatFapiaoFn`。若商户 API 开票整体不可用，退款后重开同样会失败。
- **Action**: (1) 与业务确认退款冲红后的重开是否也改手动登记（冲红 reverse API 通常可用，重开 issue 可能不可用）(2) 若改：复用 issue_mode=manual 语义，reissue 发票直落 `issued` 并在退款审批面板登记
- **Why**: 保持两种开票路径行为一致，避免退款场景静默失败。
- **How to apply**: `taskBill/src/invoice_refund.go` reissueBlueInvoice；`refund_approve.go` 联动；`invoice_refund_test.go`

### OPT-20260903-011 — 为 GitLab 订单消耗补按订单计量流水（按订单归属，替代 LIFO 分摊）

- **Status**: pending
- **Created**: 2026-09-03
- **Related**: 与 [[#OPT-20260822-051]] 同簇 —— 051 决定「已消耗配额是否按剩余比例退款」前，本项的按订单消耗归属没有资金语义、仅影响退款审批展示口径；若 051 拍板按比例退，本项正是实现该扣减所需的计量机制。
- **Context**: 退款「资源消耗」已能展示本单 GitLab 磁盘/流量的已发放数量；消耗/剩余是 `attachGitlabKind`→`gitlabOrderRemaining` 按区域当前用量（`quota − used`）对已支付订单**最新优先**分摊的近似（等效「旧订单先被算作已消耗」），不是逐笔计量归属。现状代码事实：GitLab 磁盘=绝对测量字节（`reportGitlabDiskUsage`）、流量=区域池化计数器（`meterGitlabOutboundTraffic` 只累加 `traffic_used_gb`）；计量事件**不带 order id**，购买侧也不写 `billing_resource_grant` 批次（仅 task_post 购买与 admin_grant 赠送写），故**不存在可读的本单流水**。退款金额走 `billing_payment_ledger.remaining_points`（`FrozenPoints`），与 `resource_consumption` 无关。
- **Action**: (1) 产品确认目标口径：按订单建立 grant lot 后，池化用量如何归因到具体订单（FIFO/最新优先/按仓库归属？赠送与购买如何交错？磁盘绝对用量回退场景如何冲正？）(2) 若确认要建：购买侧在 `markOrderPaid` 为 gitlab_disk/gitlab_traffic 写 `billing_resource_grant`（order_id+region+remaining，source_kind=purchase）；计量写入按策略从 lot FIFO 扣减并写 `billing_transaction`（points_source_type=quota_consumption, related_order_id）；存量已支付订单需回填 lot 或 attach 保留 LIFO 兜底 (3) `attachGitlabOrderConsumption` 改读本单 lot remaining/流水而非 LIFO；回归 `go test ./src -count=1 -timeout 180s -run TestOrderJSONGitlab`。
- **Why**: 多笔同区域购买时，LIFO 把用量算到较早订单，退款审批看到的「已消耗」可能与真实使用订单不一致；但池化计量下「真实归属」本身就是策略选择，且当前无资金影响，需产品拍板值不值得为纯展示口径引入整套按订单计量 ledger（含到期/强制/退款联动与历史回填）。
- **How to apply**: `taskBill/src/order_consumption_gitlab.go`、`taskBill/src/gitlab_resources.go`（`reportGitlabDiskUsage` / `addGitlabTrafficUsedGBForRegion`）、`taskBill/src/gitlab_traffic_meter.go`、`taskBill/src/order_payment.go`（markOrderPaid 购买侧 lot）；决策后再分流回 [OPTIMIZATION_TODOS.md](./OPTIMIZATION_TODOS.md)。

### OPT-20260904-010 — 评估通用 TASK_COMMENT_CREATED SSE（非 git_pr）

- **Status**: pending
- **Created**: 2026-09-04
- **Context**: 2026-09-05 夜跑枚举（taskTaskService）：任务评论 INSERT 仅两个入口 —— (a) `src/task_handlers_comments.go` `handleCreateComment`（HTTP POST /comments，人类/agent/内部客户端都走它；git_pr 子评论路径已在此发 `task_git_pr_reply_created` SSE）；(b) `src/auto_run_at_comment.go` `ensureAutoRunAtComment`（auto-run/软跳过启动时后端合成【自动运行】@镜像 评论，直接 DB INSERT，**不发任何 SSE**；该路径可被队列/调度/另一客户端触发，视图中的其他标签/客户端不会收到通知）。本期 git_pr SSE 只覆盖 (a) 中 `git_pr_html_url` 非空的分支；普通人类评论与 auto-run 合成评论仍靠提交方本地 fetchTaskDetail，另一标签/relay 客户端打开同一任务会陈旧。
- **Action**: (1) 产品确认：是否要「任意客户端/后端创建的评论都实时出现在所有打开该任务的标签」——即多标签/跨客户端协作是否为产品场景（auto-run 合成评论 + agent 状态评论 + 其他管理员代建）(2) 若确认要做：把 git_pr 特化 SSE 泛化为通用 `task_comment_created`（SSE_MESSAGE event_name），在 (a) 成功 INSERT 与 (b) `ensureAutoRunAtComment` INSERT 后各发一条，载荷含 task_id/comment_id/parent_comment_id/created_by，FE `establishSSEConnection` 按 comment_id 去重触发 Feed 重拉；(3) 与现有分页 Feed 合并策略一并评审（避免重复重拉/游标错位）。建议顺序：先完成 OPT-20260904-009 线上验证 git_pr SSE 链路，再泛化。
- **Why**: 避免每类评论各自打补丁；但泛化涉及 taskTaskService/taskEvents/taskSSE/taskFE 四端与分页 Feed 语义，属跨端迭代，需产品按协作场景优先级拍板。
- **How to apply**: `taskTaskService/src/task_handlers_comments.go`、`taskTaskService/src/auto_run_at_comment.go`；对照 billing_order_comment_created fanout 与 `docs/superpowers/specs/2026-09-04-git-pr-reply-sse-fanout-design.md`；决策后再分流回 [OPTIMIZATION_TODOS.md](./OPTIMIZATION_TODOS.md)。
