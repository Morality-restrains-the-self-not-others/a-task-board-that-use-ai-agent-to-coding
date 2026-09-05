# 评论级执行细节 — 价值流

## 端到端流（MVP）

```text
用户打开任务详情
  → 加载 comments + container_agent（既有 API）
  → useCommentExecutionContext 计算 activeExecutionCommentId
  → Runtime 区：ServerStartStatusPanel（启动 + SSE only）
  → Feed 每条顶层评论：
       └─ CommentExecutionDetails（折叠）
            ├─ dependency badge（默认 wait_previous）
            ├─ ContainerConnectionStatus（仅 active 或摘要）
            └─ LayerAssociationPanel（仅 activeExecutionCommentId）
  → 用户 @镜像 / auto_run 触发新 agent
  → activeContainerAgentId 更新 → 归属评论切换 → layer 面板迁移
  → 容器 heartbeat / layer job（既有路径，无新 API）
```

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 有 activeContainerAgentId | layer + 完整连接状态在该 agent 父评论下 |
| T2 | 无 agent，有 AI 评论 | 归属最新 AI 顶层评论 |
| T3 | 仅 user 评论 | 归属最新 user 顶层评论 |
| T4 | 零评论 | 顶部回退区渲染 ExecutionDetails |
| T5 | Runtime 面板 | 无「容器连接状态」标题块 |
| T6 | 非 active 评论展开 | 见依赖 badge；无 layer 命令面板 |
| T7 | SSE 断开重连 | 与评论折叠状态无关，全局 composable 仍工作 |
| T8 | dependency 默认 | 新评论 context 为 wait_previous |

## 价值流映射

- `task-detail` / `comment-feed`：执行上下文与评论对齐
- `container-runtime`：连接状态可视化下沉到评论
- 未来 `comment-container-orchestration`：dependencyMode 后端持久化（本期不启）
