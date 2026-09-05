# Code Review: 任务详情关联项目名称链接

日期：2026-05-28  
结论：**通过**

## 变更摘要

- `TaskDetailLinkedProjectsPanel.vue`：只读模式下项目名改为 `router-link` 跳转项目详情。
- 新增 Playwright `TaskDetail.linked-project-name-link.playwright.test.js`。
- 更新 `TaskDetail.repo-list-shows-only-associated-projects.playwright.test.js` 选择器。

## 检查项

| 项 | 结果 |
|----|------|
| 与设计一致 | ✅ |
| 编辑模式未受影响 | ✅ |
| 无后端/API 变更 | ✅ |
| 路由与 Projects.vue 一致 | ✅ |
| 降级路径（缺 ID） | ✅ span 回退 |
| 测试 | 见 CI / 本地 Playwright |

## 备注

- 无 critical/important 问题。
