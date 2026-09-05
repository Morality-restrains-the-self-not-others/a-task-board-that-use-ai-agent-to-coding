# 存量业务进程 ticker 迁出计划

- **日期：** 2026-08-16
- **规则：** 元规则 46 / ADR-0011 / `.ai/01_project_constraints/51_no_service_internal_poll_loop.md`
- **TODO：** `OPT-20260816-022`（总表）及 `OPT-20260816-025`～`032`（分项）
- **模板：** `taskEvents/internal/handlers/taskpostexpiryscan/` + `taskEvents/cmd/task_post_expiry_scan/1_expire_posts/main.go`

## 目标形态

```
专门 timer 进程 (taskEvents) --HTTP POST 一次性 API--> 表 owner 业务服务
外界事件 (Kafka / 用户 HTTP / 状态变更) ---------------> 表 owner 业务服务
```

业务 `main` **不再** `go ticker`。门禁 `LEGACY_INTERNAL_TICKERS` 每迁出一处删一行，最后全库扫描须 0 legacy。

## 建议顺序（依赖从弱到强）

| 序 | OPT | 存量文件 | 间隔（现状） | 一次性 API | timer 进程 |
|----|-----|----------|--------------|------------|------------|
| 1 | 025 | `taskCloudService/src/workspace_machine_recycle.go` | env 未设则 **已禁用**；真时钟已在 18044 | **已有** `POST /api/internal/cloud/compute/recycle-idle-machines/` | **已有** `task-events-workspace-machine-idle-1-recycle-idle-nodes` |
| 2 | 026 | `taskBill/src/main.go` `startReferralSettleLoop` | 1h | **已有** `POST /api/internal/taskbill/referral/settle-due/` | 新建 `billing_referral_settle_scan/1_settle_due`（建议端口 18062） |
| 3 | 027 | `taskCloudService/src/server_orphan_reconcile.go` | 默认 10min | 新建 `POST /api/internal/cloud/compute/reconcile-orphan-csc/`；**禁止**再直连 task-task MySQL | 与 028 共用一个 worker（先 orphan 后 leaked） |
| 4 | 028 | `taskCloudService/src/server_release_reconcile.go` | 默认 5min | 新建 `POST /api/internal/cloud/compute/reconcile-leaked-servers/` | `cloud_csc_reconcile/1_sweep`（建议端口 18063） |
| 5 | 029 | `taskBill/src/wechat_profit_sharing.go` | 1h | 新建 `POST /api/internal/taskbill/profit-sharing/process-pending/` | `billing_profit_sharing_scan/1_process_pending`（建议端口 18064） |
| 6 | 030 | `taskTaskService/src/queued_schedule_dispatch.go` | 30s | 新建 `POST /api/internal/tasks/queued-schedule/dispatch-once/` | `queued_auto_run_scan/1_dispatch`（建议端口 18065） |
| 7 | 031 | `taskReferral/src/referral_code.go` | 1h | 新建 `POST /api/internal/referral/expire-codes/` | `referral_code_expiry_scan/1_expire`（建议端口 18066） |
| 8 | 032 | `go_relayToTrae/src/push.go` | 1.5s 进程内心跳 | **不新建 timer**（状态在 sidecar 内存里，无表可扫） | 改为 `/v1/register|start|stop` 与真实状态变化时立刻 push |

端口 18062–18066 落盘前须对 `conf/events/domain-events/**/config.yaml` 与 `AllIntents` 再核一次，避免冲突。

## 新建 timer worker 标准清单（025 除外）

复制 `task_post_expiry_scan`，每条须同时改：

1. owner 服务：幂等 POST + `X-Internal-Secret` + 单测（handler 调已有 `*Once` 函数）
2. `taskEvents/internal/handlers/<pkg>/runner.go`（`transport: timer`，无 Kafka）
3. `taskEvents/cmd/<event>/<intent>/main.go`
4. `taskEvents/config/intent_registry.go` `AllIntents`
5. `taskEvents/run.sh` `INTENT_PATHS`（及如需 `EVENTS`）
6. `conf/events/domain-events/<event>/config.yaml`
7. `conf/runAll.yaml` 独立 service：health、`depends_on` owner、tick 间隔 env（SSOT 在该 conf）
8. 删业务 `main` 里 `go startXxxTicker()` / `NewTicker`
9. 从 `db/scripts/ci/check_no_service_internal_poll_loop.py` 的 `LEGACY_INTERNAL_TICKERS` 去掉路径
10. `python3 db/scripts/ci/check_no_service_internal_poll_loop.py` 与相关 Go 单测全绿

**禁止**双时钟并行超过一个发布窗口：先上 timer 并确认 health，再关业务 ticker（同一 PR 内先加后删即可）。

## 分项要点

### 025 空闲回收 — 只删重复时钟

`IDLE_RECYCLE_TICK_SEC` 未在 `conf/` 出现，进程内 ticker 默认不启动。`handleInternalRecycleIdleMachines` 与 18044 worker 已是目标形态。删 `startIdleRecycleTicker`、`idleRecycleTickSeconds` 及 `main.go` 调用；保留 HTTP。

### 026 推荐结算 — 只加时钟

`handleInternalReferralSettleDue` 已存在。timer 调该路径即可；删 `startReferralSettleLoop`。

### 027 + 028 云 CSC 对账 — 一条流水线

orphan 把「任务已删但仍有 instance_id」标 `terminal_released=1`，leaked 再清 `server_url`/停 sidecar。应用 **一个** timer 顺序调用两个 Once API。

027 额外：`server_orphan_reconcile.go` 现用 `dbload.ResolveMySQLDSN("task-task")` 跨服务直连，违反单库单表所有权。改为 HTTP 问 taskTaskService（批量 `task_id` 是否存在），或依赖/补强 `TASK_DELETED` 消费者后，timer 只做补偿扫描且不直连他库。

### 029 微信分账

`processPendingProfitSharings` 已是 Once。包一层 internal POST；timer 1h；`wechatLiveOK==false` 时 API 仍 200 + skip（与现 daemon 一致）。

### 030 排队节奏调度

`runQueuedScheduleDispatchOnce` + `runQueuedScheduleAutoCloseOnce` 合成一次 API。30s 间隔对 timer 进程可接受（对标现网）；勿把 30s 扫表塞回 taskTaskService。

### 031 推荐码过期

`expireReferralCodes` 已是 Once。读路径 `getActiveReferralCode` 已按 `expires_at` 过滤，timer 只为管理列表 `status` 字段；可 1h。

### 032 relay 状态推送 — 外界触发，不是 timer

`statusPushLoop` 扫的是 **本进程** `registeredTasks`，放到 taskEvents 会变成「定时 HTTP 打 sidecar」，更差。改为：

- `ensurePushThread` / ticker **删除**
- 在 register / start / stop / 导致 status 变化的子进程事件上同步 `pushStatusToBackend`
- 单测：无 ticker；一次 register 至少一次 push；stop 后再无周期请求

若后端仍要心跳保活，应改为后端短超时判定离线（外界不拉），而不是 sidecar 1.5s 盲推。

## 验收（全部 OPT 完成后）

```bash
python3 db/scripts/ci/check_no_service_internal_poll_loop.py
# 期望：ok: no new business-service ticker (0 legacy allowlisted)
rg -n 'startIdleRecycleTicker|startLeakedServerReconcileTicker|startOrphanTaskCSCReconcileTicker|startReferralSettleLoop|runProfitSharingDaemon|startQueuedScheduleTicker|startReferralExpiryLoop|statusPushLoop' \
  taskBill taskCloudService taskTaskService taskReferral go_relayToTrae --glob '*.go' | rg -v '_test.go'
```
