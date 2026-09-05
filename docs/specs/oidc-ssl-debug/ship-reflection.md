# Ship & Reflection: OIDC SSL Protocol Fix

**Date**: 2026-06-24
**Branch**: `fix/oidc-ssl-protocol-fix` (in `conf` repo)
**Commit**: `fbbc709`

## What Was Shipped

### Root Cause
`openid_connect` Ruby gem v2.3.1 (GitLab 19.0) bug — discovery `Resource` class
discards URI scheme, `SWD.url_builder` defaults to `URI::HTTPS`, forcing HTTP
issuers to be accessed via HTTPS → SSL record layer failure on HTTP-only port 8003.

### Fix
- **Container**: GitLab Rails initializer `zzz_fix_oidc_http.rb` sets `SWD.url_builder = URI::HTTP`
- **Commit (conf repo)**: OIDC provider config + value stream YAML
- **Design artifacts**: 6 documents covering design, value stream, NFR, DDD skip, plan, ship

### Verification (All Passed)
- ✅ OIDC Discovery endpoint now uses `http://` (was `https://`)
- ✅ Full OIDC Discovery from GitLab container succeeds
- ✅ GitLab logs show no new SSL errors
- ✅ YAML syntax valid, valueStream accepts config

## Reflection

### What Went Well
- Diagnosis was systematic: SSL test → Ruby source trace → exact bug line identified
- Fix was minimal (3-line initializer) and immediately verifiable
- Full pipeline artifacts created even for a small fix

### Lessons Learned
- The `openid_connect` gem bug (scheme discarded in Resource constructor) affects
  any OIDC provider serving HTTP only — not just taskAuth
- Gateway TLS port (18444) needs separate investigation — APISIX standalone SSL
  config requires `ssls` entries in the routes YAML
- GitLab container rebuild destroys the initializer — Phase 2 persistence needed

### Follow-up
- [ ] Fix gateway APISIX TLS (port 18444) for long-term HTTPS solution
- [ ] Integrate initializer into `gitService/run.sh` for container rebuild resilience
- [ ] Monitor openid_connect gem releases for upstream fix
