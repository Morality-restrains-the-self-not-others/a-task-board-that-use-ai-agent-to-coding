# ADR-0059: 微信服务号 API 出站经 Host sh egress

- **Status:** accepted
- **Date:** 2026-09-04
- **Author:** cursor
- **Deciders:** goal-mode auto (referral MP QR 40164)

---

## Context

推荐关注闸门 `POST /api/auth/wechat/mp/follow-qr/` 由 **taskAuth** 调用 `api.weixin.qq.com` 创码。微信公众平台对服务号接口强制 **IP 白名单**。INFRA 机当前出口 `120.36.185.132` 为动态家宽地址，不适合长期入白；Host sh 公网 `1.117.67.121` 为云主机固定出口。用户询问是否将「相关进程」剥离到 sh：整迁 taskAuth 会牵动 forward-auth、OIDC、auth DB 与入站回调，成本过高。

## Decision

We will **keep taskAuth on INFRA** and deploy a dedicated **wechat-mp-egress** Go process on **Host sh** that:

1. Accepts only authenticated internal HTTP from taskAuth (`X-Internal-Secret`)
2. Forwards only to `https://api.weixin.qq.com` (host allowlist)
3. Uses SH egress IP `1.117.67.121` for WeChat API calls

taskAuth reads `wechat.mpEgress.baseUrl` from conf；when empty, keeps direct dial (dev/tests). Production conf-local points to `http://1.117.67.121:8030`.

Ops **must** add `1.117.67.121` to the WeChat MP IP whitelist (verified: SH currently also receives 40164 until whitelisted).

## Alternatives Considered

### Alternative 1: Move entire taskAuth to Host sh

- **Pros:** Single process; egress naturally SH IP
- **Cons:** Splits auth hot path from INFRA DB/gateway; high blast radius
- **Why rejected:** Disproportionate to an egress-only constraint

### Alternative 2: Only whitelist INFRA current IP

- **Pros:** Zero code
- **Cons:** Residential IP churn; recurring 40164
- **Why rejected:** Unstable; may remain as short-term ops patch only

### Alternative 3: Env HTTP_PROXY for taskAuth

- **Pros:** Tiny code
- **Cons:** Violates app-startup-no-env-proxy meta-rule; risk of proxying unrelated traffic
- **Why rejected:** Must use explicit conf-driven client, never env proxy

## Consequences

### Positive

- Stable WeChat egress IP aligned with existing SH edge footprint
- taskAuth ownership / ticket DB / callbacks unchanged
- Allowlist + secret limit abuse surface

### Negative / Trade-offs

- Extra process + deploy path on SH
- Hard dependency on WeChat console whitelist update
- Extra hop latency (~tens of ms) on QR issue path

### Mitigations

- Health endpoint on egress; runAll/ssh deploy script
- BLOCK_TODO_OPS for whitelist
- Feature: empty `baseUrl` → direct (local unit tests unaffected)

## References

- Design: `docs/superpowers/specs/2026-09-04-wechat-mp-egress-host-sh-design.md`
- Trace: `81612ca5-03e6-42a7-be52-b7b77420b2c5`
- Meta-rule: `.ai/01_project_constraints/23_app_startup_no_env_proxy.md`
