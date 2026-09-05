# 意图：Chrome 插件自动运行与已安装镜像必选挂钩

## 背景与目标

- 背景：浮窗勾选自动运行后创建，服务端才拒绝（`AUTO_RUN_IMAGE_REQUIRED`）。镜像下拉默认可空。
- 目标：镜像是否必选随自动运行挂钩——要自动运行必须先选已安装镜像；不自动运行时镜像可选。创建前在 UI 阻断，不再等后端报错。

## 范围与边界

- 范围内：浮窗 `#taskplugin-image` + `#taskplugin-auto-run`；DevTools `#singleContainerImage` / `#batchContainerImage` 与对应自动运行；`lib/project-auto-run-label.js`；`validateCreateTaskForm`；使用说明。
- 范围外：不改创建任务 API；不补硬件运行模版前端门禁；不改工作面板（已有 `canEnableAutoRun`）。

## 约束与风险

- 判定：`String(container_image_id).trim()` 非空视为已选镜像。
- 自动运行启用顺序：已选项目 → 项目允许自动运行 → 已选镜像。
- 项目不允许自动运行时，镜像保持可选。

## 验收标准

1. 项目允许自动运行但未选镜像 → 自动运行 disabled、unchecked，提示含「请先选择已安装镜像」。
2. 选中镜像后 → 自动运行 enabled；允许项目默认勾选。
3. 清空镜像 → 自动运行强制关闭。
4. `auto_run=true` 且无镜像 → `validateCreateTaskForm` 返回「请先选择已安装镜像」，不发请求。
5. `auto_run=false` 且无镜像 → 表单门禁不因镜像拦截。
6. 使用说明两份 SSOT 已写挂钩规则。

## 业务意图 → 事件对照

| 业务意图 | 领域事件 | MQ Topic | 说明 |
|----------|----------|----------|------|
| UI 挂钩镜像与自动运行 | （无） | — | **无对应事件**：纯客户端；创建任务沿用既有事件 |

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-27 | 初版 | 浮窗自动运行缺镜像仅后端报错 |
