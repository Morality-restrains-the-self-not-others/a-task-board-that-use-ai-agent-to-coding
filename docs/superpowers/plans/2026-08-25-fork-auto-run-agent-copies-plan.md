# 实施计划：Fork 自动运行按智能体（模型）复制

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-fork-auto-run-agent-copies-design.md`
- **DDD**: `docs/superpowers/plans/2026-08-25-fork-auto-run-agent-copies-ddd.md`

## 任务

每步 Red → Green。删除旧双按钮与仅派生数量框（Logic-Rollback：不保留双路径）。

- [x] T1 纯函数：`buildAgentModelOptionsFromFeatureParams` / `agentModelsEnvFromContextPack` + 单测
- [x] T2 模态：单选派生方式 +「确认派生」；无 `fork-copy-count-input`；仅派生 payload `{ autoRun:false, copyCount:1 }`；更新 `ForkAutoRunConfirmModal.test.js`
- [x] T3 自动运行章节：智能体资源 + 模型多选；未选模型确认 disabled；切换资源清空并预勾默认（T14/T17）
- [x] T4 `forkTask`：仅派生 1 次无 `agent_models`/`fork_count`；自动运行 N 次每份 `agent_models`，不用 `fork_count`；`${batch}:${i}`；只 open 第一份（T10/T15/T16）
- [x] T5 Header / `useTaskDetail` 透传 featureParamsSource、agents、`createClickGuard`
- [x] T6 taskTaskService：`parseCreateTaskAgentModels`；非法 400；合法写入 `autoRunTriggerParams`；`notifyContainerAgentPending` 把 `agent_models` 写入 `context_pack`（T18）
- [x] T7 trae-agent：`maybeStartAutoRunFirstInstruction` 把 pack 中模型写入 job `env`（TASK_AGENT_MODEL）；taskAIComment `parseContainerJobContext` 已支持，补 context_pack 读取测例
- [x] T8 Playwright：仅派生 1 POST；两模型 2 POST 不同 model；更新 wsd / 意图（T11 废止）
- [x] T9 事件：每份仍 `publishTaskCreated`（已有）；文档对照表已更新

## 日志

- `[forkTask] event=fork_copy_model`：copy_index、model、provider（无密钥）
- `[taskTaskService] event=auto_run_agent_models`：task_id、model
- `[onlineServiceJS] AUTO_RUN_FIRST_INSTRUCTION_START` 增加 model 字段
