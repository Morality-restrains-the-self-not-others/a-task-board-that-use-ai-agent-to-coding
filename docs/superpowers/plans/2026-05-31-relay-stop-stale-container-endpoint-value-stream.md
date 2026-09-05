# Value Stream: 容器停止 stale 端点作废与运行会话 History

> Derived from design: `docs/superpowers/specs/2026-05-31-relay-stop-stale-container-endpoint-invalidation-design.md`

## Value Summary

任务详情用户在 relay / mock-run / 云 VM **停止**后，平台不再向已下线容器 URL 转发层图；History 记录同一次运行的启停时间与 reachability，避免 stale 502。

## Related Value Streams

- **task-detail-runtime-relay**（`value-stream.yaml`）：**extension** — 扩展 `relay-stop-refresh-status-ui`、`container-runtime-context`
- **2026-05-27-relay-stop-refresh-status-value-stream.md**：**modification** — stop 后须清 DB 端点，非仅 UI 刷新
- **2026-05-31-relay-to-trae-startup-reliability-value-stream.md**：**dependency** — register 成功后再拉层图

Greenfield 增量：History 运行会话模型（方案 F）。

## End-to-End Flow

[用户点停止] → [go_relay/worker/云 API 停进程] → **[close session + 清 CloudServerConfig reachability]** → [SSE container_endpoint_registered=false] → [前端不再拉 container-layer-graph 至 stale URL]

## Value Increments

### Increment 1: Stop 止血（Thin Slice，方案 D）

**Value to user:** 停止后刷新/再启不再立即 502 Connection refused  
**Scope:** 共享 `clear_container_reachability`；挂载 relay / mock-run / stop-vm；SSE；pytest  
**Depends on:** nothing

### Increment 2: History 运行会话（方案 F）

**Value to user:** 一次运行一条 History（started_at / stopped_at / server_url）；register 更新同条会话  
**Scope:** migration、open_session、register 改 update、get_server_start_history API  
**Depends on:** Increment 1

### Increment 3: 前端乐观收敛 + 启动门控

**Value to user:** stop 后 UI 立即收敛；再启无 stale 竞态  
**Scope:** `onContainerReachabilityCleared`、Playwright 回归  
**Depends on:** Increment 1
