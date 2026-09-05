# Vue Frontend → Task Gateway HTTP (Dev Only)

**Date:** 2026-06-02
**Status:** Approved
**Type:** Configuration change

## Intent

Change the primary SaaS Vue frontend (`task2app/front_project/app/`) to communicate with
the APISIX task-gateway over **plain HTTP** instead of HTTPS during **local development**.
Production deployments continue to use HTTPS.

## Motivation

- Eliminate self-signed certificate friction in local dev (browser warnings, cert regeneration)
- APISIX already exposes HTTP on port `:8080` alongside HTTPS on `:8443` — no gateway change needed
- Internal services already communicate over plain HTTP; TLS is only at the edge

## Current State

```
conf/vue/config.yaml
  apiBaseUrl: https://172.20.10.3:8443   ← HTTPS to gateway

↓ scripts/conf-read.py snapshot-json
↓ vite.config.js defines VITE_API_BASE_URL

Browser ──HTTPS:8443──→ APISIX ──HTTP──→ Django (:8001)
```

When `VITE_API_BASE_URL` is set (non-empty), Vite's dev proxy is **disabled** and the
browser sends cross-origin requests directly to the gateway (see
`vite.config.js:304-309`).

## Proposed Change

**Single file:** `conf/vue/config.yaml` line 6

| Before | After |
|--------|-------|
| `apiBaseUrl: https://172.20.10.3:8443` | `apiBaseUrl: http://172.20.10.3:8080` |

### Target State

```
conf/vue/config.yaml
  apiBaseUrl: http://172.20.10.3:8080    ← HTTP to gateway

Browser ──HTTP:8080──→ APISIX ──HTTP──→ Django (:8001)
```

### Cascade

1. `scripts/conf-read.py snapshot-json` reads new value
2. `vite.config.js:82-83` sets `VITE_API_BASE_URL = "http://172.20.10.3:8080"`
3. `front_project/app/src/utils/config.js` → `API_BASE_URL` = new HTTP value
4. `apiFetch()` builds URLs like `http://172.20.10.3:8080/api/...`
5. Browser makes plain HTTP requests → APISIX `:8080` → internal services

## Scope

| Item | Action |
|------|--------|
| `conf/vue/config.yaml` | Change `apiBaseUrl` to HTTP |
| `task2app/front_project/app/` | No code changes (reads config dynamically) |
| `task2app/Saas_Ai_Provider/frontend/` | No changes (already uses HTTP via Vite proxy) |
| `taskGateway/` | No changes (HTTP `:8080` already exposed) |
| Production config | No changes (scoped to local dev config) |

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| CORS mismatch on HTTP origin | Low | Medium | APISIX CORS already allows both `http://` and `https://` origins for `172.20.10.3:4000` |
| Port `:8080` conflict | Low | Low | Already mapped in docker-compose and confirmed working |
| Mixed content browser errors | None | — | All requests go to same scheme (HTTP), no mixed content |
| Cookie/CSRF issues | Low | Medium | Cross-origin HTTP cookies may need `SameSite=None; Secure=false` — verify in testing |

## Verification

1. Change `apiBaseUrl` in `conf/vue/config.yaml`
2. Restart Vite dev server → confirm `VITE_API_BASE_URL` is `http://172.20.10.3:8080`
3. Open browser at `http://172.20.10.3:4000`
4. Verify API calls succeed (check Network tab — requests go to `http://172.20.10.3:8080/api/...`)
5. Verify login, workspace listing, task operations all work
6. Verify NO browser certificate warnings

## Domain Concepts

N/A — configuration-only change with no new domain entities.

## Value Stream Impact

No existing value streams are affected. This is a development-environment configuration
change that does not alter any business flow, data field, or user-facing behavior.

## References

- [conf/vue/config.yaml](../../../conf/vue/config.yaml)
- [vite.config.js](../../../task2app/front_project/app/vite.config.js)
- [src/utils/config.js](../../../task2app/front_project/app/src/utils/config.js)
- [taskGateway docker-compose.yml](../../../taskGateway/docker-compose.yml)
