# 业务服务禁止进程内轮询 / 循环（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-16
- 维护者：Trae AI 团队
- 优先级：高（禁止忽略）
- 约束索引：`00_project_constraints.md` 第 46 条
- Cursor：`.cursor/rules/no-service-internal-poll-loop.mdc`（alwaysApply）
- ADR：`docs/adr/0011-no-service-internal-poll-loop.md`
- 门禁：`db/scripts/ci/check_no_service_internal_poll_loop.py`
- 自测：`db/scripts/ci/test_check_no_service_internal_poll_loop.py`

## 背景

业务进程（HTTP/RPC 服务）若在 `main` 里再挂 `time.NewTicker` / `for { sleep; do work }`，会把「何时执行」和「做什么」绑在同一个长驻进程上：

1. **触发不可观测** — 周期工作没有独立健康检查、独立扩缩、独立失败面；ticker 挂了 HTTP 仍 200
2. **职责膨胀** — 计费、云主机、任务调度各自偷偷扫表，重复实现同一套「隔 N 秒跑一次」
3. **无法按事件驱动扩展** — 本仓库意图成功路径须投递领域事件（Kafka）；进程内轮询绕过事件与消费者隔离
4. **前端盲轮询同构** — SPA `setInterval` 打 Describe / 状态接口会产生费用、打到错误实例、掩盖推送缺失

本仓库已有正确形态：`taskEvents` 下 **timer worker**（如 `task-events-task-post-expiry-scan-1-expire-posts`）按间隔 HTTP 调用业务服务的**一次性** internal API；Kafka 消费者按消息唤醒。业务服务只暴露「做一次」的入口。

## 核心原则

**任意业务服务都不得把自己设置成轮询 / 循环模式。服务运行的触发点只能是：**

| 触发类型 | 谁发起 | 业务服务看到什么 |
|----------|--------|------------------|
| **专门的定时服务** | `taskEvents` timer worker（`transport: timer`，独立 runAll 条目） | 一次 HTTP/RPC 调用（internal API） |
| **外界触发** | 用户/网关 HTTP、Kafka 领域事件、Webhook、显式 CLI、9999 运维按钮 | 一次请求或一条消息 |

禁止业务进程用内部时钟自己叫醒自己做周期工作。

## 强制要求

### 1. 业务 HTTP/RPC 进程

- `main` / 启动路径 **禁止** `go startXxxLoop()`、`time.NewTicker`、`time.Tick`、`for { time.Sleep; 扫表/对账/过期/回收 }`
- 周期工作拆成：**一次性 handler**（`ExpireOnce` / `RecycleIdleOnce`）+ **独立 timer 进程** 调用它
- 新增周期能力时：先在业务服务加 internal 一次性 API，再在 `taskEvents/internal/handlers/<domain>/` 增加 timer worker，并注册 `conf/runAll.yaml`

### 2. 专门的定时服务（允许 ticker 的唯一业务落点）

- 目录：`taskEvents/internal/handlers/` 下 `transport: timer` 的 runner
- 独立进程、独立 health、`OTEL_SERVICE_NAME` 与 runAll `name` 对齐
- timer **只负责叫醒**；领域逻辑仍在表 owner 服务的一次性 API 内（符合单库单表所有权）
- 参考：`taskpostexpiryscan`、`workspacemachineidle`

### 3. 外界触发（无 ticker）

允许且推荐：

- 公网 / 鉴权 HTTP、internal HTTP
- Kafka 消费者（落在 `taskEvents` intent worker，**不是**业务服务内嵌 consumer 死循环扫）
- Webhook、SSE 入站事件
- 显式一次性 CLI（`migrate`、bootstrap、运维脚本）
- 前端：**用户操作**或 **SSE/WebSocket 推送** 后再拉一次

### 4. 前端 SPA（同源约束）

- **禁止**无用户操作、无推送事件时的后台 API 轮询（如 5s/30s `setInterval` 打云厂商 Describe / runtime-status）
- 允许：本地倒计时（短信验证码秒数，不打 API）；用户正在等待的显式流程（支付结果，须有超时与停止条件）；SSE 断连后的有限次补偿（须绑定 `comment_id` 等分片键，禁止盲扫）
- 与 `docs/intents/frontend/comment_runtime_no_background_poll.intent.md` 一致

### 5. 明确允许的循环（不是「业务轮询」）

| 允许 | 原因 |
|------|------|
| HTTP `ListenAndServe` / accept loop | 等待外界连接 |
| Kafka client Fetch / consumer group 协议循环 | 等待 broker 推消息；实现须在 `taskEvents` |
| `taskEvents/broker` 连接恢复 ticker | 协议层，不是扫业务表 |
| `runAll` 进程监护 / 编译进度 | 编排器基础设施 |
| 测试文件 `*_test.go` | 夹具 |
| 第三方 `third_party/` `vendor/` `gitService/gitlab-ce/` | 不可改上游 |

**禁止**用「健康检查也要轮询」当借口在业务服务里扫表。探活由 runAll / 网关对 `/api/health/` 发起，仍是外界触发。

### 6. 新增周期工作的落地清单

1. 业务服务：幂等的一次性 internal API（带 `X-Internal-Secret` / 既有内部鉴权）
2. `taskEvents`：timer handler 调该 API；间隔走 conf / env（SSOT 在对应 `conf/<area>/<app>/`）
3. `conf/runAll.yaml`：独立 service 条目 + health
4. **禁止**在 `taskBill` / `taskCloudService` / `taskTaskService` 等 owner 服务 `main` 里 `go ticker`

## 存量

门禁对下列文件 **allowlist**（`LEGACY_INTERNAL_TICKERS`），**禁止再复制**。迁出计划与分项 TODO：

- 计划：`docs/superpowers/plans/2026-08-16-legacy-ticker-migration-plan.md`
- 总表：`OPT-20260816-022`；分项：`OPT-20260816-025`～`032`

| 文件 | 分项 |
|------|------|
| `taskCloudService/src/workspace_machine_recycle.go` | 025（已有 18044 timer，只删重复 ticker） |
| `taskBill/src/main.go` | 026 |
| `taskCloudService/src/server_orphan_reconcile.go` | 027 |
| `taskCloudService/src/server_release_reconcile.go` | 028 |
| `taskBill/src/wechat_profit_sharing.go` | 029 |
| `taskTaskService/src/queued_schedule_dispatch.go` | 030 |
| `taskReferral/src/referral_code.go` | 031 |
| `go_relayToTrae/src/push.go` | 032（改为状态变化立刻 push，不新建 timer） |

新文件出现 `time.NewTicker` / `time.Tick` 且不在允许路径 → CI **阻断**。业务服务 **不得** 用注释自我豁免。

## 验收

```bash
python3 db/scripts/ci/test_check_no_service_internal_poll_loop.py
python3 db/scripts/ci/check_no_service_internal_poll_loop.py
```

## 与其他规则的关系

| 规则 | 关系 |
|------|------|
| 意图 → MQ 事件（`.ai/08`） | 外界触发的主路径是领域事件，不是进程内 sleep |
| 第 22 条 Go 服务优先 | 新建 Go 服务同样禁止内嵌 ticker |
| `33_new_service_runall_registration.md` | timer worker 必须单独注册，不能塞进已有 HTTP 进程 |
| 第 47 条 health 端口 SSOT | 新 timer/intent 的 runAll 探活端口必须等于 domain-events YAML |
| 单库单表所有权 | timer 只调用 owner 的 API，不直连他服务的表 |
| 事件消费者幂等（元规则 49） | timer 调用的 owner API 仍须 HTTP 幂等；Kafka 消费走共享 runner |

## 变更日志

- 2026-08-16：1.0.0 — 首版；与 ADR-0011 同步。
