# auto_run 完成后交付（push/PR）未执行 — 修复设计

- **日期**: 2026-07-21
- **作者**: goal-mode / 0-auto-flow（自动采纳）
- **状态**: approved（goal-mode 跳过 USER GATE）
- **页面**: `…/task-detail/task_13759732724159256867/`（首指令 completed，ztree 仍「可推送 / 推送并创建PR」）
- **基线设计**: `2026-07-13-auto-run-first-instruction-and-delivery-design.md`
- **失败经验**: `.ai/09_failure_experience/02_runtime_errors/66_auto_run_delivery_done_locks_unpushed.md`
- **python_api_approval**: n/a（不新增 Python 接口）

## 1. 问题

「自动运行步骤说明」第 4 步要求：首指令成功后 → 身份 sync → commit → push → 创建 PR。  
现象：第 3 步 completed，第 4 步未闭环，层图长期「N 个提交可推送」。

## 2. 根因（已核实代码 HEAD）

1. `runAutoRunDelivery` 在 **push 失败与异常路径仍写** `runtime/auto_run_delivery.done`。
2. 跳过条件仅为 **文件存在**（`hasAutoRunDeliveryDone`），失败后同容器生命周期内永不重试。
3. 交付 commit 未复用嵌套感知的 `commitLayerGitWorkdirs`。
4. 无启动补跑；交付后未强制 remirror 层图。
5. （加固）工作分支若仅在 `branch_strategy.work_branch_name`，交付/oauth-refresh 只读 `task.target_branch` 可能漏传。

## 3. 方案（采纳）

| 项 | 做法 |
|----|------|
| 幂等 | `shouldSkipAutoRunDelivery`：仅 `push_ok=true` / `skipped_clean` / 2xx 成功跳过；`push_ok=false` / `error` 可重试 |
| 写 done | 仅成功或「nothing to push 且 ahead=0」 |
| 重试 | `auto_run_delivery.retry` 计数，默认上限 3；`server` listen 后 `retryPendingAutoRunDeliveries` |
| commit | `commitLayerGitWorkdirs` |
| 层图 | 交付后 `mirrorLayerGraphToTaskCloudSSE` |
| 分支 | `resolveAutoRunDeliveryTargetBranch` = `collectRepoBranchPlans().sharedWorkBranch` |

## 4. 成功标准

| # | 标准 | 验证 |
|---|------|------|
| S1 | push 失败不写成功 done；二次调用可再交付 | `autoRunOrchestration.test.mjs` |
| S2 | 启动时补跑 completed + auto_run_first 且未成功交付的 job | `autoRunDeliveryHooks.test.mjs` |
| S3 | 工作分支可从 branch_strategy 解析 | hooks 单测 |
| S4 | 文档与 failure #66 一致 | 本设计 + design changelog |

## 5. 🕸️ CRG

- 图仅索引少量文件（soft-dep unavailable for full blast radius）；已用 failure #66 + 定向读源码替代。
- 改动面：`trae-agent/onlineServiceJS` 交付编排 / oauth push 拆分；无新 HTTP 路由。

## 6. 部署说明（非代码）

存量任务容器仍跑旧镜像时：需 `DOCKER_PUSH=1 ./buildDocker.sh` 后对该 VM `stop-vm`→`start-vm`，启动补跑才会生效。
