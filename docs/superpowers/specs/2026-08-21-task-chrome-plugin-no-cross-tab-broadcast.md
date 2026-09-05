# 设计：去掉登录/快捷键跨 tab 广播

- **Date:** 2026-08-21
- **Status:** accepted（`/goal` 零交互）
- **Architecture artifacts:** 非架构变更

## 🕸️ Code Review Graph 分析

CRG 软依赖；调用链以 grep 为准。跨 tab 扇出仅三处 `tabs.query({})`：auth、shortcut、accountExpired。选元素 `broadcastPickToChildFrames(tabId)` 为同 tab 子 frame。

## 方案

Compulsory 删除扇出函数。刷新改走已有 `chrome.storage.onChanged`（content / pick-frame 已具备；补 popup / panel / page-bridge）。

同 tab 选元素与 `commands`→活动标签保留。
