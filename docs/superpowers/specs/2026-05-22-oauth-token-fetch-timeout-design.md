# OAuth Token Fetch Timeout Design

## Context

当前链路为：UI -> `onlineServiceJS` -> `task2app` -> `gitOauth` -> GitHub OAuth。  
故障表现为 `onlineServiceJS` 返回 502，`detail` 为 `This operation was aborted`，难以快速定位卡点。

## Goals

- 让超时故障可定位到明确阶段，而不是统一落成 `aborted`。
- 让返回错误可解释（结构化错误码、失败阶段、可重试标记）。
- 在不引入大规模改造的前提下，优先交付可观测和可恢复能力。

## Non-goals

- 本阶段不引入完整异步任务队列与前端轮询改造。
- 本阶段不变更 GitHub OAuth 协议或授权模型。

## Proposed Design

### Phase 1: Observability + Structured Errors

- 在 `task2app` 的 `layer-github-oauth-access-tokens` 视图链路增加分段观测：
  - `entry`
  - `token_check`
  - `binding_check`
  - `gitoauth_summary`
  - `gitoauth_access`
  - `exit`
- 统一错误载荷（`task2app` -> `onlineServiceJS` -> UI）：
  - `error_code`
  - `failed_stage`
  - `retryable`
  - `detail_safe`
- `onlineServiceJS` 保留总超时保护，但将中断错误映射为结构化错误。

### Phase 2: Timeout and Retry Policy

- 按阶段设置更细超时（connect/read）和幂等重试策略。
- 支持部分成功返回，避免多仓场景全量失败。
- 引入短时缓存减少重复换票压力。

### Phase 3: Async Evolution (Future)

- 将换票流程改造成异步任务，前端基于 job 轮询或 SSE 获取结果。

## Domain Model

### Bounded Contexts

- Container Gateway Context (`onlineServiceJS`)
- Task Orchestration Context (`task2app`)
- OAuth Token Broker Context (`gitOauth`)
- External Provider Context (GitHub OAuth)

### Entities / Aggregates

- `ContainerRuntimeRequest`
- `TaskGithubRepoOauthBinding`
- `OAuthTokenResolution`
- `TokenExchangeAttempt`

### Domain Events

- `TokenFetchRequested`
- `BindingResolved`
- `BindingMissing`
- `TokenExchangeStarted`
- `TokenExchangeTimedOut`
- `TokenFetchPartiallySucceeded`
- `TokenFetchCompleted`
- `TokenFetchFailed`
