# 计划：Chrome 插件项目单选 + 自动运行联动

- **Date:** 2026-08-27
- **Goal:** 项目单选；自动运行控件随所选项目 `default_auto_run` 启用/禁用。

## Tasks

- [x] T1 单测：`resolveAutoRunControlState` 无项目 / 不允许 / 允许 + preference
- [x] T2 单测：`pickSingleProjectId` 取第一个仍存在的 id
- [x] T3 单测：`applyAutoRunControlToElements` 写 disabled/checked/hint
- [x] T4 源码契约：浮窗与面板用 radio、无项目全选、调用 resolve
- [x] T5 实现 lib 纯函数（红→绿）
- [x] T6 浮窗 `content.js` 接入
- [x] T7 面板 `workspace.js` + single/batch tabs 接入
- [x] T8 使用说明 + user-guide + README；manifest 1.8.16
- [x] T9 `node --test` 相关文件全绿

## 事件任务

无新事件契约（见 DDD 书面例外）。不新增 publish/consumer。
