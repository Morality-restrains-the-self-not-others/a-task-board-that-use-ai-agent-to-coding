# 测试意图：评论「服务器运行状态」Tab 也展示容器名 / CSC / 启动 TraceId

## 测试目标

验证执行细节启用 Tab 后，切到「服务器运行状态」仍展示容器名、CSC、启动 TraceId；切回「执行细节」与无 Tab 模式不回归。

## 测试分层

- 单元：`taskFE/app/src/components/task-detail/TaskDetailCommentExecutionDetails.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | Tab 模式，有 containerName / cscId / startTraceId | 默认渲染 | 执行细节面板与 `comment-execution-container-meta` 同时存在；含容器名、CSC、启动 TraceId 与 `data-traceId` |
| T2 | 同上 | 点击「服务器运行状态」 | `comment-execution-panel-server-runtime` 存在；`comment-execution-container-meta` 仍在，三项值不变 |
| T3 | 切到服务器运行状态后 | 再点「执行细节」 | 元信息仍在；运行状态面板隐藏 |
| T4 | `serverRuntimeStatusTab=false` | 展开 | 无 tablist；元信息仍在展开内容中 |
| T5 | startTraceId 为空 | 切到服务器运行状态 | 无启动 TraceId 行；容器名仍在 |

## 数据与环境

- Vitest + jsdom；`@vue/test-utils` mount 组件；无需网络。

## 通过标准

```
cd taskFE/app && npx vitest run src/components/task-detail/TaskDetailCommentExecutionDetails.test.js
```

全绿。
