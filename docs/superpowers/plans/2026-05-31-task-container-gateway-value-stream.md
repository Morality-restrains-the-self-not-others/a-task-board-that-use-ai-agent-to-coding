# Value Stream: taskContainerGateway

> 设计：`docs/superpowers/specs/2026-05-31-task-container-gateway-auth-design.md`

## Related Value Streams

- **task-agent-support-phase2-internal-scoped**：扩展 — inbound 路由最终合并进同一 Go binary（P4）
- **task-detail-runtime-relay**：修改 — 出站由 Django 直连 OSJS → 经 Go 网关
- **platform-centralized-logging**：扩展 — 新 service `task-container-gateway`
- **2026-05-31-relay-git-commit-pending-trace-observability**：被取代 — thread-local 热修降为 P0 可选

## Value Summary

任务详情 zTree 对 onlineServiceJS 的操作经 **Go 网关**完成 Session 鉴权与并发安全的 HTTP 转发，消除 git-commit 无限 pending，并在 Loki 形成完整 trace 链。

## End-to-End Flow

[zTree 提交] → **taskGateway** → **taskContainerGateway**（validate session）→ Django internal（scope + target）→ onlineServiceJS git/commit → [200 返回浏览器]

## Value Increments

### Increment 1: 薄切片 — validate + git-commit forward（本迭代）

**Value：** 用户点击「提交」≤5s 得到响应，不再无限 pending  
**Scope：** Django `validate-session` + `resolve-container-target` internal；Go 网关 `container-layer-git-commit`；**taskGateway** `routes.yaml` 路由（非 Vite proxy）  
**Depends on：** taskAgentSupport 模式、现有 CloudServerConfig  
**Test：** `tests/test_gateway_validate_session.py`、`taskContainerGateway/src/handlers_test.go`

### Increment 2: L0 forward 全集 + 结构化日志

**Value：** layer-graph、files、git/add 等经网关；Grafana 可见 forward_stage  
**Scope：** 映射剩余 L0 actions  
**Depends on：** Increment 1

### Increment 3: job-stream goroutine

**Value：** 发指令后 SSE 轮询不再与 forward 争用 Python Session  
**Scope：** Go job-stream + Django publish internal  
**Depends on：** Increment 2

### Increment 4: git-push 编排 + inbound 合并

**Value：** 推送 pending 消除；单 Go 服务 inbound+outbound  
**Depends on：** Increment 3
