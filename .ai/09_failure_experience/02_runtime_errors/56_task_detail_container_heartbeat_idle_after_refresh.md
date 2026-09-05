# 任务详情：容器已启动刷新后仍「等待连接」

- **日期**: 2026-07-19
- **症状**: 服务器与镜像容器已运行时刷新任务详情，「容器连接状态」长期为「等待连接」（`idle`）。
- **根因**:
  1. `containerHeartbeatStatus` 初始 `idle`，仅靠 SSE `container_heartbeat` 更新；
  2. 应用门禁只认 `isServerRunning/Starting`，冷打开 hydrate 前早到心跳被丢；
  3. 无缓冲、端点已登记时也不离开 idle。
- **修复**:
  - `applyContainerHeartbeatSse.js`：门禁含 `containerEndpointRegistered`；早到缓冲；idle→connecting；
  - `establishSSEConnection.js` / `fetchContainerTaskUiContext` / `runtime_hydrate` 接线 sync+flush。
- **意图**: `task2app/docs/intents/frontend/task_detail/030_container_heartbeat_cold_open_idle.intent.md`
