# 测试意图：去掉 Chrome 插件跨 tab 广播

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_no_cross_tab_broadcast.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | SW 源码 | 无 `chrome.tabs.query({})`；无 `broadcastAuthStateChanged` / `broadcastElementPickerShortcut` / `broadcastAccountExpired` |
| T2 | Popup | 无 `notifyContentScriptsAuthChanged`；无对 `authStateChanged` 的 `tabs.query({})` |
| T3 | SW `setElementPickerShortcut` | 成功且 `__sent` 为空（不向其它 tab 发快捷键） |
| T4 | SW `logout` | 不向其它 tab 发 `authStateChanged` |
| T5 | 选元素 | 仍有 `broadcastPickToChildFrames`；`broadcastStartElementPick` 只命中**同一 tabId** 子 frame |
| T6 | UserGuide | 说明登录/快捷键不跨 tab 扇出；选元素仅本页 |

## 运行

```bash
cd taskChromePlugin && npm test
```
