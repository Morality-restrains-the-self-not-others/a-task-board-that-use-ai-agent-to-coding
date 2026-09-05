# 实施计划：released-server-comments-retain-layer-loading

- **日期**: 2026-07-15
- **设计**: `docs/superpowers/specs/2026-07-15-released-server-comments-retain-layer-loading-design.md`

## Tasks

### Task 1 — 抽取「非服务态」判定 + loading 门禁（红→绿）

- [ ] 在 `useTaskDetail.js`（或小工具模块）提供 `isServerNotServingUi`：`!isServerRunning && !isServerStarting`，或显式 `serverRuntimeNotServing` ref
- [ ] `showCommentLayerZtreeLoading`：非服务态 → false
- [ ] 单测：connecting + not running → loading false

### Task 2 — 停机 SSE 对称 pause（红→绿）

- [ ] `updateServerStatus.js`：`stopped`/`error` 调用 `pauseContainerHeartbeatForRelayStop`
- [ ] 单测断言 paused + endpoint false + hb idle

### Task 3 — 忽略晚到 heartbeat + endpoint watch 门禁

- [ ] `establishSSEConnection.js`：paused 或非服务态忽略 heartbeat
- [ ] `watch(containerEndpointRegistered)`：非服务态不强制 connecting

### Task 4 — 冷打开 runtime released

- [ ] `ensureServerRuntimeAllowsContainerLayerGraph` / 拉取 runtime 后：notServing → enter pause UI
- [ ] 单测或 composable 测试覆盖

### Task 5 — 空态 UI

- [ ] `TaskDetail.vue`：`showCommentLayerZtreeReleased` 静态块，`data-testid="comment-layer-ztree-released"`
- [ ] 文案按设计 §4.2

### Task 6 — 意图文档 + 失败经验

- [ ] `docs/intents/frontend/task_detail/022_*.intent.md` + test-intent
- [ ] `.ai/09_failure_experience/` 短条目

### Task 7 — 回归跑测

- [ ] 相关 vitest 通过
- [ ] 确认无误伤运行中 connecting
