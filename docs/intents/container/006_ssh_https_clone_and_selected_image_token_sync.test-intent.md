# 测试意图：SSH→HTTPS 克隆 + selected_image Token 同步

对应功能意图：`006_ssh_https_clone_and_selected_image_token_sync.intent.md`

## 单元测试

| 用例 | 位置 | 状态 |
|------|------|------|
| ResolveHttpsCloneURL SCP 本地 GitLab | `taskCredentialService/infrastructure/provider_configs_test.go` | ✅ |
| normalizeRepoUrlForHttpsClone | `trae-agent/onlineServiceJS/src/gitRemote.test.mjs` | ✅ |
| selected_image pull/run + token-sync | `go_relayToTrae/src/container_image_test.go` | ✅ |

## 手工验收

1. 打开任务详情 `?relayToTrae=true`，点「启动」
2. 容器日志应出现 `token-sync: OK`，**不应**再出现 status push 401 / unregister
3. 克隆日志应出现 `bootstrap-clone remote normalized ssh→https`，目标为 `http://183.250.1.132:8012/...`，**不应**再出现 `Host key verification failed`
