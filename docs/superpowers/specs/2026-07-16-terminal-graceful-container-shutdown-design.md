# 设计：任务终态优雅通知容器收尾 → 容器回调释放机器节点

- **日期**: 2026-07-16
- **作者**: claude
- **状态**: approved（goal-mode 自动采纳，跳过确认门）
- **迭代名**: `terminal-graceful-container-shutdown`
- **相关页面**: `https://www.daydaymoney.com/tenant/{t}/workspace/{w}/task-detail/{taskId}/`
- **架构版本**: v33 🎯 target（基于 v32 ✅ current）
- **python_api_approval**: n/a（**零新增** Python/Django HTTP 接口；全部落 Go / onlineServiceJS）
- **前置**: v17 终态释放；v27 硬释放 + 兄弟迁机

---

## 1. 问题 / 意图

用户在任务详情将进度切换为「已取消」或「已完成」时，期望：

1. **把状态变化发送给正在运行该任务的容器镜像**（优雅指令，而非直接杀机）
2. 容器收到后做**收尾**（中断 job、上报 layer-graph、落盘等）
3. 容器**通知 SaaS API** 关闭对应运行机器节点，然后**结束自身**
4. SaaS 判断：该机器节点**仅运行本容器**（或**已无任何容器**）时才**释放**机器节点

### 与现有实现的差距

| 现状（v17/v27） | 缺口 |
|-----------------|------|
| `TASK_STATUS_CHANGED` → taskEvents **直接** stop relay/mock + `CLOUD_SERVER_STOPPED` | 容器无「终态收尾」窗口；进程被外力终止 |
| 兄弟 busy 容器先 migrate 再硬释放 | 符合「多容器不误杀」；但终态任务自身无优雅通知 |
| 无容器 → SaaS 的「请释放我所属机器」inbound | 释放决策全在事件消费侧，非容器主动 |

### 与用户表述对齐

| 短语 | 本设计解释 |
|------|------------|
| 把状态变化发给容器镜像 | SaaS 经 `server_url` 调容器 `POST /api/task-lifecycle/shutdown`，body 含 `terminal_kind` |
| 收尾并通知 SaaS 关闭机器 | 容器 interrupt jobs → layer-graph-push → `POST …/server-container-token/request-machine-release/` |
| 结束容器自身 | 回调成功后 `process.exit` / 通知侧车 stop（best-effort） |
| 仅本容器或无容器时释放 | `request-machine-release` 查同 `instance_id` busy 绑定；sole/empty → 发 `CLOUD_SERVER_STOPPED` / local stop；否则仅清本任务 CSC |

---

## 2. 成功标准（SMART）

| # | 标准 | 可验证方式 |
|---|------|------------|
| S1 | 进度→已完成/已取消 且容器可达 → 容器收到 shutdown（含 terminal_kind） | Go/Node 单测 + HTTP mock |
| S2 | 容器收尾后调用 `request-machine-release`，sole/empty 时机器被释放 | 单测：bindings=1 → publish stop |
| S3 | 同机仍有其他 busy 容器 → **不**销毁实例，仅解绑终态任务 CSC | 单测：bindings>1 → no CLOUD_SERVER_STOPPED |
| S4 | 容器不可达 / 超时未回调 → 补偿硬释放（复用 v27 路径） | 单测：notify fail 或 deadline 过期 |
| S5 | 幂等：重复 shutdown / 重复 request-release 不重复破坏 | 单测 |
| S6 | 零新增 Python HTTP；Swagger 登记新 Go/L0 路由 | 路由归属 / OpenAPI |
| S7 | 意图文档 + 价值流测试点 | intents / value-stream |

---

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 结论 |
|------|------|------|
| **A. 优雅通知 + 容器回调释放 + 超时补偿硬释放** | 主路径符合用户意图；补偿保证最终释放 | **采纳** |
| B. 仅延长 v27 硬停前 sleep | 容器可能来不及收尾；非容器主动 | 拒 |
| C. 仅依赖容器轮询 task-detail | 延迟大、无强制；离线漏释 | 拒作主路径（可作辅） |
| D. 前端改状态后同步调 interrupt+stop | 多入口漏触发 | 拒 |

### 自主决策

1. **出站通知**：taskEvents 经 CSC `server_url`（或 business_api_endpoint）直连容器 `POST /api/task-lifecycle/shutdown`；HTTP 接受超时 **5s**；不经浏览器 session。
2. **L0 登记**：`taskContainerGateway` 增 `container-task-lifecycle-shutdown`（浏览器/服务凭据均可走网关；事件侧优先直连 `server_url` 减跳）。
3. **释放主权**：容器回调 `request-machine-release` 为**首选释放触发**；taskEvents 在成功投递 shutdown 后**不再立即** hard-stop，改为标记 `graceful_shutdown_pending` + 发布延迟补偿事件。
4. **补偿**：新事件 `TASK_TERMINAL_SHUTDOWN_TIMEOUT`（或同 topic 带 `phase=timeout`）；deadline 默认 **90s**；到期若 CSC 仍有 `instance_id`/`server_url` 且 `terminal_released` 未完成 → 走 v27 硬释放。
5. **兄弟迁机**：仍在优雅通知**之前**完成（v27 不变），避免「sole」误判。
6. **不可达**：notify 失败（连接拒绝/超时）→ **立即**走 v27 硬释放（无优雅窗口）。

---

## 4. 领域概念（供 `/6-ddd`）

| 概念 | Bounded Context | 说明 |
|------|-----------------|------|
| **TerminalGracefulShutdown** | Cloud Runtime / Container | 终态后先通知容器收尾 |
| **MachineReleaseRequest** | Cloud Runtime | 容器请求释放所属机器 |
| **SoleContainerGate** | Cloud Runtime | 仅本容器或无 busy 容器才销毁节点 |
| **ShutdownTimeoutCompensation** | Domain Events | 超时未回调则硬释放 |
| **TerminalKind** | Task | completed \| cancelled（复用 v17） |

### 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 |
|----------|--------|--------|---------------|
| 任务进度变为终态 | `TASK_STATUS_CHANGED`（已有） | taskTaskService | taskEvents：迁机 → 优雅通知或硬释放 |
| 已投递优雅 shutdown | `TASK_GRACEFUL_SHUTDOWN_REQUESTED`（可选日志事件；MVP 可内联 CSC 标记） | taskEvents | 观测 / 补偿调度 |
| 优雅窗口到期 | `TASK_TERMINAL_SHUTDOWN_TIMEOUT` | taskEvents（notify 成功后发布，带 `deadline_at`） | taskEvents：若未释放则硬释放 |
| 容器请求释放机器 | `CONTAINER_MACHINE_RELEASE_REQUESTED` | taskCloudService（inbound 成功后） | 可内联处理；可选下游 SSE |
| 机器节点停止 | `CLOUD_SERVER_STOPPED`（已有） | Cloud / Events | 既有删实例 |

MVP：**必须**投递 `TASK_STATUS_CHANGED`（已有）与 `TASK_TERMINAL_SHUTDOWN_TIMEOUT`（notify 成功路径）；`CONTAINER_MACHINE_RELEASE_REQUESTED` 可内联处理并记结构化日志，若需跨服务观测再升 Kafka。

---

## 5. 详细设计

### 5.1 序列（主路径）

```text
Vue PATCH progress → taskTaskService → TASK_STATUS_CHANGED
  → taskEvents:
       set terminal_released flag
       migrate foreign busy (v27)
       clear idle siblings
       POST {server_url}/api/task-lifecycle/shutdown {terminal_kind}
       if notify OK:
         mark graceful_shutdown_pending=1
         publish TASK_TERMINAL_SHUTDOWN_TIMEOUT (delay 90s)
         return success  // 不立刻 stop
       else:
         hard-release (v27 local stop + CLOUD_SERVER_STOPPED + mark)
  → onlineServiceJS:
       interrupt all running/pending jobs
       layer-graph-push
       POST …/request-machine-release/ {terminal_kind, reason}
       exit / stop self
  → taskCloudService request-machine-release:
       auth container token
       list bindings on instance_id
       if sole or empty busy:
         local stop if needed + CLOUD_SERVER_STOPPED + mark terminal_released
       else:
         clear this task CSC only (不解绑他人)
  → timeout consumer (if still pending):
       hard-release v27
```

### 5.2 onlineServiceJS

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/task-lifecycle/shutdown` | body: `{terminal_kind, reason?, task_id?}`；异步收尾；立即 202 |

收尾步骤（`runTerminalShutdown`）：

1. 幂等锁（内存 flag `shutdownInFlight`）
2. 对所有 running/pending job 调 `interruptJob`
3. 触发既有 layer-graph-push
4. 调用 SaaS `request-machine-release`（access token）
5. `setTimeout` 短延迟后 `process.exit(0)`（或写 stop 信号文件供侧车）

### 5.3 taskCloudService — inbound

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `…/server-container-token/request-machine-release/` | 容器令牌；body: `access_token`, `terminal_kind`, `reason?` |

逻辑：

1. 校验令牌与 task 边界
2. 加载 CSC；若已 `terminal_released=1` 且无 instance → 200 no-op
3. `ListBusyBindings(instance_id)` 排除已清零 server_url 的行
4. **SoleContainerGate**：busy 数 ≤ 1（仅本 task）或 0 → 释放节点
5. 释放：复用 clear-after-stop / publish `CLOUD_SERVER_STOPPED` / mark-terminal-released
6. 非 sole：仅 clear 本任务 CSC + mark terminal_released，**不** publish stop

路由归属：经 taskAgentSupport 入站（与 heartbeat 同族）；实现落 taskCloudService（或 credential 转发 — **采纳**与 heartbeat 同路径：TAS → Cloud）。

### 5.4 taskEvents 变更

扩展 `taskstatuschanged.Handler`：

1. 注入 `ContainerNotifier`：`NotifyShutdown(ctx, serverURL, terminalKind, taskID) error`
2. notify 成功分支：不调用 `StopLocal` / 不发 `CLOUD_SERVER_STOPPED`；发 timeout 事件
3. notify 失败 / 无 server_url 但有 instance_id：保留 v27 硬释放
4. 无 server_url 且无 instance：no-op（已有）

新 handler：`taskterminalshutdowntimeout.Handler` — 若 CSC 仍 busy/绑定 → hard-release。

### 5.5 taskContainerGateway

L0 增补：

```text
container-task-lifecycle-shutdown → POST {scopedAPI}/task-lifecycle/shutdown
```

供运维/前端诊断；事件主路径可直连。

### 5.6 CSC 字段（可选）

| 列 | 说明 |
|----|------|
| `graceful_shutdown_pending` INTEGER DEFAULT 0 | notify 成功置 1；释放完成清 0 |
| `graceful_shutdown_deadline_at` TEXT | ISO 截止时间 |

MVP 可用现有字段 + timeout 事件 payload 携带 deadline，避免急迁表；若单测需可观测状态则加列。

**采纳**：CSC 增 `graceful_shutdown_pending`（INTEGER NOT NULL DEFAULT 0），与 `terminal_released` 并列。

### 5.7 权限与安全

- 出站 shutdown：服务间 → 容器内网；无用户 cookie
- inbound release：仅有效容器 access_token；强制 path 三段 ID 与令牌一致
- 禁止跨任务释放

### 5.8 Python 新增接口

**不新增** → 不触发 Python 专项审批。

### 5.9 前端

**不改**进度 PATCH；可选后续：shutdown 进行中 SSE 提示（非本期必做）。

---

## 6. 非目标

- 热迁移跨主机保进程内存
- 改进度列命名体系 / `is_terminal` 元数据（仍用 v17 列名约定）
- 修复 `/switch` 不发事件（可顺手修，非本迭代核心）
- 多云非阿里云 stop-vm 501

---

## 7. 架构变更影响

| 视图 | 变更 |
|------|------|
| application-integration v33 | 新增：Events→OSJS graceful shutdown；OSJS→Cloud request-machine-release；Timeout 补偿 |
| 伴生 | `.puml` + `.archimate` + `.mermaid.md` |

---

## 8. 测试计划（摘要）

1. Go：Notifier mock → pending；timeout → hard release
2. Go：request-machine-release sole vs multi
3. Node：shutdown interrupt + 调用 release mock
4. 可选 Playwright：终态后 runtime Released（relay 环境）

---

## 9. 意图文档路径

- `task2app/docs/intents/frontend/task_detail/024_terminal_graceful_container_shutdown.intent.md`
- `task2app/docs/intents/frontend/task_detail/024_terminal_graceful_container_shutdown.testintent.md`
