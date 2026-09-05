# 价值流：顶层交付物排队自动执行调度节奏

- 日期：2026-07-19
- 设计：`docs/superpowers/specs/2026-07-19-top-deliverable-queued-auto-run-schedule-design.md`

## 影响

| Stream | Step | 动作 |
|--------|------|------|
| task-management | create-task-auto-run-backend-start | 并列保留（立即路径） |
| task-management | top-deliverable-queued-auto-run-schedule | **新增 active** |
| cloud-compute | workspace-machine-idle-policy | 正交（闲置复用仍生效） |

## 增量切片

1. **I1** 节奏字段 + PATCH 校验（仅顶层）
2. **I2** 入队/出队 membership + 事件
3. **I3** 窗口判定 + deferred
4. **I4** Dispatcher 出队顺序 + start-vm-auto(started_via)
5. **I5** 队列 GET + 前端表单/徽章

YAML 已写入 `conf/value-stream.yaml`。
