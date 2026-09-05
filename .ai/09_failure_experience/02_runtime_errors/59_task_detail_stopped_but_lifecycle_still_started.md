# [运行时] 已停机但「服务器启动状态」仍显示已启动

## 现象

任务详情页「服务器启动状态」为「已启动」（含启动成功日志 / SSE 已连接），同页「服务器运行状态」为「未知 / 未创建」且提示「该任务尚未创建云实例」。

典型路径：本会话刚启动成功后机器已 `stop_vm` / 释放，CSC `instance_id` 已清空；或实例仅写在评论级 CSC，而运行态只读任务级空绑定。

复现任务（示例）：`task_13914730557265528870`（启动态 SSE 成功日志 vs 运行态尚未创建）。

## 环境与上下文

- 「服务器启动状态」：SSE `success` → `isServerRunning=true`；`resolveServerLifecycleLabel`
- 「服务器运行状态」：`GET .../cloud/compute/server-runtime-status/` → `runtime_status`
- 后端空实例成功响应：`status=success, runtime_status=null, message=该任务尚未创建云实例`
- 多评论 CSC：`cloud_server_configs.comment_id`；任务级 `comment_id=''`

## 根因

1. **空 runtime 不回落生命周期**：`notifyRuntimeHydrate` 对空 `runtime_status` 直接 return；意图 029 的 `shouldRematchRuntimeHydrate` 只认云 `Stopped/Released` 等非空码，不认「尚未创建」。停机清绑定后运行态正确变空，SSE「已启动」残留。
2. **停机只清任务级 CSC 字段**：`clearContainerReachabilityNative` upsert 任务级空绑定，但评论级行上的 `last_runtime_status=Running`（由 `setCloudServerLastRuntimeStatus` 全任务 UPDATE 写入）可残留。
3. **运行态只读任务级 CSC**：`handleServerRuntimeStatus` 原用 `loadCloudServerConfig`（`comment_id=''`）；实例若只在评论级 CSC，也会误报「尚未创建」。

## 修复

- 前端：`isServerRuntimeAbsentPayload` / `notifyRuntimeAbsentHydrate`——在本地仍 `isServerRunning` 且 API 明确无实例时，以 `Stopped` hydrate 回落生命周期；空状态展示「未创建」而非「未知」。
- 后端：`loadCloudServerConfigForRuntime`——任务级无 instance 时回退最新带 instance 的评论级 CSC；`clearContainerReachabilityNative` 停机后 `setCloudServerLastRuntimeStatus(..., "")` 清全任务残留运行态。
- `stop-vm` 同样经 `loadCloudServerConfigForRuntime` 定位物理机。

### 补充（启动中对齐，2026-08-12）

对称问题：评论级 CSC 已 ensure、尚无 `instance_id`，启动面板为「启动中」+「容器实例已分配」，运行态却报「未找到服务器配置记录」/「未创建」。

- `loadCloudServerConfigForRuntime` 额外回退「任务下最新 CSC（可不含 instance）」。
- 无 instance 且评论级 CSC / 绑定仍在 provisioning → `runtime_status=Starting` +「云实例创建中，等待分配」。
- 前端「云实例创建中」文案不计入 `isServerRuntimeNoInstanceMessage`，避免误 hydrate 为已停止。

## 预防

- 两面板对齐：运行态「无实例」与启动态「已启动」不得长期并存；空 `runtime_status` 在**明确无绑定文案**下必须可回落 SSE。
- 启动中：评论级 CSC 已存在但无 instance 时，运行态须为 Starting/创建中，禁止「未找到服务器配置记录」。
- 新增评论级 CSC 写入点时，凡读「当前物理机」须走 `loadCloudServerConfigForRuntime`（或显式按 `comment_id`），禁止只读任务级默认行。
- 回归：本会话启动成功 → stop → 刷新/拉取 runtime 后，`data-testid="server-lifecycle-status"` 不得仍为「已启动」。
- 回归：评论绑定 starting + 评论级 CSC 无 instance → runtime 不得含「未找到服务器配置记录」。
