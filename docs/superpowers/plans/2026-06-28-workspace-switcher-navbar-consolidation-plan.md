# 实施计划: Workspace Switcher Navbar Consolidation

> 输入:
> - 设计文档: `docs/plans/2026-06-28-workspace-switcher-navbar-consolidation.md`
> - 价值流: `docs/superpowers/plans/2026-06-28-workspace-switcher-navbar-consolidation-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-28-workspace-switcher-navbar-consolidation-nfr-clarification.md` (全部跳过)
> - DDD: `docs/superpowers/plans/2026-06-28-workspace-switcher-navbar-consolidation-ddd.md` (跳过)
>
> 类型: 纯前端组件迁移 | 增量数: 1 | 预估文件数: 3-4

---

## 增量 1: 将 WorkspaceSwitcher 迁移至 Navbar（唯一增量）

### 任务清单

#### Task 1: Navbar.logic.vue — 添加 workspaceRefreshTrigger 并传递 tenant

- [ ] 在 `<script setup>` 中新增 `workspaceRefreshTrigger = ref(0)`
- [ ] 在模板中将 `workspaceRefreshTrigger` 和 `tenant` 传递给 `NavbarUI`
- [ ] `tenant` prop 值取自 `currentTenant`（已有）

**文件:** `task2app/front_project/app/src/components/Navbar.logic.vue`
**验证:** 组件渲染不报错，tenant 正确传递

#### Task 2: Navbar.ui.vue — 添加 WorkspaceSwitcher 组件

- [ ] Import `WorkspaceSwitcher` from `../components/WorkspaceSwitcher.vue`
- [ ] 在模板中公司切换器 `<select>` 之前插入 `<WorkspaceSwitcher>`，条件：仅当用户非超管且有 tenant 时显示
- [ ] 新增 props: `tenant` (String), `workspaceRefreshTrigger` (Number)
- [ ] 绑定: `<WorkspaceSwitcher v-if="tenant && !currentUser.isSuperuser" :tenant="tenant" :refresh-trigger="workspaceRefreshTrigger" />`

**文件:** `task2app/front_project/app/src/components/Navbar.ui.vue`
**验证:** Navbar 渲染工作空间切换器，下拉菜单可展开，列出工作空间

#### Task 3: WorkPanelHeader.vue — 移除 WorkspaceSwitcher

- [ ] 移除 `<WorkspaceSwitcher>` 模板节点（第 12-18 行）
- [ ] 移除 `import WorkspaceSwitcher` 语句（第 27 行）
- [ ] 移除 emit 声明中的 `workspace-switched` 和 `workspace-created`（若仅用于 WorkspaceSwitcher）
- [ ] 移除 `refreshTrigger` prop（若仅用于 WorkspaceSwitcher）

**文件:** `task2app/front_project/app/src/views/WorkPanelHeader.vue`
**验证:** 工作面板 Header 中不再显示工作空间切换器，"创建任务"和"筛选"按钮正常

#### Task 4: WorkPanel.vue — 清理 workspace-switched 事件链（按需）

- [ ] 检查 `WorkPanel.vue` 中对 `WorkPanelHeader` 的 `@workspace-switched` 和 `@workspace-created` 绑定
- [ ] 若 `handleWorkspaceSwitched` 仅用于 WorkspaceSwitcher 路径，可保留（页面刷新后这些处理函数不会被触发，但保留无害）；若需要清理则移除相关绑定
- [ ] 确认 `initData()` 在页面刷新时从 URL `?workspace_id=` 正确读取工作空间 ID（已有逻辑，不需改）

**文件:** `task2app/front_project/app/src/views/WorkPanel.vue`
**验证:** 页面刷新后工作面板正确初始化

#### Task 5: 前端构建验证

- [ ] 运行 `npm run build` 确认无编译错误
- [ ] 可选：在浏览器中手动验证切换流程

**文件:** 无新文件
**验证:** `npm run build` 零错误退出

---

## 依赖关系

```
Task 1 (Navbar.logic) ──┐
                         ├──> Task 4 (WorkPanel cleanup) ──> Task 5 (build verify)
Task 2 (Navbar.ui) ─────┤
                         │
Task 3 (WorkPanelHeader) ┘
```

Task 1-3 可并行执行（修改不同文件），Task 4 依赖前三个任务完成后的上下文确认，Task 5 是最终验证。

## 回滚方案

若出现问题，反向操作即可：
1. 从 Navbar 移除 WorkspaceSwitcher
2. 恢复 WorkPanelHeader 中的 WorkspaceSwitcher
3. 所有变更限于 3-4 个前端 Vue 文件，无数据库迁移
