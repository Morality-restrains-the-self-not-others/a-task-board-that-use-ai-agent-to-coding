# runAll: Build Before Restart

Date: 2026-05-19
Status: approved

## Summary

Add an optional `build_command` field to the service config. On restart (only), execute the build command before stopping and restarting the service process. If the build fails, the running process is left untouched and an error is returned.

## Motivation

Several services (go-run-container, go-relay, ai-provider) currently embed their build step inside the `command` field using `&&`:

```yaml
command: "go build -o go_run_container . && ./go_run_container"
```

This works for initial startup but means every restart also recompiles. By splitting build from run, the build step only runs on explicit restart, making the restart intent clear and avoiding unnecessary rebuilds on the initial `Run` pass.

## Config Change

### Service struct (config.go)

```go
type Service struct {
    // ... existing fields unchanged ...
    BuildCommand string `yaml:"build_command"` // optional, only on restart
}
```

- Optional. When empty, restart behavior is unchanged from current.
- No validation required (it's optional by design).

### Example config.yaml usage

Before:
```yaml
- name: go-run-container
  command: "go build -o go_run_container . && ./go_run_container"
  working_dir: go_run_container
```

After:
```yaml
- name: go-run-container
  build_command: "go build -o go_run_container ."
  command: "./go_run_container"
  working_dir: go_run_container
```

## Restart Flow Change

### New RestartService order

```
1. Validate: service exists, status is healthy or failed
2. Set status → StatusRestarting
3. [NEW] If BuildCommand != "":
   a. Set status → StatusBuilding
   b. Execute build_command (reuse WorkingDir, Env from service config)
   c. If build fails → set status to StatusFailed, keep old process running, return error
4. Stop old process (stopProcess)
5. Start new process (startAndCheck, existing logic)
```

The build runs **before** stopping the old process. This is the guarantee that a failed build leaves the service untouched.

### New method: runBuild

```go
func (r *Runner) runBuild(ctx context.Context, svc *Service) error
```

- Creates `exec.CommandContext(ctx, "sh", "-c", svc.BuildCommand)`
- Sets `Dir` to `svc.WorkingDir` and `Env` same as `startAndCheck`
- Streams stdout/stderr via `streamOutput`
- Returns the command's exit error (or nil on success)

## Status Change

### New status constant (status.go)

```go
const StatusBuilding = "building"
```

### Status transitions during restart

```
healthy → restarting → building → stopping → starting → retrying → healthy  (success)
                                → failed                                     (build failure)
```

On build failure, the status lands on `failed` but the old process is **still running**. The user can fix the build issue and restart again.

## Edge Cases

| Scenario | Behavior |
|---|---|
| `build_command` is empty | Skip build, behavior identical to current |
| Build times out / hangs | Killed by context (same as any shell command) |
| Double restart (restart while building) | Rejected — status is `restarting`/`building`, not `healthy`/`failed` |
| Build command working_dir | Reuses `svc.WorkingDir` |
| Build command env vars | Reuses `svc.Env` |
| Initial startup (`Run`) | `build_command` is NOT executed |

## Files Changed

| File | Change |
|---|---|
| `runAll/config.go` | Add `BuildCommand` field to `Service` |
| `runAll/runner.go` | Add `runBuild` method; reorder `RestartService` |
| `runAll/status.go` | Add `StatusBuilding` constant |
| `runAll/runner_test.go` | Tests for restart+build (success, failure, no-build-command) |
| `runAll/config_test.go` | Test `build_command` YAML parsing |
| `config.yaml` | Split `go-run-container` and `go-relay` build/run commands |

## Testing

- **Unit**: `build_command` parsed correctly from YAML
- **Unit**: `runBuild` succeeds with valid command, fails with invalid command
- **Integration**: Full restart with `build_command` — build success → old process replaced
- **Integration**: Full restart with `build_command` — build failure → old process still running, status reflects error
- **Regression**: Restart without `build_command` works as before
