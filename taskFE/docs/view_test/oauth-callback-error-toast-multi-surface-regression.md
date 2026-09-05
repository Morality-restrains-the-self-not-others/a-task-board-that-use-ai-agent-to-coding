# OAuth 回调失败 Toast — 多入口回归

## 入口

| 页面 | 组件 |
|------|------|
| 项目详情 | `ProjectDetail.vue` |
| 任务详情 | `TaskDetailLinkedProjectsPanel.vue` |
| GitHub PR 凭据 | `TaskDetailGithubPrCredentialPanel.vue` |
| 创建任务 | `CreateTaskModal.vue` |

## 验收

各入口 OAuth 失败回跳后均有 Toast；设置页仍仅内联。
