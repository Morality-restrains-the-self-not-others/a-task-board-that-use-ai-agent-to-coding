# 意图：selected_image 启动前清空 relay 状态残留

- **日期**: 2026-07-14
- **状态**: 已实现
- **背景**: `go_relayToTrae/.relay_online_state/` 可累积到数 GB（layers / job_logs / logs），且子目录常为容器 root 属主，宿主机 `rm` 失败。重建服务进程通常不需要这些残留；每次启动容器应先清空再挂载。

## 验收标准

- [x] `startSelectedImageContainer` 在 `docker run` 前清空 `RELAY_ONLINE_STATE_BASE`（默认 `go_relayToTrae/.relay_online_state`）下**全部** per-task 残留目录
- [x] 当前任务目录重建为仅含空 `runtime/`，再 bind-mount
- [x] 宿主机 `RemoveAll` 因权限失败时，用**刚 pull 的业务镜像** `docker run --entrypoint rm` 以 root 删除（不额外拉 alpine）
- [x] 仅允许删除 state base 的直接子目录（防误删）
- [x] 单元测试覆盖：残留清理、docker 回退路径

## 变更要点

| 组件 | 变更 |
|------|------|
| `go_relayToTrae/src/container_image.go` | `wipeHostStatePath` / `clearRelayOnlineStateResiduals`；`ensureSelectedImageHostStateRoot` 启动前整树清空 |
| `go_relayToTrae/src/container_image_test.go` | 残留清理与 docker 回退测例 |



## 业务意图 → 事件对照

> 精修（2026-07-15）：进程内清状态无跨边界副作用，不投递 MQ。

**无对应事件**：selected_image 启动前清空本机 `.relay_online_state` 为进程内副作用，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| selected_image 启动前清空 relay 状态残留 | — | — | — | — | 进程内清理，无 MQ 业务事件 |
## 变更记录

| 日期 | 说明 |
|------|------|
| 2026-07-14 | 初版：启动前清空 `.relay_online_state` 残留，解决数 GB 堆积风险 |
