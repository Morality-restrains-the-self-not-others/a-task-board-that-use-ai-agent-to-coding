# task_chrome_plugin_no_cross_tab_broadcast（功能意图）

## 1. 背景与目标

- 背景：描述同步已下线；登录态、快捷键、账号过期仍 `chrome.tabs.query({})` 向**所有标签页** `sendMessage`。content/pick-frame 已有 `storage.onChanged` 兜底。
- 目标：去掉这三类**跨 tab**扇出。各页靠 storage 变更刷新。选元素仍只对**当前 tab 的子 frame**发指令。

## 2. 范围与边界

- 范围内：
  - 删除 `broadcastAuthStateChanged` / `broadcastAccountExpired` / `broadcastElementPickerShortcut`
  - Popup 不再 `tabs.query({})` 发 `authStateChanged`
  - Panel / Popup / content 用 `storage.onChanged` 刷新登录/账号
  - 使用说明写明：登录与快捷键不扇出其它标签页；选元素仅本页（含 iframe）
- 范围外：
  - 同 tab `broadcastPickToChildFrames` / `toggleElementPick`（当前页）
  - `chrome.commands` → 当前活动标签
  - Popup「显示悬浮球」全 tab 开关（未列入本目标）

## 3. 行为约定

| 场景 | 期望 |
|------|------|
| 登录 / 登出 / 切账号 | 写 storage；不向其它 tab `sendMessage` |
| 修改拾取快捷键 | `commands.update` + 持久化；各页经 storage 更新兜底组合 |
| 账号过期 prune | 只改 storage；各页经 `storage.onChanged` 刷新（不再经 page-bridge 通知页面） |
| 本页选元素 / iframe | 仍向**该 tabId** 的子 frame 发 `startElementPick` |

## 4. 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-21 | 下线登录/快捷键/过期的跨 tab 广播 | 用户目标：去掉跨 tab 扇出 |
| 2026-08-27 | page-bridge 网页账号桥删除 | 无第一方调用方 |

## 业务意图 → 事件对照

**无对应事件**：纯扩展本机消息通道下线。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 去掉跨 tab 登录/快捷键扇出 | — | — | — | 纯扩展本地，无服务端聚合变更 |
