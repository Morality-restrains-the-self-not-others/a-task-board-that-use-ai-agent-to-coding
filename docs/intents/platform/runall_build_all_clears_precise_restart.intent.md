# Intent: 全部重新编译完成后清空精准编译登记

## 背景与目标

runAll 页头「精准编译重启」旁的 `#precise-restart-reg-label` 展示 `.runall/precise_restart_services.txt` 中已登记服务数（例如「已登记 53 个服务」）。点击页头「全部重新编译」会按依赖序编译全部可编译服务，等价于已经消费这批待精准重启的变更。

约束 `.ai/01_project_constraints/42_precise_restart_service_registration.md` 已规定：`POST /api/build-all` / `Runner.BuildAll` **正常完成**后须清空登记并写入 `precise_restart_consumed_at`。该收尾曾落地后又从 `BuildAll` 丢失，导致编译全部完成后标签仍显示旧登记。

目标：全部重新编译正常完成后，登记文件为空、水位线已写，页面标签不再显示「已登记 N 个服务」。

## 范围与边界

- **范围内**：页头「全部重新编译」（`BuildAll`）正常完成路径清空登记 + 写 consumed-at；进度 `done` 事件发布前完成写盘，使 `updateProgress`→`refresh()`→`refreshPreciseRestartRegistrations()` 读到空列表。
- **范围外**：分组「全部重新编译」仍只裁剪本组成功项（`trimRegistrationsAfterGroupBuild`）；精准编译重启自身的失败保留语义不变。
- **中断**：`BuildAll` 因 cancel/`ctx.Done()` 返回时**不清空**登记。

## 约束与风险

- 清空发生在循环成功结束后、发布 `done` 之前，避免 SSE 完成刷新仍读到旧登记。
- 清空失败只记日志，不改变 `BuildAll` 的编译结果。
- 写 consumed-at 防止 Stop hook 见脏工作树立刻把刚消费的服务重新登记回来。

## 验收标准

1. `BuildAll` 正常完成后 `.runall/precise_restart_services.txt` 为空（或等价空登记）。
2. 同目录 `precise_restart_consumed_at` 被写入 unix 秒水位线。
3. 已取消的 `BuildAll` 保留登记文件内容。
4. 页头 `#precise-restart-reg-label` 在完成后不再展示「已登记 N 个服务」（由既有 `refresh()` 拉空列表）。

## 实施计划

1. `clearPreciseRestartRegistrationsAfterFullRebuild(cfgPath)`：`clearRegisteredServices` + `writePreciseRestartConsumedAt`
2. `Runner.BuildAll` 非中断收尾调用该函数
3. 单测：成功清空 / 中断保留 / 水位线

## 业务意图 → 事件对照

> 运维编排器消费本地登记文件，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: 全部重新编译完成后清空精准编译登记 | — | — | — | — | 运维编排器本地登记文件，不产生业务领域事件 |

## 变更记录

- 2026-08-17：补回 `BuildAll` 收尾清空登记。根因：约束 42 / OPT-20260811-035 已描述该行为，但 `clearPreciseRestartRegistrationsAfterFullRebuild` 未再被 `BuildAll` 调用，页头标签残留「已登记 N 个服务」。
