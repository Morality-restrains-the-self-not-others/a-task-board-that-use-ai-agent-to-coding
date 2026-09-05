# 任务详情：项目文件树与变动列表 Tab

## 意图

任务详情「任务关联」里，选中可写层后，「项目文件树（所有拉取仓库）」与「文件变动列表」由上下叠放改为 **Tab 切换**：默认「文件变动」，另一 Tab 展示同一层的项目文件树。

## 背景

- 页面：`/tenant/{tenantId}/workspace/{workspaceId}/task-detail/{taskId}/`
- 锚点：评论执行细节 → 任务关联 → `TaskDetailProjectFileTree` 的 summary「项目文件树（所有拉取仓库）」
- 旧布局：`TaskDetailExecLayerChanges` 与 `TaskDetailProjectFileTree` 纵向并列，占高度且需分别展开 `<details>`

## 方案

1. 新组件 `TaskDetailLayerFilesTabs`：`role="tablist"`，两 Tab「项目文件树」「文件变动」；后者在 `displayCount>0` 时显示 `· N`
2. 默认选中「文件变动」；切换层时回到文件变动 Tab
3. 面板用 `v-show` 切换，避免卸载导致树展开/预览/变动选中丢失
4. **不新增 API / 事件**：变动数据仍来自既有 `selectedZTreeLayerChangesPanel`；提交/刷新/续拉事件原样上抛
5. Tab 点击为本地 UI 状态，无 HTTP；Anti-Replay-OK: 只读 Tab

## 变更相对旧版

| 项 | 旧 | 新 |
| --- | --- | --- |
| 布局 | 变动列表与文件树上下叠放 | 同区域 Tab 切换 |
| 默认可见 | 两者均可展开占位 | 默认文件变动；文件树在另一 Tab |
| 数据 | 层 diff / 文件树 API | 不变 |

## 业务意图 → 事件对照

**无对应事件**：纯前端展示切换，无平台业务状态变更。

| 业务意图 | 事件名 | 例外理由 |
|---------|--------|---------|
| 切换项目文件树 / 文件变动 Tab | — | 只读 UI 状态 |
| 展示层变动列表 | — | 复用既有查询载荷，无新写路径 |
