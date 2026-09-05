# Implementation Plan: runAll failed 下游级联关闭

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** git-oauth 级联关闭时纳入 failed 下游，附 Playwright 回归。

**Architecture:** 在 domain 层新增 `isCascadeStopCandidateStatus`，`filterStoppable` 改用该函数；补 Go/Playwright 测试。

**Tech Stack:** Go (runAll), Playwright

---

### Task 1: Domain 规则

**Files:**
- Modify: `runAll/src/domain/service_stop_policy_service.go`
- Modify: `runAll/src/domain/service_cascade_orchestration_service.go`
- Test: `runAll/src/domain/service_cascade_orchestration_service_test.go`

- [x] 1.1 新增 `isCascadeStopCandidateStatus`
- [x] 1.2 `filterStoppable` 改用新函数
- [x] 1.3 添加 `PlanStopCascade_IncludesFailedDownstream` 测试

### Task 2: Runner 集成

**Files:**
- Test: `runAll/src/runner_test.go`

- [x] 2.1 添加 `StopServiceCascade_StopsFailedDownstreamBeforeUpstream`
- [x] 2.2 `go test ./runAll/src/...` 全绿

### Task 3: Playwright

**Files:**
- Modify: `runAll/src/playwright_server_test.go`
- Create: `runAll/playwright/tests/runall-stop-git-oauth.playwright.test.js`

- [x] 3.1 Fixture 含 platform 链 + mixed ownership
- [x] 3.2 Playwright 测试 git-oauth 关闭
- [x] 3.3 `npx playwright test runall-stop-git-oauth.playwright.test.js` 通过

### Task 4: 价值流文档

**Files:**
- Modify: `value-stream.yaml`
- Create: `view_test/runall-stop-cascade-failed-downstream.md`

- [x] 4.1 追加 value-stream step
- [x] 4.2 view_test 验收说明
