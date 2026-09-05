# 测试意图：云主机启动成功后恢复 failed 评论容器绑定

## 测试目标

证明 failed binding 在评论级启动成功或 CSC 平台升级后回到 starting；任务级成功不污染 failed 卡片；前端不再用红条覆盖成功日志。

## 测试分层

- 单元：`taskCloudService` `comment_container_binding_server_log_test.go`、`comment_csc_ensure_test.go`
- 单元：`taskFE` `bindingServerStartupLogs.test.js`、`useCommentContainerBindings.test.js`、`TaskDetailServerStartStatusPanel.test.js`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 评论级 SSE success「aliyun服务器启动成功」且 binding=failed | status=`starting`，有 `server_started` 日志 |
| T2 | 任务级（无 comment_id）SSE success 且 binding=failed | status 仍 `failed`，无 `server_started` |
| T3 | 评论 CSC mock 升级为 aliyun | failed binding → `starting` |
| T4 | 前端 logs 含启动成功 | `serverStatus=processing`，无 error banner |
| T5 | 成功之后又有「启动服务器失败」 | 仍视为失败，保留红条 |
| T6 | mock 首轮 advance | status=`starting`，无 failed 阶段日志 |
| T7 | 评论级 SSE error | starting/pending → `failed` |

## 数据与环境

- `setupCommentContainerBindingTest` / `setupCloudTestDB` + `markCommentContainerBindingFailed`

## 通过标准

- 上表 T1–T5 全绿。
