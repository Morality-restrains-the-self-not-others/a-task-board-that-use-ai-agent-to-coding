# 测试意图：启服中不展示平台预创建的容器 Agent 空气泡

对应功能意图：`hide_pending_container_agent_placeholder.intent.md`

## 测试目标

锁定 Feed 只在容器镜像写入可见回复后展示「容器 Agent」气泡；
平台 pending 占位在启服中不可见。

## 测试分层

| 层 | 范围 |
| --- | --- |
| 单元 | `decorateAgentRepliesWithLayerPr` 过滤规则 |
| 接线 | `TaskDetailCommentsSection` 的 feedDisplayComments 包装 |

## 用例矩阵

| ID | 场景 | 期望 |
| --- | --- | --- |
| T1 | pending + `@镜像`+父正文 echo + 无 PR | `children === []` |
| T2 | 同上但 ztree 已有 PR | 子评论保留，content 空，带 git_pr |
| T3 | 有 `assistant_response` | 子评论保留 |
| T4 | `run_status=streaming` 或 live SSE | 子评论保留 |
| T5 | 容器自建非 echo 正文 | 子评论保留 |

## 数据与环境

纯函数单测，不连后端 / 不启浏览器。

## 通过标准

`decorateAgentRepliesWithLayerPr.test.js` 与
`TaskDetailCommentsSection.test.js` 相关用例全绿。

## 不测

- 公网 Playwright 真机启服（依赖云实例）
- 停止平台 `notifyContainerAgentPending` 写库（本意图只藏 UI）

## 业务意图 → 事件对照

本意图无服务端状态变更，测试不断言 MQ 投递。
