# ADR-0020: 前端副作用按钮必须具备防重放设计

- **Status:** accepted
- **Date:** 2026-08-19
- **Author:** Trae AI
- **Deciders:** 工程团队

---

## Context

用户连点、框架把 `disabled` 推迟到下一渲染帧、以及 HTTP 超时后的客户端重试，会把同一用户意图变成两次写请求。仓库已有：

- 元规则 48：NFR 阶段审视服务端/事件幂等键；
- 元规则 49：Kafka 消费走共享幂等 runner；
- `.cursor/rules/frontend-button-interaction.mdc`：仅 glob 生效，只要求 debounce + loading，**没有**请求身份，也不是 alwaysApply。

若不把「点击入口」升为元规则，Agent 会继续只加 `:disabled="pending"`，建单/支付/启停机器仍会双发。debounce 单独存在时，超时重试仍会换新 UUID，服务端视为新业务。

## Decision

We will require **anti-replay design on every side-effecting frontend button click**:

1. **L1** — synchronous in-flight latch + visible pending (`disabled` / `aria-busy`) + short debounce for accidental taps.
2. **L2** — mutating HTTP carries `Idempotency-Key` minted when the click is *accepted*; retries of that intent reuse the same key.
3. **L3** — servers remain the source of truth (NFR 48 / consumer 49). UI locks do not replace DB uniqueness.

We will ship a shared helper `taskFE/app/src/utils/clickGuard.js` (`createClickGuard`, `mergeIdempotencyHeaders`) as the default for Vue/taskFE. Other frontends must provide an equivalent. Exceptions require an `Anti-Replay-OK:` comment.

## Alternatives Considered

### Alternative 1: Keep debounce-only glob rule

- **Pros:** 已有 `frontend-button-interaction.mdc`
- **Cons:** 非 alwaysApply；无 Idempotency-Key；Agent 易漏
- **Why rejected:** 不足以防止超时重放与首帧连点

### Alternative 2: Server-only idempotency, no frontend rule

- **Pros:** 单一权威在后端
- **Cons:** 双请求仍打满网关/计费；用户看到两次 loading/错误；NFR「双击」触发源在前端未落地
- **Why rejected:** 前后端分层：前端减重放，后端兜底双写

### Alternative 3: Disable the button only (no Idempotency-Key)

- **Pros:** 实现短
- **Cons:** `disabled` 非同步；超时重试仍双写
- **Why rejected:** 必须 L1+L2 同时存在

## Consequences

### Positive

- 新写按钮有可检查的共享入口与 CI 工件
- 与 NFR「双击」触发源对齐：键从点击意图产生，而不是每次 fetch 现场发明
- 建单参考路径（`OrderCreate.vue`）可作样板

### Negative / Trade-offs

- 存量副作用按钮未一次改完，需渐进
- 后端尚未兑现 `Idempotency-Key` 的接口仍可能双写（前端只能减流量）
- 误标 `Anti-Replay-OK` 会漏检

### Mitigations

- CI 校验规则文件与 helper 存在；`--strict --files` 扫描新改写 Vue
- NFR 表要求写清前端如何产生键；资金路径仍强制服务端 ≥ L3
- 存量改造记入 OPT

## References

- `.ai/01_project_constraints/57_frontend_button_anti_replay.md`
- `.ai/01_project_constraints/53_nfr_idempotency.md`（元规则 48）
- `.ai/01_project_constraints/54_event_consumer_idempotency.md`（元规则 49）
- [ADR-0015: 领域事件消费幂等](0015-event-consumer-idempotency.md)
