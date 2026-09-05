# DDD 领域模型: 修复编译状态限制

> 输入:
> - 设计文档: `docs/design/fix-build-status-constraint.md`
> - 价值流: `docs/superpowers/plans/2026-06-21-fix-build-status-constraint-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-21-fix-build-status-constraint-nfr-clarification.md`

## 判定: 无新增领域概念

此 fix 不引入新实体、值对象、聚合、仓储接口或领域事件。变更范围是现有 `BuildService` 函数的状态门逻辑——从白名单（3 种状态）扩展为排除法（排除 `building`）。

## 现有领域模型（不变）

所有领域概念已存在于 `runAll/src/domain/`:

| 文件 | 概念 | 变更? |
|------|------|------|
| `managed_service_entity.go` | `ManagedService` 实体 + 状态常量 | 不变 |
| `build_group_result_value_object.go` | `BuildGroupResult` VO + `IsTerminalBuildStatus` | **函数体变更**（白名单→排除法） |
| `service_cascade_orchestration.go` | 级联编排 | 不变 |
| `service_stop_policy.go` | 停止策略 | 不变 |

## `IsTerminalBuildStatus` 语义迁移

```
旧: switch status { case healthy, failed, stopped → true; default → false }
新: return status != ServiceStatusBuilding
```

影响范围:
- `runner.go:BuildService` — CAS 状态门使用相同语义（显式列出 8 种允许状态）
- `runner.go:BuildGroup` — 通过 `IsTerminalBuildStatus` 过滤

## 仓储接口 & 领域事件

无变更。
