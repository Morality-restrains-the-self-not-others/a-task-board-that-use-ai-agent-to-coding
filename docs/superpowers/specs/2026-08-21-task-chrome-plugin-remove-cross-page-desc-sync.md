# 设计：taskChromePlugin 去掉跨页面任务描述同步

- **Date:** 2026-08-21
- **Status:** accepted（`/goal` 零交互采用）
- **Architecture artifacts:** 非架构变更（不下沉新服务/事件/权限），不新增 ArchiMate / `.puml` 视图

## 🕸️ Code Review Graph 分析

CRG `update --brief` 成功（nodes=108）。`taskChromePlugin` 不在当前图的主要社区内；调用链以仓库内 grep 为准：`syncDescription` → SW 全 tab `tabs.sendMessage` → content `syncDescriptionUpdate`。

## 方案

Compulsory 下线，**删除旧逻辑**（无开关、无双路径、无兼容转发）：

1. Popup 去掉开关与 `get/setSyncDescriptionConfig`
2. content 去掉 `setupDescSyncListener` 与 `syncDescriptionUpdate` 处理
3. SW 去掉三个 action；旧客户端消息走 `default` → `Unknown action`
4. Storage 去掉读写 API；遗留 `syncDescriptionEnabled` 键忽略
5. 使用说明写明各页描述独立；manifest 版本 +0.0.1

保留：auth / 快捷键 / 选元素 的跨 tab 广播（不是本需求）。
