# Permission analysis — wechat-mp-egress-host-sh

| Endpoint | Actor | Auth | Notes |
|----------|-------|------|-------|
| POST `/api/auth/wechat/mp/follow-qr/` | logged-in user | token + Idempotency-Key | unchanged; no new public API |
| POST `/internal/wechat-mp/forward` (Host sh :8030) | taskAuth only | `X-Internal-Secret` | host allowlist `api.weixin.qq.com` only |
| GET `/healthz` | ops probe | none | no secrets |

IDOR: follow-qr still session user only. Egress secret in conf-local only.

python_api_approval: not_applicable
