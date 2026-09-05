# 测试意图：评论卡片运行态按钮请求评论级服务器配置

## 测试目标

验证评论卡 Workbench / 刷新 / 停止 带 `comment_id`，且后端按该评论 CSC 解析，不回退任务级或其它评论。

## 测试分层

- 单元：`taskCloudService/src/compute_scoped_csc_test.go`、`compute_workbench_link_test.go`
- 前端单元：`cloudComputeCommentQuery.test.js`、`bindCommentRuntimePanel.test.js`、`useServerConfigRuntimeFetch.test.js`、`ServerConfigRuntimeStatusSection.test.js`、`taskDetailContainerFns.commentId.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 任务级 instance=mock-task，评论 c1 instance=mock-cmt | GET runtime-status?comment_id=c1 | instance_id=mock-cmt |
| T2 | 同上 | GET runtime-status 无 comment_id | 400 缺少评论ID |
| T3 | 任务级有 instance，评论行 instance 空 | POST stop-vm `{comment_id}` | 400 未提供实例ID |
| T4 | 两评论不同 instance | GET workbench-link?comment_id=cmt-b | URL 含 i-comment-b，不含任务/另一评论 |
| T5 | 评论卡面板 | 点击 Workbench/刷新/停止 | query 或 body 含 comment_id |
| T6 | 无评论 id | 前端 fetchContainerTaskUiContext | 不发网络请求并保持 unregistered |

## 数据与环境

- `setupCloudTestDB`；双行 `cloud_server_configs`
- 前端 vitest mock `apiFetch`

## 通过标准

```
go test ./src -count=1 -run 'TestWorkbenchLink|TestResolveScoped|TestServerRuntimeStatusCommentId|TestStopVmNativeCommentId'
```

相关 vitest 全绿。
