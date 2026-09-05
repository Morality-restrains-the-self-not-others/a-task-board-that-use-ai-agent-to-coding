# 实施计划：创建任务可选字段显隐

**日期**: 2026-07-18  
**设计**: `docs/superpowers/specs/2026-07-18-create-task-field-settings-design.md`

## 任务清单

- [x] Go：表 `workspace_create_task_field_settings` + `create_task_field_settings.go` + main 路由
- [x] Go：单元测试（默认/roundtrip/隔离/normalize）
- [x] OpenAPI：path + schemas + tag
- [x] 前端 utils：`createTaskFieldSettings.js` + unit test
- [x] 前端 composable：`useWorkPanelCreateTaskFieldSettings.js`，挂入 `useWorkPanelTaskMetaOptions`
- [x] Settings：Actions 按钮 + Modal + WorkspaceSettingsTaskPanel 接线
- [x] CreateTaskModal：prop `fieldSettings` + 子组件 v-if + 门禁跳过
- [x] 意图文档：`task2app/docs/intents/frontend/work_panel/013_*`
- [x] 架构 v35 三件套 + VERSION_HISTORY
- [x] Vitest：模态显隐；Go test 通过

## 事件契约例外

纯工作区配置 CRUD，不投递 MQ 业务事件（与 task-kind-options / code-lang-options 一致）。书面例外已记录于设计文档 §2。
