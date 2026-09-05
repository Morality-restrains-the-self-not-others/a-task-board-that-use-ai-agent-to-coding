# 测试意图：Work Panel 任务状态 SSE（后端）

对应：`work_panel_task_status_sse.intent.md`

| ID | 场景 | 期望 |
|----|------|------|
| B1 | fanout handler 缺 workspace_id | permanent error / 不 publish |
| B2 | 正常 payload | Redis channel = `sse:workspace:{ws}`，status_data.event_name=`task_status_changed` |
| B3 | hub key 在 envelope.task_id | `workspace:{ws}` |
| B4 | 网关路由指向 taskSse | routes.yaml 含 work-panel-events-sse |
| B5 | fanout 收到 TASK_CREATED / TASK_DELETED | Redis 帧 event_name=`task_created` / `task_deleted` |
| B6 | fanout intent 订阅事件 | 含 TASK_STATUS_CHANGED、TASK_CREATED、TASK_DELETED |
| B7 | clone-run CONF_ROOT 指向部署根 | taskSSE 从部署根 conf-local 读到 gatewayInternalSecret，不因 envs/current/conf/base.yaml 走空密钥 fail-closed |
