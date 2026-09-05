# NFR：去掉 Chrome 插件跨 tab 广播

- **Date:** 2026-08-21
- **Levels default:** L0（扩展本机）

## 路径分片键审视

| 路径 | 分片 ID | 判定 | 可伸缩性等级 | 结论 / 动作 |
|------|---------|------|-------------|-------------|
| `tabs.query({})` + `authStateChanged`（删除） | 无 | 本机扇出 | L0 | 删除 |
| `tabs.query({})` + `setElementPickerShortcut`（删除） | 无 | 本机扇出 | L0 | 删除 |
| `broadcastPickToChildFrames(tabId)` | tabId（本机） | 单页 iframe | L0 | 保留 |
| `chrome.storage.onChanged` | 无 | 浏览器进程内通知 | L0 | 作为刷新通道 |

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 等级 |
|------|--------|------------|--------------|--------|----------|------|
| 写登录 storage | 会话凭据 | 登录重试 | 本机 local | 最终态覆盖 | 同 token 重放同终态 | L0 |
| 已删除跨 tab sendMessage | 曾刷新它页 UI | — | — | — | 路径移除 | — |
| 同 tab 选元素指令 | 本页指针模式 | 连点 | 该 tabId | 切换/开启 | 本页内 | L0 |

禁止用 `tenant_id`/`user_id` 作本增量消费键（无 Kafka）。
