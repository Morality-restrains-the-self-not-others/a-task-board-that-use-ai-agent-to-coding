# 项目编辑页：标签与自动运行（运行模版）设置设计

日期：2026-07-06  
状态：待批准

## 问题

用户访问项目编辑页  
`/tenant/{tenant}/projects/{id}/edit/`（示例：`861581450509701120`）时，无法配置：

1. **项目标签（tags）** — 后端 `Project.tags` 已存在且 API 可读写，编辑页未展示/提交。
2. **自动运行相关设置** — 任务级 `auto_run` 依赖项目级 `server_run_template`；详情页已有 `ProjectRunTemplatePanel`，**编辑页完全缺失**，用户须在详情页与编辑页之间来回切换。

## 目标

- 编辑页可编辑 **项目标签** 并随「保存更改」一并提交。
- 编辑页可配置 **项目运行模版**（`server_run_template`），与任务创建时的「是否自动运行」能力对齐。
- 保存成功后跳转项目详情，详情页能展示标签与运行模版摘要（只读）。

## 非目标

- 不在项目级新增 `default_auto_run` 字段（自动运行仍为**任务级** `Todo.auto_run`）。
- 不改动后端 API 契约（`ProjectSerializer` 已支持 `tags`、`server_run_template`）。
- 不在本次强制改造「创建项目」页（可作为后续一致性增量）。
- 不涉及架构变更（纯前端 UX 补齐，无新服务/组件拓扑）。

---

## 现状

| 能力 | 后端 | 项目详情页 | 项目编辑页 | 创建项目页 |
|------|------|-----------|-----------|-----------|
| `tags` | ✅ `ProjectSerializer.tags` | ❌ 未展示 | ❌ 未编辑 | ❌ |
| `server_run_template` | ✅ JSONField + 校验 | ✅ `ProjectRunTemplatePanel`（独立保存） | ❌ | ❌ |
| 任务 `auto_run` | ✅ `TodoSerializer` | — | — | ✅ `CreateTaskModal` 勾选 |

**编辑页当前字段**：名称、Git 仓库、描述、已安装镜像、关联工作空间（`ProjectEdit.vue`）。

**自动运行链路**（已有，编辑页缺前置配置入口）：

```
项目配置 server_run_template
  → 创建任务时勾选 auto_run（CreateTaskModal）
  → triggerTaskAutoRun 按模版启动云服务器（workPanelAutoRun.js）
```

后端校验（`todo_serializer.py`）：`auto_run=true` 时要求关联项目已配置运行模版且任务有已安装镜像。

---

## 方案（推荐）

### 1. 项目标签 — 新增 `ProjectTagsInput.vue`

轻量可复用组件（chip 输入）：

- 展示已有标签为可删除 badge。
- 输入框 + Enter / 逗号添加；失焦时 commit 当前输入。
- 规范化：`trim`、去重、单标签 max 64 字符、最多 20 个标签（前端软限制，与后端 `ListField` 一致）。
- `v-model` 绑定 `string[]`。

**编辑页集成**：

- `formData.tags: string[]`，`fetchProjectDetail` 从 API `tags` 填充（缺省 `[]`）。
- `updateProject` PUT body 增加 `tags: formData.tags`。

### 2. 自动运行相关 — 嵌入 `ProjectRunTemplatePanel`

在 `ProjectEdit.vue` 表单内、工作空间区块之后增加：

```vue
<ProjectRunTemplatePanel
  ref="runTemplatePanelRef"
  :tenant-id="..."
  :project-id="..."
  :project="projectSnapshot"
  :hide-actions="true"
/>
```

扩展 `ProjectRunTemplatePanel`（小改）：

- 新增 prop `hideActions?: boolean` — 为 true 时隐藏「保存运行模版 / 清除模版」按钮，由父表单统一保存。
- 已有 `defineExpose({ buildPayload })` 供父组件在 submit 时读取 `server_run_template`。

**统一保存**（单按钮「保存更改」）：

```javascript
const runTpl = runTemplatePanelRef.value?.buildPayload?.() ?? {}
await apiFetch(..., {
  method: 'PUT',
  body: JSON.stringify({
    ...formData.value,
    git_repos: trimmedGitRepos(),
    tags: normalizedTags(formData.value.tags),
    server_run_template: runTpl,
  }),
})
```

说明文案（面板上方或标签旁）：

> 配置运行模版后，在工作面板创建任务时可勾选「是否自动运行」，系统将按此模版自动启动服务器。

### 3. 项目详情页 — 只读补齐

`ProjectDetail.vue` 基本信息区增加：

- **项目标签**：有则 badge 列表，无则「未设置」。
- **运行模版摘要**：复用 `summarizeRunTemplate(project.server_run_template)`（与面板一致），避免用户保存后详情页看不到标签。

### 4. 错误与校验

- API 4xx：解析 `detail` / 字段错误，表单顶部 `role="alert"` 展示（对齐 `CreateProject.vue`）。
- 标签非法：前端拦截空串、超长；后端 ValidationError 回显。
- 运行模版：沿用 `normalize_server_run_template` 后端校验，前端不重复业务规则。

---

## 领域概念清单（供 DDD / 价值流）

| 类型 | 名称 | 说明 |
|------|------|------|
| 聚合根 | Project | 含 tags、server_run_template |
| 实体 | Todo | auto_run 依赖 Project 运行模版 |
| 值对象 | RunTemplate | 云平台/地域/硬件 preset |
| 领域事件 | — | 无新增 |

---

## 价值流影响

| 现有流 | 影响 |
|--------|------|
| `project-workspace` → `project-crud` | 扩展前端编辑步骤，覆盖 `projects_project.tags`、`projects_project.server_run_template` |
| `task-management` → 创建任务 + auto_run | 间接：编辑页可配置 auto_run 前置条件 |
| `project-detail-oauth-*` | 无冲突 |

**建议 value-stream.yaml 增量**（Step 3 落地）：

```yaml
- name: project-edit-tags-run-template
  status: planned
  test_file: ../playwright/front_project/tests/ProjectEdit.tags-run-template.playwright.test.js
  fields:
    - name: saas-backend.projects_project.tags
    - name: saas-backend.projects_project.server_run_template
```

---

## 测试计划

| 层级 | 内容 |
|------|------|
| 单元 | `ProjectTagsInput` 规范化/去重/上限；`updateProject` payload 含 tags + server_run_template |
| Playwright | 编辑页加载已有 tags；修改标签 + 选择运行模版 → 保存 → 详情页断言 |
| 回归 | 现有 `ProjectViewSet` PUT/PATCH 测试无需改（字段已覆盖） |

---

## 验收标准

1. 编辑页可见并可编辑项目标签。
2. 编辑页可见并可配置项目运行模版（与详情页能力等价）。
3. 单次「保存更改」同时持久化 tags 与 server_run_template。
4. 详情页展示 tags 与运行模版摘要。
5. 配置运行模版后，创建任务页「是否自动运行」对该项目可用（沿用现有 `canEnableAutoRun` 逻辑）。

---

## 架构变更影响

**无需更新 ArchiMate 架构** — 纯 SPA 表单补齐，调用既有 `Project Management API`，无新组件/服务/数据流。

---

## 开放问题（已自主决策）

| 问题 | 决策 |
|------|------|
| 编辑页运行模版单独保存还是合并保存？ | **合并保存**，减少双按钮困惑 |
| 是否在创建项目页同步加 tags？ | **本次不做**，编辑页优先；创建页列为 follow-up |
| 项目列表卡片是否展示 tags？ | **本次不做**，详情 + 编辑即可 |
