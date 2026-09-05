# associateWorkspace 501 + 报错无 data-traceId

## 现象

- 页面：项目详情「关联工作空间」→「添加关联」
- 文案：`endpoint not yet implemented in taskProjectService`
- DOM：`<p class="taskplugin-el-highlight">…</p>`，无 `data-traceId`
- 日志：`POST .../projects/{pid}/associateWorkspace/` → `status=501`（task-project-service）

## 根因

1. `taskProjectService` `handleProjectsRoute` 未登记 `associateWorkspace` 子路径，落入 501 stub。
2. `WorkspaceAssociation.vue` 失败时 `throw new Error(errorData.error)`，未把 `response.traceId` / 响应头 `X-Trace-Id` 挂到 Error，`modalService.alert(..., { traceId: err.traceId })` 为空。

## 修复

1. 实现 `handleAssociateWorkspace`（`associate_workspace.go`），路由 `case "associateWorkspace"`；501 stub 改为 `writeNotImplemented`（body 带 `trace_id`）。
2. 前端失败路径用 `extractTraceId(response|body)` + `showRequestError`。
3. 公网 SPA：`npm run build` 后若 `collected_static/assets/` 缺新 chunk，须 rsync `front_project/static/assets/` → `STATIC_ROOT/assets/`。

## 验收

```bash
curl -sS -X POST 'http://127.0.0.1:8016/api/tenant/<tid>/projects/<pid>/associateWorkspace/' \
  -H 'Content-Type: application/json' -H 'X-Trace-Id: t' -H 'X-Parent-Span-Id: aabbccddeeff0011' \
  -d '{"workspace_id":"<wid>"}'   # 期望 200，body 含 workspaces
cd taskProjectService && go test ./src/ -count=1 -run TestAssociateWorkspace
cd taskFE/app && npm test -- --run src/components/WorkspaceAssociation.traceId.test.js
```
