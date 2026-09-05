# [运行时] Chrome 插件 DevTools 面板请求列表卡在「正在加载请求列表...」

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-17
- 最后修改：2026-07-17
- 维护者：Trae AI 团队

## 现象

- 打开 DevTools → TaskPlugin 面板「单请求创建」Tab，请求列表区域永久显示「正在加载请求列表...」。
- 刷新列表按钮无效；即使 Network 已有流量也不出现条目或空态提示。

## 根因

1. **`initRequests` 竞态**：`devtools.js` 在 `panel.onShown` 时 `postMessage({ action: 'initRequests' })`，而 `panel.js` 的 `message` 监听挂在 `await refreshAuthState()` / `checkConnection()` **之后**。早到消息无监听即丢失。
2. **SW 空兜底不刷新 UI**：1.5s 后 `getRecentRequests` 仅在 `success && data.length > 0` 时调用 `applyRequestFilters()`；空列表时 HTML 初始占位永不替换。
3. **`setRequestLoading(false)` 空操作**：只处理 `loading === true`，结束加载时不清除「正在加载请求列表...」。

## 解决方案

- `panel.html` `<head>` 同步缓冲 `initRequests` / `newRequest` / `requestUpdated`。
- `lib/panel-request-bootstrap.js`：消息缓冲 + SW 合并 +「兜底后必须刷新 UI」。
- `panel.js`：`init()` **任何 await 之前**接通缓冲 consumer；SW 兜底无论空满都 `applyRequestFilters()`；`setRequestLoading(false)` 走同一刷新路径。

## 预防

- DevTools Panel 凡依赖父页 `postMessage` 的数据：监听/缓冲须在首个 `await` 之前就绪（或 head 同步缓冲）。
- 结束 loading 的代码路径必须覆盖「空数据」；禁止只在 `length > 0` 时清占位。
- 回归：`taskChromePlugin/test/panel-request-bootstrap.test.js`。

## 验证

```bash
cd taskChromePlugin && npm test
# 手动：chrome://extensions 重新加载 → 打开任意页 DevTools → TaskPlugin
# 期望：短暂后显示请求列表，或「暂无匹配的请求」（不得永久「正在加载请求列表...」）
```

## 关联

- Popup 登录卡死：`20_chrome_plugin_popup_login_storage_hang.md`
- Popup spinner / finally：`test/async-timeout.test.js`
