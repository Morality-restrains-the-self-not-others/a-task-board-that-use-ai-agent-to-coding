# 测试意图：Chrome 插件长时间使用后 CPU 暴涨修复

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_cpu_leak.intent.md`

## 测试目标

锁定可见性感知 interval、指针监听按需挂载、page-bridge 已删除、webRequest body 热路径过滤。

## 测试分层

- 单元：`taskChromePlugin/test/hot-path-guards.test.js`
- 单元：`taskChromePlugin/test/visibility-interval.test.js`
- 单元：`taskChromePlugin/test/content-cpu-hot-path.test.js`
- 回归：`taskChromePlugin/test/page-bridge-origin.test.js`、`test/no-cross-tab-broadcast.test.js`、`test/content.test.js`

## 用例矩阵

| ID | 场景 | 层级 | 期望 |
|----|------|------|------|
| T1 | document.hidden | 单元 | `shouldRunPeriodicAuthTick` 为 false |
| T2 | inFlight=true | 单元 | 跳过下一拍，不重叠 |
| T3 | hidden 时 start interval | 单元 | 不挂 setInterval |
| T4 | hidden→visible | 单元 | 恢复 interval 并立即 tick |
| T5 | GET + capture on | 单元 | 不缓存 requestBody |
| T6 | 非命名空间 postMessage | 单元 | （page-bridge 已删除，本项作废） |
| T7 | content.js 源码 | 契约 | visibility interval + 按需 mousemove/mouseover + 单一 storage listener |
| T8 | page-bridge | 契约 | 文件不存在，见 `test/page-bridge-removed.test.js` |
| T9 | SW | 契约 | `WEB_REQUEST_BODY_TYPES` + `shouldCacheWebRequestBody` |

## 数据与环境

- 不连真实浏览器；visibility 测例注入假时钟。

## 通过标准

```bash
cd taskChromePlugin && node --test test/hot-path-guards.test.js test/visibility-interval.test.js test/content-cpu-hot-path.test.js test/page-bridge-removed.test.js test/page-bridge-origin.test.js test/no-cross-tab-broadcast.test.js test/content.test.js
```

T1–T9 全部通过。

## 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-27 | T6/T8 改为 page-bridge 已删除 | 与 remove_page_bridge 同步 |
