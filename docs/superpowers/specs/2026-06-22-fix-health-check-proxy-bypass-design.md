# Design: Health Check Proxy Bypass Fix

**Date:** 2026-06-22
**Status:** approved
**Scope:** `runAll/src/health.go` — 1 file, ~5 lines

## Problem

runAll health checks fail when `ALL_PROXY` / `HTTP_PROXY` environment variables are set:

```
[runAll] error: health check timed out after 60s (last error:
  Get "http://0.0.0.0:8003/api/health/":
  socks connect tcp 127.0.0.1:1234->0.0.0.0:8003: EOF)
```

All runAll health check targets are local/infrastructure services — they should never go through an external proxy.

## Root Cause (3 layers)

| Layer | Issue |
|-------|-------|
| **Code** | `health.go:72` uses `http.DefaultClient.Do(req)` — Go's `DefaultTransport` calls `ProxyFromEnvironment`, which reads `ALL_PROXY`/`HTTP_PROXY`/`HTTPS_PROXY` env vars |
| **Config** | `conf/auth/task-auth/config.yaml` has `host: 0.0.0.0` (bind address). runAll resolves `conf_app` + `health_path` → `http://0.0.0.0:8003/api/health/`. Even without proxy, `0.0.0.0` is not a valid connect target for remote health checks |
| **Env** | The process inherits `ALL_PROXY=socks5h://127.0.0.1:1234` (set manually or via `scripts/proxy-setup.sh`). `NO_PROXY` does not include `0.0.0.0` |

## Fix

### Change: `runAll/src/health.go`

Replace `http.DefaultClient.Do(req)` with a custom `http.Client` that explicitly disables proxy (`Proxy: nil`):

```go
// healthHTTPClient is used only for health checks — never proxy.
var healthHTTPClient = &http.Client{
    Transport: &http.Transport{
        Proxy: nil,
    },
    Timeout: 5 * time.Second,
}

func checkHealth(ctx context.Context, url string) error {
    // ... (unchanged request creation) ...
    resp, err := healthHTTPClient.Do(req)  // was: http.DefaultClient.Do(req)
    // ... (unchanged response handling) ...
}
```

### Why this approach

- **Minimal**: 1 file, 1 new variable, 1 line change
- **Safe**: Does not modify global `http.DefaultTransport` — other code that legitimately needs proxy (e.g. `django_client.go` for external API calls) is unaffected
- **Correct**: ALL runAll health check targets are internal services (localhost or LAN); none should ever use a proxy
- **Future-proof**: New services added to `runAll.yaml` automatically get proxy-free health checks

## Domain Concepts

- **Bounded Context**: Infrastructure orchestration (runAll)
- **Key Entity**: `HealthCheck` (already exists in config model)
- **Domain Event**: Service readiness probe (HTTP GET → healthy/unhealthy)

## Value Stream Impact

- Affects the **runAll orchestration** itself, not a business value stream
- No changes to `value-stream.yaml` needed
- All services with `health_path` or `url` health checks benefit from the fix
