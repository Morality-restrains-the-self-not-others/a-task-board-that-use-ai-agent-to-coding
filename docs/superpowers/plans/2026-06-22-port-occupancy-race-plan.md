# Implementation Plan: Fix Port Occupancy Race

> Inputs: design, value-stream, NFR, DDD (all 2026-06-22-port-occupancy-race-*)

## Increment 1: Port-Release Wait Loop in startAndCheck

### Task 1.1: Add waitForPortFree helper
- **File**: `runAll/src/runner.go` (near `listenerPIDs`, ~line 2168)
- **Details**: Add function `waitForPortFree(port string, timeout time.Duration) error` that polls `listenerPIDs(port)` every 500ms until empty or timeout (30s). Returns nil when port is free, error on timeout.
- **Test**: `runAll/src/runner_test.go` — new `TestWaitForPortFree` test using a temporary listener

### Task 1.2: Integrate wait loop into startAndCheck
- **File**: `runAll/src/runner.go:490`
- **Details**: Replace the "skip start" path with:
  ```go
  if len(pids) > 0 {
      log.Printf("[%s] port %s occupied by PID=%v, waiting for release...", svc.Name, port, pids)
      if err := waitForPortFree(port, 30*time.Second); err != nil {
          // timeout — force cleanup then start
          log.Printf("[%s] port %s not released after 30s, forcing start", svc.Name, port)
      }
      // fall through to normal start path below
  }
  ```
- **Behavior change**: Previously skipped → now waits then starts

### Task 1.3: Update existing tests
- **File**: `runAll/src/runner_test.go`
- **Details**: Update `TestStartAndCheck_*` tests that mock `listenerPIDsFn` to verify wait-then-start behavior

## Increment 2: SO_REUSEADDR for taskEvents Consumers

### Task 2.1: Add SO_REUSEADDR to eventbin HTTP listener
- **File**: `taskEvents/consumer/runner.go:90`
- **Details**: Replace `http.ListenAndServe(addr, ...)` with:
  ```go
  lc := net.ListenConfig{Control: func(network, address string, c syscall.RawConn) error {
      var err error
      c.Control(func(fd uintptr) {
          err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
      })
      return err
  }}
  ln, err := lc.Listen(context.Background(), "tcp", addr)
  if err != nil { ... }
  http.Serve(ln, tracelog.Middleware(mux))
  ```
- **Affects**: All 17 taskEvents consumers (shared code path)

### Task 2.2: Verify and test
- **Command**: `cd taskEvents && go build ./...`
- **Test**: `cd taskEvents && go test ./consumer/... -count=1`
- **Manual**: Start a consumer, kill -9, immediately restart — should bind without EADDRINUSE

## Verification

- `cd runAll && go test ./src/... -count=1`
- `cd taskEvents && go test ./... -count=1`
- Platform restart: all 17 consumers healthy within 60s
