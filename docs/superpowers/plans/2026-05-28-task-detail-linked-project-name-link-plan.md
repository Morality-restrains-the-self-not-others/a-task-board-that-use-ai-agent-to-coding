# Implementation Plan: 任务详情关联项目名称链接

> Design: `docs/superpowers/specs/2026-05-28-task-detail-linked-project-name-link-design.md`

## Task 1: 模板 — 项目名称 router-link

**File:** `task2app/front_project/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue`

- [ ] 将 L100–102 的 `<span>{{ tp.project_name }}</span>` 改为条件 `router-link`
- [ ] `to`: `/tenant/${tenantId}/projects/${tp.project_id}/`
- [ ] 添加 `data-testid="task-linked-project-name-link"`
- [ ] 保留徽章样式 + hover 可点击提示

## Task 2: Playwright 回归

**File:** `task2app/playwright/front_project/tests/TaskDetail.linked-project-name-link.playwright.test.js`

- [ ] 打开示例任务详情 URL（或测试 fixture）
- [ ] 断言链接 href 含 `/projects/{projectId}/`
- [ ] 点击后 URL 匹配项目详情路径

## Task 3: 验证

```bash
cd task2app/playwright/front_project && npx playwright test TaskDetail.linked-project-name-link.playwright.test.js
```
