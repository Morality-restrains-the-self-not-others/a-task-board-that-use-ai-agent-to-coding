# 意图：执行日志按 agent step 逐步推送

## 背景

任务详情「执行日志」在容器启动并 auto_run / 执行 Trae 指令后，长时间无增量，任务结束才一次性刷出全部步骤。

## 目标

1. 容器内 Trae 任务每完成一个 agent step，向 job events 写入一条 `phase=step` 事件（含摘要）。
2. 详情页收到 `container_job_stream` 的 `phase=step` 时立即刷新执行日志（步骤卡片）。
3. 无 job-stream（如容器内 auto_run 首指令）时，选中任务处于 `queued|pending|running` 期间定时拉取执行日志，仍能逐步展示。

## 非目标

- 不改变 trajectory / tae_agent_json 落盘格式。
- 不强制改造 stdout chunk 节流策略。


## 业务意图 → 事件对照

> 存量回填（自动）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。事件名若为启发式占位，可在后续迭代精修。

**无对应事件**：纯前端展示/交互或设计治理，无服务端业务状态变更意图。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 意图：执行日志按 agent step 逐步推送 | — | — | — | 纯前端展示/交互或设计治理，无服务端业务状态变更意图 |
## 变更记录

- 2026-07-14：初版；根因是 running 期间仅 status 变化才刷新，结束时整批拉取。
