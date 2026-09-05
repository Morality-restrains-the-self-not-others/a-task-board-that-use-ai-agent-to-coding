# 空评论态取消任务级运行态 fallback

- **日期**: 2026-08-11
- **状态**: implemented（goal-mode）
- **迭代**: empty-comments-no-runtime-fallback
- **作者**: claude
- **python_api_approval**: n/a（纯前端）
- **架构变更**: 无（不写 architecture target）

## 问题

任务详情评论区在 `displayComments.length === 0` 时渲染 `execution-details-fallback`（执行细节 + 服务器启动/SSE/SDKError 等任务级运行态），用户误以为「有评论但看不到正文」。

## 决策

采用 **方案 A：空评论彻底不挂 fallback**。

| 方案 | 说明 | 结论 |
|------|------|------|
| A. 移除 fallback | 空态仅 ConversationFeed「暂无评论」；有评论时运行态仍挂在气泡 `#execution-details` | **采纳** |
| B. 保留 fallback 但改文案/折叠 | 仍占位，误导风险残留 | 拒 |
| C. 把运行态挪到 Runtime 区顶栏 | 超本次范围；记 OPT | 后置 |

## 实现要点

- `TaskDetailCommentsPanel.vue`：删除 `comment-execution-details-fallback-wrap`
- `TaskDetailCommentsSection.vue`：删除 `#execution-details-fallback` 模板
- 意图：`comment_execution_details(.test).intent.md` T4 改为「零评论不挂任务级运行态」
- 单测：`TaskDetailCommentsPanel.empty-fallback.test.js`

## 业务意图 → 事件对照

| 业务意图 | 事件 | 例外理由 |
|---------|------|---------|
| 空评论 UI 不再挂任务级运行态 | — | 纯前端展示，无服务端状态变更 |

## 🕸️ Code Review Graph 分析

- skipped_non_code 近似：局部 Vue 槽位删除；CRG 未索引该 composable 符号，以源码 + 单测为准。
