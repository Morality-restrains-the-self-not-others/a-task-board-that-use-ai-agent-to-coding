# 测试意图：容器转发目标解析评论级 CSC

## 测试目标

验证内部 `container-target` 在任务级 CSC 为空模板、评论级已登记业务地址时能解析到评论级 `base_url`；带 `comment_id` 时命中对应评论行。

## 测试分层

- 单元：`taskCloudService/src/container_target_comment_csc_test.go`（httptest + 测试库夹具）
- 单元：`taskFE` `containerComputeRequest.test.js` / `containerForwardCommentId.scan.test.js` / `taskDetailContainerFns.commentId.test.js`
- 单元：`taskContainerGateway/src/handlers_comment_id_forward_test.go`（两评论 token 选择）
- 无新增 MQ 断言（只读，见功能意图例外）

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 任务级 CSC `server_url` 空；评论级 `http://127.0.0.1:1/ui/cmt-tok/` | GET container-target 无 comment_id | 200，`base_url` 为 `http://127.0.0.1:1`，token 为 `cmt-tok` |
| T2 | 评论 a/b 各有不同 token 的 server_url | GET container-target?comment_id=cmt-a | 200，token 为评论 a，不是 b |
| T3 | seedCloudConfig 任务级与评论级同有 URL（既有） | GET container-target | 200（不回归） |
| T4 | 任务级 CSC 空；评论级已登记 server_url | validateAICommentPost 无 comment_id | 200（不因空模板拒绝「发送给 AI」） |
| T5 | 两评论级 CSC 不同 token | GET container-layer-graph?comment_id=cmt-a | gateway `container-target` query 为 cmt-a，upstream token 为 tok-a |
| T6 | 两评论级 CSC 不同 token | POST container-layer-git-commit?comment_id=cmt-a | 同上 |
| T7 | 任务详情层图/git/layer-changes | FE 构造 compute URL | query（及 POST body）含当前执行评论 `comment_id` |

## 数据与环境

- `setupCloudTestDB`；`cloud_server_configs` 双行（`comment_id=''` 与非空）
- 不探测真实容器；`127.0.0.1:1` 使 stream probe 快速失败回落 `legacy`

## 通过标准

`go test ./src -count=1 -run 'TestInternalContainerTarget|TestValidateAICommentPost_UsesCommentCSC'` 全绿；gateway `go test ./src -count=1 -run 'CommentID|GitPush|ContainerTarget'`；FE vitest `containerComputeRequest` / `containerForwardCommentId` / `taskDetailContainerFns.commentId` 全绿。
