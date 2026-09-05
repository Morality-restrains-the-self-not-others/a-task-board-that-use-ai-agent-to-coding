# 测试意图：去掉任务级 ServerConfig，运行态禁止回退

## 测试目标

验证运行态 compute API 必须带 `comment_id` 且只读该评论 CSC；无 comment_id 不得落到任务级行。

## 测试分层

- 单元：`compute_scoped_csc_test.go`、`compute_workbench_link_test.go`、既有 runtime-status/stop/content 测例改为评论级种子
- 前端：`cloudComputeCommentQuery.test.js`、`useServerConfigRuntimeFetch.test.js`、`bindCommentRuntimePanel.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 任意 CSC | GET runtime-status 无 comment_id | 400 缺少评论ID |
| T2 | 任务级 mock-task + 评论 c1 mock-cmt | GET runtime-status?comment_id=c1 | instance_id=mock-cmt |
| T3 | 任务级有 instance，评论行无 | POST stop-vm `{comment_id}` | 400 未提供实例ID |
| T4 | 无 comment_id | POST stop-vm / GET workbench-link | 400 缺少评论ID |
| T5 | 评论卡 | Workbench/刷新/停止 | 请求含 comment_id |
| T6 | 无评论 id | 前端 fetch API | 不发网络请求 |

## 通过标准

`go test ./src -count=1 -run 'TestResolveScoped|TestWorkbenchLink|TestServerRuntimeStatus|TestStopVmNativeCommentId|TestHandleServerContent'` 全绿；相关 vitest 全绿；`rg loadCloudServerConfigForRuntime taskCloudService` 无命中。
