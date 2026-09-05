# 测试意图：自动运行先建评论再带 comment_id 启机

## 测试目标

证明自动运行 / 排队启机在调用 Cloud start-vm 前已持有评论 ID，并写入请求体；缺 ID 时不启机。

## 测试分层

- 单元：`taskTaskService` `auto_run_comment_id_test.go`、`queued_schedule_comment_id_test.go`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `ensureAutoRunAtComment` 返回 `cmt-mock-auto-run` | start-vm body.`comment_id` 与 `parent_comment_id` 均为该值 |
| T2 | `ensureAutoRunAtComment` 返回 error | 不调用 start-vm，`triggerTaskAutoRun` 返回错误 |
| T3 | `ensureAutoRunAtComment` 返回空 ID | 不调用 start-vm，`triggerTaskAutoRun` 返回错误 |
| T4 | `startQueuedMembership` 成功路径 | start-vm body 含同一评论 ID |

## 数据与环境

- `startAutoRunMockServices` + 可替换 `ensureAutoRunAtCommentFn` / `startVMFn`

## 通过标准

- 上表 T1–T4 全绿。
