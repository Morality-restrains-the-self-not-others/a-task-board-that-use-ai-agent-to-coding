# 实施计划：导航栏任务搜索过滤

- **日期**: 2026-07-20
- **设计**: `docs/superpowers/specs/2026-07-20-navbar-task-search-filter-design.md`

## 任务清单

### A. 意图与文档

- [x] A1. 写入 `docs/intents/frontend/navbar_task_search_filter.intent.md` + `.test-intent.md`
- [x] A2. 更新 `docs/flows/value-stream-test-integration.wsd`（导航栏任务搜索测试点）

### B. 后端（taskTaskService）

- [x] B1. RED：`TestSearchTasks` / 新测覆盖 owner、assignee LIKE、`assignee_ids`
- [x] B2. GREEN：扩展 `searchTasksInWorkspaces` + `handleSearchTasks` 解析 `assignee_ids`、丰富响应字段
- [x] B3. 复跑 Go 测试通过

### C. 前端

- [x] C1. `utils/navbarTaskSearch.js`：构建 query、解析跳转 URL
- [x] C2. `components/NavbarTaskSearch.vue`（UI）+ 接入 `Navbar.ui.vue` / `Navbar.logic.vue`
- [x] C3. WorkPanel：`task_id` query → 打开详情（抽 composable，避免继续堆行）
- [x] C4. Vitest：`NavbarTaskSearch` / util / deep-link
- [x] C5. 错误展示带 `data-traceId`

### D. 验证与交付

- [x] D1. Go + Vitest 通过
- [x] D2. Review + Log/Intent 审计
- [x] D3. PR（不直接 merge）— taskTaskService#7、task2app#47

## 事件契约

无新 publish；查询路径书面例外见设计文档。
