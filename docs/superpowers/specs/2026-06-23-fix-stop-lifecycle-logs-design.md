# Fix: stopService Missing Lifecycle Logs

**Date:** 2026-06-23
**Status:** Approved
**Scope:** `runAll/src/runner.go` — `stopService` function

## Root Cause

`stopService()` does not call `appendLifecycleLog()` anywhere. The start path (`startService`) logs multiple events (start requested, preflight status, dependency waiting), but the stop path produces zero lifecycle log entries.

## Fix

Add `appendLifecycleLog` calls in `stopService` at these key points:

1. **Stop requested** — at the beginning, after status check passes
   ```go
   r.appendLifecycleLog(name, fmt.Sprintf("stop requested (current status=%s)", current.Status))
   ```

2. **Stop completed** — after `finalizeServiceStop` succeeds
   ```go
   r.appendLifecycleLog(name, "stopped successfully")
   ```

3. **Already stopped** — when the service is already in stopped state
   ```go
   r.appendLifecycleLog(name, "already stopped, skipping")
   ```

4. **Stop failed** — when stop fails
   ```go
   r.appendLifecycleLog(name, fmt.Sprintf("stop failed: %v", err))
   ```

## Files Changed

- `runAll/src/runner.go` — `stopService` function (~10 lines added)

## Verification

- Click "全部关闭" → open each service's log panel → verify "stop requested" and "stopped successfully" messages appear
- Stop an already-stopped service → verify "already stopped" appears
