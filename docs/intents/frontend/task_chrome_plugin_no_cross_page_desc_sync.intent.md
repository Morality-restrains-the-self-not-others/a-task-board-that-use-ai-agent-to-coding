# task_chrome_plugin_no_cross_page_desc_sync（功能意图）

## 1. 背景与目标

- 背景：`taskChromePlugin` 曾默认把浮窗「任务描述」经 Service Worker 广播到其它标签页（Popup 开关「跨页面同步任务描述」）。多标签编辑时会互相覆盖，且广播扇出会放大 SW 压力。
- 目标：**去掉跨页面同步**。每个标签页的任务描述只属于本页，互不影响。

## 2. 范围与边界

- 范围内：
  - 删除 Popup「跨页面同步任务描述」开关
  - 删除 content script 输入防抖广播与 `syncDescriptionUpdate` 收听
  - 删除 SW `syncDescription` / `getSyncDescriptionConfig` / `setSyncDescriptionConfig`
  - 删除 `Storage.getSyncDescriptionConfig` / `saveSyncDescriptionConfig`
  - 使用说明（`docs/USER_GUIDE.md` + `lib/user-guide.js`）写明各页描述独立
- 范围外：
  - 登录态 / 快捷键 / 悬浮球显隐等其它跨标签广播（仍保留）
  - 历史遗留 `chrome.storage.local.syncDescriptionEnabled` 键可残留，不再读写

## 3. 行为约定

| 场景 | 期望 |
|------|------|
| 标签页 A 填写描述 | 标签页 B 描述不变 |
| Popup 请求预览区 | 无「跨页面同步任务描述」开关 |
| 旧版 content 仍发 `syncDescription` | SW 视为未知 action，不向其它 tab 转发 |

## 4. 变更记录

| 日期 | 差异 | 原因 |
|------|------|------|
| 2026-08-21 | 下线跨页任务描述同步（compulsory，无双路径） | 用户目标：去掉跨页面同步 |

## 业务意图 → 事件对照

**无对应事件**：纯扩展端本地 UI/消息通道下线，无服务端业务状态变更，不投递领域事件。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 去掉跨页面任务描述同步 | — | — | — | 纯前端/扩展本地交互，无服务端聚合状态变更 |

## 5. 角色权限

无新 HTTP 端点、无新增扩展权限。`tabs` 权限仍用于其它广播，本增量不收窄 manifest。

## 6. NFR 摘要

### 路径分片键审视

| 路径 | 是否携带可分片 ID | 判定 | 动作 |
|------|-------------------|------|------|
| `chrome.runtime` `syncDescription`（删除） | 否 | L0：扩展本机消息，无多租户伸缩 | 删除该路径 |
| Popup / content 本地描述输入 | 否 | L0：单页 DOM 状态 | 保持本页隔离 |

### 幂等性审视

| 路径 | 副作用 | 判定 | 说明 |
|------|--------|------|------|
| 填写任务描述 | 仅本页 textarea | L0 | 无跨页写、无服务端写 |
| 已删除的跨页广播 | 曾有跨 tab 覆盖 | — | 路径移除，不再评估重放 |

## 7. DDD

无领域模型变更。描述文本不是服务端聚合；创建任务仍走既有 `createTask` 路径。
