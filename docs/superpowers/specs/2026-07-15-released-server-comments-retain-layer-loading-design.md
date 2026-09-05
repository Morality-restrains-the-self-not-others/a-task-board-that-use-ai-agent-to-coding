# 设计：服务器已释放后保留 AI 评论展示，并消除可写层「连接确认中」假加载

- **日期**: 2026-07-15
- **作者**: claude
- **状态**: approved（goal-mode 自动采纳，跳过确认门）
- **迭代名**: `released-server-comments-retain-layer-loading`
- **相关页面**: `https://www.daydaymoney.com/tenant/{t}/workspace/{w}/task-detail/{taskId}/`
- **复现任务示例**: `task_13033995991120014872`（tenant `850256677331562496` / workspace `861623708318031872`）
- **架构版本**: **不更新**（纯前端状态机 / UX；无组件增删、无数据所有权变更）
- **python_api_approval**: n/a（**零新增** Python/Django HTTP 接口；不新增 Go 公网接口）
- **前置**: v27 终态硬释放；v26 统一评论区 / `container_agent_comments`；intent `006_container_heartbeat_log_and_bidirectional`

---

## 1. 问题 / 意图

### 现象

任务详情页评论区上方「任务关联（可写层串行 · 容器推送）」长期显示：

> 容器连接确认中，即将拉取可写层…

页面上评论内容可能已存在，但层图占位造成「整页卡住 / 未就绪」的观感。

### 用户判定（本迭代采纳）

| 判定 | 说明 |
|------|------|
| 根因场景 | **服务器/容器已释放**（终态硬释放或手动/自动停机后），前端仍停在 `containerHeartbeatStatus === 'connecting'` |
| 产品要求 | **AI 评论（含容器 Agent 评论）必须依旧保留并可展示**；不得因机器释放而消失或被加载态掩盖预期 |

### 与代码对齐的机制根因

文案来自 `useTaskDetail.js` → `commentLayerZtreeLoadingHint`：当层图为空且 `heartbeat === 'connecting'` 时渲染。

| 缺口 | 说明 |
|------|------|
| G1 | `showCommentLayerZtreeLoading` **不感知** `isServerRunning` / `runtime_status ∈ {released,stopped,…}` |
| G2 | SSE `stopped` / `error` 仅 `stopContainerHeartbeat()`（→ `idle`），**未**走 `pauseContainerHeartbeatForRelayStop`；晚到的 `container_heartbeat` 可再次把状态打回 `connecting` |
| G3 | **冷打开**已释放任务时：`server-runtime-status` 已拒绝拉层图，但 UI 心跳/loading 未同步进入「已释放」空态 |
| G4 | 释放且无层图时，若 loading=false，层图区域目前可能**完全空白**，缺少「服务器已释放」说明 |

### 非问题（已确认）

- AI / 容器 Agent 评论**已落库**（`taskAIComment`：`ai_task_comments` / `container_agent_comments`），读路径不依赖活容器。
- `#layer-association` 与 `#comments-container` 为兄弟槽位；loading **不会 DOM 级隐藏** Feed。本迭代修的是**假连接态 + 空态文案**，并写明评论保留契约。

---

## 2. 成功标准（SMART）

| # | 标准 | 可验证方式 |
|---|------|------------|
| S1 | 服务器非服务态（`!isServerRunning && !isServerStarting`，或 runtime `released/stopped/…`）时，**不出现**「容器连接确认中，即将拉取可写层…」 | 单测 + Playwright |
| S2 | 同上条件下，层图区展示**静态空态**（见 §4 文案），而非无限 spinner | UI 断言 `data-testid` |
| S3 | 任务详情刷新后，历史 `comments` + `ai_comments` + `container_agent_comments` **全部仍出现在 Feed** | 单测 `buildDisplayComments` + 详情页断言 |
| S4 | SSE `stopped`/`error` 与「停止虚拟机成功」对称：pause 心跳、清 endpoint、忽略后续 heartbeat 直至再次启动 | `updateServerStatus` 单测 |
| S5 | 冷打开已释放任务：进入详情即空态，不先闪 connecting 再卡住 | 单测 / E2E |
| S6 | 服务器运行中且双向未达成时，connecting 文案**仍可用**（不误伤正常启动窗口） | 回归现有 heartbeat 测试 |
| S7 | 零新增 Python/Go 公网 API；无需架构 target 文件 | 代码审查 |

---

## 3. 方案对比

| 方案 | 描述 | 结论 |
|------|------|------|
| **A. 前端「非服务态」门禁 + 停机路径对称 pause + 释放空态** | 扩展 loading 条件；对齐 stop SSE；runtime/UI context 驱动 pause；评论契约写死 | **采纳** |
| B. 后端释放时广播专用 SSE `container_released` 再驱动 UI | 有用但非必要；现有 `stopped` / runtime-status 足够 | 拒（本期） |
| C. 释放后仍轮询心跳直到超时再提示 | 浪费请求；与「机器已释放」事实冲突 | 拒 |
| D. 隐藏整个评论区直至层图就绪 | 与「评论必须展示」直接冲突 | 拒 |

### 自主决策

1. **评论保留**：只读展示路径不变；本迭代**禁止**任何「释放时清空评论缓存 / 跳过 fetch 评论」的改动。
2. **层图与评论解耦**：层图不可用 ≠ 评论不可用；空态文案须明示「历史评论仍可查看」。
3. **停机对称**：凡进入非服务态，一律等价于 `pauseContainerHeartbeatForRelayStop`（或抽共享 `enterServerNotServingUiState`）。
4. **正常启动窗口**：仅在 `isServerRunning \|\| isServerStarting`（或 runtime transitional）时允许 connecting loading。

---

## 4. UX 约定

### 4.1 可写层区域状态机（简）

```
[无层图]
  ├─ 启动中 / 运行中 + connecting/refreshing → spinner（现有文案）
  ├─ 运行中 + connected + 等待层 → 「等待可写层就绪…」（现有）
  └─ 非服务态（已释放/已停止）→ 静态空态（本期）
[有层图] → 正常 zTree（释放后若本地快照被 pause 清空，则走空态）
```

### 4.2 空态文案（建议）

> **服务器已释放，可写层暂不可用**  
> 历史评论（含 AI / 容器 Agent）仍可在下方查看。重新启动服务器后将恢复层图。

`data-testid`: `comment-layer-ztree-released`（或等价）

### 4.3 容器连接状态面板

非服务态时：心跳点应为灰/空闲或「已断开」，**不应**长期黄点「连接中 / 单向连接中」。与 pause 后 `idle` 一致。

---

## 5. 实现要点（前端）

| # | 位置 | 改动 |
|---|------|------|
| 1 | `useTaskDetail.js` `showCommentLayerZtreeLoading` | 非服务态 → `false` |
| 2 | `TaskDetail.vue` / CommentsPanel slot | 非服务态 + 无层图 → 静态空态块 |
| 3 | `updateServerStatus.js` `stopped`/`error` | 调用 `pauseContainerHeartbeatForRelayStop`（与「停止虚拟机成功」对齐） |
| 4 | `establishSSEConnection.js` | `paused` **或** 非服务态时忽略 `container_heartbeat`（防回写 connecting） |
| 5 | `fetchContainerTaskUiContext` / runtime gate | 冷打开：`runtime_status` ∈ notServing → 主动 enter not-serving UI |
| 6 | endpoint `watch` | 非服务态时**禁止** idle→connecting 强制提升 |

**不改**：评论 fetch/merge（`buildDisplayComments`）、`taskAIComment` 存储、硬释放后端编排（v27）。

---

## 6. 领域概念（轻量，供后续 /5-ddd 若需要）

| 概念 | 说明 |
|------|------|
| **ServerNotServingUiState** | 前端聚合：机器不可服务时的 UI 态（pause 心跳、层图空态） |
| **RetainedTaskComments** | 任务级持久评论（人/AI/容器 Agent），生命周期独立于 CSC/容器 |
| **WritableLayerAssociation** | 依赖活容器的层图视图；非服务态仅展示空态 |

### 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 打开已释放任务详情并展示历史评论 | — | — | — | **纯查询/只读**；评论已在既有写入意图落库 |
| 前端进入「服务器非服务」UI 态 | — | — | — | **纯前端状态机**；不改变服务端事实 |
| （既有）终态硬释放机器 | `CLOUD_SERVER_STOPPED` 等 | taskEvents / Cloud | 停机清绑定 | 已由 v27 覆盖；本期不新增 |

---

## 7. 价值流影响（输入给 `/3-value-stream`）

| 问题 | 评估 |
|------|------|
| 影响流 | 任务协作 / 任务详情查看与评论阅读（`conf/value-stream.yaml` 内 task-detail 相关步） |
| 新流？ | 否 |
| 字段 | 无新 `<service>.<table>.<field>`；评论表只读 |
| 测试 | 新增前端单测 + 可选 Playwright：已释放任务详情评论可见、层图非 connecting |
| 跨流依赖 | 依赖 v27 释放事实；本迭代只消费 runtime/SSE，不改编排 |

---

## 8. 🏛️ 架构变更影响

- **结论**: **不创建**新版本 `docs/architecture/` target 文件。
- **理由**: 无 Application_Component / Rel_Flow / 数据所有权变更；属前端状态机与空态 UX。
- **关联已有**: v27 硬释放（后端事实）；v26 统一评论区（评论模型，仍为 target 时亦不被本期推翻）。

---

## 9. 测试计划（实现阶段）

1. 单测：`showCommentLayerZtreeLoading` — `isServerRunning=false` + `hb=connecting` → false  
2. 单测：`updateServerStatus` — `status=stopped` → paused + endpoint false + hb idle  
3. 单测：非服务态下 SSE `container_heartbeat` 不改变状态  
4. 单测：冷打开 runtime=`released` → 空态，无 connecting hint  
5. 回归：运行中双向未达成仍显示 connecting 系列文案  
6. Playwright（可选）：mock runtime released + 预制 `ai_comments` → Feed 可见且无 `comment-layer-ztree-loading` connecting 文案  

---

## 10. 非目标

- 不恢复已释放机器上的可写层内容（无快照回放）
- 不在释放后继续生成新的容器 Agent 流式回复（仍须重新启动）
- 不改评论存储引擎（见 `2026-07-15-task-comments-sql-vs-nosql-scale-design.md`）
- 不改 v27 migrate/hard-release 编排

---

## 11. 变更记录

| 日期 | 变更 |
|------|------|
| 2026-07-15 | 初版 draft：释放后评论保留 + 消除可写层假 connecting |
