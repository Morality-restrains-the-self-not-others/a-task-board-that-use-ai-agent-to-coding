# ADR-0031: 指令闲置回收 — 平台释放优先，STS 不进评论

- **Status:** accepted
- **Date:** 2026-08-22
- **Author:** Trae AI 团队
- **Deciders:** 总体设计已批准（2026-08-22）

---

## Context

工作空间已有 `idle_recycle_minutes`，但现网只在容器 **unregister**（`server_url` 清空）后把机器标闲置，由 taskEvents timer 调 stop-vm。容器仍注册、仅「指令已跑完」时不会回收。

用户要求：容器在每条指令结束后按该字段倒计时；到期无新指令则释放机器；并评估把阿里云 STS 随评论交给容器直连释放。同时提出：容器可能连不上平台；指令结果可能提交失败。

CPA 目前是长期 AccessKey，容器从未持有阿里云密钥。`request-machine-release` 与 `CLOUD_SERVER_STOPPED` 已是释放主权路径。

## Decision

**We will:**

1. 把「指令闲置」定义为：本评论容器上一条指令已**成功交付平台**且无新指令。倒计时从交付成功开始，不从 job 进程退出或指令下发开始。
2. 容器 `task-detail` 返回工作空间 `idle_recycle_minutes`。STS **不**写入评论 / `context_pack` / 用户可见 SSE。
3. 释放顺序：容器 `request-machine-release` → 平台 timer 读 `instruction_idle_since`（CPA 密钥）→ 仅当交付已成功且 L1 因平台不可达失败、且 CPA 配置了回收专用 RAM Role 时，才用短时 STS 直连 `DeleteInstance`（Resource 仅本实例）。
4. 交付失败时禁止任何释放路径（含 STS）。指令执行中不因短时心跳失败按 N 分钟拆机。
5. 不恢复跨任务闲置复用（ADR-0013）。

设计全文：`docs/superpowers/specs/2026-08-22-instruction-idle-recycle-design.md`。

## Alternatives Considered

### Alternative 1: STS 随每条评论下发，容器直连阿里云

- **Pros:** 平台宕机时容器仍能拆机
- **Cons:** 评论/日志/Kafka 可能落临时密钥；权限难限制到单实例；交付失败时易误拆丢结果
- **Why rejected:** 密钥面过大；与「提交失败不得拆机」冲突

### Alternative 2: 仅容器 `request-machine-release`，无平台 instruction_idle 列

- **Pros:** 实现面小
- **Cons:** 交付成功后容器崩溃或再也连不上平台时，机器永不回收（`server_url` 仍在，现网卸载闲置不触发）
- **Why rejected:** 无法回答「连不上平台」顾虑

### Alternative 3: GetSessionToken（与 CPA 同等权限的会话密钥）塞进 task-detail

- **Pros:** 无需 RAM Role
- **Cons:** 容器获得与租户云账号几乎同等的 ECS 权限
- **Why rejected:** 不可接受

## Consequences

### Positive

- 指令跑完后按工作空间策略收机，费用可控
- 结果先上平台再允许拆机
- 密钥不进评论；STS 为可选窄权限兜底

### Negative / Trade-offs

- 需配置 RAM Role 才有 L3 STS；未配置时依赖 L1/L2
- 执行中长时间完全断网不会按 N 分钟拆机（保护未交付结果）
- Credential 须 internal 读 Cloud policy（禁止直连 `task_cloud`）

### Mitigations

- L2 `instruction_idle_since` 在交付成功时由平台落库，不依赖容器稍后还活着
- Session policy 锁死单实例 DeleteInstance
- `idle_recycle_minutes=0` 关闭指令闲置回收

## References

- [ADR-0013](0013-remove-idle-machine-reuse.md)
- [ADR-0011](0011-no-service-internal-poll-loop.md)（SaaS 业务进程禁 ticker；回收仍走 taskEvents timer）
- `docs/skills/saas-container/saas-machine-container.md` `request-machine-release`
