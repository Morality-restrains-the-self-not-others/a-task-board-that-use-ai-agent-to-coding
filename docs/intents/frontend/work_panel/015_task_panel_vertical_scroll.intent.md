# 工作面板任务区纵向可滚动

## 意图

用户在 `/tenant/{id}/work-panel` 的 `#task-panel-container` 内，当过滤栏 + 看板内容超出可视高度时，应能上下滚动并到达底部任务卡片；不得因整链 `overflow-hidden` 被裁切且滚轮无响应。

## 验收

1. `#task-panel-container` 使用 `overflow-y-auto`（非 `overflow-hidden`）。
2. 进度列 / `.task-cards-container` 在有界高度下可出现列内纵向滚动（`min-height: 0`，无固定 `min-height: 360px` 撑破父级）。
3. 「其他」看板在剩余空间充足时仍贴近视口底边；空间不足时由面板容器滚动，不压缩至不可读。
4. 单测：`taskFE/app/src/views/TaskPanel.fillViewport.test.js`。
