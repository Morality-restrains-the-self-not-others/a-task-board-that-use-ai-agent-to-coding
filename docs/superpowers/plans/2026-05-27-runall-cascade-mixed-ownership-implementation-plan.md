# Implementation Plan: runAll 链式关闭混合所有权

**Goal:** 修复 git-oauth 链式关闭时 taskFE takeover 错误。

## Tasks

- [x] Task 1: `CascadeStepActorResolver` 领域服务 + 单测
- [x] Task 2: `Runner.resolveCascadeStepActor` + `executeLifecyclePlan` 接线
- [x] Task 3: `StopGroupWithActor` 同样委托 per-step actor
- [x] Task 4: `TestRunner_StopServiceCascade_DelegatesOwnershipPerStep` 集成测试
- [x] Task 5: value-stream.yaml + view_test 文档

## Verify

```bash
cd runAll && go test ./src/... -run 'CascadeStepActor|StopServiceCascade|StopService_RejectsNonOwner' -v
cd runAll && go test ./src/...
```
