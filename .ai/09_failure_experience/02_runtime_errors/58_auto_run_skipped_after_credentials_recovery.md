# [运行时] auto_run 在凭证恢复克隆成功后未启动首条 Agent 指令

## 症状

- 任务 `auto_run=true`，start-vm 成功；容器最终完成仓库克隆与 feature-params，任务详情可展开「自动运行步骤说明」。
- **没有**出现以任务标题/描述为内容的 trae job（智能体未执行）。
- 典型任务例：`task_13639601097151956877`（2026-07-20）。

## 时间线证据（Loki / 本地服务日志）

| 时间 (CST) | 事件 |
|------------|------|
| 13:57 | `task_auto_run_start_vm_ok` |
| 13:59–14:01 | `repo-clone-credentials` **409**（凭证未齐） |
| 14:03:10 | `repo-clone-credentials` **200**（用户批准 Git 后恢复） |
| 14:03:19 | `feature-params-env` **200** → bootstrap 完成 |
| 之后 | 仅 heartbeat / layer-graph-push；无 `AUTO_RUN_FIRST_INSTRUCTION_*` |

## 根因

1. `server.mjs` 仅在 **首次** `runBootstrapAfterListen` **成功返回后**调用 Agent kickoff。
2. 首听遇 `REPO_CLONE_CREDENTIALS_INCOMPLETE` 时：抛错 → 调度 `scheduleBootstrapCredentialsRecovery` → **跳过** kickoff。
3. 恢复轮询成功后只 `registerBootstrapCloneJob`，**未**调用 `maybeStartAutoRunFirstInstruction` / at_mention 路径。

## 修复

- 抽取 `postBootstrapAgentKickoff.mjs`（合法 at_mention 优先，否则 auto_run；残缺 at_mention 回退 auto_run）。
- `scheduleBootstrapCredentialsRecovery` 成功分支调用 `kickoffAfterCredentialsRecovery`。
- 单测：`postBootstrapAgentKickoff.test.mjs`。

## 验收

```bash
cd trae-agent/onlineServiceJS && node --test src/postBootstrapAgentKickoff.test.mjs
# 公网：提交后 DOCKER_PUSH=1 ./buildDocker.sh，stop-vm→start-vm 滚动新镜像后
# 容器日志应含 AUTO_RUN_FIRST_INSTRUCTION_START / STARTED（或合法 AT_MENTION job）
```

## 相关

- 意图：`006_auto_run_first_instruction_and_delivery`
- 近亲：`47_bootstrap_credentials_incomplete_skips_clone.md`（凭证 409 本身；本条为恢复后的 Agent 漏触发）
