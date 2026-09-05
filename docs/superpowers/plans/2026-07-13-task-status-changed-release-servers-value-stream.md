# 价值流：任务状态变更事件与终态释放服务器

- 日期：2026-07-13
- 设计：`docs/superpowers/specs/2026-07-13-task-status-changed-release-servers-design.md`

## 价值主张

任务进入终态后自动释放计算资源，避免僵尸服务器费用与端口占用，且不增加用户操作步骤。

## 增量切片（按价值排序）

### Increment 1 — 状态变更发事件（MVP 核心）

| Step | 说明 | 验证 |
|------|------|------|
| S1 | PATCH 成功且 column/completed 变化 → 发 `TASK_STATUS_CHANGED` | Go 单测 mock Kafka |
| S2 | 无变化不发 | Go 单测 |

### Increment 2 — 终态释放 ECS

| Step | 说明 | 验证 |
|------|------|------|
| S3 | Consumer 识别终态 | 单测列名 / completed |
| S4 | 有 instance_id → 发 `CLOUD_SERVER_STOPPED` | 单测 publisher |
| S5 | 无配置 no-op | 单测 |

### Increment 3 — 终态释放 relay/mock

| Step | 说明 | 验证 |
|------|------|------|
| S6 | 有本地运行态 → HTTP stop | 单测 HTTP client mock |

## 影响的既有流

- 任务协作：`progress_column_id` 变更
- 云资源：`CLOUD_SERVER_STOPPED`、relay/mock stop

## YAML 字段（拟写入 conf/value-stream）

- `task-task-service.tasks.progress_column_id`
- `task-task-service.tasks.completed`
- `cloud.cloud_server_config.instance_id`
- `cloud.cloud_server_config.server_url`
