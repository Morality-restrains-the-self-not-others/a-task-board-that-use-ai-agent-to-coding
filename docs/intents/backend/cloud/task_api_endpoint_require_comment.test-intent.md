# 测试意图：TaskApiEndPoint 必须含 /comment/{cid}/

## 测试目标

无 `commentId` 时不得生成或解析出旧 `…/task/{taskId}/cloud`；有 cid 时必须带 `/comment/{cid}/`。

## 测试分层

- Go 单元：`taskCloudService` userdata prefix；`taskEvents` userdata replace；`go_relayToTrae` token/push/process
- 前端单元：`taskFE` userdata 占位符
- 容器单元：`trae-agent/onlineServiceJS` `taskApiPrefix` / `buildTaskCloudPrefix`
- inbound 单元：`taskAgentSupport` `parseCloudInboundPath`；`taskCredentialService` `parseContainerAPIPath` / `parseTokenInitPath`；`taskContainerGateway` token-init / repo-clone URL

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 合法 cid | 生成 TaskApi 前缀 | 含 `/comment/{cid}/cloud` |
| T2 | cid 空或 `-` | 生成前缀 | 空字符串，不含 `…/task/{id}/cloud` |
| T3 | 无 COMMENT_ID | `expandEnvForRuntime` | 不写入旧 `TASK_API_ENDPOINT` |
| T4 | 无 comment_id | `cloudTokenAPIPrefix` / `statusPushURL` | error / 空 URL |
| T5 | 旧 `/cloud` 无 COMMENT_ID | `taskApiPrefix()` | throw |
| T6 | 旧 `/cloud` + COMMENT_ID | `taskApiPrefix()` | 重建为含 `/comment/{cid}/` 的新前缀 |
| T7 | path 已含 `/comment/{cid}/` | `taskApiPrefix()` | 原样规范前缀 |
| T8 | 无 `/comment/{cid}/` 的 inbound path | `parseCloudInboundPath` / `parseContainerAPIPath` | `ok=false` |
| T9 | token init 9 段无 comment | `parseTokenInitPath` | `ok=false` |
| T10 | 网关 container-inbound-token | `routes.yaml` | 仅 `…/task/*/comment/*/cloud/…` |

## 自动化落点

- `taskCloudService/src/userdata_replace_test.go`
- `taskEvents/internal/cloud/userdata/replace_test.go`
- `go_relayToTrae/src/token_test.go` `push_test.go` `process_test.go`
- `taskFE/app/src/utils/userdataContainerImageReplace.test.js`
- `taskFE/app/src/composables/userdataContainerImageReplace.test.js`
- `trae-agent/onlineServiceJS/src/saasTaskCloud.taskApiPrefix.test.mjs`
- `trae-agent/onlineServiceJS/src/scopedUiPath.test.mjs`
- `taskAgentSupport/src/handlers_test.go`
- `taskCredentialService/interfaces/container_api_path_test.go`
- `taskCredentialService/interfaces/token_init_path_test.go`
- `taskContainerGateway/src/credential_client_test.go`
- `taskGateway/scripts/ci/test_container_inbound_comment_path.py`
