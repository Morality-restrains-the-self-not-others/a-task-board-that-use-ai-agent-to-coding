# 设计文档：工作空间切换器统一到导航栏

**日期**: 2026-06-28  
**状态**: 待审批  
**类型**: 前端 UX 优化

---

## 1. 问题分析

### 现状

当前 `/tenant/:tenant/work-panel/` 页面存在工作空间切换器的视觉重叠问题：

| 位置 | 文件 | 说明 |
|------|------|------|
| 工作面板 Header | `WorkPanelHeader.vue:12-18` | 包含 `<WorkspaceSwitcher>` 组件 |
| 导航栏 (Navbar) | `Navbar.ui.vue` | **当前仅有公司切换器**，无工作空间切换器 |

### 根因

用户反馈页面中看到两个工作空间切换器重叠。经代码审查，`WorkspaceSwitcher` 仅在 `WorkPanelHeader.vue` 中使用，Navbar 中并无此组件。用户可能将 Navbar 中的公司切换器（`<select>` 元素，`Navbar.ui.vue:56-69`）误认为工作空间切换器，或页面渲染出现了意外的视觉重叠。

### 用户期望

1. **去除工作面板内的工作空间切换器** — 从 `WorkPanelHeader.vue` 中移除
2. **将工作空间切换器放到导航栏** — 作为全局导航元素，不在页面内容区重复
3. **切换后工作空间 ID 放到 URL 参数** — `?workspace_id=<id>` 已由 `WorkspaceSwitcher.vue:156` 实现

---

## 2. 设计方案

### 2.1 组件迁移

```
变更前:                          变更后:
┌──────────────────────┐        ┌──────────────────────────────────┐
│ Navbar               │        │ Navbar                           │
│  [公司切换器]         │        │  [工作面板] [工作空间切换器] [公司切换器] [用户] │
└──────────────────────┘        └──────────────────────────────────┘
┌──────────────────────┐        ┌──────────────────────┐
│ WorkPanelHeader      │        │ WorkPanelHeader      │
│  工作面板 [工作空间切换器] │  →     │  工作面板 [创建任务] [筛选] │
│  [创建任务] [筛选]    │        │                      │
└──────────────────────┘        └──────────────────────┘
```

### 2.2 具体变更

#### A. Navbar.ui.vue — 添加 WorkspaceSwitcher

在现有公司切换器 `<select>` 之前插入 `<WorkspaceSwitcher>`：

```vue
<!-- 工作空间切换器：不同工作空间对应不同项目/任务面板 -->
<WorkspaceSwitcher
  v-if="tenant"
  :tenant="tenant"
  :refresh-trigger="workspaceRefreshTrigger"
  @workspace-switched="/* 组件内部已处理 URL 更新 + 页面刷新 */"
/>
```

需要新增 props:
- `tenant` (String) — 当前租户 ID
- `workspaceRefreshTrigger` (Number) — 刷新触发器

#### B. Navbar.logic.vue — 传递 props

- `tenant` 已有：`currentTenant.value`（从 `/api/user/{id}/accounts/users/me/` 获取）
- `workspaceRefreshTrigger`：新增 ref，初始 0

#### C. WorkPanelHeader.vue — 移除 WorkspaceSwitcher

移除第 12-18 行的 `<WorkspaceSwitcher>` 及相关 import 和 emit 声明。

#### D. WorkPanel.vue — 移除 workspace-switched 事件传递

`WorkPanelHeader` 的 `@workspace-switched` emit 链不再需要。`WorkPanel.initData()` 在页面加载时已从 URL `?workspace_id=` 读取工作空间 ID，页面刷新后自动正确初始化。

### 2.3 数据流（变更后）

```
用户点击 Navbar 工作空间切换器
  → WorkspaceSwitcher.switchToWorkspace(id)
    → POST /api/tenant/{tid}/projects/switch/ { workspace_id }
    → 成功后: URL 设置 ?workspace_id={id}
    → window.location.href 整页刷新
      → WorkPanel.initData()
        → 从 URL 读取 workspace_id
        → 初始化工作面板数据
```

关键点：`WorkspaceSwitcher` 内部已通过 `window.location.href` 做全页刷新，Navbar 不需要额外处理事件 — 页面刷新后一切从 URL 参数重建。

### 2.4 WorkspaceSwitcher 显示条件

- **仅在用户已登录且有租户上下文时显示**（`v-if="tenant"`）
- 在系统管理员视图（`isSuperuser`）下**不显示**，因为管理员使用 `/system-admin/` 而非工作面板
- 与现有公司切换器的显示逻辑一致

---

## 3. 影响文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `front_project/app/src/components/Navbar.ui.vue` | 修改 | 添加 WorkspaceSwitcher 组件 |
| `front_project/app/src/components/Navbar.logic.vue` | 修改 | 传递 tenant、workspaceRefreshTrigger props |
| `front_project/app/src/views/WorkPanelHeader.vue` | 修改 | 移除 WorkspaceSwitcher |
| `front_project/app/src/views/WorkPanel.vue` | 可能修改 | 清理 workspace-switched 事件链（若不需保留） |
| `front_project/app/src/components/WorkspaceSwitcher.vue` | 不变 | 已支持 URL 参数，无需改动 |

---

## 4. 领域概念清单

| 概念 | 类型 | 所属上下文 |
|------|------|-----------|
| Workspace（工作空间） | 实体 | 项目与工作空间 |
| Tenant/Company（租户/公司） | 实体 | 组织与成员 |
| Workspace Switch | 领域服务 | 项目与工作空间 |
| Navbar | UI 组件（非领域概念） | 前端展示层 |

---

## 5. 价值流影响分析

### 受影响的价值流

| 价值流 | 步骤 | 影响 |
|--------|------|------|
| `project-workspace` | `switch-workspace` | UX 入口从 WorkPanelHeader 移至 Navbar，后端 API 不变 |
| `people-management` | `join-team` | `PeopleJoin.vue` 跳转 `/tenant/{id}/work-panel/` 后，Navbar 中的 WorkspaceSwitcher 加载新公司工作空间列表 — 行为不变 |

### 影响评估

- **后端 API**: 无变化（`/api/tenant/{tid}/projects/switch/` 不变）
- **数据库字段**: 无变化
- **测试文件**: 无需更新后端测试；前端 Playwright 测试可能需要更新选择器（若存在相关 E2E 测试）
- **新增价值流**: 不需要
- **状态变更**: 无

---

## 6. 风险评估

| 风险 | 概率 | 缓解措施 |
|------|------|---------|
| Navbar 中 tenant 未就绪时 WorkspaceSwitcher 提前渲染 | 低 | `v-if="tenant"` 守卫 |
| 页面刷新导致 Navbar 状态丢失 | 无风险 | 设计如此 — 全页刷新后重新初始化 |
| WorkPanel 中其他页面（如 task-detail）也使用了 WorkspaceSwitcher | 无 | 经搜索仅 `WorkPanelHeader.vue` 引用 |

---

## 7. 未涉及范围

- 不改变 WorkspaceSwitcher 的 API 调用逻辑
- 不改变后端 workspace switch 接口
- 不改变 URL 参数命名或格式（保持 `?workspace_id=`）
- 不涉及移动端适配（WorkspaceSwitcher 本身为响应式组件）
