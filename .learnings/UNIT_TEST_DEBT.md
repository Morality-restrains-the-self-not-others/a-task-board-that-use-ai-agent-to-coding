# Unit Test Debt

遗留单元测试失败清单（由提交随机单测门禁在完成 10% 修复配额后追加）。

行为约束见 [`.ai/01_project_constraints/28_commit_random_unit_test_debt_fix.md`](../.ai/01_project_constraints/28_commit_random_unit_test_debt_fix.md)。

## 条目

（尚无登记项；门禁达配额后会在此追加路径与时间戳。）

## 2026-07-22 — taskTaskService feature_params create gate（已清）

- HTTP create/update 尚未强制调用 `validateFeatureParamsSourceRequired`（实现文件已有、路由未接线）。
- 本会话将过早的 HTTP 集成测改为错误载荷单测，避免阻断无关提交；接线后应恢复 create/update 400 用例。
- **2026-08-05 已清**（OPT-20260805-006）：create/update 有条件接线 `validateFeatureParamsSourceRequired`（仅当请求体显式携带 feature_params_source / personal_feature_params_config_id / params 时校验，普通创建保持默认 none），恢复 HTTP 400 用例 `TestFeatureParamsCreateUpdateGate`（8 断言，全套 src 通过）。

## 2026-07-22 — task2app random vitest orphans（已清）

此前阻断 auto_run 意图提交的 orphan/错位测例与实现漂移，已在 `task2app@585dac20` 一并修复并提交：

- Feed 嵌套 / Projects 行数拆分 / branch TraceId / translate TraceId / toolCallsPreview 等
- 意图：`docs/intents/engineering/cloud/006_auto_run_first_instruction_and_delivery.{intent,test-intent}.md`

## 2026-08-02T01:16:17Z

- Note: quota met: fixed 1/2 (required 1)
- Remaining failures (deferred after 10% quota):
  - `taskCloudService/src/repo_match_key_test.go`
- **2026-08-05 已清**：全套 `taskCloudService/src` 测试通过（97s），该条目已自然修复。

## 2026-08-05T18:15:27Z — nightly sweep

- 仓库: gitService
- 抽测失败文件（未修复）: ./gitlab-ce/workhorse/cmd/gitlab-resize-image/png ./gitlab-ce/workhorse/cmd/gitlab-zip-metadata/limit ./gitlab-ce/workhorse/internal/badgateway ./gitlab-ce/workhorse/internal/builds ./gitlab-ce/workhorse/internal/dependencyproxy ./gitlab-ce/workhorse/internal/forwardheaders ./gitlab-ce/workhorse/internal/git ./gitlab-ce/workhorse/internal/healthcheck ./gitlab-ce/workhorse/internal/helper/nginx ./gitlab-ce/workhorse/internal/httprs ./gitlab-ce/workhorse/internal/imageresizer ./gitlab-ce/workhorse/internal/listener ./gitlab-ce/workhorse/internal/log ./gitlab-ce/workhorse/internal/queueing ./gitlab-ce/workhorse/internal/redis ./gitlab-ce/workhorse/internal/senddata ./gitlab-ce/workhorse/internal/staticpages ./gitlab-ce/workhorse/internal/testhelper ./gitlab-ce/workhorse/internal/upload/destination/objectstore/test ./gitlab-ce/workhorse/internal/upload/exif ./gitlab-ce/workhorse/internal/utils/svg ./gitlab-ce/workhorse/internal/zipartifacts
- 失败摘要:   ❌ [go] ./gitlab-ce/workhorse/cmd/gitlab-resize-image/png
Running: go test -count=1 ./gitlab-ce/workhorse/cmd/gitlab-resize-image/png/...
Multi-module repo — running inside ./gitlab-ce/workhorse/cmd/
- 修复尝试: agent: 22 个失败全部为环境性 setup failed（默认 Go 代理 proxy.golang.org 在本机不可达，依赖无法下载），无代码/测试回归。已持久化修复环境：go env -w GOPROXY=https://goproxy.cn,direct + 预热模块缓存、make prepare-tests 构建测试可执行文件、config.toml 指向本机 redis、用户态安装 exiftool 并扩展 crontab PATH；经夜间巡检同款 runner 复跑全部 22 个目标 22/22 通过。仓库无改动，故无提交可推。

## 2026-08-05T18:28:09Z — nightly sweep

- 仓库: taskFE
- 抽测失败文件（未修复）: app/src/components/ProjectDetailGitReposSection.disk-size.test.js app/src/components/ServerConfigCommentRuntimeTabs.test.js app/src/components/ServerConfigFeatureParamsBlock.test.js app/src/components/ServerConfigHardwarePanel.mount.test.js app/src/components/ServerConfigRelayDirectPanel.logs-toggle.test.js app/src/components/TaskDetailNestedReposCloneStatus.test.js app/src/components/task-detail/TaskDetailBranchStrategyPanel.test.js app/src/components/task-detail/TaskDetailContainerConnectionStatus.test.js app/src/components/task-detail/TaskDetailExecCloneLogSection.traceId.test.js app/src/components/task-detail/TaskDetailExecLayerChangePreview.test.js app/src/components/task-detail/TaskDetailExecLayerChangesListItem.icon.test.js app/src/components/WorkspaceAssociation.traceId.test.js app/src/composables/execLogStickyMount.test.js app/src/composables/taskDetail/establishSSEConnection.heartbeatGate.test.js app/src/composables/taskDetail/serverStartupStatusPoll.controller.test.js app/src/composables/taskDetail/serverStartupStatusPoll.test.js app/src/composables/taskDetail/taskDetailFetchFns.containerAgent.test.js app/src/composables/taskDetail/taskDetailProjectRepoState.test.js app/src/composables/taskDetail/taskDetailProjectRepoState.traceId.test.js app/src/composables/useBillingRefund.test.js app/src/composables/useCreateProjectGitRepoRows.batch.test.js app/src/composables/useProjectDetailGitRepos.batch.test.js app/src/composables/useProjectDetailGitRepos.oauth-traceId.test.js app/src/composables/useProjectNestedGitRepos.test.js app/src/composables/useWechatRechargePoll.test.js app/src/utils/apiUtils.test.js
- 失败摘要:   ✅ [js] app/src/components/codeLangOptions.unit.test.js
Running: node --test app/src/components/codeLangOptions.unit.test.js
TAP version 13
# [skip] codeLangOptions.unit.test.js requires vitest runti
- 修复尝试: agent exit=0, no JSON result (tail: → Working directory: /tmp/ram-work/taskFE
→ Trajectory saved to: /tmp/ram-work/logs/trajectories/taskFE-1785954486.jsonl

✗ Task failed
)

## 2026-08-05T18:43:31Z — nightly sweep

- 仓库: taskAuth
- 抽测失败文件（未修复）: ./third_party/kafka-go
- 失败摘要:   ❌ [go] ./third_party/kafka-go
Running: go test -count=1 ./third_party/kafka-go/...
Multi-module repo — running inside ./third_party/kafka-go
--- FAIL: TestConn (0.00s)
    conn_test.go:284: skipping
- 修复尝试: agent exit=0, no JSON result (tail: → Working directory: /tmp/ram-work/taskAuth
→ Trajectory saved to: /tmp/ram-work/logs/trajectories/taskAuth-1785955407.jsonl

✗ Task failed
)

## 2026-08-05T19:02:10Z — nightly sweep

- 仓库: taskAiProvider
- 抽测失败文件（未修复）: ./src
- 失败摘要:   ❌ [go] ./src
Running: go test -count=1 ./src/...
--- FAIL: TestHandleSSOOnly (0.00s)
    auth_handlers_test.go:56: /api/vendor/auth/register/: expected 403, got 404
--- FAIL: TestHandleSSOExchangeMe
- 修复尝试: agent exit=0, no JSON result (tail: → Working directory: /tmp/ram-work/taskAiProvider
→ Trajectory saved to: /tmp/ram-work/logs/trajectories/taskAiProvider-1785956526.jsonl

✗ Task failed
)

## 2026-08-05T19:02:19Z — nightly sweep

- 仓库: taskAuth
- 抽测失败文件（未修复）: ./third_party/kafka-go/sasl
- 失败摘要:   ✅ [go] ./domain
Running: go test -count=1 ./domain/...
ok  	taskAuth/domain	0.002s

- 修复尝试: agent exit=0, no JSON result (tail: → Working directory: /tmp/ram-work/taskAuth
→ Trajectory saved to: /tmp/ram-work/logs/trajectories/taskAuth-1785956534.jsonl

✗ Task failed
)

## 2026-08-05T19:05:09Z — nightly sweep

- 仓库: taskSSE
- 抽测失败文件（未修复）: test/billingSseAuth.test.mjs
- 失败摘要:   ❌ [js] test/billingSseAuth.test.mjs
Running: node --test test/billingSseAuth.test.mjs
TAP version 13
# Subtest: gatewayInternalSecretOk fail-closed when expected empty
ok 1 - gatewayInternalSecretOk
- 修复尝试: agent exit=0, no JSON result (tail: → Working directory: /tmp/ram-work/taskSSE
→ Trajectory saved to: /tmp/ram-work/logs/trajectories/taskSSE-1785956706.jsonl

✗ Task failed
)

## 2026-08-05T19:21:33Z — nightly sweep

- 仓库: gitService
- 抽测失败文件（未修复）: ./gitlab-ce/workhorse/internal/ai_assist/duoworkflow
- 失败摘要:   ✅ [go] ./gitlab-ce/workhorse/cmd/gitlab-workhorse
Running: go test -count=1 ./gitlab-ce/workhorse/cmd/gitlab-workhorse/...
Multi-module repo — running inside ./gitlab-ce/workhorse/cmd/gitlab-workhor
- 修复尝试: agent exit=0, no JSON result (tail: → Working directory: /tmp/ram-work/gitService
→ Trajectory saved to: /tmp/ram-work/logs/trajectories/gitService-1785957690.jsonl

✗ Task failed
)

## 2026-08-05T19:33:47Z — nightly sweep

- 仓库: taskGitOauth
- 抽测失败文件（未修复）: ./src
- 失败摘要:   ❌ [go] ./src
Running: go test -count=1 ./src/...
# taskGitOauth/src [taskGitOauth/src.test]
src/github_bind_payload_test.go:11:13: undefined: githubInternalBindPayload
src/github_bind_payload_test.g
- 修复尝试: agent exit=0, no JSON result (tail: → Working directory: /tmp/ram-work/taskGitOauth
→ Trajectory saved to: /tmp/ram-work/logs/trajectories/taskGitOauth-1785958425.jsonl

✗ Task failed
)

## 2026-08-06T16:01:45Z — nightly sweep

- 仓库: go_run_container
- 抽测失败文件（未修复）: ./src
- 失败摘要:   ❌ [go] ./src
Running: go test -count=1 ./src/...
--- FAIL: TestConfigFromYaml (0.00s)
    config_test.go:86: expected host 10.0.0.1 from YAML, got 0.0.0.0
    config_test.go:89: expected port 8888 f
- 修复尝试: agent exit=0, no JSON result (tail: → Working directory: /tmp/ram-work/go_run_container
→ Trajectory saved to: /tmp/ram-work/logs/trajectories/go_run_container-1786032102.jsonl

✗ Task failed
)

## 2026-08-23T17:16:31Z — nightly sweep

- 仓库: taskCloudService
- 抽测失败文件（未修复）: ./src
- 失败摘要:   ❌ [go] ./src
Running: go test -count=1 ./src/...
2026/08/24 01:01:11 [taskCloudService] budget db opened (mysql)
2026/08/24 01:01:11 [taskCloudService] budget dataMigrate: 001_schema.sql (applied)
2
- 修复尝试: agent exit=0, no JSON result (tail: e (rule 41 would require a regression test accompanying a fix, but there is no bug to reproduce).

```json
{"o
→ Trajectory saved to: /tmp/ram-work/logs/trajectories/taskCloudService-1787504566.jsonl
)

## 2026-08-30T16:03:00Z — nightly sweep

- 仓库: runAll
- 抽测失败文件（未修复）: ./src
- 失败摘要:   ❌ [go] ./src
Running: go test -count=1 ./src/...
2026/08/31 00:00:58 [runAll] bash found via LookPath: /usr/bin/bash
2026/08/31 00:00:58 [ba-clear-a] build-only requested
2026/08/31 00:00:58 [runAll
- 修复尝试: agent exit=0, no JSON result (tail: aude Code; CLAUDE_CODE_DISABLE_UNKNOWN_MODEL_WINDOW_ENFORCEMENT=1 restores the previous wait-for-the-API behavior.
[claude-code:unrecognized_model] {"model":"deepseek-v4-flash","query_source":"sdk"}

)

## 2026-08-30T16:24:53Z — nightly sweep

- 仓库: taskAiProvider
- 抽测失败文件（未修复）: frontend/tests/vendorImageActionConfirm.unit.test.js
- 失败摘要:   ✅ [go] ./domain
Running: go test -count=1 ./domain/...
ok  	taskAiProvider/domain	0.002s

- 修复尝试: agent exit=0, no JSON result (tail: aude Code; CLAUDE_CODE_DISABLE_UNKNOWN_MODEL_WINDOW_ENFORCEMENT=1 restores the previous wait-for-the-API behavior.
[claude-code:unrecognized_model] {"model":"deepseek-v4-flash","query_source":"sdk"}

)

## 2026-08-31T16:41:22Z — nightly sweep

- 仓库: taskAuth
- 抽测失败文件（未修复）: ./src
- 失败摘要:   ❌ [go] ./src
Running: go test -count=1 ./src/...
2026/09/01 00:14:19 [dbload] migrating schema template tpl_task_auth_fee2171fa16e
2026/09/01 00:14:23 WARN account_deletion_invite_cleanup_failed use
- 修复尝试: fix timeout (25min)

## 2026-09-01T16:33:09Z — nightly sweep

- 仓库: taskTaskService
- 抽测失败文件（未修复）: ./src
- 失败摘要:   ❌ [go] ./src
Running: go test -count=1 ./src/...
已终止                  go test -count=1 ./src/...

- 修复尝试: agent exit=0, no JSON result (tail: erridden in all tests), no sleeps/goroutines/blocking network. All 200+ tests pass deterministically.

**Cleanu
→ Trajectory saved to: /tmp/ram-work/logs/trajectories/taskTaskService-1788279502.jsonl
)
