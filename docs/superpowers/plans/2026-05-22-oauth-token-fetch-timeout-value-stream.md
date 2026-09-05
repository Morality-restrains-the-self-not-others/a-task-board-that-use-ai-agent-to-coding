# Value Stream: OAuth Token Fetch Timeout Governance

> Derived from design: `docs/superpowers/specs/2026-05-22-oauth-token-fetch-timeout-design.md`

## Value Summary

当用户在任务页面拉取仓库 AccessToken 失败时，系统可以快速返回可解释错误并定位卡点，减少盲排与重复重试成本。

## End-to-End Flow

用户点击「拉取 AccessToken」 -> `onlineServiceJS` 组装仓库请求 -> `task2app` 鉴权并解析绑定 -> `gitOauth` 换票 -> `task2app` 返回按仓库 token 映射 -> `onlineServiceJS` 落盘并返回结果给用户。

## Value Stage Classification

- **Core value**
  - 结构化错误回传（用户和开发都能知道失败原因与阶段）
  - 分段观测（可直接定位卡在绑定、summary、access、或外部 provider）
- **Essential support**
  - 统一 trace_id 贯穿跨服务日志
  - 错误脱敏与安全输出
- **Enhancement**
  - 多仓部分成功（partial success）
  - 重试与短时缓存策略
- **Future**
  - 全链路异步化（job 化）

## Wait / Dependency Points

- `onlineServiceJS` 等待 `task2app` 返回（总超时保护点）
- `task2app` 等待 `gitOauth` `summary-for-user` / `access-for-user`
- `gitOauth` 等待 GitHub OAuth 响应（最不稳定依赖点）

## Delivery Point

用户在 UI 收到明确结果：成功写入 token 文件，或失败但含 `error_code`、`failed_stage`、`retryable` 的可执行反馈。

## Value Increments

### Increment 1: End-to-End Error Contract (Thin Slice)

**Value to user:** 失败不再是裸 `aborted`，能看到可解释错误。  
**Scope:** `onlineServiceJS` + `task2app` 建立最小结构化错误契约并打通到 UI 响应。  
**Depends on:** nothing.

### Increment 2: Stage-level Observability (Core Value)

**Value to user:** 故障定位速度显著提升，减少重复点击与长时间等待。  
**Scope:** `task2app` 增加分段日志与阶段耗时；`onlineServiceJS` 记录上游阶段信息。  
**Depends on:** Increment 1.

### Increment 3: Safe Timeout/Retry Strategy (Essential Support)

**Value to user:** 超时场景更稳定，失败可重试且副作用可控。  
**Scope:** 为 `summary/access` 配置 connect/read 超时与有限重试，保持幂等边界。  
**Depends on:** Increment 2.

### Increment 4: Partial Success for Multi-repo (Enhancement)

**Value to user:** 多仓场景下成功仓库可先用，减少全量失败。  
**Scope:** 返回 `token_files` + `partial_error` 的可消费结构，前端展示部分成功。  
**Depends on:** Increment 3.

### Increment 5: Async Job-based Token Fetch (Future)

**Value to user:** 消除同步长等待，体验更平滑。  
**Scope:** 改造为任务化执行与轮询/SSE 回传。  
**Depends on:** Increment 4.
