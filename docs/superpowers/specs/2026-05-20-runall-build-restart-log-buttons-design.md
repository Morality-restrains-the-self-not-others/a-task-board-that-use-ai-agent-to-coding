# runAll: Split Build/Restart Buttons and Add Startup Logs Viewer

Date: 2026-05-20
Status: approved

## Summary

Update the `runAll` Web UI (`http://localhost:9999/`) so each service row has three separate actions:

- `编译` (build only, no restart)
- `重启` (existing restart behavior)
- `日志` (open inline modal to view recent startup/build logs)

The implementation keeps current architecture, extends existing API handlers, and adds a lightweight in-memory per-service log buffer.

## Confirmed Decisions

- Buttons are **per service row**, not global.
- Build action is **build only** and must **not restart** the service.
- Log viewing is via **inline modal** in the same page.
- Preferred path is a minimal-intrusion extension of current `status.html + ui.go + runner.go`.

## Goals

- Improve operational clarity by separating build and restart intent.
- Let users inspect startup/build logs without leaving the dashboard.
- Keep the change small, testable, and consistent with existing `runAll` patterns.

## Non-Goals

- No SSE/WebSocket log streaming in this iteration.
- No persistent log storage to files or database.
- No full page navigation or dedicated log route for humans.

## Current Baseline

- UI currently has a restart button on healthy/failed services.
- API currently provides:
  - `GET /api/status`
  - `POST /api/restart`
- `Runner` already supports `runBuild(...)` and uses `streamOutput(...)` for process stdout/stderr.
- Logs are currently emitted to process logger only and not queryable by UI.

## Architecture

### UI Layer

- In `status.html`, render 3 per-row buttons:
  - `编译` -> `POST /api/build`
  - `重启` -> `POST /api/restart` (existing)
  - `日志` -> open modal and fetch `GET /api/logs`
- Add a modal component in plain JS/CSS:
  - title: service name
  - body: monospaced log lines
  - controls: close, refresh
  - optional polling every 2s while modal is open

### API Layer

Extend `registerUIHandlers(...)` in `ui.go` with:

- `POST /api/build` with body `{ "name": "<service>" }`
- `GET /api/logs?name=<service>&lines=<n>`

Response shape:

- Build success: `{ "status": "ok" }`
- Build error: `{ "error": "..." }`
- Logs success: `{ "name": "...", "lines": [ ... ] }`

### Application Layer

- Add `BuildService(ctx, name)` in `Runner`:
  - validate service exists
  - guard disallowed transient states (for example `starting/restarting/building`)
  - allowed trigger states: `healthy` and `failed`
  - set status to `building`
  - execute `runBuild(...)`
  - on success, restore to the previous stable status (`healthy` stays `healthy`, `failed` stays `failed`)
  - on failure, set `failed` with error
- Keep `RestartService(...)` semantics unchanged.

### Log Buffer Layer

- Introduce an in-memory per-service ring buffer (fixed max lines per service).
- `streamOutput(...)` appends structured log entries to that buffer in addition to `log.Printf`.
- `/api/logs` reads from this buffer via tail query.

## Domain Model

### Bounded Context

- `ServiceOperationsContext`: build/restart/log-read operations for a named service.

### Entity

- `ServiceProcess` (existing concept): `name`, `status`, `pid`, runtime metadata.

### Value Objects

- `OperationRequest`:
  - `serviceName`
  - `operationType` (`build` | `restart` | `logs`)
  - optional params (for example `lines`)
- `LogEntry`:
  - `timestamp`
  - `serviceName`
  - `stream` (`stdout` | `stderr`)
  - `message`

### Domain Service

- `ServiceOperationService` responsibility is implemented by `Runner` methods:
  - `BuildService`
  - `RestartService`
  - `TailLogs` (directly or via repository helper)

### Repository Interface

- `ServiceLogRepository` (in-memory impl this iteration):
  - `Append(service, entry)`
  - `Tail(service, lines)`

### Domain Events (internal only for now)

- `ServiceBuildStarted`
- `ServiceBuildFinished`
- `ServiceLogsRead`

These events are conceptual in this iteration and do not require external event bus publishing.

## Data Flow

### Build Button

1. User clicks `编译` in a service row.
2. Frontend sends `POST /api/build`.
3. Handler validates payload and calls `runner.BuildService(...)`.
4. Runner executes build only (no process stop/restart).
5. Build output is appended to service log buffer.
6. Frontend status polling reflects state transitions.

### Restart Button

1. User clicks `重启`.
2. Existing `POST /api/restart` flow runs unchanged.
3. Existing health-check and monitor resume flow remains the source of truth.

### Logs Button

1. User clicks `日志`.
2. Modal opens and requests `GET /api/logs?name=<service>&lines=200`.
3. UI renders latest lines.
4. Optional periodic refresh while modal is visible.
5. Close modal stops refresh timer.

## Error Handling

- Invalid payload or missing `name` -> `400`.
- Unknown service -> `400` (align with current restart handler style).
- Operation-state conflict (already building/restarting/starting) -> `400` with explicit error message.
- Build execution failure -> `400` with `error`, service status set to `failed`.
- Logs for valid service but empty buffer -> return empty `lines` array.

## Testing Strategy

### Backend

- `Runner.BuildService`:
  - build success
  - build failure
  - service not found
  - disallowed state conflict
- Log repository:
  - append and tail
  - per-service isolation
  - ring buffer truncation
  - concurrent append safety
- UI handlers:
  - method checks
  - JSON/body validation
  - response schema on success/error

### Frontend (status page behavior)

- Three buttons render for each service row.
- Build action calls correct endpoint with correct payload.
- Logs modal open/close behavior and periodic refresh lifecycle.
- Error display on failed build/log fetch.

## Implementation Scope

- Primary files:
  - `runAll/src/status.html`
  - `runAll/src/ui.go`
  - `runAll/src/runner.go`
  - `runAll/src/status.go` (if extra status transitions need adjustment)
  - new in-memory log buffer source file in `runAll/src/` (small focused unit)
- Tests:
  - `runAll/src/ui_test.go`
  - `runAll/src/runner_test.go`
  - additional log-buffer test file

## Acceptance Criteria

- Each service row shows separate `编译` / `重启` / `日志` actions.
- `编译` does not restart service process.
- `重启` behavior stays backward compatible.
- `日志` opens inline modal showing recent startup/build logs for selected service.
- Existing status refresh still works and no regression in current restart path.

