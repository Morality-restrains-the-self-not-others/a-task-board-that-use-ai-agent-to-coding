# 意图：评论运行态以容器推送为真源，禁止前后端轮询

## 背景与目标

`OPT-20260816-018` 要把 Describe 后台轮询扩到全部 live 评论。用户纠正：直播应由**容器/启动工作流主动推送**，前端只 apply 下发字段；对账只通过「刷新状态」按钮。前端轮询会卡死主线程；服务端轮询云再转推同样泄漏。

## 范围与边界

- 范围内：删除 runtime Describe 定时器、挂载自动 GET、SSE 后自动 GET、`serverStartupStatusPoll`、binding 30s timer；SSE payload 带 `comment_id` + `runtime_status` 时只 apply。
- 范围外：运维 reconcile ticker；启动命令内有界等公网 IP（须在就绪时 push，不驱动前端 timer）；不新增 HTTP。

## 约束与风险

- 禁止 `forPoll()` 作为直播 id。
- 推送字段不足时修发布点，不恢复自动 Describe。
- 「刷新状态」必须带该面板 `comment_id`。

## 验收标准

1. 无按钮点击时无周期性 runtime-status / startup-status。
2. 心跳与启动成功 SSE 更新该评论面板且不跟 GET Describe。
3. OPT-018 取消。

## 实施计划

见 `docs/superpowers/specs/2026-08-16-comment-runtime-no-background-poll-design.md`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 容器心跳 | container_heartbeat | SSE | 容器 → taskSSE | 前端 apply | 不新增 MQ |
| 启动/停止完成 | 现有启动 SSE | SSE | taskCloudService → taskSSE | 前端 apply runtime_status | 禁止随后自动 Describe |
| 用户刷新状态 | — | — | 按钮 | GET server-runtime-status | 纯查询、一次性 |
| 删除 UI 轮询 | — | — | — | — | 纯前端/定时器删除 |

## 前端 API 轮询 allowlist（OPT-20260816-024）

业务服务的「无操作无 SSE 后台 API 轮询」一律禁止；以下为既有例外，**只减不增**（复制前须经评审）：

| 位置 | 轮询对象 | 触发源 / 收敛机制 |
|------|---------|------------------|
| `useWorkPanelMachineSummary` | 工作面板机器摘要 | 仅工作面板挂载后启动；`document.visibilityState=hidden` 即停、回前台刷新一次恢复（OPT-20260808-021） |
| `useServerConfigRelayToTrae` | relay-to-trae 服务状态 | 启动/重启服务时的 bootstrap 轮询，服务就绪或达上限即自停 |
| `useBillingOrderActions` payPollTimer | 支付结果 | 用户点支付后的有界轮询，成功/超时自停 |
| `useServerConfigRuntime` runtimeUptime | 运行时长展示 | 纯本地时钟，不打 API |
| 验证码/倒计时 timer（PhoneVerificationGate / PhoneCodeLogin / PhoneRegister 等） | — | 纯倒计时，不打 API |
