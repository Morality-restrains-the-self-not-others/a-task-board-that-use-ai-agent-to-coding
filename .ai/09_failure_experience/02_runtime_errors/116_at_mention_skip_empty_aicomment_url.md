# [运行时] @trae-agent 评论已启机但未自动执行指令

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 编号：116
- 维护者：Trae AI 团队
- TraceId：`be794568-25eb-4646-9b53-1d912f50b2db`
- 任务：`task_878865840278106112` / 评论 `cmt_878866202137489408`

## 现象

- 任务详情「是否自动运行」为否；用户评论 `@trae-agent 删除 用 java 写的 hello world`。
- 容器已启动并完成 bootstrap 克隆；层图仅有 `[bootstrap] 容器引导克隆 completed`，无 Agent job。
- Loki 在 kickoff 处为 `AUTO_RUN_FIRST_SKIP reason=auto_run_false`。

## 时间线（Loki，UTC）

| 时间 | 事件 |
|------|------|
| 05:00:31 | `at_mention_comment_create` + `TASK_COMMENT_IMAGE_MENTIONED` |
| 05:00:39 | `start_vm_instance_persisted` |
| 05:00:43 | `by-parent/.../status` **PATCH 404**（无 pending Agent 评论） |
| 05:06:02 | `BOOTSTRAP_COMPLETE` → `AUTO_RUN_FIRST_SKIP reason=auto_run_false` |
| 全程 | **无** `at_mention_notify_agent_ok` / `container_agent_created` / POST `container-agent-comments` |

## 根因

1. `conf/taskTaskService/config.yaml` 未声明 `services.taskAIComment`。
2. `loadConfig` 仅在 `TaskAIComment.Port != 0` 时赋值 `AICommentServiceURL`，缺省保持空串。
3. `notifyContainerAgentPending` 对空 URL **直接 return nil**，不写结构化日志。
4. `handleCreateComment` 先发 Kafka 再 notify；kickoff 依赖 task-detail 注入的 `at_mention_run`（来自 Agent 评论 ContextPack）。Pack 缺失时回退 `auto_run`，任务 `auto_run=false` 则跳过首指令。

## 解决方案

1. `resolveLoopbackServiceURL`：空 host / `0.0.0.0` → `127.0.0.1`，缺省端口 **8019**。
2. conf 补 `taskAIComment: host 127.0.0.1 port 8019`（同节点调用注释）。
3. 空 URL 返回 error + `logInfo`；notify **先于** `TASK_COMMENT_IMAGE_MENTIONED`。
4. notify 成败改走 `logInfo`（可进 Loki）。

## 验证

```bash
cd taskTaskService && go test ./src -count=1 -timeout 90s \
  -run 'TestResolveLoopbackServiceURL|TestNotifyContainerAgentPendingEmptyURL|TestCreateCommentWithMentionNotifiesAgentBeforeEvent'
# 部署：精准编译重启 task-task-service
# Loki：新 @mention 应有 at_mention_notify_agent_ok，且容器 BOOTSTRAP 后出现 at_mention_job_start / AUTO_RUN_FIRST_INSTRUCTION 而非仅 auto_run_false skip
```

## 存量容器

已 bootstrap 且 skip 的旧容器不会自动补跑；需新镜像/重启或页面「重新执行」/「发送给AI」。新评论在 task-task-service 重启后走修复路径。
