# 意图：Chrome 插件项目列表标注是否可自动运行

## 背景与目标

- 背景：浮窗与 DevTools 拉取工作空间项目后，复选框只显示项目名。用户勾选「是否自动运行」时才在创建失败后得知项目未允许自动运行（`server_run_template.default_auto_run`）。
- 目标：拉取并渲染项目列表时，在每个项目名称旁标注该项目是否允许自动运行，与工作面板项目设置「是否允许自动运行」语义一致。

## 范围与边界

- 范围内：浮窗 `#taskplugin-projects`、DevTools 单请求/批量「项目」复选框列表；纯函数 `lib/project-auto-run-label.js`；使用说明。
- 范围外：~~不在本增量根据标注自动勾选/禁用「是否自动运行」复选框~~ **已由** `task_chrome_plugin_project_single_select_auto_run` **交付**（项目单选 + 控件联动）。本意图仍只覆盖列表徽章。

## 约束与风险

- 判定键：`project.server_run_template.default_auto_run === true` 为可自动运行；缺字段、非对象模版、`false` 均为不可自动运行。
- 项目名须 HTML 转义；标注文案为固定中文，禁止把未转义的 API 字段写入 HTML。
- 复选框 `value` 仍为项目 id，提交 payload 不变。

## 验收标准

1. 列表项文案含项目名，旁侧有「可自动运行」或「不可自动运行」。
2. `default_auto_run: true` → `data-auto-run="true"` 且文案「可自动运行」。
3. 缺 `server_run_template` 或 `default_auto_run` 非 true → `data-auto-run="false"` 且文案「不可自动运行」。
4. 浮窗与 DevTools 面板共用同一纯函数。
5. `node --test test/project-auto-run-label.test.js` 全绿。

## 实施计划

1. 纯函数 + 单测先行。
2. 浮窗 `loadProjects`、面板 `renderProjectCheckboxes` 接入；注入 `lib/project-auto-run-label.js`。
3. 使用说明两份 SSOT；`manifest.json` version bump。

## 业务意图 → 事件对照

| 业务意图 | 领域事件 | MQ Topic | 说明 |
|----------|----------|----------|------|
| 插件展示项目是否可自动运行 | （无） | — | **无对应事件**：只读渲染已有 GET 项目字段 |

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-27 | 范围外「禁用自动运行」改由后续意图交付 | 项目改为单选后可联动 |
