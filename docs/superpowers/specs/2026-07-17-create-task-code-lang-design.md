# 创建设计：创建任务可选字段 code_lang（主要编程语言）

**日期**: 2026-07-17  
**状态**: 已采用  
**范围**: 工作台创建任务 + 工作区可选值配置；与 `task_kind` 同模式

## 目标

创建/编辑任务时增加可选字段 `code_lang`（主要编程语言）；默认可选值为 `go`、`rust`、`js`；工作区可配置列表。

## 方案

| 层级 | 落点 |
|------|------|
| 任务持久化 | `taskTaskService.tasks.code_lang`（TEXT，默认可空） |
| 工作区可选值 | `taskProjectService`：`GET/PUT .../workspaces/{id}/code-lang-options/`，表 `workspace_code_lang_options`，无记录时返回默认 `["go","rust","js"]` |
| 前端 | 创建任务下拉；工作区设置「编程语言」入口；提交/回填 `code_lang` |

网关：既有 `/api/tenant/*/workspaces/*` 通配，无需新路由条目。

## 交互

- 下拉含「未指定」（空字符串）与工作区配置项。
- 可选；不选则存空，列表/详情 JSON 中可为 `null`/省略（与 `task_kind` 一致的 `nilIfEmpty`）。
- 工作区设置：每行一个值；空保存回退默认三语言。

## 非目标

- 不按 `code_lang` 过滤看板（本期）。
- 不改 Chrome 插件创建任务（可另开 OPT）。
