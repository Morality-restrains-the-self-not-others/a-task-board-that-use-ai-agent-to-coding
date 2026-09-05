# 意图：评论「服务器运行状态」Tab 也展示容器名 / CSC / 启动 TraceId

## 背景与目标

任务详情评论执行细节已拆成两个 Tab：「执行细节」与「服务器运行状态」。容器名、CSC、启动 TraceId 原先只挂在「执行细节」面板内；切到「服务器运行状态」后这三项随面板一起消失。

目标：切到「服务器运行状态」时，**同样**可见容器名、CSC、启动 TraceId（有值才渲染对应行），且「执行细节」Tab 行为不变。启用「任务关联」（ztree）Tab 后，元信息仍在 Tab 面板外共享（见 `035_comment_ztree_independent_tab`）。

## 范围与边界

- 范围内：`TaskDetailCommentExecutionDetails.vue` 将 `comment-execution-container-meta` 从仅「执行细节」面板提升为两 Tab 共享；对应单测。
- 范围外：不改数据源、不改 start-vm / SSE 写 TraceId、不改 Teleport、不新增接口或 Kafka 事件。

## 约束与风险

- 无 Tab 模式（`serverRuntimeStatusTab=false`）仍展开即见元信息。
- CSC 未分配时仍显示「CSC 尚未分配…」；`startTraceId` 为空仍不渲染 TraceId 行。
- 禁止 Teleport；就地复用同一块 DOM。

## 验收标准

1. 仅运行状态 Tab 模式默认「执行细节」：可见 `comment-execution-container-meta`（容器名 / CSC / 启动 TraceId）。启用任务关联 Tab 时默认该 Tab，元信息同样可见。
2. 点击「服务器运行状态」后：`comment-execution-panel-server-runtime` 存在，且同一 `comment-execution-container-meta` 仍可见，三项文案与 `data-traceId` 与执行细节 Tab 一致。
3. 切回「执行细节」后元信息仍在。
4. 无 Tab 模式：元信息仍在展开内容中。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 在服务器运行状态 Tab 查看容器元信息 | — | — | — | — | 纯前端展示，无服务端事实变更 |

## 实施计划

1. 将 `comment-execution-container-meta` 移到 Tab 面板之外（tablist 与两面板之间），两 Tab 共享一块 DOM。
2. 单测：切到 serverRuntime Tab 后断言容器名 / CSC / 启动 TraceId 仍存在。

## 变更记录

- 2026-08-13：由任务详情页元素调整提出——「服务器运行状态」也要显示容器名、CSC、启动 TraceId。
- 2026-08-13：实现将 `comment-execution-container-meta` 移到 tablist 与两面板之间，两 Tab 共享一块 DOM（不 Teleport、不复制）。
