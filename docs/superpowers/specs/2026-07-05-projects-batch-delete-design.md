# 设计文档：项目列表页 — 批量删除项目

**日期**：2026-07-05  
**状态**：已实现  
**迭代**：项目列表批量删除  
**关联页面**：[项目列表](http://183.250.1.132:4000/tenant/850256677331562496/projects/)（`Projects.vue`）

---

## 1. 背景与问题

### 1.1 现状

| 能力 | 位置 | 行为 |
|------|------|------|
| 项目列表 | `Projects.vue` | 卡片网格，点击整卡跳转详情；支持搜索筛选 |
| 单个删除 | `ProjectDetail.vue` | `DELETE /api/tenant/{tid}/projects/{id}/`，使用浏览器 `confirm()` |
| 后端 | `ProjectViewSet`（ModelViewSet） | 已有 `destroy`；`get_queryset` 限定用户所属租户 |

租户内项目较多时（例如 GitLab 同步批量创建后），用户需逐个进入详情页删除，效率低且无批量操作入口。

### 1.2 诉求

在**项目列表页**提供**批量删除**：勾选多个项目 → 确认 → 一次性删除，并刷新列表。

### 1.3 架构上下文（只读）

- 当前架构基线 **v5 ✅ shipped**；本次为前端交互 + 单 API 扩展，**不更新** `docs/architecture/`。
- 删除仍走 Django `ProjectViewSet` 同租户权限边界；无新增微服务。

---

## 2. 目标与非目标

### 2.1 目标

1. 列表页进入「选择模式」，可多选当前租户下可见项目。
2. 提供「批量删除（N）」按钮，二次确认后执行删除。
3. 新增批量删除 API，返回 `{ deleted, errors }` 结构化结果。
4. 删除成功后刷新列表；部分失败时展示摘要，保留选择模式便于重试。
5. 单元测试 + Playwright（Mock API）覆盖主路径。
6. 确认弹窗使用**自定义 Modal**，禁止 `alert`/`confirm`（遵循前端规范）。

### 2.2 非目标

- 不改造项目详情页的单删交互（可后续统一 Modal，本迭代不强制）。
- 不做软删除 / 回收站。
- 不限制「有关联任务的项目不可删」（与现有单删 CASCADE 行为一致；若单删能删，批量也能删）。
- 不新增独立权限角色（与现有「租户成员可删可见项目」一致）。

---

## 3. 领域概念（供 DDD）

| 概念 | 说明 |
|------|------|
| Aggregate | `Project`（租户内） |
| 领域服务 | `BatchDeleteProjectsService` — 校验 ID 归属 + 批量 destroy |
| 值对象 | `ProjectId`（Snowflake string 传输） |

---

## 4. 价值流影响

| 流 | 步骤 | 影响 |
|----|------|------|
| `project-workspace` / `project-crud` | 项目 CRUD | 新增 batch-delete 步骤与测试 |
| 字段 | `saas-backend.projects_project.id` | 批量 DELETE 操作 |

---

## 5. 交互设计

### 5.1 模式切换

工具栏新增 **「选择」** 按钮（`data-testid="projects-select-mode-btn"`）：

| 状态 | 行为 |
|------|------|
| 默认 | 卡片为 `<a>` 链接，点击进详情 |
| 选择模式 | 卡片改为 `<div>` + 左上角 checkbox；点击卡片切换勾选（不跳转） |

再次点击 **「取消选择」** 或批量删除完成后退出选择模式并清空勾选。

### 5.2 选择范围

- **全选**：工具栏 checkbox「全选当前列表」—— 仅作用于 **筛选后** 的 `filteredProjects`（与搜索框一致）。
- 勾选状态按 `project.id` 存储；切换搜索词后，已选但不可见的 ID 仍保留，工具栏显示「已选 N 个（当前可见 M 个）」可选简化文案为「已选 N 个」。

### 5.3 批量删除按钮

选择模式下，工具栏显示：

```
[ 取消选择 ]  [ 批量删除 (N) ]   （N = 已选数量，0 时 disabled）
```

- 样式：`btn-secondary` + 危险色边框/文字（如 `text-red-600 border-red-200`），与「创建项目」主按钮区分。
- `data-testid="projects-batch-delete-btn"`

### 5.4 确认弹窗 `BatchDeleteProjectsModal.vue`

独立组件，props：

- `show`, `projectNames: string[]`, `deleting`, `error`, `result`

内容示例：

> 确定删除以下 **3** 个项目吗？此操作不可恢复。  
> · valuestream  
> · other-repo  
> · …（最多展示 5 条，超出显示「等 N 个项目」）

按钮：**取消** / **确认删除**（删除中 disabled +「删除中…」）。

### 5.5 结果反馈

| 结果 | UI |
|------|-----|
| 全部成功 | 关闭确认框 + 退出选择模式 + toast/绿色条「已删除 N 个项目」+ `loadProjects()` |
| 部分失败 | 确认框内或列表上方展示 `errors`；已删项从勾选移除；保留失败项选中 |
| 全部失败 | 展示错误，不退出选择模式 |

列表页顶部可用现有绿色条样式（与 GitLab 同步 `createResult` 类似）展示摘要。

---

## 6. API 设计

### 6.1 端点

```
POST /api/tenant/{tenant_id}/projects/batch-delete/
```

**Request**

```json
{
  "project_ids": ["848537693488873472", "848537693488873473"]
}
```

**Response 200**

```json
{
  "deleted": [
    { "id": "848537693488873472", "name": "valuestream" }
  ],
  "errors": [
    { "id": "999", "error": "项目不存在或无权访问" }
  ]
}
```

- `project_ids` 为空 → **400** `{ "error": "project_ids 不能为空" }`
- 权限：复用 `_resolve_tenant_company` + `get_queryset` 语义，仅删除用户在该租户下可见的项目
- 实现：逐 ID `get` + `delete()`；单条失败记入 `errors`，不中断其余（与 GitLab batch 创建一致）
- 日志：`logger.info("batch_delete_projects tenant=%s deleted=%s errors=%s", ...)`

### 6.2 与单删关系

保留现有 `DELETE .../projects/{pk}/`；批量端点内部可调用相同 queryset 过滤，避免重复权限逻辑。

### 6.3 Swagger

在 Swagger 中登记新端点（路径、body schema、响应 schema、4xx 说明）。

---

## 7. 前端实现结构

| 文件 | 职责 |
|------|------|
| `composables/useProjectsBatchDelete.js` | 选择模式 state、`selectedProjectIds`、`toggleSelectAll`、`executeBatchDelete` |
| `components/BatchDeleteProjectsModal.vue` | 确认 UI |
| `views/Projects.vue` | 接入 composable、卡片 checkbox、工具栏按钮 |

**`executeBatchDelete` 流程**

1. `POST batch-delete` with selected IDs  
2. 解析 `deleted` / `errors`  
3. 调用 `onDeleted` → `loadProjects()`  
4. 从 `selectedProjectIds` 移除已成功删除的 ID  

---

## 8. 卡片结构变更（示意）

选择模式下，卡片由：

```html
<a :href="..."> ... </a>
```

变为：

```html
<div role="button" tabindex="0" @click="toggleProject(id)" @keydown.enter="...">
  <input type="checkbox" :checked="selected" @click.stop @change="..." />
  ...原有内容（详情链接改为 span 或次要链）...
</div>
```

可选：保留卡片内小字「查看详情」链， `@click.stop` 避免触发行选择。

---

## 9. 测试计划

### 9.1 后端 `test_projects_batch_delete_api.py`

- 认证用户批量删除 2 个项目 → `deleted.length === 2`，DB 不存在  
- 含无权限/不存在 ID → `errors` 有记录，合法项仍删除  
- 空数组 → 400  
- 未认证 → 401/403  

### 9.2 前端 Vitest

- `useProjectsBatchDelete`：toggle、全选 filtered、API mock 后清空选择  

### 9.3 Playwright `Projects.batch-delete.playwright.test.js`

- Mock `batch-delete` → 选择模式 → 勾选 2 卡 → 确认 → 列表刷新（或 mock 后断言请求体）  
- `project_ids` 与勾选一致  

---

## 10. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 误删 | 二次确认 + 列出项目名称 |
| 删除含 TaskProject 关联的项目 | 与单删相同 CASCADE；确认文案注明「不可恢复」 |
| 卡片 `<a>` 改结构影响 E2E | 新增 `data-testid="project-card-{id}"` |
| 列表页超 500 行 | composable + modal 独立文件，符合前端拆分规范 |

---

## 11. 实施清单（批准后）

- [ ] `utility_views.py` + `urls.py` — `batch_delete_projects_view`
- [ ] `BatchDeleteProjectsModal.vue`
- [ ] `useProjectsBatchDelete.js`
- [ ] `Projects.vue` — 选择模式 UI
- [ ] 后端 / Vitest / Playwright 测试
- [ ] Swagger 同步

---

## 12. 架构变更影响

**无** — 不新增 Application 组件；Django 项目模块内 API + Vue 列表页增强。

---

## 13. 关键决策摘要

| 决策 | 选择 | 理由 |
|------|------|------|
| 新 API vs 循环 DELETE | **新 batch-delete API** | 单次确认、结构化 partial result、减少往返 |
| 选择模式 | **显式切换** | 避免误触 checkbox；与日常浏览进详情互不干扰 |
| 确认 UI | **独立 Modal 组件** | 符合项目禁止原生 confirm 规范 |
| 权限 | **与 list queryset 一致** | 不引入新角色模型 |

---

**批准后可进入 `/8-build-构建` 实现。**
