# [运行时] 任务详情 clone-log 报错「Invalid or missing access token」缺少 data-traceId

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- 页面：`https://www.daydaymoney.com/tenant/.../workspace/.../task-detail/.../`
- 可见错误：`Invalid or missing access token`（onlineServiceJS 容器侧 401 `detail`）
- 承载元素：`p.text-xs.text-red-600.mb-1`（`TaskDetailExecCloneLogSection`）
- **无** `data-traceId`，无法从 DOM 一键对齐后端 `trace_id` 日志

## 根因

1. `refreshZTreeExecutionLog`（`taskDetailExecLog.js`）在 `container-clone-log` / `container-job-execution-log` 非 2xx 时只把 `j.detail` 写入 `layerCloneLogFetchError` / `layerJobLogFetchError`。
2. `apiFetch` 已在响应上挂 `response.traceId`，但拉取路径未读取、未下传。
3. 错误 `<p>` 模板未绑定 `data-traceId`（与元规则 24、NestedReposCloneStatus 等已对齐路径不一致）。

## 解决方案

- 失败结果携带 `traceId`（`extractTraceId(r)` / 响应体）。
- 新增 `layerCloneLogFetchErrorTraceId` / `layerJobLogFetchErrorTraceId`，经 Comments / LayerAssociation / ExecLog 组件链下传。
- `TaskDetailExecCloneLogSection` / `TaskDetailExecJobBody` 错误节点：`:data-traceId="… || undefined"`。
- 单测：`taskDetailExecLog.test.js`、`TaskDetailExecCloneLogSection.traceId.test.js`。

## 预防

- 凡把 API `detail` 写到 inline 错误文案，必须同步保存并挂载该次请求的 `data-traceId`。
- 新增错误 UI 时优先复用 toast/modal/`requestErrorDisplay` 统一出口；自绘 `<p class="…text-red…">` 须自检 `[data-traceId]`。

## 验证

```bash
cd taskFE/app && npx vitest run \
  src/composables/taskDetail/taskDetailExecLog.test.js \
  src/components/task-detail/TaskDetailExecCloneLogSection.traceId.test.js
```

公网生效：`bash scripts/runall-lifecycle.sh build`（build + collectstatic）。
