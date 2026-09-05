# task_terminal_progress_skip_auto_run（测试意图 · 后端）

## 测试目标

证明任务更新进入终态时不校验、不触发自动运行；非终态仍校验。

## 测试分层

| 层 | 落点 |
|----|------|
| 单元 | `taskTaskService/src/terminal_kind_test.go` — `updateEntersTerminal` |
| 集成 | `taskTaskService/src/auto_run_test.go` — PATCH 进度 × 镜像已删 |
| 前端契约 | `taskFE/.../taskDetailSubtreeFetch.test.js` — PATCH body 不含 `auto_run` |

## 用例矩阵

| ID | 前置 | 动作 | 期望 |
|----|------|------|------|
| T1 | auto_run=true，镜像 lookup 失败 | PATCH `progress_column_id` → 已完成 | 200；不 start-vm |
| T2 | 同 T1 | PATCH → 已取消 | 200；不 start-vm |
| T3 | 同 T1 | PATCH → 进行中 | 400 `AUTO_RUN_RUNTIME_ENV_REQUIRED` |
| T4 | T1 成功 | 发布 TASK_STATUS_CHANGED | 1 次，column=已完成列 |
| T5 | 详情页改进度 | `onProgressStatusChange` | body 仅 `progress_column_id` |

## 数据与环境

沿用 `startAutoRunMockServices`；创建成功后把 `lookupInstalledImageFn` 改为返回 nil。进度列校验走 `X-Task-Test-Skip-Django-Validate` + 列名 header。

## 通过标准

T1–T5 全绿；既有 `auto_run_test.go` 回归全绿。
