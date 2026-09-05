# 实施计划 — Work Panel 任务状态 SSE

设计：`docs/superpowers/specs/2026-07-18-work-panel-task-status-sse-design.md`

## 任务清单

- [x] **P1** taskSSE：`workspaceHubKey` + path `work-panel-events-sse` + normalize
- [x] **P2** taskSSE 单测：403 / connected / normalize
- [x] **P3** taskEvents：intent `2_fanout_work_panel_sse`（port 18048）+ handler + 单测
- [x] **P4** 注册：intent_registry、domain-events yaml（两份）、run.sh、runAll.yaml、bin/README、check_event_health
- [x] **P5** Gateway routes + api_route_ownership + apisix 生成（若需）
- [x] **P6** 前端 `useWorkPanelTaskStatusSse` + WorkPanel 接线 + Vitest
- [x] **P7** intents / value-stream 测试点（已写文档）
- [x] **P8** 构建验证：taskSSE test、Go test、前端相关 vitest + SPA collectstatic

## 事件契约任务

- [x] TASK_STATUS_CHANGED 已存在
- [x] publish 路径：fanout → Redis
- [x] 消费者：taskSSE hub
