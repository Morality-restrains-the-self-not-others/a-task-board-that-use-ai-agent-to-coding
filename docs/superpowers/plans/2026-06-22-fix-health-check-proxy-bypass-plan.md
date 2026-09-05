# Implementation Plan: Health Check Proxy Bypass Fix

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-fix-health-check-proxy-bypass-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-fix-health-check-proxy-bypass-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-22-fix-health-check-proxy-bypass-nfr-clarification.md`
> - DDD 文档: `docs/superpowers/plans/2026-06-22-fix-health-check-proxy-bypass-ddd.md`

## 变更范围

`runAll/src/health.go` — 1 文件，~5 行改动

## 任务清单

### Task 1: Add proxy-free health HTTP client
- [ ] 在 `runAll/src/health.go` 中新增包级变量 `healthHTTPClient`，使用 `http.Transport{Proxy: nil}`
- **文件**: `runAll/src/health.go`
- **命令**: 编辑文件

### Task 2: Update checkHealth to use proxy-free client
- [ ] 将 `checkHealth()` 中的 `http.DefaultClient.Do(req)` 替换为 `healthHTTPClient.Do(req)`
- **文件**: `runAll/src/health.go`
- **命令**: 编辑文件

### Task 3: Verify existing tests pass
- [ ] 运行 `cd runAll && go test ./... -race` 确认无回归
- **命令**: `cd runAll && go test ./... -race`

### Task 4: Build and verify binary compiles
- [ ] 运行 `cd runAll && ./build.sh` 确认编译通过
- **命令**: `cd runAll && ./build.sh`
