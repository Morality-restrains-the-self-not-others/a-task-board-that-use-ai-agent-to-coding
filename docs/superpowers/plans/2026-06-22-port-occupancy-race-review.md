# Code Review: Fix Port Occupancy Race

> Reviewed: `runAll/src/runner.go`, `taskEvents/consumer/runner.go`

## Summary

✅ **PASS** — Ready to ship.

## Changes

### 1. runAll/src/runner.go — Port-release wait loop

**Before**: Port occupied → skip start, mark dependency failed, return (service permanently lost)
**After**: Port occupied → health check → if healthy (genuinely running) → skip; if unhealthy (dying) → `waitForPortFree(30s, 500ms poll)` → fall through to normal start

**New function**: `waitForPortFree(port, timeout)` — polls `listenerPIDs` until empty or timeout.

### 2. taskEvents/consumer/runner.go — SO_REUSEADDR

**Before**: `http.ListenAndServe(addr, handler)`
**After**: `net.ListenConfig{SO_REUSEADDR} → Listen → http.Serve(ln, handler)`

Affects all 17 taskEvents consumers via shared code path.

## Findings

### Correctness
- ✅ `waitForPortFree` uses `listenerPIDs` (same detection mechanism), 500ms poll, 30s timeout
- ✅ Health check gate prevents waiting for genuinely-running services (backward compatible)
- ✅ `SO_REUSEADDR` set before `Listen`, both errors handled
- ✅ `http.Serve` used instead of `http.ListenAndServe` (equivalent behavior)

### Code Quality
- ✅ `waitForPortFree` is a pure function, uses ticker for clean polling
- ✅ `net.ListenConfig.Control` pattern matches Go best practices for socket options
- ✅ No new dependencies

### Tests
- ✅ `TestStartAndCheck_*` suite passes (all 8 tests)
- ✅ Consumer builds and starts (verified with email_sent consumer)

### Edge Cases
- ✅ Port free immediately → `waitForPortFree` returns instantly
- ✅ Port stays occupied → timeout after 30s, force-start anyway
- ✅ Health check passes → skip start (existing behavior preserved)
- ✅ `listenerPIDs` error during wait → returned as error

## Verdict

| Dimension | Result |
|-----------|--------|
| Correctness | ✅ Pass |
| Code Quality | ✅ Pass |
| Tests | ✅ Pass |
| Backward Compat | ✅ Pass |
| Edge Cases | ✅ Pass |

**0 issues found.**
