# 任务详情：ECS 已 Running 但连接「等待连接」且 zTree 空白

- **日期**: 2026-07-22
- **页面**: 任务详情评论「执行细节」
- **症状**:
  - 服务器启动状态「已启动」、评论 binding「运行中」；
  - 容器连接状态长期「等待连接」（`idle`）；
  - `comment-layer-association-body` 为空（无 loading、无 zTree）。
- **根因**:
  1. CSC `last_runtime_status=Running` 但 `server_url` / `business_api_endpoint` 仍空 → `container_endpoint_registered=false`（镜像 userdata 尚未完成 register-reachability，或回调失败）；
  2. 评论 `comment_container_bindings.status=running` 仅为编排 MVP 的 mock 推进，**不等于**真实容器已登记；
  3. `activeExecutionCommentId` 取最新评论，可能落在 `waiting_previous`，与真实 running binding 错位；
  4. `shouldShowCommentLayerZtreeLoading` 在 `idle` 且无 endpoint 时返回 false → 空白面板。
- **修复（前端）**:
  - VM Running 且无 endpoint：心跳 idle→`connecting`（文案「等待容器登记」）；
  - zTree 显示加载 hint（等待 register-reachability）；
  - active 优先 `running`/`starting` binding。
- **残余**: 若 userdata/拉镜像卡住，仍需运维查实例 `/root/init_from_task2app.sh.log` 或安全组出站；前端只能展示等待态。
