# 测试意图：compute 转发 comment_id 进 path

| 编号 | 场景 | 期望 | 用例位置 |
|------|------|------|----------|
| T1 | func-first URL 带 comment_id kv | parser 填 Scope.CommentID | `taskContainerGateway/src/handlers_test.go` |
| T2 | kv-last URL 带 comment_id 再 action | 同上且 Cloud rest 仍为 compute/container-* | `shareLib/gatewayauth/pathparams_test.go` |
| T3 | path / query / body 同时出现 | 取 path | `handlers_test.go`、`compute_scoped_csc_test.go` |
| T4 | FE helper 生成 URL | `/comment_id/` 且无 query comment_id | `containerComputeRequest.test.js` 等 |
| T5 | 无 comment_id 不发执行日志 | apiFetch 未调用 | `taskDetailExecLog.test.js` |
| T6 | TaskApiEndPoint 含 `/comment/{cid}/` | `taskApiPrefix` 不剥 comment 段 | `saasTaskCloud.taskApiPrefix.test.mjs` |
| T7 | inbound path 含 comment 段 | AgentSupport / Credential 解析出 action 与 CommentID | `taskAgentSupport/src/handlers_test.go`、`taskCredentialService/interfaces/container_api_path_test.go` |
| T8 | UserData 有 comment_id | cloud prefix 含 `/comment/{cid}/` | `userdata_replace_test.go`、`replace_test.go` |
