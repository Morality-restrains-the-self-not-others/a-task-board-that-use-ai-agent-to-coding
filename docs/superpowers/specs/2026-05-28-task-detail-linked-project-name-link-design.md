# 任务详情「关联项目」项目名称可点击跳转设计

日期：2026-05-28  
状态：待批准

## 问题

用户在任务详情页（示例 URL：`/tenant/{tenant}/workspace/{workspace}/task-detail/{taskId}/?relayToTrae=true`）的「关联项目」区域看到项目名称，但名称仅为静态徽章，无法跳转到项目详情页查看项目配置、仓库与 OAuth 等信息。

## 目标

- 「关联项目」卡片中的**项目名称**可点击。
- 点击后在新路由打开对应**项目详情页**：`/tenant/{tenantId}/projects/{projectId}/`。
- 编辑任务模式（`isEditing=true`）下保持现有下拉选择行为，不改为链接。

## 非目标

- 不修改后端 API 或序列化字段（`taskProjectsWithDetails` 已含 `project_id`）。
- 不改变仓库行、OAuth、同步按钮等现有交互。
- 不在新标签页打开（与项目列表页 `Projects.vue` 一致，使用 SPA 内导航）。

## 现状

| 位置 | 行为 |
|------|------|
| `TaskDetailLinkedProjectsPanel.vue` L100–102 | 只读模式：`{{ tp.project_name }}` 渲染为 `<span>` 徽章 |
| `Projects.vue` | 项目名通过 `tenantPath + '/projects/' + project.id + '/'` 链到详情 |
| `router.js` | 路由 `project_detail`：`/tenant/:tenant/projects/:id/` |

`useTaskDetail.js` 的 `taskProjectsWithDetails` 已提供 `project_id`、`project_name`；组件 props 已有 `tenantId`。

## 方案（推荐）

### 前端：`TaskDetailLinkedProjectsPanel.vue`

1. 将只读模式下的项目名称 `<span>` 改为 `<router-link>`（或 `<a>` + `router.push`，优先 `router-link` 与项目列表一致）。
2. 链接目标：`/tenant/${tenantId}/projects/${tp.project_id}/`。
3. 样式：保留现有蓝色徽章视觉，增加 `hover:underline` / `hover:bg-blue-100` 等可点击 affordance；`cursor-pointer`。
4. 守卫：`tenantId` 与 `tp.project_id` 均非空时才渲染链接；否则回退为不可点击的 `<span>`（与现有一致）。
5. 辅助：`title="查看项目详情"`；可选 `data-testid="task-linked-project-name-link"` 供 E2E。

### 测试

1. **Playwright（新增或扩展）**：在任务详情页断言关联项目名称链接 `href` 含正确 `project_id`；点击后 URL 变为项目详情路径（可 mock 或使用已有 seed 数据）。
2. **可选单元测试**：纯函数 `buildProjectDetailPath(tenantId, projectId)` 若抽取则测路径拼接；若内联在模板则 E2E 足够。

## 价值流影响

受影响流（`value-stream.yaml`）：

- **项目与工作空间 / 项目详情与导航**：任务详情 → 项目详情交叉导航（前端 UX，无数据字段变更）。
- **任务协作 / task-detail UI**：关联项目展示增强。

| 维度 | 影响 |
|------|------|
| 数据字段 | 无 |
| 后端测试 | 无 |
| 前端测试 | 新增/扩展 Playwright |
| 跨流依赖 | 依赖既有 `project_detail` 路由与权限（`requiresAuth: true`） |

## 领域概念清单（轻量）

| 概念 | 说明 |
|------|------|
| Bounded Context | 任务协作 UI、项目与工作空间 |
| Entity | Task、Project（只读导航，无聚合变更） |
| Domain Event | 无 |

## 风险与约束

- **权限**：用户能打开任务详情即应能访问关联项目详情；若项目被删或无权访问，详情页现有错误处理兜底。
- **relayToTrae 查询参数**：跳转项目详情时不携带 `relayToTrae`（项目页无关参数）。
- **组件体积**：`TaskDetailLinkedProjectsPanel.vue` 已超 500 行；本改动仅替换一行模板 + 可选 helper，不触发进一步拆分。

## 验收标准

- [ ] 只读模式下，每个关联项目的名称显示为可点击链接。
- [ ] 点击后进入 `/tenant/{tenantId}/projects/{projectId}/`。
- [ ] 编辑模式下仍为项目下拉，无链接。
- [ ] Playwright 回归通过。

## 实施增量（供 Step 3 引用）

**Increment 1（唯一增量）**：关联项目名称 → 项目详情链接 + E2E。
