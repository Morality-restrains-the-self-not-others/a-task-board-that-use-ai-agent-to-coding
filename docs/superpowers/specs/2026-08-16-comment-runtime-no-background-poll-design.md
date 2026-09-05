# 评论运行态：容器推送为真源，禁止前后端轮询

- **日期**: 2026-08-16
- **作者**: cursor
- **迭代**: comment-runtime-push-sync
- **状态**: accepted（goal-mode 已交付；架构 v84 current）
- **意图**: `docs/intents/frontend/comment_runtime_no_background_poll.intent.md`
- **取代**: `OPT-20260816-018`；修订 `docs/superpowers/specs/2026-08-16-comment-runtime-no-background-poll-design.md` 同文件
- **python_api_approval**: n/a（零新增 Python 接口；GET `server-runtime-status` 仅供同步按钮）
- **架构**: 批准后写 **v84** `application-integration`（直播 Rel_Flow：容器/SSE 推送；Describe 降为按需按钮）。v82/v83 target 积压，与本迭代正交。

---

## 1. 用户纠正（本版要锁死的原则）

上一问已否决「把轮询扩到全部 comment_id」。本次进一步纠正：

> 最合适的方式是**容器主动推送**，前端**只接收**服务端下发的信息来同步，并提供**同步按钮**做主动同步。采用前端轮询仍可能导致**前端卡死**。**前端和服务端都不适合轮询**。

**结论先行**：直播通道只有推送。拉通道只有用户点「刷新状态」。禁止用 `setInterval` / 服务端 ticker 维持 UI 与云状态一致；也禁止用「SSE 到了再自动 GET Describe」当补偿（那是把轮询藏进事件回调，主线程同样会被并发 Describe 打满）。

---

## 2. 对当前架构的理解

根据 `docs/architecture/`：

- current 视图：`enterprise-landscape` v81、`application-integration` v81
- 业务层：任务协作、云资源
- 应用层：`taskFE`；`taskCloudService`（CSC / 可选 Describe）；`taskSSE`；容器 heartbeat
- 积压 target：v82 终态释放评论 CSC、v83 评论级仓库身份

📋 架构版本历史（节选）：

- v81 (2026-08-14) ✅ current — 工作空间任务帖人读序号
- v82 / v83 🎯 target — 释放 CSC / 评论级身份（积压，与本次正交）
- 本次批准后 → **v84** 标注直播数据流从轮询改为推送

---

## 3. 🕸️ Code Review Graph 分析

- **CRG**: `.code-review-graph/graph.db` 存在，索引面仅 17 文件。本次以 Grep 为准。
- **CRG unavailable for impact**: 未覆盖 `useServerConfigRuntime` / `useBindingAdvancePolling` / `serverStartupStatusPoll`。

残留直播轮询（均须按本原则处理）：

| 位置 | 间隔 | 打什么 | 本版处置 |
|------|------|--------|----------|
| `createRuntimeStatusPollController` / `startingFallbackPollId` | 5s / 30s | GET `server-runtime-status` → 云 Describe | **删除**（WIP 已动手） |
| `ServerConfigRuntimeStatusSection` `watch(commentId) immediate` | 每条评论挂载 | 自动 Describe | **删除**；只留按钮 |
| SSE 成功文案 → `watchStatusMessageForRefresh` → GET Describe | 每次成功/停止 | 自动 Describe | **删除**；只应用推送 payload |
| `serverStartupStatusPoll` | 5s/15s | GET `server-startup-status` | **删除**作为 UI 同步；SSE 断连只提示重连 + 按钮 |
| `useBindingAdvancePolling` 30s timer | 30s | POST advance | **删除定时器**；只保留 SSE `binding_advanced` / `container_heartbeat` 驱动的一次 advance |
| `runtimeUptimeTick` 30s | 30s | 无网络，只刷新「已运行时长」文案 | **保留**（非 I/O） |
| `taskCloudService` 启动命令内 `pollInstancePublicIP` | 命令路径有界重试 | 云 Describe 等公网 IP | **不作为 UI 同步**；IP 就绪后 **push SSE**，不让前端空转等 |
| 泄漏实例 reconcile ticker | 运维 | 与任务详情 UI 无关 | 本期不改 |

---

## 4. 为什么轮询会卡死 / 泄漏

- **前端**：`setInterval` + 异步 GET 重叠（`usePaymentPoll` 注释已写明此坑）；多评论展开后 N 路 Describe；JSON 大包解析堵主线程；`isServerStarting` 卡住则定时器永不退出。
- **服务端若轮询云再转推**：把限流和费用搬到 taskCloudService，页面开着就持续 Describe，仍是泄漏，且用户没点同步。
- **OPT-018「扫全部 live binding」**：泄漏 × N，且逻辑错时不会自停。

正确模型：**状态变化的那一端在变化当下 push**（容器 heartbeat、启动工作流在 Aliyun API 返回时发 SSE），前端 apply；对账只走按钮。

---

## 5. 选定方案：Push 直播 + 按钮对账

```
容器 / 启动工作流 --SSE(comment_id + 状态字段)--> taskSSE --> taskFE 写入该评论 snapshot
用户点「刷新状态」 --一次 GET server-runtime-status?comment_id=--> taskCloudService Describe --> 覆盖该评论 snapshot
```

### 允许

| 通道 | 行为 |
|------|------|
| 容器 `container_heartbeat` + `comment_id` | 前端只 apply，不跟 GET |
| 启动/停止 SSE（须带 `comment_id`，成功时带 `runtime_status` 等展示字段） | 前端只 apply |
| `comment_container_binding_advanced` | 前端只 apply；必要时 **事件驱动** 调一次 advance（不是 timer） |
| 「刷新状态」按钮 | **唯一** Describe；必须带该面板 `comment_id` |
| stop-vm / start-vm **HTTP 响应体** | 命令回包里已有的状态可 apply；后续仍靠 push，不启动 timer |

### 禁止

| 禁止 | 原因 |
|------|------|
| 任何 `setInterval` 打 runtime-status / startup-status / advance | 卡死 + 泄漏 |
| 服务端 ticker Describe 再 SSE 刷 UI | 把轮询换到服务端，仍泄漏 |
| 挂载 Tab / 切 commentId 自动 Describe | 展开两条评论 = 两路立即打云 |
| SSE 到达后再自动 GET Describe | 隐藏轮询，并发同样卡死 |
| `scopedComment.forPoll()` 当直播 id | 刷错评论 |

### 推送契约缺口（实施时补，不靠拉来补）

启动成功 SSE 今天主要是文案 `aliyun服务器启动成功！`，前端再 GET Describe。本版改为发布点写入 `comment_id` + `runtime_status`（及已有 instance 字段），前端 `updateServerStatus` apply 到该评论 snapshot。字段不足则修发布点，不恢复自动 GET。

### 拒绝的方案

| 方案 | 拒绝原因 |
|------|----------|
| OPT-018 全量 live 轮询 | 泄漏 × N |
| 只刷可见 Tab 的短轮询 | 忘关 Tab 仍卡死 |
| SSE 断连时 REST 轮询启动进度 | 仍是前端轮询；改为提示 SSE 重连 + 按钮 |
| 服务端 5s Describe 再推前端 | 服务端轮询，用户未授权对账 |

---

## 6. 成功标准

| # | 标准 |
|---|------|
| S1 | Initializing / `isServerStarting` / 多评论展开时，**零**周期性 `server-runtime-status` 与 `server-startup-status` |
| S2 | 只有点「刷新状态」才出现 Describe；path 含该评论 `comment_id` |
| S3 | 心跳 / 启动成功 SSE 到达后，该评论面板更新且 **不** 自动跟 GET Describe |
| S4 | 无 `comment_id` 的成功文案：不 Describe，打 warn，不回落 `forPoll()` |
| S5 | `OPT-20260816-018` 取消 |
| S6 | 不新增 HTTP/Python 接口 |

---

## 7. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者 | 例外理由 |
|---------|----------------|--------|--------|----------|
| 容器心跳 | 现有 `container_heartbeat` | 容器 → taskSSE | taskFE apply 该 comment snapshot | 不新增 MQ |
| 实例启动/停止完成 | 现有启动 SSE（补 `runtime_status`） | taskCloudService → taskSSE | taskFE apply | 不新增 MQ；禁止随后自动 Describe |
| 用户点刷新状态 | — | taskFE 按钮 | GET server-runtime-status | 纯查询，一次性 |
| 删除前后端 UI 轮询 | — | — | — | 纯定时器删除 |

---

## 8. 价值流影响

- 触及评论级 start-vm / `cloud_comment_container_bindings`：直播从「前端盲拉 Describe」改为「SSE apply + 按钮对账」。
- 不改表字段；不新增 stream。切片由 `/4-value-stream` 处理。

---

## 9. 🏛️ 架构变更影响

- **迭代版本**: v84 🎯 target
- **迭代名称**: comment-runtime-push-sync
- **作者**: cursor
- **设计日期**: 2026-08-16 13:28
- **新增文件**（每个视图四类伴生，缺一不可）:
  - 🆕 `docs/architecture/v84-enterprise-landscape-20260816-1328-cursor.puml`
  - 🆕 `docs/architecture/v84-application-integration-20260816-1328-cursor.puml`
  - 🆕 `docs/architecture/v84-enterprise-landscape-20260816-1328-cursor.diff.archimate`（增量变迁：v81→v84）
  - 🆕 `docs/architecture/v84-application-integration-20260816-1328-cursor.diff.archimate`
  - 🆕 `docs/architecture/v84-enterprise-landscape-20260816-1328-cursor.full.archimate`（全量拓扑）
  - 🆕 `docs/architecture/v84-application-integration-20260816-1328-cursor.full.archimate`
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**: `docs/architecture/v81-*-*.puml` (current)
- **变更明细**: 🟢 评论容器 heartbeat 直播 / 🟡 taskFE 只 apply + 按钮 Describe、taskCloud SSE 带 runtime_status / 🔴 FE/BE UI 轮询

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v81 → Gap（UI 轮询）→ WP-comment-runtime-push-sync → Plateau v84；目标拓扑：容器/Cloud → SSE → FE，Describe 仅按钮 |
| **`.full.archimate`** | 变迁后直播+对账拓扑，含 `[DEPRECATED v84]` UI poll |

> 老文件未被修改。目标架构将在 `/10-ship` 时切换为 current。

---

## 10. 与已开工代码的关系

已落地：删除 runtime/startup/binding UI 定时器；挂载与 SSE 不再自动 Describe；启动成功 SSE 带 `runtime_status`；停止 SSE 走 `publishTaskSSE` 注入 `comment_id`；`OPT-20260816-018` 已取消。
