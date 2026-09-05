# 价值流：容器镜像 @ 模式 + 统一评论区

**日期**: 2026-07-15  
**设计**: `docs/superpowers/specs/2026-07-14-container-image-at-mention-design.md`  
**状态**: 完成（goal-mode）

## Related Value Streams

| Stream / step | 关系 |
|---------------|------|
| `task-management` / `task-comments-api` | **扩展**：mentions + 分流 |
| `cloud-integration` / `workspace-machine-idle-policy` | **依赖**：`@` 触发必须走 idle reuse |
| `task-detail-runtime-relay` / `container-runtime-context` | **扩展**：ContextPack + Agent 回写 |
| `project-workspace` / workspace-crud | **扩展**：新字段 |

非 greenfield；增量挂靠既有 stream，并新增 step `container-image-at-mention`。

## Value Increments（交付顺序）

| # | Increment | 用户价值 | 依赖 |
|---|-----------|----------|------|
| I1 | 工作空间开关 CRUD + TaskPanel UI | 管理员可默认关闭能力 | — |
| I2 | 评论 mentions 契约 + 后端硬闸（无开机器） | 安全关闭路径可测 | I1 |
| I3 | `@` 触发 → Kafka → 编排 start-vm + idle reuse | `@` 真的开跑 | I2 |
| I4 | ContextPack + 容器自动执行 | Agent 拿到线程上下文 | I3 |
| I5 | ContainerAgentComment + 容器 stream/SSE + 统一 Feed | 流式回写可见 | I4 |
| I6 | 前端 `@` picker + 按钮文案 | 完整 UX | I1–I2 |

## YAML 登记（写入 conf/value-stream.yaml 的 step 草案）

```yaml
  - name: container-image-at-mention
    status: planned
    test_file: tests/test_container_image_at_mention.py
    fields:
    - name: task-project-service.workspaces.container_image_at_mode_enabled
      description: 工作空间是否开启容器镜像@模式（默认0）
    - name: task-task-service.comments.mentions_json
      description: 评论结构化镜像mention JSON
    - name: task-ai-comment.container_agent_comments.run_status
      description: Agent评论运行状态
    - name: task-ai-comment.container_agent_comments.assistant_response
      description: Agent最终回复正文
    - name: task-ai-comment.container_agent_comments.parent_comment_id
      description: 挂载的人类评论ID
```

## 测试影响

- 新建：`tests/test_container_image_at_mention.py`（或分服务 Go 单测 + 前端单测）
- 更新：`task-comments-api`、workspace PATCH、idle reuse 回归

## 结论

优先 I1→I2 竖切可独立验收；I3–I5 为运行时竖切；I6 可与 I2 并行。
