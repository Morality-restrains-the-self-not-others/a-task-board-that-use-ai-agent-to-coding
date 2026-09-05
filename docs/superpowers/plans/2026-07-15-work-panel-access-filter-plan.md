# 实施计划：工作面板按可访问人或小组过滤

- 日期：2026-07-15
- 设计：`docs/superpowers/specs/2026-07-15-work-panel-access-filter-design.md`

## Task 1 — 过滤纯函数（红→绿）

- [ ] 新增 `task2app/front_project/app/src/utils/workPanelAccessFilter.js`
- [ ] 新增 `workPanelAccessFilter.test.js`（T1–T7）
- [ ] 命令：`cd task2app/front_project/app && npm test -- workPanelAccessFilter`

## Task 2 — Header UI

- [ ] `WorkPanelHeader.vue`：人/小组分段 + 下拉 + chip
- [ ] `data-alias`：`access-filter-toggle`、`access-filter-person-option`、`access-filter-group-option`、`access-filter-chip`
- [ ] props/emits：`accessFilter`、`accessSubjects`、`access-filter-select`、`clear-access-filter`

## Task 3 — WorkPanel 接线

- [ ] 加载 `workspace-permissions`，解析人/小组选项
- [ ] 选组时 GET `accounts/groups/{id}/members/`，经 collaborators 映射 memberIds
- [ ] `filteredTodos`：先/后接 `filterTodosByAccess`（与机器过滤 AND）
- [ ] 切 workspace 清除 accessFilter

## Task 4 — Playwright

- [ ] `WorkPanel.accessFilter.playwright.test.js`（mock permissions/members）
- [ ] 覆盖 T8–T10

## Task 5 — 意图勾选与文档

- [ ] 更新 `009` intent 验收勾选
- [ ] 若有 value-stream 图测试点，按需补一行（本功能前端过滤，可标注于 work_panel 相关流）

## 验证清单

```bash
cd task2app/front_project/app && npm test -- workPanelAccessFilter
# Playwright（环境可用时）
cd task2app/playwright/front_project && npx playwright test WorkPanel.accessFilter
```

## 事件契约

无（纯前端例外，见 domain 文档）。
