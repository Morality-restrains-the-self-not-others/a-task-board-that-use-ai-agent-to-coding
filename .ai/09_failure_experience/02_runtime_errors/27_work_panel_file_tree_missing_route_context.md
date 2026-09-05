# [运行时] work-panel 任务详情「缺少路由上下文，无法拉取文件树」

## 失败现象

在 `https://www.daydaymoney.com/tenant/{tenant}/work-panel/` 打开任务详情弹窗，选中可写层后，项目文件树区域出现红色错误：

```text
缺少路由上下文，无法拉取文件树
```

对应 DOM：`[data-testid="project-file-tree-error"]`。独立任务详情页（URL 含 `/workspace/.../task-detail/...`）通常正常。

## 环境与上下文

- 组件：`TaskDetailProjectFileTree.vue`
- 入口：`WorkPanel.vue` 模态框挂载 `TaskDetailContent`（非任务详情路由）
- 工作面板路由形态：`/tenant/:tenant/work-panel/`（**无** `workspace` / `taskId` 路由参数）
- 父链已解析并下传：`WorkPanel` → `TaskDetailContent.logic`（`resolvedTenantId/WorkspaceId/TaskId`）→ `TaskDetailCommentsSection`

## 排查过程

1. 错误文案由 `fetchFiles` 在 `!tenantId || !workspaceId || !taskId` 时写入
2. 组件仅从 `useRoute().params` 取上下文，忽略父组件已有的 props
3. work-panel 弹窗下 `route.params` 只有 `tenant`，`workspace`/`taskId` 为空 → 永不发起 `container-layer-files` 请求
4. 同目录 `TaskDetailGithubPrCredentialPanel` / `TaskDetailTaskIdentityPanel` 已采用「props 优先、route 回退」

## 根因

文件树（及同面板的 `TaskDetailExecLayerChanges`）把「任务 API 路径所需的 tenant/workspace/taskId」绑定在 vue-router 参数上，未适配 work-panel 弹窗这种「路由不完整、上下文由父组件注入」的嵌入场景。

## 解决方案

1. `TaskDetailProjectFileTree` / `TaskDetailExecLayerChanges` 增加 `tenantId` / `workspaceId` / `taskId` props，`requestContext` 优先 props、再 route
2. `TaskDetailCommentsSection` → `TaskDetailTaskLayerAssociationPanel` 向下透传上述 ID
3. watch 依赖包含 requestContext，避免 props 晚于 layerId 就绪时漏拉
4. 单测：`TaskDetailProjectFileTree.route-context.unit.test.js`（work-panel 形态路由 + props）

## 预防措施

- 任务详情子组件拼装 `/api/tenant/.../workspace/.../task/...` 时，不得只读 `route.params`；须与 `TaskDetailContent.logic` 的 resolved IDs 对齐（props 优先）
- 新增依赖任务路径的子面板时，补一条「route 仅有 tenant、靠 props」的回归测例
- 公网 SPA 改动后执行 `front_project/app/scripts/runall-lifecycle.sh build`（含 collectstatic）

## 后续加固（2026-07-17）

任务详情链路已统一经 `front_project/app/src/utils/resolveTaskRouteIds.js` 解析 IDs，覆盖：

- `TaskDetailContent.logic` / `useTaskDetail` / `useServerConfigHardwarePanel`
- 文件树、层变更、身份面板、关联项目、GitHub PR 凭证、ApplyPatch、LLM 预算、评论面板、遗留 `ServerConfig.vue`

禁止在任务详情内再手写 `props.x || route.params.x`。回归：`resolveTaskRouteIds.test.js`、`TaskDetailContent.route-ids.unit.test.js`、`TaskDetailProjectFileTree.route-context.unit.test.js`。
