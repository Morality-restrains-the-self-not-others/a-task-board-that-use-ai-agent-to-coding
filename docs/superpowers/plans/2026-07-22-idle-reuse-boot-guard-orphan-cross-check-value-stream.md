# 价值流：闲置复用启动保护 + 孤儿交叉校验

- **日期**: 2026-07-22
- **设计**: `docs/superpowers/specs/2026-07-22-idle-reuse-boot-guard-orphan-cross-check-design.md`

## 影响流

| Stream | Step | 变化 |
|--------|------|------|
| cloud-integration / workspace-machine-idle-policy | idle reuse | 候选须 `idle_since`；boot 中不复用 |
| cloud-integration / ecs orphan reconcile | orphan delete | cross-CSC 持有则 skip |

## 字段

- `task-cloud-service.cloud_server_configs.idle_since` — 复用门闩（读）
- `task-cloud-service.cloud_server_configs.instance_id` — orphan owned 集合
- `task-cloud-service.cloud_server_configs.last_runtime_status` — Starting 排除

## 测试文件

- `taskCloudService/src/compute_start_vm_idle_reuse_test.go`（扩展）
- `taskCloudService/src/orphan_instance_reconcile_test.go`（扩展）

## 增量切片（单增）

1. **Inc-1**: A+C 代码 + 单测（本迭代全部交付）
