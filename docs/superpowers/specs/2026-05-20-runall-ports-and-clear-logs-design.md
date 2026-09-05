# runAll: Show Service Ports and Add Clear-Logs Action

Date: 2026-05-20
Status: approved

## Summary

Update `runAll` Web UI (`http://localhost:9999/`) with two capabilities:

- Show each service's listening port from **two sources**:
  - `health_check.url` parsed port (`health_port`)
  - `command` best-effort parsed port (`command_port`)
- Add a per-service **clear logs** action that only clears runAll in-memory log buffer (not external files).

## Confirmed Decisions

- Port display rule: **C** (show both health-derived and command-derived ports).
- Clear logs scope: **A** (clear only UI-backed in-memory logs).
- Keep current architecture (`status.html` + `ui.go` + `runner.go` + in-memory log repository), avoid new API surface unless needed.

## Goals

- Improve observability by exposing port information directly in service rows.
- Reduce operator friction by allowing one-click log reset per service.
- Keep changes minimal, testable, and backward compatible with existing status/restart/build/log flows.

## Non-Goals

- No attempt to clear external log files or process-managed file outputs.
- No shell-based process-port probing (`lsof`, `netstat`) in this iteration.
- No persistent audit/event storage for clear-log operations.

## Current Baseline

- `/api/status` already returns service status and metadata (`command`, `url`, `pid`, etc.).
- `status.html` already renders per-service actions (`编译`, `重启`, `日志`).
- `/api/logs` reads from `ServiceLogRepository.Tail(...)`.
- In-memory repository currently supports append + tail, but no clear operation.

## Architecture

### UI Layer (`status.html`)

- Add dual-port display in each service row, format suggestion:
  - `Ports: health <x> / command <y>`
- Add a new per-service button:
  - `清空日志` -> `POST /api/logs/clear`
- In logs modal, keep current behavior; after clear success, immediately refresh current modal content.

### API Layer (`ui.go`)

Extend handlers with:

- `POST /api/logs/clear` with JSON body:
  - `{ "name": "<service>" }`

Response shape:

- Success: `{ "status": "ok" }`
- Failure: `{ "error": "..." }`

Validation/error behavior:

- Method not `POST` -> `405` JSON error.
- Missing/blank `name` -> `400`.
- Unknown service -> `400`.
- Missing repository -> `400`.

### Application Layer (`runner.go` / store wiring)

- During service metadata initialization, derive and store:
  - `health_port` from `health_check.url`
  - `command_port` from `command` (best-effort parser)
- Keep parser failures non-fatal; unresolved values remain empty.

### Log Repository Layer

- Extend `domain.ServiceLogRepository` with:
  - `Clear(service string)`
- Implement clear in in-memory repository by removing/resetting buffer for that service.

## Data Model Changes

Update `ServiceStatus` payload:

- `health_port` (string or empty)
- `command_port` (string or empty)

No behavior change for existing fields.

## Port Parsing Rules

### `health_port` (deterministic)

- Parse `health_check.url`.
- If explicit numeric port exists, use it.
- If no explicit port:
  - `http` -> `80`
  - `https` -> `443`
- Invalid URL -> empty value.

### `command_port` (best effort)

Extract first matching pattern from command string (priority order):

1. `--port <n>` or `--port=<n>`
2. `-p <n>`
3. `PORT=<n>` (environment-style prefix)
4. host/addr fragments ending in `:<n>` (safe pattern only)

If parsing fails or ambiguous, return empty value. Never block service startup for this.

## Data Flow

### Status Refresh

1. Backend initializes status metadata, including two port fields.
2. Frontend polls `/api/status` every 2s as before.
3. UI renders dual ports for each row (`-` when absent).

### Clear Logs

1. User clicks `清空日志` for a service row.
2. Frontend posts `{name}` to `/api/logs/clear`.
3. Backend validates and calls `logRepository.Clear(name)`.
4. Success response returns `status=ok`.
5. Frontend triggers status refresh and, if logs modal for same service is open, refreshes logs view.

## Error Handling

- Clear operation is idempotent: clearing an already-empty service buffer still returns success.
- Clear does not affect process state (`status`, `pid`, health monitoring untouched).
- If clear API fails, show `alert("Clear logs failed: ...")` (align current UI action style).

## Testing Strategy

### Backend

- `ui_test.go`
  - `POST /api/logs/clear` success.
  - validation failures (method, missing name, unknown service).
- repository tests
  - clear removes target service logs only.
  - clear is idempotent.
  - clear does not affect other services.
- parser tests
  - health URL: explicit/default/invalid cases.
  - command parser: supported patterns + no-match path.
- status API serialization
  - includes `health_port` and `command_port`.

### Frontend

- home page snippets include:
  - clear-logs action button hook
  - port display text rendering branch
  - clear endpoint call path (`/api/logs/clear`)
- modal refresh guard remains correct after clear action.

## Domain Model

### Bounded Context

- `ServiceRuntimeObservabilityContext`: service runtime state, logs, and operator-facing metadata.

### Aggregate / Aggregate Root

- `ServiceRuntimeAggregate` (root can be represented by existing `ServiceStatus` runtime view).

### Value Objects

- `PortSnapshot`
  - `healthPort`
  - `commandPort`
- `LogClearRequest`
  - `serviceName`

### Domain Service

- `PortResolverService`
  - resolve ports from URL/command using deterministic + best-effort strategy.

### Repository Interface

- `ServiceLogRepository`
  - `Append`
  - `Tail`
  - `Clear`

### Domain Events (conceptual, optional for future)

- `ServiceLogsCleared`

Not required to publish externally in this iteration.

## Implementation Scope

- `runAll/src/status.go`
- `runAll/src/runner.go`
- `runAll/src/ui.go`
- `runAll/src/status.html`
- `runAll/src/domain/service_log_repository.go`
- `runAll/src/infrastructure/inmemory_service_log_repository.go`
- tests under `runAll/src/*_test.go`

## Acceptance Criteria

- Each service row shows `health_port` and `command_port` (or `-` when unresolved).
- Each service row has a `清空日志` action.
- `POST /api/logs/clear` clears only in-memory logs for target service.
- Existing build/restart/log-view behavior remains backward compatible.
- All new tests pass, and there is no regression in existing tests.
