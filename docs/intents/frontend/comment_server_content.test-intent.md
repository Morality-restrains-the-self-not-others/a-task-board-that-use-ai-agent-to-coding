# 测试意图：评论级「服务器内容」Tab

## 测试目标

验证「服务器内容」挂在评论执行细节 Tab 内、按 comment_id 拉取与隔离，任务级不再直播拉取。

## 测试分层

| 层 | 落点 |
|----|------|
| 单元 | `taskFE/app/src/composables/taskDetail/commentContentSnapshotStore.test.js` |
| 单元 | `taskFE/app/src/composables/taskDetail/bindCommentServerContentPanel.test.js` |
| 单元 | `taskFE/app/src/composables/taskDetail/useServerConfigRuntimeFetch.test.js` |
| 单元 | `taskFE/app/src/composables/taskDetail/commentExecutionPanelPolicy.test.js` |
| 组件 | `taskFE/app/src/components/task-detail/TaskDetailCommentExecutionDetails.server-content-tab.test.js` |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | `serverRuntimeStatusTab=true` | 挂载执行细节 | 出现「服务器内容」Tab |
| T2 | 同上 | 点击该 Tab | `comment-execution-panel-server-content` 展示插槽内容 |
| T3 | `serverRuntimeStatusTab=false` | 挂载 | 无「服务器内容」Tab |
| T4 | 两评论快照并发写入 | 读 store | 互不覆盖 |
| T5 | bind 面板 + comment C9 | 调 fetch | `fetchServerContent('C9')` 且覆盖 snapshot 字段 |
| T6 | `commentId` 为空 | `fetchServerContentApi` | 不发请求，文案「缺少评论ID」 |
| T7 | 有 commentId 且成功 | fetch | URL 含 `comment_id` |
| T8 | ServerConfig.logic 源码 | 静态断言 | 有 `server-content-moved-hint`，无 `<ServerConfigServerContentSection` |
| T9 | CommentsSection 源码 | 静态断言 | `#server-content` + `bindCommentServerContentPanel` |

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-20 | 初版 | 与 comment_server_content 意图同期 |
