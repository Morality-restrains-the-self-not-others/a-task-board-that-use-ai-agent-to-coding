# 实施计划：Chrome 插件自动运行与已安装镜像挂钩

- **Date:** 2026-08-27
- **Design:** `docs/superpowers/specs/2026-08-27-task-chrome-plugin-auto-run-image-gate-design.md`

## 事件契约 → publish → 消费者

无新事件。创建任务成功路径沿用既有投递。

## Tasks

- [x] **T1** Red：扩展 `test/project-auto-run-label.test.js`（T1–T6, T9）与 `test/create-task-payload.test.js`（T7–T8）；确认失败。
- [x] **T2** Green：`lib/project-auto-run-label.js` 增加 `hasInstalledImageId` / `resolveImageFieldAppearance` / `applyImageFieldAppearance` / `validateAutoRunRequiresImage`；`resolveAutoRunControlState` 增加 `hasInstalledImage` 门。
- [x] **T3** `validateCreateTaskForm` 调用镜像门禁。
- [x] **T4** 浮窗 `content.js`：镜像 change → sync；标签必选标记；load 镜像后 sync。
- [x] **T5** DevTools `workspace-projects.js` + `workspace.js` + `single-request.js` / `batch.js`：三处镜像 select 接入。
- [x] **T6** `panel.html` 镜像标签 span；`USER_GUIDE.md` + `user-guide.js`；`manifest.json` version bump。
- [x] **T7** `node --test` 相关文件全绿（`npm test` 472 passed）。

## 验收对照

见 `docs/intents/frontend/task_chrome_plugin_auto_run_image_gate.intent.md`。
