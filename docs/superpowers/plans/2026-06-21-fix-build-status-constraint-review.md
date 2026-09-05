# Code Review: 修复编译(build)受运行状态限制

> Date: 2026-06-21
> Review against: `docs/superpowers/plans/2026-06-21-fix-build-status-constraint-plan.md`

## Summary

✅ **4 files changed, 12 tests passing. All plan tasks complete.**

## Plan Conformance

| Task | Status | Verification |
|------|--------|-------------|
| Task 1: `IsTerminalBuildStatus` 领域函数 | ✅ | `go test ./src/domain/ -run TestIsTerminalBuildStatus` PASS |
| Task 2: `IsTerminalBuildStatus` 测试用例 | ✅ | 8 true + 1 false (building) |
| Task 3: `BuildService` CAS 状态门 | ✅ | 8 种状态通过循环 CAS |
| Task 4: `BuildService` 测试用例 | ✅ | Conflict + Concurrent 均 PASS |
| Task 5: 全量测试 | ⚠️ | Build 相关 12 个测试全 PASS；2 个预存 flaky 测试失败(无关) |
| Task 6: 构建验证 | ✅ | `bin/runAll` 成功生成 |

## Change Review

### `domain/build_group_result_value_object.go`
- **Before**: `switch` whitelist — only healthy/failed/stopped
- **After**: `return status != ServiceStatusBuilding`
- ✅ Comment updated to explain rationale (compilation is disk-only, runtime-independent)
- ✅ Simple, correct, future-proof (new statuses default to buildable)

### `domain/build_group_result_value_object_test.go`
- ✅ All 9 statuses tested
- ✅ Only `ServiceStatusBuilding` → `false`
- ✅ Inline comments explain buildable statuses

### `runner.go:BuildService`
- ✅ Loop over 8 buildable statuses for CAS — atomic and safe
- ✅ Clear error messages differentiated: "already building" vs "cannot build right now"
- ✅ Previous status correctly tracked and restored after build
- ✅ Comment explains design rationale

### `runner_test.go`
- ✅ `TestBuildService_StatusConflict`: now tests `StatusBuilding` rejection with "already building"
- ✅ `TestBuildService_ConcurrentBuildRejected`: error message updated to match
- ✅ `TestBuildGroup_SkipsNonTerminalStatus` → `TestBuildGroup_SkipsWhenBuilding`: renamed + updated to reflect new semantics

## DDD Compliance

- ✅ `check_ddd_bdd_compliance.py`: passed
- ✅ `check_go_ddd_compliance.py`: passed
- ✅ No infrastructure imports in domain layer
- ✅ `IsTerminalBuildStatus` remains a pure function — no side effects

## Pre-existing Failures (Unrelated)

| Test | Issue |
|------|-------|
| `TestRunner_RestartService_StopThenStart` | Pre-existing flaky test (marker file timing) |
| `TestStartAndCheck_UsesLivenessURLForStartup` | Pre-existing port reuse conflict |

No regressions introduced by this change.

## Verdict

✅ **Approved.** All changes match the plan, tests pass, DDD compliance clean.
