# 意图：启服中不展示平台预创建的容器 Agent 空气泡

## 背景与目标

`@镜像` / `auto_run` 路径上，taskTaskService 在人类评论落库后立刻
`notifyContainerAgentPending`，于 taskAIComment 预创建
`run_status=pending` 的容器 Agent 子评论（正文为父评论 prompt echo）。
此时云实例仍是「服务器启动状态=启动中」，容器镜像尚未写入回复。
Feed 却已渲染空的「容器 Agent」气泡（作者名 + 徽章 + 时间戳），
让人误以为容器已回复。

**目标**：评论 Feed 中的容器 Agent **回复气泡**只在容器镜像已写入可见
内容（正文、`assistant_response`、PR、或正在流式写入）后出现。
父评论「执行细节 → 服务器启动状态」仍负责展示启服进度。

## 范围与边界

- **改**：`decorateAgentRepliesWithLayerPr` 过滤无可见载荷的 pending
  占位；`TaskDetailCommentsSection.feedDisplayComments` 传入当前流式
  agent id。
- **不改**：平台仍预创建内部 pending 记录（编排 / `agent_comment_id` /
  ContextPack / 单活跃 run）；容器 `stream`/`complete` 契约不变。
- **非目标**：把 pending 行的创建权完全搬进容器镜像（见 OPT-20260823-008）。

## 约束与风险

- 过滤仅作用于 Feed 展示层，不删库、不影响执行细节绑定
  （仍看人类父评论）。
- 流式开始或 `run_status` 为 starting/running/streaming 时必须露出气泡，
  避免 SSE 落到 fallback。
- 有 ztree PR 回填时仍展示 `CommentGitPrReply`
  （既有 prompt-echo 隐藏逻辑）。

## 验收标准

1. pending + prompt echo + 无 assistant_response + 无 PR →
   子评论不出现在 Feed。
2. 容器已写 `assistant_response`、非 echo 正文、git_pr、或正在 stream →
   气泡保留。
3. 父评论执行细节仍可显示「服务器启动状态=启动中」。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
| --- | --- | --- | --- | --- | --- |
| 隐藏启服前平台占位 Agent 回复 | — | — | — | — | 无对应事件：纯前端展示过滤，不改变服务端状态 |

## 实施计划

1. `isPlaceholderContainerAgentReply` + decorate 过滤。
2. Feed 传入 `activeContainerAgentId` / stream 状态。
3. 单测覆盖占位隐藏与流式/正文/PR 保留。
