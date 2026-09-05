# 测试意图：页内浮窗顶部 × 关闭浮窗

## 对应功能意图

`docs/intents/frontend/task_chrome_plugin_float_panel_close.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 面板 header | 含 `type=button` 的 `#taskplugin-float-close`，`aria-label="关闭浮窗"` |
| T2 | 点击 × | 调用 `hideFloatPanel`；不写 `saveFloatBallConfigToStorage`、不改 root `display` |
| T3 | 样式与防重放 | `content.css` 含 `#taskplugin-float-close`；点击处理标注 `Anti-Replay-OK: ui-only` |
| T4 | 使用说明 | `float-create` 步骤写明面板顶部「×」关闭浮窗 |
| T5 | Popup | `#floatBallSection` 独立于请求预览；登录/未登录均显示「显示悬浮球」 |

## 运行

```bash
cd taskChromePlugin && node --test test/content.test.js test/user-guide.test.js test/popup-layout.test.js
```
