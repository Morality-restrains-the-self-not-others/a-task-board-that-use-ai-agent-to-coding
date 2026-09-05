# 任务详情变动文件列表滚动加载更多

## 意图

任务详情 zTree「变动文件」列表过长时，采用**滚动触发加载更多**，替代「仅展示部分文件」的静态截断提示；相关请求失败的 inline 错误须挂载 `data-traceId`。

## 背景

- 页面：`/tenant/.../workspace/.../task-detail/.../`
- 原提示：`变动文件列表过长，当前仅展示部分文件。`（`TaskDetailExecLayerChangesHints`）
- 容器 `GET /api/layers/{id}/diff/parent/files` 原先一次返回全量（扫描上限 `MAX_DIFF_ENTRIES` 时 `truncated=true`）

## 方案

1. onlineService：`diff/parent/files` 支持 `offset`/`limit`，响应 `change_count`/`next_offset`/`has_more`
2. Gateway：执行日志首屏 `limit=100`；新增 `container-layer-diff-parent-files` 转发续拉
3. 前端：列表底部滚动触发 `loadMoreSelectedLayerChanges`；提示文案区分「可续拉」与「扫描上限」
4. `data-traceId`：刷新/续拉失败、文件预览失败路径补齐；**扫描上限琥珀提示**挂载产生该状态的成功请求 `fetch_trace_id`（见 `task_detail_layer_changes_scan_cap.intent.md`）
5. 扫描上限根因治理：`collectIndex` 跳过 `node_modules` 等噪声目录（同文件树 SKIP 集合）；**扫描上限琥珀提示**挂载产生该状态的成功请求 `fetch_trace_id`（便于对照容器扫描日志）
5. **扫描上限根因治理**（解决「目录扫描已达上限」误伤）：
   - **近端**：`collectIndex` 与文件树共用 `shouldSkipListingDirName`，跳过 `node_modules` / `.venv*` / `dist` 等噪声目录，避免占满 `MAX_DIFF_ENTRIES`（4000）
   - **中期（待办）**：优先用 git 状态/与父层 commit 对比生成变动集，减少全量目录 walk
   - **验收**：大仓含 `node_modules` 时，真实源码变动仍应出现在列表，且尽量不再出现扫描上限提示

## 变更相对旧版

| 项 | 旧 | 新 |
| --- | --- | --- |
| 截断提示 | 静态「仅展示部分」 | 有更多页时引导滚动；仅扫描截断时说明上限 |
| 首屏数据 | 可能一次灌入全量 | 执行日志侧默认首屏 100 条 |
| 错误可观测 | 刷新错误缺 traceId | refresh/load-more/preview 错误挂 `data-traceId` |
| 扫描上限提示 | 无 traceId | 挂载最近成功拉取的 `data-traceId` |
| 索引扫描 | 含 node_modules 等噪声 | 跳过与文件树一致的噪声目录 |

## 业务意图 → 事件对照

**无对应事件**：纯前端列表分页展示与容器查询转发，无平台业务状态变更。

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 变动文件列表滚动加载更多 | — | 只读查询/展示例外 |
| 刷新/预览错误挂载 data-traceId | — | 前端可观测属性，无 MQ 事件 |
