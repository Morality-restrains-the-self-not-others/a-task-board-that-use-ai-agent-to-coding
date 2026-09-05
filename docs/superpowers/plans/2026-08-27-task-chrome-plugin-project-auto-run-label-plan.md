# 计划：Chrome 插件项目列表自动运行标注

- **Date:** 2026-08-27

## 任务

- [ ] T1 红：`test/project-auto-run-label.test.js` 覆盖 T1–T9
- [ ] T2 绿：`lib/project-auto-run-label.js` 纯函数
- [ ] T3 浮窗 `loadProjects` 使用 caption HTML；`manifest.json` 注入 lib（content 之前）
- [ ] T4 面板 `renderProjectCheckboxes` 接入；`panel.html` 在 workspace.js 之前注入
- [ ] T5 CSS：浮窗 `content.css`、面板 `panel.css`
- [ ] T6 使用说明：`docs/USER_GUIDE.md` + `lib/user-guide.js`；`user-guide.test.js` 断言
- [ ] T7 `docs/intents/INDEX.md`、价值流图、`docs/flows/chrome插件/004_*.wsd`
- [ ] T8 version bump `manifest.json`；e2e `LIB_FILES` 补注入
- [ ] T9 `node --test` 相关测例全绿

无事件契约任务：只读展示。
