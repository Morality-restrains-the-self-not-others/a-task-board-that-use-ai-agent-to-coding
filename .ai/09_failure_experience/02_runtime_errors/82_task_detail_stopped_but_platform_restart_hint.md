# [运行时] 服务器已停止却显示「平台服务重启中」

## 现象

任务详情「服务器启动状态」为**已停止**，同面板 SSE 行却出现「平台服务重启中」（或等价误导文案），用户误以为云服务器正在重启。

典型可见文本组合：`服务器启动状态 已停止` + `SSE 连接 正在重连（第 N 次）…` + `平台服务重启中`。

## 根因

1. **两行语义独立**（意图 027）：生命周期反映云 VM；SSE 行反映推送通道。
2. **假阳性置位**：`establishSSEConnection` 在 `onerror` 且「尚未 open + 耗时 < 2s」时**立即**把 `ssePlatformRestartHint=true`；异步探测仅在确认 502 时再写 true，**从未在非 502 时清除**。普通鉴权/瞬断也会误标。
3. **文案过宽**：「平台服务重启中」易被读成「我的服务器在重启」，而非网关/taskSSE。

## 修复

- 仅当探测响应为 nginx HTML 502（或 HTML 5xx）时置位 hint；非 502 / 探测失败则清除。
- 文案改为「状态推送服务暂不可用」，并加 title 说明与云服务器启停无关。
- 测例：`establishSSEConnection.platformRestart.test.js`、面板 T9/T10。

## 相关

- `.ai/09_failure_experience/02_runtime_errors/58_task_detail_startup_sse_502_during_runall_restart.md`
- `task2app/docs/intents/frontend/task_detail/027_sse_vs_server_lifecycle_status_labels.intent.md`
