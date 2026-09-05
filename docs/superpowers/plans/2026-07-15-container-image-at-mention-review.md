# Code Review：container-image-at-mention（goal-mode Step 9）

**日期**: 2026-07-15  
**对照计划**: `docs/superpowers/plans/2026-07-15-container-image-at-mention-plan.md`

## 结论

**v26 follow-up 已于 2026-07-15 关闭**：镜像归属校验、Kafka→start-vm(-auto)+idle reuse、ContextPack→task-detail+自动 job、前端 `container_agent_stream` Feed 均已落地。

## 对照检查

| 项 | 状态 |
|----|------|
| 工作空间开关 + 单测 | ✅ |
| 评论 mentions 硬闸 + 单测 | ✅ |
| Kafka 事件 + 通知创建 Agent 评论 | ✅ |
| ContainerAgentComment CRUD/stream/complete/fail + SSE | ✅ |
| Gateway 路由 | ✅ |
| OpenAPI | ✅ taskProjectService / taskAIComment |
| TaskPanel Toggle + 评论 @ picker | ✅ |
| ContextPack 规范化 + 截断 + bootstrap job | ✅（OSJS `maybeStartAtMentionJob`；Credential task-detail 注入） |
| start-vm idle reuse 编排消费 | ✅（taskEvents `1_start_vm_for_at_mention` :18045） |
| Feed 合并 Agent 评论 / SSE UI | ✅（`container_agent_stream` + displayComments） |
| mention 镜像本租户 installed-images 校验 | ✅（Cloud lookup） |
| 日志 event=at_mention_* / container_agent_* | ✅ |
| 无 Python 新接口 | ✅ |

## Log Audit

- [x] 无 token/secret 明文落日志  
- [x] 关键路径有 event= 结构化标记  
- [x] 错误路径有 status/err 记录  

## Follow-up（已关闭 2026-07-15）

1. ~~taskEvents：消费 `TASK_COMMENT_IMAGE_MENTIONED` → start-vm(-auto)~~ ✅  
2. ~~ContextPack 注入容器 task-detail + auto job~~ ✅  
3. ~~前端订阅 `container_agent_stream`~~ ✅  
4. ~~mention 镜像本租户 installed-images 校验~~ ✅  
