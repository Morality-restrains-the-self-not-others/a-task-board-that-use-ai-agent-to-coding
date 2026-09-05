# Value Stream: Health Check Proxy Bypass Fix

> Derived from design: `docs/superpowers/specs/2026-06-22-fix-health-check-proxy-bypass-design.md`
> **Type:** Infrastructure fix (orchestrator reliability) — no new business value stream.

## Value Summary

Developers and operators can run `./runAll` without health checks failing due to proxy environment variables (`ALL_PROXY`, `HTTP_PROXY`). All managed services start reliably regardless of the shell's proxy configuration.

## Related Value Streams

- **runAll-stability-first** (`2026-05-25-runall-stability-first-value-stream.md`): extension — this fix further hardens runAll's startup reliability
- **runAll-cascade-lifecycle** (`2026-05-27-runall-cascade-lifecycle-value-stream.md`): extension — proxy-bypassed health checks make cascade startup more reliable
- **runAll-port-based-stop-liveness** (`2026-06-03-runall-port-based-stop-liveness-value-stream.md`): extension — split liveness/readiness probes also benefit from proxy bypass

## End-to-End Flow

```
[runAll starts a service] → [health probe begins] → [HTTP GET to local service URL] → [service responds 2xx/3xx] → [service marked healthy]
                                                                     ↑
                                                           [proxy bypassed]
```

## Value Increments

### Increment 1: Proxy-Free Health HTTP Client (Only Increment)
**Value to user:** runAll health checks succeed even when `ALL_PROXY` / `HTTP_PROXY` / `HTTPS_PROXY` env vars are set.
**Scope:** Replace `http.DefaultClient.Do(req)` with a custom `http.Client` with `Proxy: nil` in `runAll/src/health.go`.
**Depends on:** nothing.
**Test file:** `runAll/src/health_test.go` (existing) + verify no regression on manual runAll smoke.

## YAML Config

**No new value stream entries required.** This is an orchestrator bug fix. No fields, test files, or steps are added to `value-stream.yaml`. All existing value streams benefit automatically from reliable health checks.
