# 设计：评论级「执行细节」与容器连接状态归属

- **日期**: 2026-07-22
- **状态**: approved（goal-mode 锁定方案）
- **迭代名**: `comment-execution-details`
- **作者**: claude
- **架构**: v51 🎯 target（基于 v48 current）
- **python_api_approval**: n/a（零新增 Python/Go HTTP；纯前端重组 + 展示契约）

---

## 1. 问题

任务详情页当前存在两类与「执行/容器」相关的 UI，挂载位置与用户心智不一致：

| 现状 | 位置 | 问题 |
|------|------|------|
| 容器连接状态（心跳、seq/ack、探测日志） | `TaskDetailServerStartStatusPanel` 内，与服务器启动/SSE 混排 | Runtime 区信息过载；难与触发执行的评论关联 |
| 任务关联（可写层 zTree、层变更、文件树） | 评论区顶部全局 `#layer-association` slot | 无法表达「哪条评论触发了当前容器/层图」；阻碍评论级独立容器 |
| 依赖关系 | 无 UI | 未来 `wait_previous` / `independent` 编排无前端契约 |

用户期望：每条顶层评论（user / ai）下可展开「执行细节」，当前活跃执行的容器连接状态与任务关联面板应挂在**该评论**下，而非全局顶部。

---

## 2. 方案对比

| 方案 | 描述 | 优点 | 缺点 | 结论 |
|------|------|------|------|------|
| **A（采纳）** | 抽出容器连接状态组件；评论 Feed 增可折叠「执行细节」；layer-association 迁入 active 评论 | 与评论时间线对齐；为 per-comment 容器铺路；改动集中前端 | 需归属启发式；单容器期 layer 面板仅一处展示 | ✅ |
| B | 保留全局 layer-association，仅加评论内只读摘要 | 改动小 | 双源真相；仍无法表达依赖 | ❌ |
| C | 本期后端先落 Comment→Container 表再改 UI | 数据模型完整 | 违背 Go-first 后置编排；拖慢 MVP | ❌（非目标） |

---

## 3. 选定方案（MVP）

### 3.1 组件树（目标）

```text
TaskDetail.vue
├── TaskDetailRuntimeSection
│   └── TaskDetailServerStartStatusPanel          [MODIFIED] 仅服务器启动 + SSE
│       └── （移除内联「容器连接状态」块）
├── TaskDetailCommentsSection
│   └── TaskDetailCommentsPanel
│       ├── （移除顶部 #layer-association slot 全局挂载）
│       └── TaskDetailConversationFeed
│           └── 评论气泡（div.flex.gap-3…，每条顶层 user/ai）
│               └── 正文 / AI 回复 / 流式输出
│               └── TaskDetailCommentExecutionDetails   [嵌入气泡内，非兄弟独立区块]
│                   ├── dependency badge: wait_previous | independent
│                   ├── TaskDetailContainerConnectionStatus [NEW] 抽自 ServerStartStatusPanel
│                   └── TaskDetailTaskLayerAssociationPanel（仅 activeExecutionCommentId）
```

> **布局约束（2026-07-22）**：`#execution-details` slot 必须挂在评论气泡内部（`flex-1` 内容区末尾），不得作为气泡外的兄弟节点；视觉上使用轻量内嵌样式（`bg-gray-50/70`），避免再呈现独立卡片区块。

### 3.2 执行归属（activeExecutionCommentId）

启发式优先级（纯前端，本期无持久化）：

1. `activeContainerAgentId` 对应 container_agent 的**父评论** id
2. 否则 Feed 中**最新 AI 顶层评论** id
3. 否则**最新顶层 user 评论** id
4. 无任何评论 → **回退**：在评论区顶部渲染一条「全局执行细节」占位（等价于现 `#layer-association` 位置，但使用同一 `CommentExecutionDetails` 组件）

### 3.3 依赖 UI（前端契约）

每条执行细节展示依赖模式 badge（**只读**）：

| 值 | 标签 | 含义（本期展示） | 默认 |
|----|------|------------------|------|
| `wait_previous` | 串行 · 等待上一条 | 须等前序评论执行完成后再启容器（未来编排） | ✅ |
| `independent` | 可并行 | 可与前序评论并行执行（未来编排） | |

**变更（2026-07-22）**：评论发出后不再提供「依赖模式 串行/可并行」切换控件；模式仅在添加评论 composer 的执行依赖选择器中设定。`canEditMode` 默认 `false`，摘要 badge 仍展示当前模式。

数据来源：TTS `comments.execution_mode`（及绑定同步）；无字段时默认 `wait_previous`。

### 3.4 容器连接状态抽取

从 `TaskDetailServerStartStatusPanel.vue` 抽出 `TaskDetailContainerConnectionStatus.vue`：

- 保留：心跳状态点、seq/ack 面板、探测日志折叠、复制连接状态
- Props：沿用现有 heartbeat / seqInfo / logLines composable 输出
- `TaskDetailServerStartStatusPanel` 仅保留：服务器生命周期、SSE 行、启动进度/日志/错误 banner

### 3.5 layer-association 迁移

- **移除** `TaskDetailCommentsPanel` 顶部 `<slot name="layer-association" />` 的全局挂载路径
- **仅**在 `activeExecutionCommentId` 匹配的评论 `CommentExecutionDetails` 内渲染 `TaskDetailTaskLayerAssociationPanel`（及 `TaskDetailCommentLayerZtreeStatus` 加载/释放态）
- 非 active 评论的执行细节：可折叠展示依赖 badge + 容器连接摘要（只读）；不渲染完整 layer 面板（避免多 zTree 实例）

### 3.6 数据契约（CommentExecutionContext）

```typescript
/** 前端展示契约；MVP 不落库 */
interface CommentExecutionContext {
  commentId: string
  dependencyMode: 'wait_previous' | 'independent'  // default wait_previous
  containerRef?: {
    agentCommentId?: string   // container_agent 回复 id
    containerAgentId?: string // 与 activeContainerAgentId 对齐
  }
  isActiveExecution: boolean  // commentId === activeExecutionCommentId
}
```

Composable 建议：`useCommentExecutionContext(displayComments, activeContainerAgentId)` → `{ contexts, activeExecutionCommentId }`。

---

## 4. 领域模型

| 概念 | 说明 |
|------|------|
| **CommentExecutionBinding** | `{ commentId, waitPrevious: boolean, containerRef? }`；`waitPrevious=true` ↔ `dependencyMode=wait_previous` |
| **CommentExecutionContext** | 前端聚合视图：binding + 实时容器心跳 + layer 图是否挂载 |
| **ActiveExecutionComment** | 当前唯一展示完整 layer 面板 + 读写容器状态的评论 |

**Go 领域包例外**：本期**不新增** Go 包或表；`CommentExecutionBinding` 仅为前端展示契约与未来编排预留。后端多容器编排（taskTaskService / taskCloudService）在后续迭代持久化 `comment_id ↔ container_agent_id` 与 `dependency_mode`。

### 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 说明 |
|---------|--------|--------|------|
| 评论执行细节展示重组 | — | — | 纯前端 UI 重组，无新服务端事件 |
| 依赖模式展示 | — | — | 展示契约；编排事件后置 |

---

## 5. 成功标准

| # | 标准 | 验证 |
|---|------|------|
| S1 | Runtime 区无「容器连接状态」块；SSE/启动状态保留 | 组件单测 + 冒烟 |
| S2 | 每条顶层评论有「执行细节」折叠区 | Feed 渲染测试 |
| S3 | layer-association 仅出现在 active 评论（或无评论时顶部回退） | E2E / 单测 |
| S4 | `activeContainerAgentId` 变化时归属评论跟随 | composable 单测 |
| S5 | 依赖 badge 默认 `wait_previous` | 快照/单测 |
| S6 | 架构 v51 + 意图文档同步 | docs 审查 |

---

## 6. 非目标

- 本期不实现「每评论一容器」云编排或多 CSC 并行
- 不新增 Django / Go HTTP API
- 不改 container_agent 表结构
- 不在非 active 评论渲染完整 layer 命令面板

---

## 7. 实现落点（前端）

| 文件 | 变更 |
|------|------|
| `TaskDetailServerStartStatusPanel.vue` | 移除容器连接块；保留 SSE/启动 |
| `TaskDetailContainerConnectionStatus.vue` | 🆕 抽出 |
| `TaskDetailCommentExecutionDetails.vue` | 🆕 折叠 + 依赖 badge + 条件子组件 |
| `TaskDetailCommentsPanel.vue` | 移除 `#layer-association` slot |
| `TaskDetailCommentsSection.vue` | 改 Feed 子项挂载；移除顶部 layer slot |
| `TaskDetailConversationFeed.vue` / `TaskDetailCommentItem` | 注入 ExecutionDetails |
| `composables/taskDetail/useCommentExecutionContext.js` | 🆕 归属启发式 |
| `TaskDetail.vue` | 透传调整；移除全局 layer ref 路径 |

---

## 8. 🏛️ 架构变更影响

- **迭代版本**: v51 🎯 target
- **视图**: `application-integration`（基于 v48 current）
- **新增文件**:
  - 🆕 `docs/architecture/v51-application-integration-20260722-1850-claude.puml`
  - 🆕 `docs/architecture/v51-application-integration-20260722-1850-claude.archimate`（含 Plateau v48→Gap→v51 + sourceConnection）
  - 🆕 `docs/architecture/v51-application-integration-20260722-1850-claude.mermaid.md`
- **变更明细**:
  - 🟡 [MODIFIED] Vue TaskDetail — 执行 UI 评论内聚；CommentExecutionContext
  - 🟢 [NEW] 前端契约 CommentExecutionBinding / dependencyMode
  - ⚪ [UNCHANGED] taskAIComment、taskCloudService、container SSE/heartbeat API
  - 🎯 Plateau v51；Gap：执行状态与评论 Feed 脱节

### .archimate 架构变迁要点

| 元素类型 | 内容 |
|----------|------|
| **Plateau v48** | Current — 超管充值消费总览基线（后端未变） |
| **Plateau v51** | Target — 评论级执行细节 + CommentExecutionContext |
| **Gap** | 容器/层图挂全局评论区，无法 per-comment 绑定 |
| **WorkPackage** | WP-v51-comment-execution-details |
| **视图** | `架构变迁 v48→v51 — comment-execution-details`（含 sourceConnection） |

---

## 9. 验收计划（摘要）

1. 打开任务详情：Runtime 仅见启动/SSE；容器连接在 active 评论「执行细节」内。
2. `@镜像` 触发新 agent：layer 面板随 `activeContainerAgentId` 迁移到对应父评论下。
3. 无评论任务：顶部回退区仍可见 layer（与现行为等价）。
4. 折叠/展开执行细节不影响 SSE 订阅（全局单例 composable）。
5. 前端单测：`useCommentExecutionContext` 三条启发式 + 默认 `wait_previous`。

---

## 10. 审批记录

- **方案**: A（goal-mode 2026-07-22 锁定）
- **总体设计**: approved
