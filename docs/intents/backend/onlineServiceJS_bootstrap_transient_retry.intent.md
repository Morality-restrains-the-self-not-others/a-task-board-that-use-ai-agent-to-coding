# onlineServiceJS 引导在 task-detail 后卡住（瞬时断连）

## 意图

修复任务详情页「启动」后日志停在「容器已启动，开始拉取任务详情…」、无法进入克隆/配置完成态的问题。

## 现象

1. `task-detail` / `repo-clone-credentials` / 仓库克隆实际已成功。
2. 收尾 `git-clone-progress` 与 `feature-params-env` 时 `127.0.0.1:8011` 出现 `UND_ERR_SOCKET` / `ECONNREFUSED`。
3. `postJson` 仅做 localhost↔127.0.0.1 互备，**无时间维退避重试**，引导失败后 UI 仍像卡在「拉取任务详情」。
4. 同期 `runAll` 因 `shutdown-self` 重启，新实例 `cleanupOrphanManagedServices` 把仍在跑的托管服务当孤儿 `SIGKILL`，放大断连窗口。

## 验收

1. `postJson` 对 `ECONNREFUSED` / `ECONNRESET` / `UND_ERR_SOCKET` 等瞬时错误按 `TASK_API_POST_JSON_TRANSIENT_RETRIES`（默认 5）与退避重试，成功后引导继续。
2. 业务 4xx（如 401）不重试。
3. 引导关键步骤有明确控制台日志，且带稳定前缀供前端检索/高亮：
   - `BOOTSTRAP_PHASE=task_detail_begin|clone_begin|feature_params_begin`
   - `BOOTSTRAP_COMPLETE`（含「任务引导完成…」）
   - `BOOTSTRAP_FAILED phase=…`（失败必现于启动日志，避免误判卡在拉详情）
4. 前端启动日志面板对上述行分类高亮（`classifyBootstrapMilestoneLogLine`），并用 `resolveBootstrapStatusFromLogs` 显示状态条；**失败态有独立徽章**（不仅红字）。
5. `runAll`：若上一实例 `/api/shutdown-self` 成功，新实例**跳过**托管端口 orphan 强杀；热替换须用已含 skip-orphan 的二进制（`build.sh` 校验 `skip orphan port cleanup`）。
6. 单测：`saasTaskCloud.transientRetry.test.mjs`；`relayToTraeUtils` 里程碑分类与状态条；`TestCleanupOrphanManagedServices_SkipsWhenShutdownSelf`；`TestCapabilitySkipOrphanOnShutdownSelfMarker`。



## 业务意图 → 事件对照

> 精修（2026-07-15）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。

**无对应事件**：瞬时断连重试修复，无新增业务事件契约。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| onlineServiceJS 引导在 task-detail 后卡住（瞬时断连） | — | — | — | — | 瞬时断连重试修复，无新增业务事件契约 |
## 变更记录

- 2026-07-10：初版 — 根因定位 + postJson 瞬时重试 + runAll shutdown-self 保活托管服务 + 引导进度日志。
- 2026-07-10：补充 `BOOTSTRAP_*` 稳定标记与前端启动日志高亮，减少误判为卡在 task-detail。
- 2026-07-10：前端引导失败独立徽章/状态条；runAll `build.sh` 强制校验 skip-orphan 能力标记。
