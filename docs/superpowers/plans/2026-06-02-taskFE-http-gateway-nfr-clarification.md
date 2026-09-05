# NFR Clarification: Vue Frontend → Gateway HTTP (Dev Only)

> Input:
> - Design: `docs/superpowers/specs/2026-06-02-taskFE-http-gateway-design.md`
> - Value Stream: `docs/superpowers/plans/2026-06-02-taskFE-http-gateway-value-stream.md`

## NFR Skip Declaration

**All NFR categories skipped.** This change is a single-line configuration value change
(`apiBaseUrl` from HTTPS to HTTP in `conf/vue/config.yaml`). Per the NFR skill's skip
conditions:

| Condition | Status |
|-----------|--------|
| Configuration change only (no code, no new data flow) | ✅ Matches |
| No new external dependencies | ✅ Matches |
| Design doc declares no special NFR requirements | ✅ Matches |

**Justification:** Changing the protocol scheme from `https://` to `http://` for local
development does not introduce new data flows, consistency concerns, performance
characteristics, or security considerations beyond what already exists (the gateway
already serves HTTP on port 8080, and all internal services already communicate over
plain HTTP).

**Production:** Unaffected — production config remains HTTPS.
