# runAll — Multi-Service Orchestrator Design

**Date:** 2026-05-19
**Language:** Go (single binary)
**Scope:** `runAll/` directory in mono-repo root

## Overview

A Go CLI tool that reads a YAML config describing multiple services (start command, health check URL, dependencies), starts them in the correct order based on a DAG, checks their health, monitors their status, and provides a Web UI dashboard.

## YAML Configuration

File: `config.yaml` (default), overridable via `--config`.

```yaml
version: "1"
groups:
  - name: infrastructure
    services:
      - name: redis
        command: "redis-server --port 6379"
        health_check:
          url: "http://localhost:6379"
          timeout: 30
          retries: 10
          backoff:
            initial: 1.0
            max: 8.0
            multiplier: 2.0
        on_failure: exit

      - name: kafka
        command: "docker start kafka"
        health_check:
          url: "http://localhost:9092"

  - name: apps
    services:
      - name: saas-backend
        command: "python manage.py runserver 0.0.0.0:8000"
        health_check:
          url: "http://localhost:8000/api/health"
        depends_on: [redis, kafka]
        working_dir: ./task2app/Saas_project

      - name: go-relay
        command: "./go_relayToTrae"
        health_check:
          url: "http://localhost:9090/healthz"
        depends_on: [kafka]

      - name: frontend
        command: "npm run dev"
        health_check:
          url: "http://localhost:5173"
        depends_on: [saas-backend]
        on_failure: skip
```

### Field Reference

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | yes | — | Unique service name across all groups |
| `command` | yes | — | Shell command to start the service |
| `health_check.url` | yes | — | HTTP GET endpoint for health check |
| `health_check.timeout` | no | 30 | Total timeout in seconds |
| `health_check.retries` | no | 10 | Max retry attempts |
| `health_check.backoff.initial` | no | 1.0 | First retry interval (seconds) |
| `health_check.backoff.max` | no | 8.0 | Maximum interval cap (seconds) |
| `health_check.backoff.multiplier` | no | 2.0 | Exponential multiplier |
| `depends_on` | no | [] | Service names this service depends on |
| `on_failure` | no | "exit" | `exit` (stop all) or `skip` (continue without) |
| `working_dir` | no | current dir | Working directory, relative to YAML file |
| `env` | no | {} | Extra environment variables |

`groups` is for logical organization only; execution order is determined solely by `depends_on`.

### Validation Rules

- All service names must be unique across groups
- `depends_on` must reference existing service names
- DAG must have no circular dependencies
- `on_failure` must be `"exit"` or `"skip"`
- `command` and `health_check.url` are required

## Architecture

### File Structure

```
runAll/
├── main.go          # Entry: flag parsing, signal handling, HTTP server start
├── config.go        # YAML structs, parsing, validation, defaults
├── dag.go           # DAG construction, Kahn topological sort, cycle detection
├── runner.go        # Start orchestration, health scheduling, status store, graceful shutdown
├── health.go        # HTTP health check with exponential backoff
├── ui.go            # HTTP handlers: GET /api/status (JSON), GET / (embedded HTML)
├── status.html      # Embedded Web UI dashboard
├── config.yaml      # Example configuration
└── go.mod
```

### DAG Execution Algorithm

1. Flatten all services across groups into a single list
2. Build adjacency list from `depends_on`, compute in-degree per node
3. Kahn's algorithm: nodes with in-degree 0 form level 0; processing each level decrements downstream in-degrees, yielding subsequent levels
4. Detect cycles: if not all nodes are emitted after Kahn, report `cycle: a → b → c → a`
5. Execute level by level: all services in a level start in parallel goroutines, then health-checked before advancing

### Service Status State Machine

```
pending ──► starting ──► retrying ──► healthy
                       │
                       ▼
                 failed ──► (on_failure=exit → shutdown all, exit(1))
                       │
                       ▼
                 (on_failure=skip → skipped)
```

If an upstream service is `skipped` or `failed`, all downstream dependents become `skipped` automatically.

### Process Lifecycle

- **Start:** `os/exec.Command("sh", "-c", command)` — supports shell syntax
- **Stdout/stderr:** line-buffered, prefixed with `[service-name]`, printed to terminal
- **Shutdown:** reverse topological order, SIGTERM, wait 5s, then SIGKILL
- **Daemon mode:** exit after all health checks pass; daemon mode does NOT start the Web UI
- **Foreground mode:** block on signal (SIGINT/SIGTERM), then shutdown; Web UI runs during foreground

### Health Check

```go
func waitHealthy(ctx context.Context, url string, cfg HealthCheckConfig) error
```

- Respects `ctx` cancellation for graceful shutdown during retry
- Interval progression: 1s → 2s → 4s → 8s (capped at `max`)
- Success: HTTP 2xx or 3xx response
- Failure: any error (connection refused, timeout, non-2xx/3xx)

### Status Store (Thread-Safe)

```go
type StatusStore struct {
    mu       sync.RWMutex
    services map[string]*ServiceStatus
}

type ServiceStatus struct {
    Name      string      `json:"name"`
    Status    Status      `json:"status"`
    DependsOn []DepStatus `json:"depends_on"`
    Command   string      `json:"command"`
    URL       string      `json:"url"`
    PID       int         `json:"pid"`
    StartedAt string      `json:"started_at"`
    Error     string      `json:"error,omitempty"`
}

type DepStatus struct {
    Name   string `json:"name"`
    Status Status `json:"status"`
}
```

### Web UI

- Embedded via `embed.FS`, compiled into the binary
- `GET /` — single HTML page, pure HTML + CSS + `fetch()`, zero external dependencies
- `GET /api/status` — JSON array of `ServiceStatus`
- Auto-refresh every 2 seconds via `setInterval` + `fetch`
- Services displayed in DAG level order
- Each service row: colored dot + name + status text + dependencies (each with its own colored dot)
- Dot colors: green (healthy), yellow (starting/retrying), gray (pending), red (failed), dark gray (skipped)

### CLI Interface

```bash
runAll --config ./config.yaml                  # foreground with Web UI
runAll --config ./config.yaml --daemon         # start and exit, no UI
runAll --config ./config.yaml --ui-port 8888   # custom UI port (default :9999)
```

## Error Handling Summary

| Phase | Error | Behavior |
|-------|-------|----------|
| YAML parse | Invalid syntax, missing fields | Print error, exit(1) |
| Validation | Duplicate names, bad refs, cycles, bad on_failure | Print error, exit(1) |
| Service start | Command not found, workdir invalid | Per-service on_failure policy |
| Health check | Timeout, non-2xx/3xx, connection refused | Per-service on_failure policy |
| Runtime signal | SIGINT, SIGTERM | Graceful reverse-order shutdown |
| Runtime crash | Subprocess unexpected exit | Log warning, mark failed |

## Edge Cases

- **Circular dependency:** detected during DAG build, printed as `cycle: a → b → c → a`, exit(1)
- **Upstream skipped/failed:** all downstream dependents auto-skipped
- **Daemon mode + health failure:** health checks complete before exit; failures still trigger on_failure
- **Process dies after healthy:** foreground mode polls every 5s, logs dead processes and marks them failed in status
- **Relative working_dir:** resolved relative to the YAML file's directory, not the current working directory
