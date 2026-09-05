# go_run_container — Go rewrite of mock_run_container

## Purpose

Rewrite the Python Flask `mock_run_container` service (port 8796) as a single Go binary using only the standard library. Drop-in replacement: same API contract, same config source, same client (Django `cloud/services/mock_run_container.py`).

## Scope

- Rewrite the **worker service** (`mock_run_container/server.py` + `bootstrap.py` + `start.sh`)
- The Django **client** (`cloud/services/mock_run_container.py`) does NOT change
- The `port_config.json` `mock_run_container` block does NOT change

## Project Structure

```
go_run_container/
├── main.go          # entry point, wires config, starts HTTP server
├── config.go        # port_config.json reader + env var overrides
├── server.go        # HTTP handlers, routing (net/http)
├── jobs.go          # in-memory job store with sync.Mutex
├── docker.go        # docker CLI via os/exec
├── secret.go        # X-Mock-Run-Container-Secret auth middleware
├── go.mod           # module go_run_container
├── start.sh         # go build && ./go_run_container
└── README.md
```

All files ≤ 500 lines.

## API Endpoints (identical to Python)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | no | health check |
| `POST` | `/v1/jobs` | secret | create container run job |
| `GET` | `/v1/jobs/{job_id}?cursor=N` | secret | poll logs with cursor pagination |
| `POST` | `/v1/jobs/{job_id}/stop` | secret | stop by job_id |
| `GET` | `/v1/tasks/{task_id}/container/status` | secret | container status by task_id |
| `POST` | `/v1/tasks/{task_id}/container/stop` | secret | stop by task_id |

### Request/Response Schemas

**POST /v1/jobs** — body:
- `image` (string, required): Docker image reference
- `env` (object, optional): environment variables for container
- `task_id` (string, optional): used to derive `container_name = taskId_{task_id}`
- `docker_platform` (string, optional): `linux/amd64` or `linux/arm64`

→ `202 {"job_id":"...", "container_name":"..."}`

**GET /v1/jobs/{job_id}?cursor=0** →
```json
{
  "logs": ["line1", "line2"],
  "next_cursor": 2,
  "done": false,
  "error": "",
  "result": {},
  "container_id": ""
}
```

**GET /v1/tasks/{task_id}/container/status** →
```json
{
  "running": true,
  "container_name": "taskId_xxx",
  "container_id": "abc123",
  "status": "running"
}
```

## Concurrency Model

- `sync.Mutex` + `map[string]*Job` (equivalent to Python `threading.Lock` + `_jobs` dict)
- Follow the concurrency rules from `.ai/03_technical_implementation/10_python_sidecar_web_concurrency.md`:
  - Lock only for in-memory state mutations (map reads/writes, log append)
  - No `os/exec` or blocking I/O while holding the lock
  - Background goroutine per job: pull → run → logs → finish
  - `_append`-style short-lock helper for log writes

## Job Lifecycle

```
POST /v1/jobs → create Job{logs:[], done:false}
  → go runJob(job):
      1. docker pull (stream stdout → append to job.Logs)
         - detect arm64 manifest mismatch → retry with linux/amd64
      2. docker run -d --rm --network host [--platform X] [--name Y] -e ... image
      3. docker logs -f (stream stdout → append to job.Logs)
      4. set job.Done=true, job.Result={container_id, container_name, image, docker_platform}
```

## Config Loading (priority order)

1. Read `../task2app/conf/port_config.json` → `mock_run_container` block (host, port, secret)
2. Override with env vars: `MOCK_RUN_CONTAINER_HOST`, `MOCK_RUN_CONTAINER_PORT`, `MOCK_RUN_CONTAINER_SECRET`
3. Defaults: host=`127.0.0.1`, port=`8796`, secret=empty (no auth)

## Authentication

- If `MOCK_RUN_CONTAINER_SECRET` is set, require `X-Mock-Run-Container-Secret` header match
- If secret is empty, allow all requests
- `/health` is always public

## Docker Operations

All via `os/exec` (same approach as Python `subprocess`):
- `docker pull --platform X image`
- `docker run -d --rm --network host --platform X --name Y -e K=V image`
- `docker inspect --type container name`
- `docker logs -f container_id`
- `docker stop container_ref`
- `docker rm -f container_name`

Platform detection:
- Image name contains `x86_64`/`amd64` → `linux/amd64`
- Image name contains `arm64`/`aarch64` → `linux/arm64`
- Auto-retry with `linux/amd64` on manifest mismatch

## Sensitive Data Handling

Same as Python: mask env values in logs for keys containing TOKEN, SECRET, PASSWORD, KEY, AUTH — but NOT ACCESS_TOKEN (needed for debugging token exchange issues).

## start.sh

```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"
go build -o go_run_container .
exec ./go_run_container
```

## Key Design Decisions

- **No Docker SDK** — shell out to `docker` CLI, same as Python, no CGO
- **No database** — job state is ephemeral (in-memory), same as Python
- **net/http** — Go 1.22+ method+path routing, zero external dependencies
- **Single binary** — no virtualenv, no pip install
- **No Go framework** — standard library covers all 6 endpoints cleanly

## Non-Goals

- Container log persistence beyond job lifetime
- Multi-worker / horizontal scaling (same as Python version)
- Graceful shutdown of running containers on server exit (same as Python)
- Swagger/OpenAPI docs

## Testing

- Unit tests: config loading, job CRUD, secret validation, container name generation, platform inference, sensitive key detection
- Integration tests: full pull→run→logs→stop against real Docker daemon
- Deadlock regression: health endpoint remains responsive during job execution
