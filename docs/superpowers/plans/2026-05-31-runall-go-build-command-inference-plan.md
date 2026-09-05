# Implementation Plan: runAll Go Build Command Inference

> Design: `docs/superpowers/specs/2026-05-31-runall-go-build-command-inference-design.md`  
> Value stream: `docs/superpowers/plans/2026-05-31-runall-go-build-command-inference-value-stream.md`

## Tasks

- [ ] **Task 1:** `extractGoBuildFromRunScript` 跳过含 `$` 的行 + 单元测试
- [ ] **Task 2:** `runBuild` / `BuildService` / `restartService` 使用 `resolveBuildCommand`
- [ ] **Task 3:** `TestBuildService_InferredFromRunScript` 集成测试（taskAuth working_dir）
- [ ] **Task 4:** `runAll.yaml` 为 task-events-* 添加 `build_command`
- [ ] **Task 5:** `runAll/config.yaml` 同步 container-stack 编排
- [ ] **Task 6:** `TestProductionConfig_TaskEventsHaveBuildCommand` 生产配置断言
- [ ] **Task 7:** `cd runAll && go test ./...` 全绿

## Verification

```bash
cd runAll && go test ./...
curl -s -X POST http://127.0.0.1:9999/api/build -H 'Content-Type: application/json' -d '{"name":"task-auth"}'
```
