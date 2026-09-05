# 测试意图：执行日志按 agent step 逐步推送

## 用例

### T1 — step 事件格式与去重

- **给定** trajectory 或 tae_agent_json 中陆续出现 step 1、step 2
- **当** `startAgentStepPoller` / `takeNewAgentSteps` 轮询
- **则** 每个 step_number 只产生一次 `phase=step` 消息，且含 delivery_summary（若有）

### T2 — SSE phase=step 刷新执行日志

- **给定** 详情页已订阅 server-startup-status-sse
- **当** 收到 `status=container_job_stream, phase=step, message=step 1: …`
- **则** 调用 `refreshZTreeExecutionLog`，且 live output 追加该摘要行

### T3 — running 兜底轮询

- **给定** 选中 job 状态为 `running` 且容器端点已注册
- **当** `createActiveJobExecLogPoller.sync`
- **则** 启动定时刷新；状态变为 `completed` 后停止定时器

## 自动化落点

- `trae-agent/onlineServiceJS/src/jobStepEvents.test.mjs`
- `task2app/front_project/app/src/composables/taskDetail/updateServerStatus.test.js`
- `task2app/front_project/app/src/composables/taskDetail/activeJobExecLogPoller.test.js`
