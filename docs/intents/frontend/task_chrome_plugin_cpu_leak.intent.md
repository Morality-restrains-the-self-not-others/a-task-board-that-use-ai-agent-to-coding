# 意图：Chrome 插件长时间使用后 CPU 暴涨修复

## 背景与目标

- 背景：`taskChromePlugin` 以 content script 注入全部页面。使用一段时间后浏览器 CPU 持续升高。已有跨 tab `tabs.query({})` 扇出与 storage 写风暴修复（OPT-20260808-019/023），但仍残留：每个标签页无条件 `setInterval` 打 `getAuthStatus`、document 级 `mousemove`/`mouseover` 常驻、page-bridge 双 `message` 监听并对页面全量 postMessage 检查、webRequest 对 GET/静态资源走 requestBody 热路径。
- 目标：闲置与后台标签页不再把定时器、指针监听、消息广播留在热路径上；可见页才跑角标刷新；选元素/拖拽仅在对应模式挂载指针监听。

## 范围与边界

- 范围内：`content.js`、`pick-frame.js`、SW `onBeforeRequest`、panel/popup 角标定时器。
- 范围外：不改变默认捕获 2xx 的产品语义；不改 DevTools HAR 监听（仅在打开 DevTools 时存在）。

## 约束与风险

- 隐藏页停表后，角标文案在回到前台时立即补刷，不得长期显示过期态。
- 选元素快捷键（keydown）仍常驻，以便从闲置进入 pick 模式。
- page-bridge 网页账号桥已删除（见 `task_chrome_plugin_remove_page_bridge`）；勿再注入。

## 验收标准

1. 隐藏 document 时不存在 auth 角标 `setInterval`；变为可见后恢复并立即 tick 一次。
2. 未拖拽时 document 上无 `mousemove` 监听；未选元素时无 capture `mouseover`。
3. 网页账号桥 `page-bridge.js` 已删除；全站页面不再挂 `taskfe-account-bridge` message 监听。
4. GET/HEAD/OPTIONS 与静态资源 types 不进入 requestBody 缓存。
5. `node --test test/hot-path-guards.test.js test/visibility-interval.test.js test/content-cpu-hot-path.test.js` 全绿。

## 实施计划

1. 纯函数 `lib/hot-path-guards.js` + `lib/visibility-interval.js`（单测先行）。
2. content / pick-frame / SW / panel / popup 接入。
3. 源码契约测试锁定不再回归。

## 业务意图 → 事件对照

| 业务意图 | 领域事件 | MQ Topic | 说明 |
|----------|----------|----------|------|
| 降低插件闲置 CPU | （无） | — | **无对应事件**：浏览器扩展本地热路径，不投递领域事件 |

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-27 | 初版 | 排查 setInterval / 消息广播后的 CPU 暴涨 |
| 2026-08-27 | 删除 page-bridge 网页账号桥（无第一方调用方）；验收改为「文件不存在」 | 用户确认删除 |
