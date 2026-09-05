# 测试意图：去掉 Chrome 插件跨页面任务描述同步

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_no_cross_page_desc_sync.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `Storage` | 无 `getSyncDescriptionConfig` / `saveSyncDescriptionConfig` |
| T2 | `popup.html` | 无 `#syncDescriptionToggle`，无「跨页面同步任务描述」文案 |
| T3 | `content.js` / `popup.js` / `service-worker.js` | 无 `syncDescription` 广播与配置 action |
| T4 | SW 收到旧版 `syncDescription` | 不 `tabs.sendMessage`；返回 `Unknown action` |
| T5 | UserGuide `popup-extras` | 说明各标签页描述独立；不含「跨页面同步任务描述」 |
| T6 | E2E Popup | 真实 `popup.html` 无同步开关 |

## 运行

```bash
cd taskChromePlugin && npm test
```
