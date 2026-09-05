# 实施计划：容器镜像 @ 模式

**日期**: 2026-07-15  
**设计**: `docs/superpowers/specs/2026-07-14-container-image-at-mention-design.md`  
**状态**: 执行中（goal-mode → build）

## Tasks

### I1 — 工作空间开关
- [x] T1.1 taskProjectService：DB 列 + GET/PATCH 读写；默认 0；Go 单测
- [x] T1.2 OpenAPI / Swagger 字段
- [x] T1.3 WorkspaceSettingsTaskPanel Toggle UI

### I2 — 评论 mentions + 硬闸
- [x] T2.1 taskTaskService：POST comments 接受 `mentions`；开关关拒绝；多 mention 拒绝
- [x] T2.2 校验 installed image 属本租户（taskTaskService → Cloud lookup）
- [x] T2.3 落库 mentions_json；无 mention 路径回归绿

### I3 — 编排开机器
- [x] T3.1 发 Kafka `TASK_COMMENT_IMAGE_MENTIONED`
- [x] T3.2 taskAIComment：创建 pending ContainerAgentComment（HTTP notify）
- [ ] T3.3 调 start-vm(-auto) 带 container_image_id（follow-up 消费者）

### I4 — ContextPack
- [x] T4.1 聚合形状 + 256KB 截断（onlineServiceJS `atMentionContext.mjs`）
- [ ] T4.2 onlineServiceJS 读取 at_mention_run + 自动 job（follow-up）

### I5 — 流式回写
- [x] T5.1 container-agent stream/complete 接口 + token 校验
- [x] T5.2 SSE `container_agent_stream`
- [ ] T5.3 前端 Feed 合并展示（follow-up）

### I6 — @ picker UX
- [x] T6.1 开关 ON 时 composer `@` 候选（已安装镜像）
- [x] T6.2 提交 payload 带结构化 mentions

## 验证命令（按任务）

```bash
cd taskProjectService && go test ./...
cd taskTaskService && go test ./...
cd taskAIComment && go test ./...
# 前端相关 vitest
```

## 日志要求

每条关键路径打结构化日志：`event=at_mention_*` / `run_id` / `reuse` / `phase`；禁止记录 token 明文。
