# [运行时] 任务详情变动文件「列表过长」→ 滚动加载更多 + data-traceId

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-19
- 最后修改：2026-07-19
- 维护者：Trae AI 团队

## 现象

- 页面：`/tenant/.../workspace/.../task-detail/.../`
- 提示：`变动文件列表过长，当前仅展示部分文件。`（琥珀字 `text-amber-700`）
- 刷新/预览失败时错误节点可能缺少 `data-traceId`

## 根因

1. 容器 `diff/parent/files` 无分页；执行日志一次嵌入全量（或扫描截断）变动列表。
2. `layerChangesRefreshErrorTraceId` 面板已接 props，但 state 未赋值；预览失败未下传 traceId。

## 解决方案

1. onlineService 分页（`offset`/`limit`/`has_more`）；Gateway 首屏 `limit=100` + `container-layer-diff-parent-files` 续拉。
2. 前端列表滚动触发 `loadMoreSelectedLayerChanges`；提示区分「可续拉」与「扫描上限」。
3. 刷新/续拉/预览失败路径挂 `data-traceId`。

## 验证

见 `docs/intents/frontend/task_detail_layer_changes_scroll_load_more.test-intent.md`。

公网 SPA：`bash scripts/runall-lifecycle.sh build`（已 collectstatic）。
容器镜像变更需 commit 后 `DOCKER_PUSH=1 ./buildDocker.sh`（`onlineServiceJS/ai.md`）。
