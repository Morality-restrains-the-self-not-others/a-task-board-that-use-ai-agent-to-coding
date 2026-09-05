# 任务详情：服务器已释放后可写层假「连接确认中」

- **日期**: 2026-07-15
- **症状**: 评论区上方「任务关联（可写层串行 · 容器推送）」长期显示「容器连接确认中，即将拉取可写层…」；机器实际已释放；AI 评论本应仍可展示。
- **根因**: `showCommentLayerZtreeLoading` 仅看 `containerHeartbeatStatus===connecting` 与空层图，不感知 `isServerRunning`；SSE `stopped`/`error` 未 pause 心跳，晚到 `container_heartbeat` 可回写 connecting。
- **修复**: `commentLayerZtreeUiState.js` 非服务态门禁 + 释放空态；`stopped`/`error` 对称 `pauseContainerHeartbeatForRelayStop`；非服务态忽略 heartbeat；runtime notServing 冷打开 enter 非服务 UI。
- **设计**: `docs/superpowers/specs/2026-07-15-released-server-comments-retain-layer-loading-design.md`
- **意图**: `task2app/docs/intents/frontend/task_detail/022_released_server_comments_retain_layer_loading.intent.md`
