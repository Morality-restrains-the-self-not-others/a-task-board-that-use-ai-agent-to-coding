# 价值流：Fork 自动运行按智能体（模型）复制

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-fork-auto-run-agent-copies-design.md`

## Related Value Streams

扩展既有 Fork 确认流（`docs/intents/frontend/task_detail_fork_auto_run_confirm`）与 2026-08-23 副本数量增量，不是全新价值流。

| 既有流 | 交付 | 本增量变化 |
|--------|------|------------|
| `todo-fork-from` / Fork 确认 auto_run | 确认后 POST `fork_from` + `auto_run` | 仅派生始终 1 份；自动运行按所选模型循环 POST |
| 2026-08-23 copy-count | 数量框 1–99；N>1 可用 `fork_count` 同构批 | 本弹窗删除数量框；自动运行禁止 `fork_count`（异构 `agent_models`） |
| `create-task-auto-run-backend-start` | 创建后 pending agent + start-vm | 每份把该份 `agent_models` 写入 context_pack，首 job 覆盖 `TASK_AGENT_MODEL` |
| 层级图「选择模型」 | 人类评论 `agent_models` → instruct env | 复用同一契约接到 auto_run 首指令 |

**冲突检查**: 服务端 `fork_count` 批接口保留给其他调用方；本弹窗不再走该路径。不撤销 copy-count API。

## 端到端增量

用户打开任务详情 → Fork → 单选「仅派生」或「自动运行并派生」→（自动运行）选智能体资源并多选模型 → 确认派生 → 仅派生 1 次 POST；自动运行 N 次独立 POST（每份不同 `agent_models`）→ 打开第一份详情 → 工作面板出现 N 张新卡 → 各副本 auto_run 首 job 使用对应模型。

## 切片（垂直，按交付顺序）

1. 模态：派生方式单选 + 底部「确认派生」；删除数量框与双动作按钮
2. 自动运行：智能体资源 + 模型多选；N = 勾选数；confirm payload
3. `forkTask`：仅派生恒 1；自动运行循环 POST + 每份 `agent_models`，不用 `fork_count`
4. taskTaskService：解析 `agent_models` → `autoRunTriggerParams` → pending agent `context_pack`
5. trae-agent / taskAIComment：首 job `env` 覆盖 `TASK_AGENT_MODEL`
6. 单测 + Playwright T3/T10/T14–T18

## 测试点

见 `docs/intents/frontend/task_detail_fork_auto_run_confirm.test-intent.md` T3、T9–T18（T11 废止）。
索引图 `docs/flows/value-stream-test-integration.wsd` Fork 注记改为按模型复制。

## 字段

- `task-task-service.task_tasks.feature_params_source`
- `task-task-service.task_tasks.personal_feature_params_config_id`
- pending agent `context_pack.agent_models`（JSON；description：与层级图同形 `[{provider, model}]`）
