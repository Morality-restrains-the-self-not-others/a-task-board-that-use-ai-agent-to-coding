# 设计：Chrome 插件项目列表标注是否可自动运行

- **Date:** 2026-08-27
- **Status:** accepted（`/goal` 零交互）
- **Architecture artifacts:** 非架构变更（无新服务/API/事件；只读渲染既有 GET 字段）

## 🕸️ Code Review Graph 分析

CRG `update --brief` 已执行。本增量不改后端符号。既有调用链：

- 浮窗 `content.js#loadProjects` → `swApi('getProjects')` → `API.getProjects` → 约定路径 workspace 项目列表
- DevTools `panel/lib/workspace.js#loadProjects` / `renderProjectCheckboxes` 同一 API
- 门禁语义与 `taskFE` `useCreateTaskAutoRun.projectAllowsAutoRun`、`taskTaskService` `default_auto_run` 一致

## 当前架构理解（裁剪）

本需求不改企业景观。应用层仍是 taskChromePlugin 消费 taskProjectService 项目列表；`server_run_template` 已在列表响应中返回。

## 方案（选定）

**纯函数标注，不新增请求。**

1. SSOT：`lib/project-auto-run-label.js`
   - `projectAllowsAutoRun(project)`：`server_run_template` 为普通对象且 `default_auto_run === true`
   - 徽章文案：`可自动运行` / `不可自动运行`
   - `renderProjectCheckboxCaptionHtml(project, esc)`：转义后的名称 + `<span class="taskplugin-project-auto-run" data-auto-run="true|false">`
2. 浮窗与面板复选框在名称后插入该 span；`value` 仍为项目 id。
3. 样式：允许为绿色弱强调，不允许为 muted 灰；窄列表不换行挤掉复选框。

### 未采用

| 方案 | 拒绝原因 |
|------|----------|
| 为每个项目再 GET 详情 | 已有列表字段；违反无后台轮询/多余请求 |
| 标注完整启机门禁（镜像+模版齐） | 产品问的是项目是否允许自动运行，与项目设置开关对齐 |
| 按标注禁用自动运行勾选 | 超出本增量；多选项目时门禁仍由创建路径处理 |

## 风险

- 列表若某环境未返回 `server_run_template`：一律显示不可自动运行（安全默认，不假装可跑）。
