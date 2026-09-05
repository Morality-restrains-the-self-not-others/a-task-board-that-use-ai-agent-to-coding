# DDD 领域模型：容器镜像 @ 模式

**日期**: 2026-07-15  
**状态**: 完成（goal-mode）  
**落点**: 扩展现有 Go 服务领域概念；骨架代码在各服务 `domain/` 或紧邻 handler 的 domain 包（与存量一致）

## Bounded Contexts

| Context | 所有者服务 | 职责 |
|---------|------------|------|
| Workspace Policy | taskProjectService | `AtModeEnabled` 开关 |
| Task Conversation | taskTaskService | HumanComment + ImageMention |
| Container Agent Conversation | taskAIComment | ContainerAgentComment + AtMentionRun |
| Compute Scheduling | taskCloudService | 复用 IdleReuse（已有） |
| Container Runtime | onlineServiceJS | ContextPack 消费与自动 job |

## Aggregates / Entities

### Workspace（扩展）
- 属性：`containerImageAtModeEnabled bool`（默认 true）

### HumanComment（扩展）
- VO：`ImageMention { InstalledImageID, DisplayName }`（0..1）
- 不变量：开关关则 Mentions 必须空；Mentions 最多 1

### ContainerAgentComment（新）
- ID、ParentCommentID、TaskID、TenantID、WorkspaceID、InstalledImageID
- RunStatus：pending|starting|running|streaming|completed|failed
- AssistantResponse（最终）
- 不变量：同一 ParentCommentID 仅一个活跃 run（非 terminal）

### CommentThreadContextPack（VO）
- TaskSnapshot、Comments[]（含 trigger）、Truncated flag

## Domain Services

- `MentionGate.Validate(atMode, mentions, tenantImages)`  
- `AtMentionOrchestrator.Start(run)` — 应用服务，调 Cloud + Runtime 端口  
- `AgentReplyAppender.AppendChunk / Complete`

## Ports

- `WorkspaceAtModeReader`
- `InstalledImageVerifier`
- `ComputeStarter`（start-vm + idle reuse）
- `ContainerContextInjector`
- `AgentStreamPublisher`（SSE）
- `EventBus.Publish(TASK_COMMENT_IMAGE_MENTIONED)`

## Domain Events

- `TaskCommentImageMentioned`
- `ContainerAgentRunStatusChanged`
- `ContainerAgentReplyCompleted`

## 包路径约定（实现）

| 服务 | 路径 |
|------|------|
| taskProjectService | 字段进现有 workspace handlers；可选 `domain/workspace_at_mode.go` |
| taskTaskService | `domain/image_mention.go` + comment handler |
| taskAIComment | `domain/container_agent_comment.go` + store/handlers |
