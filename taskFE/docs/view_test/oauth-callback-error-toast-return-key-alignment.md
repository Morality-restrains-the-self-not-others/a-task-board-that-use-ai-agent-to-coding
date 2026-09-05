# OAuth 回调失败 Toast — return_key 入口对齐

## 范围

- `CreateTaskModal.vue` 发起 OAuth 时携带 `return_key` + `setGithubAppReturnTarget`
- 与 `ProjectDetail.vue` / `TaskDetailLinkedProjectsPanel.vue` 行为一致

## 验收

创建任务弹窗内 OAuth 失败后回跳原页，触发与项目详情相同的 Toast 与 query 清理。
