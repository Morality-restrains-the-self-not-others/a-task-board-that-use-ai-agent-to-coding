# [运行时] auto_run completed 但未自动提交/创建 PR：close 钩子被空 mountedAgentId 短路

## 基本信息

- 版本：1.0.0
- 日期：2026-08-17
- TraceId：`6a2a9c37101acc53f59a8487`
- 任务：`task_877171547301769216` / 评论 `cmt_877174604639006720`
- 容器：`task_877171547301769216_cmt_877174604639006720`
- Job：`49e04df5-be1e-412b-a1eb-567790d7fd77`（`auto_run_first`）
- 关联：失败经验 66（交付 done 锁死 / runtime-event）、edit-run 交付设计 `2026-07-23-edit-run-auto-delivery-pr-comment`

## 现象

- 层图/ztree 任务节点 **completed**，但未执行自动 commit → push → 创建 PR。
- Loki **无任何** `AUTO_RUN_DELIVERY_BEGIN|COMPLETE|FAILED`。

## 时间线（Loki，UTC）

| 时间 | 事件 |
|------|------|
| 13:09:17 | `BOOTSTRAP_PHASE` task_detail / clone_begin |
| 13:16:06 | `BOOTSTRAP_COMPLETE` |
| 13:16:06 | `AUTO_RUN_FIRST_INSTRUCTION_START`（`mounted_agent_comment_id=""`，`command_len=23`） |
| 13:16:07 | `AUTO_RUN_FIRST_INSTRUCTION_STARTED` job=`49e04df5-…` layer=`20260817_131605_9c3a93` |
| 13:16:08–13:16:51 | job_stream `running` → `step`/`chunk` → **`completed`** |
| （之后） | **缺失** `AUTO_RUN_DELIVERY_*` |

## 根因

`jobsRuntimeRunJob.mjs` 在 `proc.on('close')` 异步收尾中：

```js
if (!mountedAgentId || wasInterrupted) return;
// … completeMountedAgentComment …
// … triggerAutoRunDeliveryForJobAndMirror …
```

普通 `auto_run`（无 `at_mention_run.agent_comment_id`）创建 job 时 **不挂载** `mounted_agent_comment_id`。  
Job **已 completed**，但因 `mountedAgentId` 为空提前 return，**交付代码永不执行**。  
`edit_run` 亦常先交付再 `ensureEditRunMountedAgentComment`，同样会被该短路误伤。

契约单测 `shouldRunAutoRunDeliveryOnClose` 未覆盖「空 mount 仍应交付」，故未拦住回归。

## 修复

1. 抽出 `finalizeJobCloseSideEffects`：仅 `wasInterrupted` 跳过；Agent 评论收尾依赖 mount；**交付与 mount 解耦**。
2. `jobsRuntimeRunJob` close 钩子改调该函数。
3. 回归：`jobsRuntimeCloseSideEffects.test.mjs`（空 mount + auto_run/edit_run 仍触发 delivery）。

## 验收

```bash
cd trae-agent/onlineServiceJS && node --test \
  src/jobsRuntimeCloseSideEffects.test.mjs \
  src/jobsRuntime.autoRunDelivery.test.mjs
# 部署：commit 后 DOCKER_PUSH=1 ./buildDocker.sh，再 stop-vm→start-vm（或重建评论容器）
# Loki：同任务 completed 后应出现 AUTO_RUN_DELIVERY_BEGIN → COMPLETE|FAILED
```

## 存量容器

已 completed 且无 `auto_run_delivery.done` 的旧镜像容器：需拉新镜像并重启；启动补跑 `retryPendingAutoRunDeliveries` 仅对新代码生效。
