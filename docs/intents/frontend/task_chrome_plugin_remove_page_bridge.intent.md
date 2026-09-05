# 意图：删除 Chrome 插件网页账号桥 page-bridge

## 背景与目标

- 背景：`lib/page-bridge.js` 在每个网页上监听 `window` `message`，把 `taskfe-account-bridge` 请求中继到 Service Worker。taskFE 已改为 `localStorage` 存账号，仓库内无第一方调用方。
- 目标：删除该桥及其向页面 `postMessage` 的 `accountStateChanged`；浮窗/弹窗/DevTools 建任务不变。DevTools 面板的 HAR `message` 监听保留。

## 范围与边界

- 范围内：删除 `lib/page-bridge.js`、manifest 注入、`content.js` 的页面账号广播、`hot-path-guards` 中仅服务该桥的函数。
- 范围外：`panel-core` 的 `initRequests` / `newRequest` / `requestUpdated`；SW 给扩展页的 `chrome.runtime` 账号接口。

## 约束与风险

- 外部页面若仍 ping `__taskChromePlugin` / 发 `taskfe-account-bridge` 将不再得到响应。当前仓库无此调用方。

## 验收标准

1. `lib/page-bridge.js` 不存在；manifest `content_scripts` 不含该文件。
2. `content.js` 不含 `taskfe-account-bridge`。
3. `node --test test/page-bridge-removed.test.js` 通过；全套 `test/*.test.js` 全绿。

## 实施计划

1. 契约测试锁定文件/注入/协议字符串消失。
2. 删除实现与仅服务该桥的守卫函数。

## 业务意图 → 事件对照

| 业务意图 | 领域事件 | MQ Topic | 说明 |
|----------|----------|----------|------|
| 删除无调用方的网页账号桥 | （无） | — | **无对应事件**：浏览器扩展本地通道下线 |

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-27 | 初版 | 用户确认删除 page-bridge message 监听 |
