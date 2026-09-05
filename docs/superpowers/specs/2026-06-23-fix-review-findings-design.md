# Fix Code Review Findings — Design Document

**Date:** 2026-06-23
**Status:** Approved
**Scope:** runAll status page — 3 bug fixes from code review

## Findings to Fix

### F1: `handleBuildAllAction` Synchronous Request Context
- **File:** `src/ui.go:569-585`
- **Root Cause:** Handler uses `runner.BuildAll(r.Context())` directly — blocks HTTP handler and propagates HTTP request cancellation to build subprocesses
- **Fix:** Wrap in `runLifecycleActionAsync` (same pattern as `handleStopAllAction`), return `202 Accepted` immediately
- **Trade-off:** Client won't get build result in HTTP response — same behavior as stop-all/start-all; frontend already handles this via polling

### F2: `PlanStopAll` Error When All Services Already Stopped
- **File:** `src/domain/service_cascade_orchestration_service.go:142-167`
- **Root Cause:** `filterStoppableAll` returns empty slice → `NewServiceLifecyclePlan` validation rejects empty plan
- **Fix:** Check `len(filtered) == 0` after filtering, return empty plan instead of calling `NewServiceLifecyclePlan`

### F3: `collapseAllGroups` Button Text Desync on Empty Data
- **File:** `src/status.html:539-553`
- **Root Cause:** `[].every(cb) === true` — empty `lastStatusData` causes `allCollapsed = true`, toggles groups to expanded but button shows "折叠全部"
- **Fix:** Guard `collapseAllGroups` with empty-data early return

## Domain Concept Inventory

No new domain concepts — these are pure bug fixes in existing infrastructure.

## Value Stream Impact

No value stream impact — fixes don't change any value flow, data fields, or test coverage.

## Implementation Checklist

- [ ] F1: Make `handleBuildAllAction` async (ui.go)
- [ ] F2: Guard empty filtered list in `PlanStopAll` (service_cascade_orchestration_service.go)
- [ ] F3: Guard empty data in `collapseAllGroups` (status.html)
- [ ] Build & verify
