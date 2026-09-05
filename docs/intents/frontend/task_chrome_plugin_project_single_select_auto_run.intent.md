# 意图：Chrome 插件项目单选且自动运行随项目能力变化

## 背景与目标

- 背景：浮窗与 DevTools 项目为多选复选框；「是否自动运行」与项目是否允许自动运行无关。上一增量只在名称旁标注能力，未钳制开关。
- 目标：项目改为单选；下方自动运行根据所选项目是否允许自动运行启用或禁用（不允许则取消勾选并提示）。

## 范围与边界

- 范围内：浮窗 `#taskplugin-projects`、DevTools `singleProjects` / `batchProjects`；`lib/project-auto-run-label.js` 状态推导；使用说明。
- 范围外：不新增 API；不改创建任务后端契约；不在本增量做镜像/硬件模版完整启机门禁 UI。

## 约束与风险

- 判定键仍为 `server_run_template.default_auto_run === true`。
- payload `projects` 长度为 0 或 1。
- 切换到允许自动运行的项目时默认勾选；用户可取消。不允许或未选项目时强制不勾选且 disabled。

## 验收标准

1. 项目列表为 radio，同一入口不能同时选中两个项目；无「全选」。
2. 未选项目或项目不可自动运行 → 自动运行 disabled 且 unchecked。
3. 选中可自动运行项目 → 自动运行 enabled，默认 checked，提示为启机说明。
4. `node --test test/project-auto-run-label.test.js test/user-guide.test.js` 全绿。

## 业务意图 → 事件对照

| 业务意图 | 领域事件 | MQ Topic | 说明 |
|----------|----------|----------|------|
| 单选项目并钳制自动运行 | （无） | — | **无对应事件**：纯 UI；创建任务沿用既有事件 |

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-27 | 初版 | 项目改为单选，自动运行随项目能力变化 |
