# [运行时] 任务详情项目文件树周期性刷新 + 提交日志缺 data-traceId

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-18
- 最后修改：2026-07-18
- 维护者：Trae AI 团队

## 现象

- 页面：`/tenant/.../workspace/.../task-detail/.../`
- 元素：`[data-testid="comment-layer-ztree-project-file-tree"]`（`details.mt-3...`）
- 文件树区域持续闪烁/重新加载，像浏览器定时轮询，而非容器推送
- 选中仓库目录后「提交日志」显示 `Invalid or missing access token`，错误 `<p>` **无** `data-traceId`

## 根因

1. **轮询误伤文件树**：`activeJobExecLogPoller`（约 2s）调用 `refreshZTreeExecutionLog` → `ingestLayerChangesFromExecutionPayload` → **无条件** `bumpProjectFileTreeRefresh()` → `TaskDetailProjectFileTree` watch nonce → `fetchFiles()`。即使 `layer_changes` 内容未变也会刷树（`filesLoading` opacity 闪烁）。
2. **提交日志缺 traceId**：`handleSelectDir` / `handleSelectFile` 对非 2xx 只 `throw new Error(detail)`，未读取 `apiFetch` 挂载的 `response.traceId`；`TaskDetailExecLayerChangePreview` 错误节点也未绑定 `data-traceId`。

## 解决方案

1. `layerChangesContentFingerprint` + ingest 时仅 fingerprint 变化才 `bumpProjectFileTreeRefresh`。
2. 预览失败路径：`extractTraceId(resp|body)` → `previewErrorTraceId` → 预览错误节点 `:data-traceId`；文件树列表错误同理。
3. 超 500 行：纯函数抽至 `utils/taskDetailProjectFileTreeHelpers.js`。

## 验证

```bash
cd taskFE/app && npx vitest run \
  src/composables/taskDetail/taskDetailZTreeExecLogState.test.js \
  src/components/task-detail/TaskDetailExecLayerChangePreview.test.js \
  src/components/task-detail/TaskDetailProjectFileTree.preview-traceId.unit.test.js
bash scripts/runall-lifecycle.sh build
```

## 预防

- 凡由 exec-log / SSE 轮询驱动的副作用，须对载荷做变更判定，禁止「每次成功响应都 bump UI」。
- 自绘红字错误须同步 `data-traceId`（元规则 24）；与 clone-log 修复（`38_...`）同一检查清单。
