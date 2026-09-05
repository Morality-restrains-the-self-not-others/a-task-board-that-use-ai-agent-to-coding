# Value Stream: Vue Frontend → Gateway HTTP (Dev Only)

> Derived from design: `docs/superpowers/specs/2026-06-02-taskFE-http-gateway-design.md`

## Value Summary

Developers working locally no longer deal with self-signed certificate warnings when the
Vue frontend communicates with the APISIX gateway — plain HTTP on port 8080 instead of
HTTPS on 8443.

## Related Value Streams

| Stream | Relationship |
|--------|-------------|
| `taskgateway-apisix` | Extension — uses the gateway's existing HTTP port (:8080) that was already configured but unused by the frontend in dev mode |

## End-to-End Flow

```
[Developer opens http://172.20.10.3:4000]
  → [apiFetch → http://172.20.10.3:8080/api/...]
  → [APISIX routes to upstream over HTTP (unchanged)]
  → [User completes login / operations — no cert warnings]
```

## Value Increments

### Increment 1: Config Change (Thin Slice — Single Line)

**Value to developer:** No HTTPS cert warnings in local dev
**Scope:** `conf/vue/config.yaml` line 6: `apiBaseUrl` → HTTP
**Depends on:** nothing
**Test:** Manual — open browser, verify API calls succeed via HTTP, no cert errors

**Type:** Configuration-only change. No code, no new entities, no new tests.
**Production impact:** None (production config unchanged).
