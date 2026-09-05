# 实施计划: 修复编译(build)受运行状态限制

> 输入:
> - 设计: `docs/design/fix-build-status-constraint.md`
> - 价值流: `docs/superpowers/plans/2026-06-21-fix-build-status-constraint-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-21-fix-build-status-constraint-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-21-fix-build-status-constraint-ddd.md`

## 任务清单

### Task 1: 更新 `IsTerminalBuildStatus` 领域函数

- [ ] **文件**: `runAll/src/domain/build_group_result_value_object.go:79-86`
- [ ] **变更**: 将白名单 `switch { case healthy, failed, stopped → true; default → false }` 改为排除法 `return status != ServiceStatusBuilding`
- [ ] **验证**: `cd runAll && go test ./src/domain/ -run TestIsTerminalBuildStatus -v`

### Task 2: 更新 `IsTerminalBuildStatus` 测试用例

- [ ] **文件**: `runAll/src/domain/build_group_result_value_object_test.go:109-130`
- [ ] **变更**: `pending/starting/retrying/restarting/skipped` 的 expected 从 `false` 改为 `true`；`building` 保持 `false`
- [ ] **验证**: `cd runAll && go test ./src/domain/ -run TestIsTerminalBuildStatus -v`

### Task 3: 扩展 `BuildService` CAS 状态门

- [ ] **文件**: `runAll/src/runner.go:1604-1616`
- [ ] **变更**: 将 3 个 `CompareAndSwapStatus` 分支扩展为 8 个（所有非 `StatusBuilding` 状态），更新错误消息为 `"service %q is already building"`（当 status=building 时）和 `"service %q is %s, cannot build right now"`（其他意外情况）
- [ ] **验证**: `cd runAll && go test ./src/ -run TestBuildService -v`

### Task 4: 更新 `BuildService` 测试用例

- [ ] **文件**: `runAll/src/runner_test.go:360-379` (`TestBuildService_StatusConflict`)
- [ ] **变更**: 将测试状态从 `StatusRestarting`（现在允许编译）改为 `StatusBuilding`（唯一拒绝的状态），验证错误消息包含 `"already building"`
- [ ] **文件**: `runAll/src/runner_test.go:410-416` — 无需变更（并发 build 测试仍正确）
- [ ] **验证**: `cd runAll && go test ./src/ -run "TestBuildService_StatusConflict|TestBuildService_ConcurrentBuildRejected" -v`

### Task 5: 运行全量测试

- [ ] **命令**: `cd runAll && go test ./... -count=1`
- [ ] **预期**: 全部测试通过
- [ ] **特别注意**: `TestBuildGroup_*` 系列测试的行为变化——此前会被 skip 的服务现在会通过 `IsTerminalBuildStatus` 检查

### Task 6: 构建验证

- [ ] **命令**: `cd runAll && ./build.sh`
- [ ] **预期**: 编译成功，`bin/runAll` 生成

## 执行顺序

```
Task 1 → Task 2 → Task 3 → Task 4 → Task 5 → Task 6
  ↓        ↓        ↓        ↓        ↓        ↓
 领域层    领域测试  应用层    应用测试   回归     构建
```

Task 1-2 (领域层) 必须先完成，因为 Task 3 (runner.go) 的 `BuildGroup` 通过 `IsTerminalBuildStatus` 间接依赖 Task 1 的变更。

## 风险点

| 风险 | 缓解 |
|------|------|
| `TestBuildGroup_*` 系列测试对 skip 行为有断言 | Task 5 全量测试会发现，需要检查所有 `BuildGroup` 相关测试 |
| 生产环境已有服务处于非 terminal 状态 | 向后兼容——只是让更多状态允许编译，不影响现有行为 |
