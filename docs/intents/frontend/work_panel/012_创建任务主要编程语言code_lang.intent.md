# 012_创建任务主要编程语言 code_lang（功能意图）

## 1. 背景与目标

- 背景：创建任务已有结构化说明与 `task_kind`；需要标注任务主要编程语言，便于 Agent/人工分流。
- 目标：创建/编辑任务可选填写 `code_lang`；默认选项 go / rust / js；工作区可配置可选值。

## 2. 范围与边界

- 范围内：
  - 创建任务弹窗「主要编程语言」下拉（`data-testid="create-task-field-codeLang"`）。
  - 提交 todos 时携带 `code_lang`；编辑回填。
  - 工作区设置「编程语言」维护可选值（`GET/PUT .../code-lang-options/`）。
  - `taskTaskService` 持久化 `tasks.code_lang`。
- 范围外：
  - 看板按语言过滤。
  - Chrome 插件对齐。

## 3. 验收标准

- [x] 创建任务可见主要编程语言下拉，默认含 go / rust / js，可「未指定」。
- [x] 提交后任务 JSON 含所选 `code_lang`。
- [x] 工作区设置可改可选值并影响创建下拉。
- [x] 相关单测通过（前端 normalize、Go CodeLang options、任务 create/list）。

## 4. 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-07-17 | 初版：`code_lang` 字段 + 工作区可选值 API + 创建下拉 | 用户要求创建任务增加主要编程语言 |

## 业务意图 → 事件对照

<<<<<<< Updated upstream
**无对应事件**：本变更仅扩展既有 todos / workspace options HTTP 写路径上的可选字段与配置列表，不新增独立领域事件或消息总线发布。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 012_创建任务主要编程语言 code_lang | — | — | — | 无对应事件：复用既有 create/update todos 与 code-lang-options PUT，无新增 publish |
=======
| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 创建任务写入 code_lang | TaskCreated / TaskUpdated（既有） | taskTaskService create/update | 任务列表/详情展示字段 | 复用既有任务写路径，不新增独立领域事件 |
| 工作区保存 code_lang 可选值 | WorkspaceCodeLangOptionsUpdated（可选） | taskProjectService PUT code-lang-options | 创建任务下拉刷新 | 配置类写操作；前端以 GET 刷新为准，可不发总线事件 |
>>>>>>> Stashed changes
