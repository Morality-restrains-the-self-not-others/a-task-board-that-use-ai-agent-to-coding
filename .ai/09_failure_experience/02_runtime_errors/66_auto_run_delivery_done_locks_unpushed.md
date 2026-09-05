# [运行时] auto_run 任务 completed 但未推远端（交付失败写 done + runtime-event 404）

## 基本信息

- 版本：1.1.0
- 创建日期：2026-07-20
- 更新日期：2026-07-21
- 编号：66
- 维护者：Trae AI 团队

## 现象

- 任务详情 ztree：首指令 **completed**，仍显示「N 个提交可推送」/「推送并创建PR」。
- 「自动运行步骤说明」第 4 步（完成后交付：commit → push → PR）未完成。
- 典型任务：`task_13646028863068037877`（层 `20260720_074701_4ea0bc`）；复现：`task_13759732724159256867`（2026-07-21）。
- 手动点推送：`container-layer-git-push` → 上游 `…/layers/…/git/push` **400**（trace `758d0f7c-7f04-4a00-9ada-4d4cb883a9e4`）。

## 时间线证据（Loki）

| 时间 (CST) | 事件 |
|------------|------|
| 15:46–15:47 | 容器 `runtime-event` 经 taskAgentSupport → Django **404**（事件未进 Loki） |
| 15:47:01 | 首指令层创建 / 作业运行 |
| 15:53:18 | `layer-github-oauth-access-tokens` **200**（交付已开始换票） |
| 15:57:15 | 手动 `container-layer-git-push` → 上游 `git/push` **400** |

## 根因

1. **交付失败仍写 `auto_run_delivery.done`**：`runAutoRunDelivery` 在 push 失败 / 异常时也写 done，导致同容器生命周期内不再重试第 4 步；ztree 长期「可推送」。
2. **简单 commit 路径**：交付侧未复用嵌套感知的 `commitLayerGitWorkdirs`，与层图「相对父层差异 / 嵌套 ahead」不一致时易漏提交。
3. **可观测性缺口**：`taskAgentSupport.forwardsToCloudService` **未包含** `runtime-event`，AUTO_RUN_DELIVERY_* 落到 Django 404，Loki 无法检索交付成败。
4. **PR 元数据缺失**：`layer-github-oauth-access-tokens` 未返回 `pr_base_branch` / `pr_title`，即便 push 成功也难建 PR。

## 解决方案

1. `shouldSkipAutoRunDelivery`：仅 `push_ok=true` / `skipped_clean` 跳过；`push_ok=false` / `error` 允许重试（次数上限）。
2. 交付改用 `commitLayerGitWorkdirs`；push 后若仍 ahead 视为失败。
3. 交付结束后 `mirrorLayerGraphToTaskCloudSSE`；启动时 `retryPendingAutoRunDeliveries`。
4. `runtime-event` 转发至 taskCloudService。
5. OAuth token 响应附带 `pr_base_branch`（merge_target）与 `pr_title`。

## 验证

```bash
cd trae-agent/onlineServiceJS && node --test \
  src/autoRunOrchestration.test.mjs \
  src/autoRunDeliveryHooks.test.mjs \
  src/jobsRuntime.autoRunDelivery.test.mjs
cd taskAgentSupport && go test ./src/ -count=1
# 部署：重启 taskAgentSupport；容器侧 DOCKER_PUSH=1 ./buildDocker.sh 后 stop-vm→start-vm
# Loki：{job=~".+"} |= "AUTO_RUN_DELIVERY" 应可见 BEGIN/COMPLETE 或 FAILED（非 Django 404）
```

## 关联

- `docs/superpowers/specs/2026-07-13-auto-run-first-instruction-and-delivery-design.md`
- `trae-agent/onlineServiceJS/autoRunStep.md` 步骤 4
- 近亲：`58_auto_run_skipped_after_credentials_recovery.md`、`56_nested_committed_no_push_ahead_and_clone_seal.md`
