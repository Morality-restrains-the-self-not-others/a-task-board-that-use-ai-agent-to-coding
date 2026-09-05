# 测试意图：025_machine_owner_hint

| # | 场景 | 断言 |
|---|------|------|
| 1 | container_running=false | 组件不渲染 |
| 2 | owner=[viewer] | 文案含本任务 ID 与「本任务」 |
| 3 | owner=[viewer, other] | 列出两者；other 有 task-detail href |
| 4 | Go status 有 server_url | 响应含 machine_owner_task_ids |
| 5 | Go 无 server_url | container_running=false，可不带 owner 或 owner 仍可附带但不驱动 UI |
