# 测试意图：删除 Chrome 插件网页账号桥 page-bridge

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_remove_page_bridge.intent.md`

## 测试目标

锁定 page-bridge 文件、manifest 注入、content 账号桥 postMessage、相关守卫函数不再出现。

## 测试分层

- 契约：`taskChromePlugin/test/page-bridge-removed.test.js`
- 回归：`taskChromePlugin/test/page-bridge-origin.test.js`（仅保留 panel-core origin）、`test/content-cpu-hot-path.test.js`、`test/hot-path-guards.test.js`

## 用例矩阵

| ID | 场景 | 层级 | 期望 |
|----|------|------|------|
| T1 | 文件系统 | 契约 | `lib/page-bridge.js` 不存在 |
| T2 | manifest | 契约 | content_scripts 不含 `lib/page-bridge.js` |
| T3 | content.js | 契约 | 无 `taskfe-account-bridge` / `notifyPageAccountStateChanged` |
| T4 | hot-path-guards | 单元 | 不再导出 `shouldInspectPageBridgeMessage` 等 |
| T5 | panel-core | 回归 | DevTools HAR message 仍校验 origin |

## 数据与环境

- 不连真实浏览器。

## 通过标准

```bash
cd taskChromePlugin && node --test test/page-bridge-removed.test.js test/page-bridge-origin.test.js test/content-cpu-hot-path.test.js test/hot-path-guards.test.js
```

T1–T5 全部通过。

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-27 | 初版 | 与功能意图同步 |
